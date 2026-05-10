// SPDX-License-Identifier: AGPL-3.0-or-later

// Package auth manages local user accounts and session cookies.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/argon2"
)

// argon2id parameters (spec §14.2).
const (
	argonTime    = 2
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
)

// Store manages the accounts + sessions SQLite database.
type Store struct {
	db         *sql.DB
	SessionTTL time.Duration
}

// User is a registered account.
type User struct {
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Admin       bool      `json:"admin"`
	CreatedAt   time.Time `json:"created_at"`
}

// Open opens (or creates) the auth database at dbPath.
func Open(dbPath string, sessionTTL time.Duration) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating auth db dir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening auth db: %w", err)
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, SessionTTL: sessionTTL}
	if err := s.createSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database connection.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) createSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS users (
    email         TEXT PRIMARY KEY,
    display_name  TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    admin         INTEGER NOT NULL DEFAULT 0,
    created_at    INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    user_email TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_email);
CREATE TABLE IF NOT EXISTS tokens (
    id         TEXT PRIMARY KEY,
    user_email TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    prefix     TEXT NOT NULL,
    expires_at INTEGER,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tokens_user ON tokens(user_email);
CREATE INDEX IF NOT EXISTS idx_tokens_prefix ON tokens(prefix);
`)
	if err != nil {
		return err
	}
	// Idempotent migrations for existing databases.
	for _, stmt := range []string{
		`ALTER TABLE users ADD COLUMN admin INTEGER NOT NULL DEFAULT 0`,
	} {
		if _, err := s.db.Exec(stmt); err != nil {
			// Ignore "duplicate column" errors for idempotency.
			if !strings.Contains(err.Error(), "duplicate column") {
				return fmt.Errorf("schema migration: %w", err)
			}
		}
	}
	return nil
}

// UserCount returns the total number of registered users.
func (s *Store) UserCount() (int, error) {
	var n int
	return n, s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
}

// CreateUser registers a new account with an argon2id-hashed password.
func (s *Store) CreateUser(email, displayName, password string, admin bool) error {
	hash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	adminInt := 0
	if admin {
		adminInt = 1
	}
	_, err = s.db.Exec(
		`INSERT INTO users (email, display_name, password_hash, admin, created_at) VALUES (?, ?, ?, ?, ?)`,
		email, displayName, hash, adminInt, time.Now().Unix(),
	)
	return err
}

// GetUser returns the user with the given email, or nil if not found.
func (s *Store) GetUser(email string) (*User, error) {
	var u User
	var ts int64
	var adminInt int
	err := s.db.QueryRow(
		`SELECT email, display_name, admin, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.Email, &u.DisplayName, &adminInt, &ts)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.Admin = adminInt != 0
	u.CreatedAt = time.Unix(ts, 0)
	return &u, nil
}

// ListUsers returns all users ordered by created_at.
func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(
		`SELECT email, display_name, admin, created_at FROM users ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var ts int64
		var adminInt int
		if err := rows.Scan(&u.Email, &u.DisplayName, &adminInt, &ts); err != nil {
			return nil, err
		}
		u.Admin = adminInt != 0
		u.CreatedAt = time.Unix(ts, 0)
		users = append(users, u)
	}
	return users, rows.Err()
}

// DeleteUser removes a user and all their sessions and tokens.
func (s *Store) DeleteUser(email string) error {
	if _, err := s.db.Exec(`DELETE FROM users WHERE email = ?`, email); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM sessions WHERE user_email = ?`, email); err != nil {
		return err
	}
	return s.DeleteTokensForUser(email)
}

// ResetPassword updates a user's password hash. Returns an error if the user does not exist.
func (s *Store) ResetPassword(email, newPassword string) error {
	hash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	res, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE email = ?`, hash, email)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("user %q not found", email)
	}
	return nil
}

// Authenticate verifies email + password. Returns the user on success, nil on failure.
func (s *Store) Authenticate(email, password string) (*User, error) {
	var u User
	var hash string
	var ts int64
	var adminInt int
	err := s.db.QueryRow(
		`SELECT email, display_name, password_hash, admin, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.Email, &u.DisplayName, &hash, &adminInt, &ts)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !verifyPassword(password, hash) {
		return nil, nil
	}
	u.Admin = adminInt != 0
	u.CreatedAt = time.Unix(ts, 0)
	return &u, nil
}

// CreateSession generates a new session token for the user and stores it.
func (s *Store) CreateSession(userEmail string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b)
	now := time.Now()
	_, err := s.db.Exec(
		`INSERT INTO sessions (id, user_email, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		id, userEmail, now.Unix(), now.Add(s.SessionTTL).Unix(),
	)
	return id, err
}

// GetSession looks up a session by ID and returns the owning user.
// Returns nil if the session is missing or expired.
func (s *Store) GetSession(id string) (*User, error) {
	var u User
	var ts, expiresAt int64
	var adminInt int
	err := s.db.QueryRow(`
		SELECT u.email, u.display_name, u.admin, u.created_at, s.expires_at
		FROM sessions s
		JOIN users u ON s.user_email = u.email
		WHERE s.id = ?`, id,
	).Scan(&u.Email, &u.DisplayName, &adminInt, &ts, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if time.Now().Unix() > expiresAt {
		_ = s.DeleteSession(id)
		return nil, nil
	}
	u.Admin = adminInt != 0
	u.CreatedAt = time.Unix(ts, 0)
	return &u, nil
}

// DeleteSession removes a session (logout).
func (s *Store) DeleteSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// CreateToken generates a bearer token for the user, stores its hash, and returns
// the plaintext token (the only time it is available in plain form).
// expires is optional; pass nil for a non-expiring token.
func (s *Store) CreateToken(userEmail string, expires *time.Time) (plaintext string, err error) {
	// Generate 32 random bytes → hex-encode as the plaintext token (64 hex chars).
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generating token bytes: %w", err)
	}
	plaintext = hex.EncodeToString(raw)

	// First 8 hex chars as a non-secret prefix for fast lookup.
	prefix := plaintext[:8]

	hash, err := hashPassword(plaintext)
	if err != nil {
		return "", fmt.Errorf("hashing token: %w", err)
	}

	// Generate a short random ID for the primary key.
	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		return "", fmt.Errorf("generating token id: %w", err)
	}
	id := hex.EncodeToString(idBytes)

	var expiresUnix *int64
	if expires != nil {
		v := expires.Unix()
		expiresUnix = &v
	}

	_, err = s.db.Exec(
		`INSERT INTO tokens (id, user_email, token_hash, prefix, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, userEmail, hash, prefix, expiresUnix, time.Now().Unix(),
	)
	if err != nil {
		return "", err
	}
	return plaintext, nil
}

// ValidateToken checks a plaintext bearer token against stored hashes.
// Returns the associated user on match, nil on mismatch or expiry.
func (s *Store) ValidateToken(plaintext string) (*User, error) {
	if len(plaintext) < 8 {
		return nil, nil
	}
	prefix := plaintext[:8]

	rows, err := s.db.Query(`
		SELECT t.token_hash, t.expires_at, u.email, u.display_name, u.admin, u.created_at
		FROM tokens t
		JOIN users u ON t.user_email = u.email
		WHERE t.prefix = ?`, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now().Unix()
	for rows.Next() {
		var tokenHash string
		var expiresAt sql.NullInt64
		var u User
		var ts int64
		var adminInt int
		if err := rows.Scan(&tokenHash, &expiresAt, &u.Email, &u.DisplayName, &adminInt, &ts); err != nil {
			return nil, err
		}
		if expiresAt.Valid && now > expiresAt.Int64 {
			continue
		}
		if verifyPassword(plaintext, tokenHash) {
			u.Admin = adminInt != 0
			u.CreatedAt = time.Unix(ts, 0)
			return &u, nil
		}
	}
	return nil, rows.Err()
}

// DeleteTokensForUser revokes all tokens for the given user.
func (s *Store) DeleteTokensForUser(email string) error {
	_, err := s.db.Exec(`DELETE FROM tokens WHERE user_email = ?`, email)
	return err
}

// ----- password helpers -----

// hashPassword produces a "$argon2id-style" encoded string: base64(salt):base64(hash).
func hashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return base64.StdEncoding.EncodeToString(salt) + ":" + base64.StdEncoding.EncodeToString(hash), nil
}

// verifyPassword checks a plaintext password against an encoded hash.
func verifyPassword(password, encoded string) bool {
	parts := strings.SplitN(encoded, ":", 2)
	if len(parts) != 2 {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expected, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
