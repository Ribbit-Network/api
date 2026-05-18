// Package ratelimit provides per-API-key rate limiting middleware.
package ratelimit

import (
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

// Limiter holds per-key token-bucket limiters.
type Limiter struct {
	mu      sync.Mutex
	entries map[string]*rate.Limiter
	r       rate.Limit
	b       int
}

// New creates a Limiter allowing r tokens per second with a burst of b.
func New(r rate.Limit, b int) *Limiter {
	return &Limiter{
		entries: make(map[string]*rate.Limiter),
		r:       r,
		b:       b,
	}
}

// Middleware returns HTTP middleware that rate-limits by API key.
// Keys are read from "Authorization: Bearer <key>" or "X-API-Key: <key>",
// matching the auth middleware extraction logic. Requests with no key
// are passed through — the auth middleware upstream handles rejection.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := extractKey(r)
		if key != "" && !l.get(key).Allow() {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) get(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.entries[key]
	if !ok {
		lim = rate.NewLimiter(l.r, l.b)
		l.entries[key] = lim
	}
	return lim
}

func extractKey(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if rest, ok := strings.CutPrefix(h, "Bearer "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return strings.TrimSpace(r.Header.Get("X-API-Key"))
}
