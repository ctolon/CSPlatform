package xsession

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type connEntry struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type CodeServerSessionRegistry struct {
	mu         sync.Mutex
	cancels    map[string]map[string]connEntry
	transports map[string][]*http.Transport
	withTLS    bool
	revoker    *Revoker
	log        zerolog.Logger
}

func NewSessionRegistry(withTLS bool, revoker *Revoker, log zerolog.Logger) *CodeServerSessionRegistry {
	return &CodeServerSessionRegistry{
		cancels:    make(map[string]map[string]connEntry),
		transports: make(map[string][]*http.Transport),
		withTLS:    withTLS,
		revoker:    revoker,
		log:        log,
	}
}

func (r *CodeServerSessionRegistry) AddConn(sessionID, connID string, ctx context.Context, cancel context.CancelFunc, tr *http.Transport) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.cancels[sessionID]; !ok {
		r.cancels[sessionID] = make(map[string]connEntry)
	}
	r.cancels[sessionID][connID] = connEntry{ctx, cancel}
	if tr != nil {
		r.transports[sessionID] = append(r.transports[sessionID], tr)
	}
}

func (r *CodeServerSessionRegistry) RemoveConn(sessionID, connID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.cancels[sessionID]; ok {
		delete(m, connID)
		if len(m) == 0 {
			delete(r.cancels, sessionID)
			delete(r.transports, sessionID)
		}
	}
}
func (r *CodeServerSessionRegistry) CancellAll(sessionID string, addRevekoList bool) int {
	r.mu.Lock()
	cancels := r.cancels[sessionID]
	delete(r.cancels, sessionID)
	transports := r.transports[sessionID]
	delete(r.transports, sessionID)
	parts := strings.Split(sessionID, ":")
	uname := parts[1]
	if addRevekoList {
		r.revoker.AddRevokeUser(uname, cancels)
	}
	r.log.Info().Msgf("User Revoked: %s", uname)
	r.mu.Unlock()

	count := 0
	for _, cancel := range cancels {
		cancel.cancel()
		count++
	}
	for _, tr := range transports {
		tr.CloseIdleConnections()
	}
	return count
}

func (r *CodeServerSessionRegistry) CancelConn(sessionID, connID string) bool {
	r.mu.Lock()
	m, ok := r.cancels[sessionID]
	if !ok {
		r.mu.Unlock()
		return false
	}
	cancel, ok := m[connID]
	if !ok {
		r.mu.Unlock()
		return false
	}
	empty := len(m) == 0
	var transports []*http.Transport
	if empty {
		delete(r.cancels, sessionID)
		transports = r.transports[sessionID]
		delete(r.transports, sessionID)
	}
	r.mu.Unlock()

	cancel.cancel()
	if empty {
		for _, tr := range transports {
			tr.CloseIdleConnections()
		}
	}
	return true
}

func (r *CodeServerSessionRegistry) ListConns(sessionID string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var conns []string
	if m, ok := r.cancels[sessionID]; ok {
		for connID := range m {
			conns = append(conns, connID)
		}
	}
	return conns
}

func (r *CodeServerSessionRegistry) ListSessions() map[string]int {
	out := make(map[string]int)
	r.mu.Lock()
	defer r.mu.Unlock()
	for sid, m := range r.cancels {
		out[sid] = len(m)
	}
	return out
}

func (r *CodeServerSessionRegistry) CloseIdle(sessionID string) bool {
	r.mu.Lock()
	transports, ok := r.transports[sessionID]
	r.mu.Unlock()
	if !ok {
		return false
	}
	for _, tr := range transports {
		tr.CloseIdleConnections()
	}
	return true
}

func (r *CodeServerSessionRegistry) SweepClose() (removed int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for sid, m := range r.cancels {
		for cid, e := range m {
			if e.ctx.Err() != nil {
				delete(m, cid)
				removed++
			}
		}
		if len(m) == 0 {
			delete(r.cancels, sid)
			delete(r.transports, sid)
		}
	}
	return
}

func (r *CodeServerSessionRegistry) StartJanitor(parent context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-t.C:
				_ = r.SweepClose()
			case <-parent.Done():
				return
			}
		}
	}()
}
