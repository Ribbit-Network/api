// Package auth provides API-key authentication for the Ribbit Network API.
//
// Keys are 256-bit random tokens formatted as "rbnt_<base64url>". Only the
// SHA-256 of the raw key is stored. SHA-256 is appropriate here (rather than
// bcrypt/argon2) because the keys carry full entropy — slowing per-request
// verification adds latency without making brute-force any less infeasible.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

const (
	keyPrefix     = "rbnt_"
	rawKeyBytes   = 32
	prefixDisplay = 12 // characters of the raw key kept for display (e.g. "rbnt_AbCdEf")
)

var (
	ErrInvalidKey = errors.New("invalid api key")
	ErrNotFound   = errors.New("api key not found")
)

type Key struct {
	ID        int64
	Prefix    string
	Owner     string
	CreatedAt time.Time
	RevokedAt *time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS api_keys (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			key_hash    TEXT    NOT NULL UNIQUE,
			prefix      TEXT    NOT NULL,
			owner       TEXT    NOT NULL,
			created_at  INTEGER NOT NULL,
			revoked_at  INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
	`)
	return err
}

// Issue generates a new key, stores its hash, and returns the raw key.
// The raw key is only returned here — callers must surface it to the user
// immediately, because it cannot be recovered later.
func (s *Store) Issue(owner string) (string, *Key, error) {
	raw, err := generate()
	if err != nil {
		return "", nil, err
	}
	hash := hashKey(raw)
	prefix := raw[:prefixDisplay]
	now := time.Now().UTC()

	res, err := s.db.Exec(
		`INSERT INTO api_keys (key_hash, prefix, owner, created_at) VALUES (?, ?, ?, ?)`,
		hash, prefix, owner, now.Unix(),
	)
	if err != nil {
		return "", nil, fmt.Errorf("insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return "", nil, fmt.Errorf("last insert id: %w", err)
	}
	return raw, &Key{ID: id, Prefix: prefix, Owner: owner, CreatedAt: now}, nil
}

// Verify returns nil iff the raw key matches a non-revoked record.
func (s *Store) Verify(raw string) error {
	if raw == "" {
		return ErrInvalidKey
	}
	hash := hashKey(raw)
	var revokedAt sql.NullInt64
	err := s.db.QueryRow(
		`SELECT revoked_at FROM api_keys WHERE key_hash = ?`, hash,
	).Scan(&revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidKey
	}
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	if revokedAt.Valid {
		return ErrInvalidKey
	}
	return nil
}

func (s *Store) List() ([]Key, error) {
	rows, err := s.db.Query(
		`SELECT id, prefix, owner, created_at, revoked_at FROM api_keys ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Key
	for rows.Next() {
		var k Key
		var created int64
		var revoked sql.NullInt64
		if err := rows.Scan(&k.ID, &k.Prefix, &k.Owner, &created, &revoked); err != nil {
			return nil, err
		}
		k.CreatedAt = time.Unix(created, 0).UTC()
		if revoked.Valid {
			t := time.Unix(revoked.Int64, 0).UTC()
			k.RevokedAt = &t
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) Revoke(id int64) error {
	res, err := s.db.Exec(
		`UPDATE api_keys SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`,
		time.Now().UTC().Unix(), id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func generate() (string, error) {
	b := make([]byte, rawKeyBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return keyPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
