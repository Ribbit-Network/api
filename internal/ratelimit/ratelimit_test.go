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
