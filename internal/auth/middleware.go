package auth

import (
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
			if err := v.Verify(key); err != nil {
				unauthorized(w, "invalid api key")
				return
			}
			next.ServeHTTP(w, r)
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
