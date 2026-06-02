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

// KeyFromContext returns the verified API key set by Require, or "" if absent.
// Only Require populates this value; the setter is intentionally unexported so
// downstream middleware cannot be fooled by callers stashing arbitrary strings.
func KeyFromContext(ctx context.Context) string {
	s, _ := ctx.Value(ctxKey{}).(string)
	return s
}

func withKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, ctxKey{}, key)
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
				next.ServeHTTP(w, r.WithContext(withKey(r.Context(), key)))
			case errors.Is(err, ErrInvalidKey):
				unauthorized(w, "invalid api key")
			default:
				log.Printf("auth: verify error: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		})
	}
}

// Optional returns middleware for endpoints that are open to anonymous callers
// but grant a higher tier to authenticated ones. A missing key passes through
// anonymously (no key on the context); a valid key is stashed on the context
// like Require does; a present-but-invalid key is still rejected with 401, so a
// wrong key surfaces as a mistake rather than being silently downgraded.
func Optional(v Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := extractKey(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}
			switch err := v.Verify(key); {
			case err == nil:
				next.ServeHTTP(w, r.WithContext(withKey(r.Context(), key)))
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
