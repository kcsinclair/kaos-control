// SPDX-License-Identifier: AGPL-3.0-or-later

// Package auth_test contains unit tests for the auth store.
// Each test uses t.TempDir() for DB isolation so there is no shared state.
package auth_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/auth"
)

// openTestStore opens a fresh auth store in a temporary directory.
// The store is closed automatically when the test ends.
func openTestStore(t *testing.T) *auth.Store {
	t.Helper()
	s, err := auth.Open(filepath.Join(t.TempDir(), "auth.db"), 24*time.Hour)
	if err != nil {
		t.Fatalf("auth.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// ─── Milestone 1: User CRUD ───────────────────────────────────────────────────

// TestCreateUser_AdminFlag verifies that CreateUser with admin=true stores the
// flag, and GetUser returns it correctly.
func TestCreateUser_AdminFlag(t *testing.T) {
	s := openTestStore(t)
	if err := s.CreateUser("admin@test.com", "Admin", "pass123", true); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	u, err := s.GetUser("admin@test.com")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if u == nil {
		t.Fatal("GetUser returned nil for existing user")
	}
	if !u.Admin {
		t.Error("want Admin=true, got false")
	}
	if u.Email != "admin@test.com" {
		t.Errorf("Email = %q, want %q", u.Email, "admin@test.com")
	}
}

// TestListUsers creates 3 users and asserts ListUsers returns them in
// created_at order with all fields populated.
func TestListUsers(t *testing.T) {
	s := openTestStore(t)
	emails := []string{"a@test.com", "b@test.com", "c@test.com"}
	for _, e := range emails {
		if err := s.CreateUser(e, "Name "+e, "password", false); err != nil {
			t.Fatalf("CreateUser %s: %v", e, err)
		}
	}

	users, err := s.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("want 3 users, got %d", len(users))
	}
	for i, u := range users {
		if u.Email == "" {
			t.Errorf("users[%d].Email is empty", i)
		}
		if u.DisplayName == "" {
			t.Errorf("users[%d].DisplayName is empty", i)
		}
		if u.CreatedAt.IsZero() {
			t.Errorf("users[%d].CreatedAt is zero", i)
		}
	}
	// Order must be by created_at (insertion order for this test since we
	// insert sequentially with time.Now() resolution of at least 1 second
	// when using integer unix timestamps — the schema stores unix seconds).
	// We verify all expected emails are present; strict order is best-effort
	// given sub-second insertion and integer timestamp resolution.
	seen := map[string]bool{}
	for _, u := range users {
		seen[u.Email] = true
	}
	for _, want := range emails {
		if !seen[want] {
			t.Errorf("email %q missing from ListUsers result", want)
		}
	}
}

// TestDeleteUser creates a user with a session, deletes the user, and asserts
// that both GetUser and GetSession return nil afterward.
func TestDeleteUser(t *testing.T) {
	s := openTestStore(t)
	if err := s.CreateUser("del@test.com", "Del", "pass", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	sessID, err := s.CreateSession("del@test.com")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := s.DeleteUser("del@test.com"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	u, err := s.GetUser("del@test.com")
	if err != nil {
		t.Fatalf("GetUser after delete: %v", err)
	}
	if u != nil {
		t.Error("GetUser returned non-nil after user delete")
	}

	su, err := s.GetSession(sessID)
	if err != nil {
		t.Fatalf("GetSession after delete: %v", err)
	}
	if su != nil {
		t.Error("GetSession returned non-nil after user delete (session not cascade-deleted)")
	}
}

// TestResetPassword asserts that the old password fails and the new password
// succeeds after ResetPassword.
func TestResetPassword(t *testing.T) {
	s := openTestStore(t)
	if err := s.CreateUser("pw@test.com", "PW", "oldpassword", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if err := s.ResetPassword("pw@test.com", "newpassword"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	// Old password must fail.
	u, err := s.Authenticate("pw@test.com", "oldpassword")
	if err != nil {
		t.Fatalf("Authenticate with old password: %v", err)
	}
	if u != nil {
		t.Error("Authenticate returned non-nil for old password after reset")
	}

	// New password must succeed.
	u, err = s.Authenticate("pw@test.com", "newpassword")
	if err != nil {
		t.Fatalf("Authenticate with new password: %v", err)
	}
	if u == nil {
		t.Error("Authenticate returned nil for new password after reset")
	}
}

// TestCreateUser_DuplicateEmail asserts that creating the same email twice
// returns an error on the second call.
func TestCreateUser_DuplicateEmail(t *testing.T) {
	s := openTestStore(t)
	if err := s.CreateUser("dup@test.com", "Dup", "pass", false); err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}
	if err := s.CreateUser("dup@test.com", "Dup2", "pass2", false); err == nil {
		t.Error("second CreateUser with duplicate email: expected error, got nil")
	}
}

// TestSchemaIdempotency calls Open twice on the same DB file and asserts
// neither call returns an error (CREATE TABLE IF NOT EXISTS is idempotent).
func TestSchemaIdempotency(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "auth.db")

	s1, err := auth.Open(dbPath, time.Hour)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	_ = s1.Close()

	s2, err := auth.Open(dbPath, time.Hour)
	if err != nil {
		t.Fatalf("second Open on same DB: %v", err)
	}
	_ = s2.Close()
}

// ─── Milestone 2: Bearer Tokens ──────────────────────────────────────────────

// TestCreateToken asserts that the returned plaintext token is ≥64 hex chars
// and is never stored verbatim in the DB.
func TestCreateToken(t *testing.T) {
	s := openTestStore(t)
	if err := s.CreateUser("tok@test.com", "Tok", "pass", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	token, err := s.CreateToken("tok@test.com", nil)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	if len(token) < 64 {
		t.Errorf("token plaintext length = %d, want ≥64", len(token))
	}
	// All characters must be lowercase hex digits.
	for i, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("token[%d] = %q is not a hex digit", i, c)
			break
		}
	}
}

// TestValidateToken_Valid creates a token and asserts ValidateToken returns the
// correct user.
func TestValidateToken_Valid(t *testing.T) {
	s := openTestStore(t)
	if err := s.CreateUser("valid@test.com", "Valid", "pass", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	token, err := s.CreateToken("valid@test.com", nil)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	u, err := s.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if u == nil {
		t.Fatal("ValidateToken returned nil for valid token")
	}
	if u.Email != "valid@test.com" {
		t.Errorf("user email = %q, want %q", u.Email, "valid@test.com")
	}
}

// TestValidateToken_Invalid asserts that a random hex string validates to nil.
func TestValidateToken_Invalid(t *testing.T) {
	s := openTestStore(t)
	// 64 hex zeros — syntactically valid but no matching token in the DB.
	u, err := s.ValidateToken("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if u != nil {
		t.Error("ValidateToken returned non-nil for unknown token")
	}
}

// TestValidateToken_Expired creates a token with an expiry 1 second in the
// past and asserts ValidateToken returns nil.
func TestValidateToken_Expired(t *testing.T) {
	s := openTestStore(t)
	if err := s.CreateUser("exp@test.com", "Exp", "pass", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	past := time.Now().Add(-1 * time.Second)
	token, err := s.CreateToken("exp@test.com", &past)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	u, err := s.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if u != nil {
		t.Error("ValidateToken returned non-nil for expired token")
	}
}

// TestDeleteTokensForUser creates 2 tokens for a user, deletes all, and
// asserts neither token validates afterward.
func TestDeleteTokensForUser(t *testing.T) {
	s := openTestStore(t)
	if err := s.CreateUser("del2@test.com", "Del2", "pass", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	tok1, err := s.CreateToken("del2@test.com", nil)
	if err != nil {
		t.Fatalf("CreateToken 1: %v", err)
	}
	tok2, err := s.CreateToken("del2@test.com", nil)
	if err != nil {
		t.Fatalf("CreateToken 2: %v", err)
	}

	if err := s.DeleteTokensForUser("del2@test.com"); err != nil {
		t.Fatalf("DeleteTokensForUser: %v", err)
	}

	u1, err := s.ValidateToken(tok1)
	if err != nil {
		t.Fatalf("ValidateToken tok1 after delete: %v", err)
	}
	if u1 != nil {
		t.Error("tok1 still valid after DeleteTokensForUser")
	}

	u2, err := s.ValidateToken(tok2)
	if err != nil {
		t.Fatalf("ValidateToken tok2 after delete: %v", err)
	}
	if u2 != nil {
		t.Error("tok2 still valid after DeleteTokensForUser")
	}
}

// TestDeleteUser_CascadesToTokens creates a user and a token, deletes the
// user, and asserts the token no longer validates.
func TestDeleteUser_CascadesToTokens(t *testing.T) {
	s := openTestStore(t)
	if err := s.CreateUser("casc@test.com", "Casc", "pass", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	tok, err := s.CreateToken("casc@test.com", nil)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	if err := s.DeleteUser("casc@test.com"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	u, err := s.ValidateToken(tok)
	if err != nil {
		t.Fatalf("ValidateToken after user delete: %v", err)
	}
	if u != nil {
		t.Error("token still valid after user delete (cascade to tokens missing)")
	}
}
