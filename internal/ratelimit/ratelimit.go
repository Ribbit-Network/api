// Package ratelimit provides per-API-key rate limiting middleware.
package ratelimit

import (
	"net/http"
	"sync"

	"github.com/Ribbit-Network/api/internal/auth"
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

// Middleware returns HTTP middleware that rate-limits by API key, reading the
// key from the request context (set by auth.Require). Requests without a key
// pass through — the auth middleware upstream is responsible for rejecting them.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := auth.KeyFromContext(r.Context())
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
