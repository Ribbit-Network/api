package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Ribbit-Network/api/internal/auth"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// allowAll is a permissive auth.Verifier for tests — every key is valid.
// Combined with auth.Require, this is how tests populate the context-stashed
// key that Limiter.Middleware reads.
type allowAll struct{}

func (allowAll) Verify(string) error { return nil }

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// newLim builds a Limiter with a ttl far longer than any test runs, so the
// rate-limit tests below aren't affected by eviction.
func newLim(r rate.Limit, b int) *Limiter {
	return New(r, b, time.Hour)
}

func limited(l *Limiter) http.Handler {
	return auth.Require(allowAll{})(l.Middleware(okHandler()))
}

func reqWithKey(key string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("X-API-Key", key)
	return req
}

func reqWithBearer(key string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	return req
}

func TestRateLimit_AllowsWithinBurst(t *testing.T) {
	h := limited(newLim(rate.Limit(1), 5))

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, reqWithKey("testkey"))
		require.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i+1)
	}
}

func TestRateLimit_BlocksAfterBurst(t *testing.T) {
	h := limited(newLim(rate.Limit(1), 3))

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, reqWithKey("testkey"))
		require.Equal(t, http.StatusOK, rec.Code)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, reqWithKey("testkey"))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}

// Confirms the same rate-limit gate fires when the key arrives via
// "Authorization: Bearer ..." instead of X-API-Key — i.e. the auth+ratelimit
// chain works end-to-end regardless of which header the client uses.
func TestRateLimit_BlocksAfterBurst_Bearer(t *testing.T) {
	h := limited(newLim(rate.Limit(1), 2))

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, reqWithBearer("bearer-key"))
		require.Equal(t, http.StatusOK, rec.Code)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, reqWithBearer("bearer-key"))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_IndependentPerKey(t *testing.T) {
	h := limited(newLim(rate.Limit(1), 1))

	for _, key := range []string{"key-a", "key-b", "key-c"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, reqWithKey(key))
		require.Equal(t, http.StatusOK, rec.Code, "first request for %s should pass", key)
	}
}

// If something mounts Limiter.Middleware without auth in front, a request with
// no context key should pass through rather than gate on an empty string.
func TestRateLimit_NoKeyInContext_PassesThrough(t *testing.T) {
	h := newLim(rate.Limit(1), 1).Middleware(okHandler())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/data", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}

// Idle entries should be evicted once ttl elapses; active ones should remain.
// Clock is injected so the test doesn't sleep.
func TestRateLimit_EvictsIdleEntries(t *testing.T) {
	clock := time.Unix(1_000_000, 0)
	l := New(rate.Limit(1), 1, time.Minute)
	l.now = func() time.Time { return clock }

	l.get("idle-a")
	l.get("idle-b")
	require.Len(t, l.entries, 2)

	// Jump past the ttl and touch a new key — triggers the lazy sweep.
	clock = clock.Add(2 * time.Minute)
	l.get("fresh")

	require.Len(t, l.entries, 1)
	_, ok := l.entries["fresh"]
	require.True(t, ok, "freshly-touched key should remain")
}

// tiered wires Optional auth in front of Tiered, mirroring main.go: keyed
// requests are limited by key, anonymous ones by IP.
func tiered(keyed, anon *Limiter) http.Handler {
	return auth.Optional(allowAll{})(Tiered(keyed, anon)(okHandler()))
}

func anonReq(ip string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Fly-Client-IP", ip)
	return req
}

// Anonymous callers are throttled at the (slower) anon limiter's rate.
func TestTiered_AnonThrottledByIP(t *testing.T) {
	// Generous keyed tier so it can't be the thing doing the limiting here.
	h := tiered(newLim(rate.Limit(100), 100), newLim(rate.Limit(1), 2))

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, anonReq("203.0.113.7"))
		require.Equal(t, http.StatusOK, rec.Code, "anon request %d should be within burst", i+1)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, anonReq("203.0.113.7"))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.NotEmpty(t, rec.Header().Get("Retry-After"), "429 should advertise Retry-After")
}

// A keyed caller gets the keyed limits, not the anon ones — even when the anon
// tier is tiny, a key lets them sail past it.
func TestTiered_KeyedUsesKeyedLimits(t *testing.T) {
	h := tiered(newLim(rate.Limit(1), 10), newLim(rate.Limit(1), 1))

	for i := 0; i < 10; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, reqWithKey("testkey"))
		require.Equal(t, http.StatusOK, rec.Code, "keyed request %d should pass", i+1)
	}
}

// Different anonymous IPs draw from independent buckets.
func TestTiered_AnonIndependentPerIP(t *testing.T) {
	h := tiered(newLim(rate.Limit(100), 100), newLim(rate.Limit(1), 1))

	for _, ip := range []string{"198.51.100.1", "198.51.100.2", "198.51.100.3"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, anonReq(ip))
		require.Equal(t, http.StatusOK, rec.Code, "first request from %s should pass", ip)
	}
}

func TestClientIP(t *testing.T) {
	t.Run("prefers Fly-Client-IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Fly-Client-IP", "1.2.3.4")
		req.Header.Set("X-Forwarded-For", "5.6.7.8")
		require.Equal(t, "1.2.3.4", clientIP(req))
	})
	t.Run("first XFF hop", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", "5.6.7.8, 9.10.11.12")
		require.Equal(t, "5.6.7.8", clientIP(req))
	})
	t.Run("falls back to RemoteAddr host", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.5:5555"
		require.Equal(t, "192.0.2.5", clientIP(req))
	})
}

func TestRateLimit_KeepsActiveEntries(t *testing.T) {
	clock := time.Unix(1_000_000, 0)
	l := New(rate.Limit(1), 1, time.Minute)
	l.now = func() time.Time { return clock }

	// Touch the same key every 30s for several minutes — well past one ttl
	// of cumulative time, but never idle for a full ttl.
	for i := 0; i < 10; i++ {
		l.get("active")
		clock = clock.Add(30 * time.Second)
	}

	require.Len(t, l.entries, 1)
	_, ok := l.entries["active"]
	require.True(t, ok)
}
