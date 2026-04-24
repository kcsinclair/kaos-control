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
    created_at    INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    user_email TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_email);
`)
	return err
}

// UserCount returns the total number of registered users.
func (s *Store) UserCount() (int, error) {
	var n int
	return n, s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
}

// CreateUser registers a new account with an argon2id-hashed password.
func (s *Store) CreateUser(email, displayName, password string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO users (email, display_name, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		email, displayName, hash, time.Now().Unix(),
	)
	return err
}

// GetUser returns the user with the given email, or nil if not found.
func (s *Store) GetUser(email string) (*User, error) {
	var u User
	var ts int64
	err := s.db.QueryRow(
		`SELECT email, display_name, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.Email, &u.DisplayName, &ts)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt = time.Unix(ts, 0)
	return &u, nil
}

// Authenticate verifies email + password. Returns the user on success, nil on failure.
func (s *Store) Authenticate(email, password string) (*User, error) {
	var u User
	var hash string
	var ts int64
	err := s.db.QueryRow(
		`SELECT email, display_name, password_hash, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.Email, &u.DisplayName, &hash, &ts)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !verifyPassword(password, hash) {
		return nil, nil
	}
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
	err := s.db.QueryRow(`
		SELECT u.email, u.display_name, u.created_at, s.expires_at
		FROM sessions s
		JOIN users u ON s.user_email = u.email
		WHERE s.id = ?`, id,
	).Scan(&u.Email, &u.DisplayName, &ts, &expiresAt)
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
	u.CreatedAt = time.Unix(ts, 0)
	return &u, nil
}

// DeleteSession removes a session (logout).
func (s *Store) DeleteSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
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
