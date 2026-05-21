package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

func limited(l *Limiter) http.Handler {
	return auth.Require(allowAll{})(l.Middleware(okHandler()))
}

func reqWithKey(key string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("X-API-Key", key)
	return req
}

func TestRateLimit_AllowsWithinBurst(t *testing.T) {
	h := limited(New(rate.Limit(1), 5))

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, reqWithKey("testkey"))
		require.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i+1)
	}
}

func TestRateLimit_BlocksAfterBurst(t *testing.T) {
	h := limited(New(rate.Limit(1), 3))

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, reqWithKey("testkey"))
		require.Equal(t, http.StatusOK, rec.Code)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, reqWithKey("testkey"))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_IndependentPerKey(t *testing.T) {
	h := limited(New(rate.Limit(1), 1))

	for _, key := range []string{"key-a", "key-b", "key-c"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, reqWithKey(key))
		require.Equal(t, http.StatusOK, rec.Code, "first request for %s should pass", key)
	}
}

// If something mounts Limiter.Middleware without auth in front, a request with
// no context key should pass through rather than gate on an empty string.
func TestRateLimit_NoKeyInContext_PassesThrough(t *testing.T) {
	h := New(rate.Limit(1), 1).Middleware(okHandler())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/data", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}
