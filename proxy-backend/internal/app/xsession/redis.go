package xsession

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type RedisSessionManager struct {
	rdb    *redis.Client
	prefix string
	aead   cipher.AEAD
	log    zerolog.Logger
	macKey []byte
}

func NewRedisClient(opt *redis.Options) (*redis.Client, error) {
	if opt == nil {
		return nil, fmt.Errorf("redis options required")
	}

	if opt.MaxRetries == 0 {
		opt.MaxRetries = 5
	}

	if opt.MinRetryBackoff == 0 {
		opt.MinRetryBackoff = 100 * time.Millisecond
	}

	if opt.MaxRetryBackoff == 0 {
		opt.MaxRetryBackoff = 2 * time.Second
	}

	if opt.DialTimeout == 0 {
		opt.DialTimeout = 3 * time.Second
	}

	if opt.WriteTimeout == 0 {
		opt.WriteTimeout = 4 * time.Second
	}

	if opt.OnConnect == nil {
		opt.OnConnect = func(ctx context.Context, cn *redis.Conn) error {
			return cn.Ping(ctx).Err()
		}
	}

	rdb := redis.NewClient(opt)
	return rdb, nil
}

// NewRedisSessionManager sets up go-redis client with retry/backoff and validates connectivity.
func NewRedisSessionManager(rdb *redis.Client, prefix string, encKey []byte, log zerolog.Logger) (*RedisSessionManager, error) {
	l := len(encKey)
	if l != 32 && l != 64 {
		return nil, fmt.Errorf("encKey must be 32 or 64 bytes (got %d)", l)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis unavailable: %w", err)
	}
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	mk := sha256.Sum256(append(encKey, byte(1)))
	macKey := make([]byte, len(mk))
	copy(macKey, mk[:])

	return &RedisSessionManager{
		rdb:    rdb,
		prefix: strings.TrimSuffix(prefix, ":"),
		aead:   aead,
		macKey: macKey,
		log:    log,
	}, nil
}

func (m *RedisSessionManager) NewSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (m *RedisSessionManager) key(parts ...string) string {
	return m.prefix + ":" + strings.Join(parts, ":")
}

// Encrpyts plain text using AES-GCM; returns base64(nonce|ciphertext).
func (m *RedisSessionManager) encrpyt(plain string) (string, error) {
	nonce := make([]byte, m.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := m.aead.Seal(nil, nonce, []byte(plain), nil)
	out := append(nonce, ct...)
	return base64.RawStdEncoding.EncodeToString(out), nil
}

func (m *RedisSessionManager) decrypt(b64 string) (string, error) {
	raw, err := base64.RawStdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	ns := m.aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}

	nonce, ct := raw[:ns], raw[ns:]
	pt, err := m.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func (m *RedisSessionManager) tokenMAC(token string) string {
	h := hmac.New(sha256.New, m.macKey)
	h.Write([]byte(token))
	return base64.RawStdEncoding.EncodeToString(h.Sum(nil))
}

func (m *RedisSessionManager) parseUserSID(pair string) (string, string, error) {
	parts := strings.SplitAfterN(pair, "|", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("corrupt index value")
	}
	return parts[0], parts[1], nil
}

// SaveSessionsTokens:
// acc:<user>:<sid> H{access_token: <enc>}      EX accessTTL
// ref:<user>:<sid> H{refresh_token <enc>}      EX refreshTTL
// usessions:<user> SADD <sid>                  EX refreshTTL
// idx:sid:<sid> -> <user>                      EX refreshTTL (sid->user index)
// idx:accsha:<mac(access)>  -> "<user>|<sid>"  EX accessTTL
// idx:refsha:<mac(refresh)> -> "<user>|<sid>"  EX refreshTTL
func (m *RedisSessionManager) SaveSessionTokens(
	ctx context.Context,
	userID, sessionID, accessToken, refreshToken string,
	accessTTL, refreshTTL time.Duration,
) error {
	if userID == "" || sessionID == "" || accessToken == "" || refreshToken == "" {
		return fmt.Errorf("userID, sessionID and tokens required")
	}
	encA, err := m.encrpyt(accessToken)
	if err != nil {
		return err
	}
	encR, err := m.encrpyt(refreshToken)
	if err != nil {
		return err
	}
	accMac := m.tokenMAC(accessToken)
	refMac := m.tokenMAC(refreshToken)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pipe := m.rdb.Pipeline()
	pipe.HSet(ctx, m.key("acc", userID, sessionID), "access_token", encA)
	pipe.Expire(ctx, m.key("acc", userID, sessionID), accessTTL)

	pipe.HSet(ctx, m.key("ref", userID, sessionID), "refresh_token", encR)
	pipe.Expire(ctx, m.key("ref", userID, sessionID), refreshTTL)

	pipe.SAdd(ctx, m.key("usessions", userID), sessionID)
	pipe.Expire(ctx, m.key("usessions", userID), refreshTTL)

	pipe.Set(ctx, m.key("idx", "sid", sessionID), userID, refreshTTL) // sid -> user index
	pipe.Set(ctx, m.key("idx", "accsha", accMac), fmt.Sprintf("%s|%s", userID, sessionID), accessTTL)
	pipe.Set(ctx, m.key("idx", "refsha", refMac), fmt.Sprintf("%s|%s", userID, sessionID), refreshTTL)

	_, err = pipe.Exec(ctx)
	return err
}

// Get Access Token By SessionID
func (m *RedisSessionManager) GetAccessBySID(ctx context.Context, userID, sessionID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	val, err := m.rdb.HGet(ctx, m.key("acc", userID, sessionID), "access_token").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("not found")
		}
		return "", err
	}
	return m.decrypt(val)
}

// Get Refresh Token By SessionID
func (m *RedisSessionManager) GetRefreshBySID(ctx context.Context, userID, sessionID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	val, err := m.rdb.HGet(ctx, m.key("ref", userID, sessionID), "refresh_token").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("not found")
		}
		return "", err
	}
	return m.decrypt(val)
}

// Get Username by SessionID
func (m *RedisSessionManager) LookupUserBySID(ctx context.Context, sessionID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5000*time.Millisecond)
	defer cancel()
	u, err := m.rdb.Get(ctx, m.key("idx", "sid", sessionID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("not found")
		}
		return "", err
	}
	return u, nil
}

// Update/Rotate Access Token By SessionID
func (m *RedisSessionManager) UpdateAccessBySID(ctx context.Context, userID, sessionID, newAccess string, accessTTL time.Duration) error {
	encA, err := m.encrpyt(newAccess)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	oldEnc, _ := m.rdb.HGet(ctx, m.key("acc", userID, sessionID), "access_token").Result()
	if oldEnc != "" {
		if oldPlain, err := m.decrypt(oldEnc); err == nil {
			oldMac := m.tokenMAC(oldPlain)
			_ = m.rdb.Del(ctx, m.key("idx", "accsha", oldMac)).Err()
		}
	}

	pipe := m.rdb.Pipeline()
	pipe.HSet(ctx, m.key("acc", userID, sessionID), "access_token", encA)
	pipe.Expire(ctx, m.key("acc", userID, sessionID), accessTTL)
	pipe.Set(ctx, m.key("idx", "accsha", m.tokenMAC(newAccess)), fmt.Sprintf("%s|%s", userID, sessionID), accessTTL)
	_, err = pipe.Exec(ctx)
	return err
}

// Update/Rotate Refresh Token By SessionID
func (m *RedisSessionManager) UpdateRefreshBySID(ctx context.Context, userID, sessionID, newRefresh string, refreshTTL time.Duration) error {
	encR, err := m.encrpyt(newRefresh)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	oldEnc, _ := m.rdb.HGet(ctx, m.key("ref", userID, sessionID), "refresh_token").Result()
	if oldEnc != "" {
		if oldPlain, err := m.decrypt(oldEnc); err == nil {
			oldMac := m.tokenMAC(oldPlain)
			_ = m.rdb.Del(ctx, m.key("idx", "refsha", oldMac)).Err()
		}
	}

	pipe := m.rdb.Pipeline()
	pipe.HSet(ctx, m.key("ref", userID, sessionID), "refresh_token", encR)
	pipe.Expire(ctx, m.key("ref", userID, sessionID), refreshTTL)
	pipe.Set(ctx, m.key("idx", "refsha", m.tokenMAC(newRefresh)), fmt.Sprintf("%s|%s", userID, sessionID), refreshTTL)
	_, err = pipe.Exec(ctx)
	return err
}

// Refresh Token Rotation (One-Time-Use)
// - If Refresh Token input true, it asummes new refresh and access token generated and updated as atomic
// - If Refresh Token input false, session will be revoked (theft detection).
func (m *RedisSessionManager) RotateOnRefresh(
	ctx context.Context,
	userID string,
	sessionID string,
	providedRefresh string,
	newAccess string, accessTTL time.Duration,
	newRefresh string, refreshTTL time.Duration,
) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// sid -> user
	if userID == "" {
		userIDLookup, err := m.LookupUserBySID(ctx, sessionID)
		if err != nil {
			return err
		}
		userID = userIDLookup
	}

	refEnc, err := m.rdb.HGet(ctx, m.key("ref", userID, sessionID), "refresh_token").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fmt.Errorf("refresh not found")
		}
		return err
	}
	storedRefresh, err := m.decrypt(refEnc)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare([]byte(providedRefresh), []byte(storedRefresh)) != 1 {
		// cancel session due to theft
		_ = m.RevokeBySessionID(ctx, sessionID)
		return fmt.Errorf("invalid refresh; session revoked")
	}

	oldAccEnc, _ := m.rdb.HGet(ctx, m.key("acc", userID, sessionID), "access_token").Result()
	var oldAccMac string
	if oldAccEnc != "" {
		if oldAccPlain, err := m.decrypt(oldAccEnc); err == nil {
			oldAccMac = m.tokenMAC(oldAccPlain)
		}
	}
	oldRefMac := m.tokenMAC(storedRefresh)

	encA, err := m.encrpyt(newAccess)
	if err != nil {
		return err
	}
	encR, err := m.encrpyt(newRefresh)
	if err != nil {
		return err
	}

	pipe := m.rdb.Pipeline()
	if oldAccMac != "" {
		pipe.Del(ctx, m.key("idx", "accsha", oldAccMac))
	}
	pipe.Del(ctx, m.key("idx", "refsha", oldRefMac))

	pipe.HSet(ctx, m.key("acc", userID, sessionID), "access_token", encA)
	pipe.Expire(ctx, m.key("acc", userID, sessionID), accessTTL)
	pipe.HSet(ctx, m.key("ref", userID, sessionID), "refresh_token", encR)
	pipe.Expire(ctx, m.key("ref", userID, sessionID), refreshTTL)

	pipe.Set(ctx, m.key("idx", "accsha", m.tokenMAC(newAccess)), fmt.Sprintf("%s|%s", userID, sessionID), accessTTL)
	pipe.Set(ctx, m.key("idx", "refsha", m.tokenMAC(newRefresh)), fmt.Sprintf("%s|%s", userID, sessionID), refreshTTL)

	_, err = pipe.Exec(ctx)
	return err
}

// Revoke By UserID and SessionID
func (m *RedisSessionManager) RevokeByUserIDAndSessionID(ctx context.Context, userID, sessionID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	accEnc, _ := m.rdb.HGet(ctx, m.key("acc", userID, sessionID), "access_token").Result()
	refEnc, _ := m.rdb.HGet(ctx, m.key("ref", userID, sessionID), "refresh_token").Result()
	var accMac, refMac string
	if accEnc != "" {
		if accPlain, err := m.decrypt(accEnc); err == nil {
			accMac = m.tokenMAC(accPlain)
		}
	}
	if refEnc != "" {
		if refPlain, err := m.decrypt(refEnc); err == nil {
			refMac = m.tokenMAC(refPlain)
		}
	}

	pipe := m.rdb.Pipeline()
	pipe.Del(ctx, m.key("acc", userID, sessionID))
	pipe.Del(ctx, m.key("ref", userID, sessionID))
	pipe.SRem(ctx, m.key("usessions", userID), sessionID)
	if accMac != "" {
		pipe.Del(ctx, m.key("idx", "accsha", accMac))
	}
	if refMac != "" {
		pipe.Del(ctx, m.key("idx", "refsha", refMac))
	}
	_, err := pipe.Exec(ctx)
	return err
}

// Revoke By SessionID
func (m *RedisSessionManager) RevokeBySessionID(ctx context.Context, sessionID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	userID, err := m.LookupUserBySID(ctx, sessionID)
	if err != nil {
		return err
	}
	return m.RevokeByUserIDAndSessionID(ctx, userID, sessionID)

}

// Revoke By Access Token
func (m *RedisSessionManager) RevokeByAccessToken(ctx context.Context, accessToken string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pair, err := m.rdb.Get(ctx, m.key("idx", "accsha", m.tokenMAC(accessToken))).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fmt.Errorf("access token not found")
		}
		return err
	}
	userID, sid, err := m.parseUserSID(pair)
	if err != nil {
		return err
	}
	return m.RevokeByUserIDAndSessionID(ctx, userID, sid)
}

// Revoke By Refresh Token
func (m *RedisSessionManager) RevokeByRefreshToken(ctx context.Context, refreshToken string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pair, err := m.rdb.Get(ctx, m.key("idx", "refsha", m.tokenMAC(refreshToken))).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fmt.Errorf("refresh token not found")
		}
		return err
	}
	userID, sid, err := m.parseUserSID(pair)
	if err != nil {
		return err
	}
	return m.RevokeByUserIDAndSessionID(ctx, userID, sid)
}

// Revoke User's session by UserID
func (m *RedisSessionManager) RevokeAllByUserID(ctx context.Context, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	sids, err := m.rdb.SMembers(ctx, m.key("usessions", userID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	for _, sid := range sids {
		if err := m.RevokeByUserIDAndSessionID(ctx, userID, sid); err != nil {
			return err
		}
	}
	m.rdb.Del(ctx, m.key("usessions", userID))
	return nil
}

// Revoke All Sessions Globally
func (m *RedisSessionManager) RevokeAllSessions(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	iter := m.rdb.Scan(ctx, 0, m.key("idx", "sid", "*"), 1000).Iterator()
	for iter.Next(ctx) {
		key := iter.Val() // prefix:idx:sid:<sid>
		parts := strings.Split(key, ":")
		if len(parts) == 0 {
			continue
		}
		sid := parts[len(parts)-1]
		if err := m.RevokeBySessionID(ctx, sid); err != nil {
			return err
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return nil
}
