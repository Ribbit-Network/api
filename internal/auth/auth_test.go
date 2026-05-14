package auth

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	s, err := NewStore(db)
	require.NoError(t, err)
	return s
}

func TestIssueAndVerify(t *testing.T) {
	s := newTestStore(t)

	raw, k, err := s.Issue("test@example.com")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(raw, "rbnt_"))
	require.Equal(t, "test@example.com", k.Owner)
	require.NotZero(t, k.ID)

	require.NoError(t, s.Verify(raw))
}

func TestVerify_RejectsUnknownKey(t *testing.T) {
	s := newTestStore(t)
	require.ErrorIs(t, s.Verify("rbnt_doesnotexist"), ErrInvalidKey)
	require.ErrorIs(t, s.Verify(""), ErrInvalidKey)
}

func TestVerify_RejectsRevokedKey(t *testing.T) {
	s := newTestStore(t)
	raw, k, err := s.Issue("test@example.com")
	require.NoError(t, err)

	require.NoError(t, s.Revoke(k.ID))
	require.ErrorIs(t, s.Verify(raw), ErrInvalidKey)
}

func TestRevoke_UnknownID(t *testing.T) {
	s := newTestStore(t)
	require.ErrorIs(t, s.Revoke(9999), ErrNotFound)
}

func TestList(t *testing.T) {
	s := newTestStore(t)
	_, _, err := s.Issue("a@example.com")
	require.NoError(t, err)
	_, _, err = s.Issue("b@example.com")
	require.NoError(t, err)

	keys, err := s.List()
	require.NoError(t, err)
	require.Len(t, keys, 2)
	require.Equal(t, "a@example.com", keys[0].Owner)
	require.Equal(t, "b@example.com", keys[1].Owner)
}

// stubVerifier lets middleware tests run without a real store.
type stubVerifier struct{ valid string }

func (s stubVerifier) Verify(raw string) error {
	if raw == s.valid {
		return nil
	}
	return ErrInvalidKey
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestMiddleware_MissingKey(t *testing.T) {
	h := Require(stubVerifier{valid: "good"})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_BadKey(t *testing.T) {
	h := Require(stubVerifier{valid: "good"})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_GoodKey_Bearer(t *testing.T) {
	h := Require(stubVerifier{valid: "good"})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Authorization", "Bearer good")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
}

func TestMiddleware_GoodKey_XAPIKey(t *testing.T) {
	h := Require(stubVerifier{valid: "good"})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("X-API-Key", "good")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}
