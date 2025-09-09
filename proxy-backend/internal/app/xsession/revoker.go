package xsession

import (
	"context"
	"errors"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"v0/internal/app/security"
)

type Revoker struct {
	mu            sync.RWMutex
	logoutAfter   map[string]map[string]connEntry
	sessionDriver *RedisSessionManager
	log           zerolog.Logger
	sessionCookie string
}

func NewRevoker(s *RedisSessionManager, log zerolog.Logger, sessionCookie string) *Revoker {
	data := make(map[string]map[string]connEntry)
	return &Revoker{logoutAfter: data, sessionDriver: s, log: log, sessionCookie: sessionCookie}
}

func (r *Revoker) AddRevokeUser(userID string, entry map[string]connEntry) {
	r.mu.Lock()
	r.logoutAfter[userID] = entry
	r.mu.Unlock()
}

func (r *Revoker) ShouldLogout(userID string) bool {
	r.mu.RLock()
	_, ok := r.logoutAfter[userID]
	r.mu.RUnlock()
	return ok
}

func (r *Revoker) DeleteRevokeUser(userID string) {
	r.mu.RLock()
	delete(r.logoutAfter, userID)
	r.mu.RUnlock()
}

func (r *Revoker) ConsumeIfRevokedWithLoggedIn(c echo.Context, withTLS bool, isLoggedIn func(echo.Context) (string, string, []string, bool, error)) (string, string, []string, bool, error) {
	r.mu.Lock()
	username, sessionID, groups, ok, err := isLoggedIn(c)
	_, shouldLogout := r.logoutAfter[username]
	if shouldLogout {
		sessionCookie, err := c.Cookie(r.sessionCookie)
		if err != nil {
			return "", "", []string{}, false, err
		}
		sessionID := sessionCookie.Value
		ctx := context.Background()
		username, err := r.sessionDriver.LookupUserBySID(ctx, sessionID)
		if err != nil {
			return "", "", []string{}, false, err
		}

		if err := r.sessionDriver.RevokeByUserIDAndSessionID(ctx, username, sessionID); err != nil {
			return "", "", []string{}, false, err
		}

		security.ExpireSessionCookie(c, withTLS, r.sessionCookie)
		for _, v := range r.logoutAfter[username] {
			v.cancel()
		}
		delete(r.logoutAfter, username)
		ok = false
		err = errors.New("user revoked")
	}
	r.mu.Unlock()
	return username, sessionID, groups, ok, err
}
