package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
)

type Verifier interface {
	Verify(raw string) error
}

type ctxKey struct{}

// WithKey returns ctx carrying key, so downstream middleware can read the
// verified API key without re-parsing request headers.
func WithKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, ctxKey{}, key)
}

// KeyFromContext returns the verified API key set by Require, or "" if absent.
func KeyFromContext(ctx context.Context) string {
	s, _ := ctx.Value(ctxKey{}).(string)
	return s
}

// Require returns middleware that rejects requests without a valid API key
// and stashes the verified key on the request context for downstream handlers.
// Keys are accepted in either "Authorization: Bearer <key>" or "X-API-Key: <key>".
func Require(v Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := extractKey(r)
			if key == "" {
				unauthorized(w, "missing api key")
				return
			}
			switch err := v.Verify(key); {
			case err == nil:
				next.ServeHTTP(w, r.WithContext(WithKey(r.Context(), key)))
			case errors.Is(err, ErrInvalidKey):
				unauthorized(w, "invalid api key")
			default:
				log.Printf("auth: verify error: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		})
	}
}

func extractKey(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if rest, ok := strings.CutPrefix(h, "Bearer "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return strings.TrimSpace(r.Header.Get("X-API-Key"))
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="ribbit-api"`)
	http.Error(w, msg, http.StatusUnauthorized)
}
