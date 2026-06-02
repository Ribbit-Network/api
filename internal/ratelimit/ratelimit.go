// Package ratelimit provides per-API-key rate limiting middleware.
package ratelimit

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Ribbit-Network/api/internal/auth"
	"golang.org/x/time/rate"
)

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Limiter holds per-key token-bucket limiters. Idle entries are evicted
// lazily — a sweep runs inline with get() at most once per ttl, so there is
// no background goroutine to manage.
type Limiter struct {
	mu        sync.Mutex
	entries   map[string]*entry
	r         rate.Limit
	b         int
	ttl       time.Duration
	lastSweep time.Time
	now       func() time.Time
}

// New creates a Limiter allowing r tokens per second with a burst of b. Entries
// untouched for ttl are evicted; ttl must be positive. Choose ttl >= b/r so an
// evicted bucket would already have refilled — otherwise eviction effectively
// grants a fresh burst.
func New(r rate.Limit, b int, ttl time.Duration) *Limiter {
	if ttl <= 0 {
		panic("ratelimit: ttl must be > 0")
	}
	return &Limiter{
		entries: make(map[string]*entry),
		r:       r,
		b:       b,
		ttl:     ttl,
		now:     time.Now,
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

// Tiered returns middleware that rate-limits authenticated callers by API key
// against the keyed limiter, and anonymous callers by client IP against the
// (slower) anon limiter. The key is read from the request context, so this must
// be mounted behind auth.Optional. Rejections carry a Retry-After header.
func Tiered(keyed, anon *Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var lim *rate.Limiter
			if key := auth.KeyFromContext(r.Context()); key != "" {
				lim = keyed.get(key)
			} else {
				lim = anon.get(clientIP(r))
			}
			if res := lim.Reserve(); !res.OK() || res.Delay() > 0 {
				if res.OK() {
					// We're rejecting, so don't actually consume the token; just
					// use the reservation to report when the caller may retry.
					w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(res.Delay().Seconds()))))
					res.Cancel()
				}
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP returns the originating client address. On Fly.io the edge proxy
// sets Fly-Client-IP (and X-Forwarded-For); we trust those because the app is
// only reachable through that proxy. A directly-exposed service must not trust
// these headers, since clients can forge them.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("Fly-Client-IP"); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i]) // first hop is the original client
		}
		return strings.TrimSpace(xff)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func (l *Limiter) get(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastSweep) >= l.ttl {
		for k, e := range l.entries {
			if now.Sub(e.lastSeen) >= l.ttl {
				delete(l.entries, k)
			}
		}
		l.lastSweep = now
	}

	e, ok := l.entries[key]
	if !ok {
		e = &entry{limiter: rate.NewLimiter(l.r, l.b)}
		l.entries[key] = e
	}
	e.lastSeen = now
	return e.limiter
}
