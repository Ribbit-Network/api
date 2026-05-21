package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ribbit-Network/api/internal/auth"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func requestWithKey(key string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	if key != "" {
		req = req.WithContext(auth.WithKey(req.Context(), key))
	}
	return req
}

func TestRateLimit_AllowsWithinBurst(t *testing.T) {
	l := New(rate.Limit(1), 5)
	h := l.Middleware(okHandler())

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, requestWithKey("testkey"))
		require.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i+1)
	}
}

func TestRateLimit_BlocksAfterBurst(t *testing.T) {
	l := New(rate.Limit(1), 3)
	h := l.Middleware(okHandler())

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, requestWithKey("testkey"))
		require.Equal(t, http.StatusOK, rec.Code)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, requestWithKey("testkey"))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_IndependentPerKey(t *testing.T) {
	l := New(rate.Limit(1), 1)
	h := l.Middleware(okHandler())

	for _, key := range []string{"key-a", "key-b", "key-c"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, requestWithKey(key))
		require.Equal(t, http.StatusOK, rec.Code, "first request for %s should pass", key)
	}
}

func TestRateLimit_NoKey_PassesThrough(t *testing.T) {
	l := New(rate.Limit(1), 1)
	h := l.Middleware(okHandler())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, requestWithKey(""))
	require.Equal(t, http.StatusOK, rec.Code)
}
