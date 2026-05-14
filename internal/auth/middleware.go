package auth

import (
	"errors"
	"log"
	"net/http"
	"strings"
)

type Verifier interface {
	Verify(raw string) error
}

// Require returns middleware that rejects requests without a valid API key.
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
				next.ServeHTTP(w, r)
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
