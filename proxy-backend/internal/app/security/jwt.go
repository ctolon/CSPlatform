package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

const (
	ErrJWTClaimUsernameMissing = "JWT_ERR_CODE: 001"
	ErrJWTClaimIPAddrMissing   = "JWT_ERR_CODE: 002"
	ErrJWTClaimUAMissing       = "JWT_ERR_CODE: 003"
	ErrJWTClaimIpAddrMissMatch = "JWT_ERR_CODE: 004"
	ErrJWTClaimUAMissMatch     = "JWT_ERR_CODE: 005"
	ErrJWTClaimGroupsMissing   = "JWT_ERR_CODE: 006"
)

const jwtLeeway = 30 * time.Second

type JWTService struct {
	accessSecret  []byte
	refreshSecret []byte
	issuer        string
	audience      string
	log           zerolog.Logger
}

func NewJWTService(accessSecret string, refreshSecret string, issuer string, audience string, log zerolog.Logger) *JWTService {
	return &JWTService{[]byte(accessSecret), []byte(refreshSecret), issuer, audience, log}
}

func (s *JWTService) JWTCreateAccessToken(username string, groups []string, ipAddr string, userAgent string) (string, error) {
	ct := time.Now().Unix()
	claims := jwt.MapClaims{
		"username": username,
		"groups":   groups,
		"ip":       ipAddr,
		"ua":       userAgent,
		// Standard fields
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": ct,
		"nbf": ct,
		"sub": username,
		"iss": s.issuer,
		"aud": s.audience,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.accessSecret)
}

func (s *JWTService) JWTCreateRefreshToken(username string, groups []string, ipAddr string, userAgent string) (string, error) {
	ct := time.Now().Unix()
	claims := jwt.MapClaims{
		"username": username,
		"groups":   groups,
		"ip":       ipAddr,
		"ua":       userAgent,
		// Standard fields
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat": ct,
		"nbf": ct,
		"sub": username,
		"iss": s.issuer,
		"aud": s.audience,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.refreshSecret)
}

func (s *JWTService) JWTValidateAccessToken(tokenString, ipAddr, userAgent string) (string, []string, error) {
	now := time.Now()
	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.accessSecret, nil
	})
	if err != nil {
		return "", []string{}, err
	}

	// token encode validation
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", []string{}, fmt.Errorf("invalid token")
	}

	// exp validation
	if expRaw, ok := claims["exp"]; ok {
		exp := s.toTime(expRaw)
		if now.After(exp.Add(jwtLeeway)) {
			return "", []string{}, jwt.ErrTokenExpired
		}
	} else {
		return "", []string{}, jwt.ErrTokenExpired
	}

	// nbf validation
	if nbfRaw, ok := claims["nbf"]; ok {
		nbf := s.toTime(nbfRaw)
		if now.Add(jwtLeeway).Before(nbf) {
			return "", []string{}, jwt.ErrTokenNotValidYet
		}
	} else {
		return "", []string{}, jwt.ErrTokenNotValidYet
	}

	// username
	username, _ := claims["username"].(string)
	if username == "" {
		return "", []string{}, fmt.Errorf(ErrJWTClaimUsernameMissing)
	}

	// groups
	groups, err := s.ClaimToStringSlice(claims["groups"])
	if err != nil {
		return "", []string{}, fmt.Errorf(ErrJWTClaimGroupsMissing)
	}

	// IP Addr
	claimIP, _ := claims["ip"].(string)
	if claimIP == "" {
		return "", []string{}, fmt.Errorf(ErrJWTClaimIPAddrMissing)
	}
	if !s.ipEqual(claimIP, ipAddr) {
		return "", []string{}, fmt.Errorf(ErrJWTClaimIpAddrMissMatch)
	}

	// User-Agent
	claimUA, _ := claims["ua"].(string)
	if claimUA == "" {
		return "", []string{}, fmt.Errorf(ErrJWTClaimUAMissing)
	}
	if claimUA != userAgent {
		return "", []string{}, fmt.Errorf(ErrJWTClaimUAMissMatch)
	}

	return username, groups, nil
}

func (s *JWTService) JWTValidateRefreshToken(tokenString, ipAddr, userAgent string) (string, []string, error) {
	now := time.Now()
	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.refreshSecret, nil
	})
	if err != nil {
		return "", []string{}, err
	}

	// token encode validation
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", []string{}, fmt.Errorf("invalid token")
	}

	// exp validation
	if expRaw, ok := claims["exp"]; ok {
		exp := s.toTime(expRaw)
		if now.After(exp.Add(jwtLeeway)) {
			return "", []string{}, jwt.ErrTokenExpired
		}
	} else {
		return "", []string{}, jwt.ErrTokenExpired
	}

	// nbf validation
	if nbfRaw, ok := claims["nbf"]; ok {
		nbf := s.toTime(nbfRaw)
		if now.Add(jwtLeeway).Before(nbf) {
			return "", []string{}, jwt.ErrTokenNotValidYet
		}
	} else {
		return "", []string{}, jwt.ErrTokenNotValidYet
	}

	// username
	username, _ := claims["username"].(string)
	if username == "" {
		return "", []string{}, fmt.Errorf(ErrJWTClaimUsernameMissing)
	}

	// groups
	groups, err := s.ClaimToStringSlice(claims["groups"])
	if err != nil {
		return "", []string{}, fmt.Errorf(ErrJWTClaimGroupsMissing)
	}

	// IP Addr
	claimIP, _ := claims["ip"].(string)
	if claimIP == "" {
		return "", []string{}, fmt.Errorf(ErrJWTClaimIPAddrMissing)
	}
	if !s.ipEqual(claimIP, ipAddr) {
		return "", []string{}, fmt.Errorf(ErrJWTClaimIpAddrMissMatch)
	}

	// User-Agent
	claimUA, _ := claims["ua"].(string)
	if claimUA == "" {
		return "", []string{}, fmt.Errorf(ErrJWTClaimUAMissing)
	}
	if claimUA != userAgent {
		return "", []string{}, fmt.Errorf(ErrJWTClaimUAMissMatch)
	}

	return username, groups, nil
}

func (s *JWTService) ipEqual(a, b string) bool {
	na := s.normalizeIP(a)
	nb := s.normalizeIP(b)
	if na == "" || nb == "" {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}
	return na == nb
}

func (s *JWTService) normalizeIP(ipAddr string) string {
	ipAddr = strings.TrimSpace(ipAddr)
	if host, _, err := net.SplitHostPort(ipAddr); err == nil {
		ipAddr = host
	}
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return ""
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.String()
}

func (s *JWTService) ClaimToStringSlice(v any) ([]string, error) {
	switch vv := v.(type) {
	case nil:
		return nil, errors.New("missing groups claim")
	case []string:
		out := make([]string, 0, len(vv))
		for _, s := range vv {
			if t := strings.TrimSpace(s); t != "" {
				out = append(out, t)
			}
		}
		return out, nil
	case []any:
		out := make([]string, 0, len(vv))
		for _, x := range vv {
			s, ok := x.(string)
			if !ok {
				return nil, errors.New("groups contains non-string")
			}
			if t := strings.TrimSpace(s); t != "" {
				out = append(out, t)
			}
		}
		return out, nil
	case string:
		s := strings.TrimSpace(vv)
		if s == "" {
			return nil, errors.New("empty groups string")
		}
		if strings.HasPrefix(s, "[") {
			var arr []string
			if err := json.Unmarshal([]byte(s), &arr); err == nil {
				for i := range arr {
					arr[i] = strings.TrimSpace(arr[i])
				}
				return arr, nil
			}
		}
		var parts []string
		if strings.Contains(s, ",") {
			parts = strings.Split(s, ",")
		} else {
			parts = strings.Fields(s)
		}
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				out = append(out, t)
			}
		}
		return out, nil
	default:
		return nil, errors.New("unsupported groups type")
	}
}

func (s *JWTService) toTime(v any) time.Time {
	switch t := v.(type) {
	case float64:
		return time.Unix(int64(t), 0)
	case int64:
		return time.Unix(t, 0)
	case json.Number:
		n, _ := t.Int64()
		return time.Unix(n, 0)
	default:
		return time.Time{}
	}
}
