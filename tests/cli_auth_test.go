// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// CLI integration tests for the `kaos-control auth` subcommand family.
// These tests invoke the compiled binary (built once by TestMain in
// cli_init_test.go) and exercise the full CLI path from argument parsing
// through auth-store operations to output formatting.
//
// Each test creates its own temporary config dir and data dir, so tests are
// fully isolated even when run in parallel.
package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/auth"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

// writeAuthConfig creates a temporary XDG_CONFIG_HOME directory tree with a
// minimal kaos-control config.yaml pointing data_dir at a second temp dir.
// It returns the XDG_CONFIG_HOME value (to set in subprocess env) and the
// data dir path (to open the auth DB directly in tests).
func writeAuthConfig(t *testing.T) (cfgHome, dataDir string) {
	t.Helper()
	cfgHome = t.TempDir()
	dataDir = t.TempDir()
	cfgDir := filepath.Join(cfgHome, "kaos-control")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", cfgDir, err)
	}
	content := fmt.Sprintf("data_dir: %q\n", dataDir)
	cfgFile := filepath.Join(cfgDir, "config.yaml")
	if err := os.WriteFile(cfgFile, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", cfgFile, err)
	}
	return
}

// runAuth invokes `kaos-control auth <args...>` with XDG_CONFIG_HOME set to
// cfgHome, optionally piping stdinData to the process's stdin.
// Returns stdout, stderr, and exit code.
func runAuth(t *testing.T, cfgHome, stdinData string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	fullArgs := append([]string{"auth"}, args...)
	cmd := newBinCmd(t, fullArgs...)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+cfgHome)
	if stdinData != "" {
		cmd.Stdin = strings.NewReader(stdinData)
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			code = ex.ExitCode()
		} else {
			t.Fatalf("exec: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), code
}

// ─── Milestone 3: CLI Subcommand Integration Tests ───────────────────────────

// TestAuthCreateUser runs `auth create-user` with a password on stdin and
// asserts exit 0 plus a confirmation message.
func TestAuthCreateUser(t *testing.T) {
	cfgHome, _ := writeAuthConfig(t)
	stdout, _, code := runAuth(t, cfgHome, "secret123\n",
		"create-user", "--email", "user@test.com", "--name", "Test User", "--password-stdin",
	)
	if code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "user@test.com") {
		t.Errorf("stdout missing email confirmation\ngot: %s", stdout)
	}
}

// TestAuthCreateUser_DuplicateEmail creates the same user twice and asserts
// the second attempt exits 1 with an error message.
func TestAuthCreateUser_DuplicateEmail(t *testing.T) {
	cfgHome, _ := writeAuthConfig(t)
	_, _, code := runAuth(t, cfgHome, "secret123\n",
		"create-user", "--email", "dup@test.com", "--name", "Dup", "--password-stdin",
	)
	if code != 0 {
		t.Fatalf("first create-user: want exit 0, got %d", code)
	}

	_, stderr, code := runAuth(t, cfgHome, "secret456\n",
		"create-user", "--email", "dup@test.com", "--name", "Dup2", "--password-stdin",
	)
	if code == 0 {
		t.Error("second create-user with duplicate email: want exit 1, got 0")
	}
	if stderr == "" {
		t.Error("second create-user: expected error output in stderr, got nothing")
	}
}

// TestAuthListUsers creates 2 users and asserts `list-users` output contains
// both emails in tabular format.
func TestAuthListUsers(t *testing.T) {
	cfgHome, _ := writeAuthConfig(t)
	for _, email := range []string{"alice@test.com", "bob@test.com"} {
		_, _, code := runAuth(t, cfgHome, "password1\n",
			"create-user", "--email", email, "--name", email, "--password-stdin",
		)
		if code != 0 {
			t.Fatalf("create-user %s: want exit 0, got %d", email, code)
		}
	}

	stdout, _, code := runAuth(t, cfgHome, "", "list-users")
	if code != 0 {
		t.Fatalf("list-users: want exit 0, got %d", code)
	}
	for _, want := range []string{"alice@test.com", "bob@test.com"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("list-users stdout missing %q\ngot: %s", want, stdout)
		}
	}
	// Must contain a header row.
	if !strings.Contains(stdout, "EMAIL") {
		t.Errorf("list-users stdout missing header row\ngot: %s", stdout)
	}
}

// TestAuthDeleteUser creates a user, deletes them, and asserts `list-users`
// no longer contains their email.
func TestAuthDeleteUser(t *testing.T) {
	cfgHome, _ := writeAuthConfig(t)
	_, _, code := runAuth(t, cfgHome, "pw\n",
		"create-user", "--email", "gone@test.com", "--name", "Gone", "--password-stdin",
	)
	if code != 0 {
		t.Fatalf("create-user: want exit 0, got %d", code)
	}

	_, _, code = runAuth(t, cfgHome, "", "delete-user", "--email", "gone@test.com")
	if code != 0 {
		t.Fatalf("delete-user: want exit 0, got %d", code)
	}

	stdout, _, code := runAuth(t, cfgHome, "", "list-users")
	if code != 0 {
		t.Fatalf("list-users after delete: want exit 0, got %d", code)
	}
	if strings.Contains(stdout, "gone@test.com") {
		t.Errorf("list-users still contains deleted user\ngot: %s", stdout)
	}
}

// TestAuthResetPassword creates a user, resets the password via CLI, then
// authenticates programmatically with the new password.
func TestAuthResetPassword(t *testing.T) {
	cfgHome, dataDir := writeAuthConfig(t)
	_, _, code := runAuth(t, cfgHome, "oldpw123\n",
		"create-user", "--email", "reset@test.com", "--name", "Reset", "--password-stdin",
	)
	if code != 0 {
		t.Fatalf("create-user: want exit 0, got %d", code)
	}

	_, _, code = runAuth(t, cfgHome, "newpw456\n",
		"reset-password", "--email", "reset@test.com", "--password-stdin",
	)
	if code != 0 {
		t.Fatalf("reset-password: want exit 0, got %d", code)
	}

	// Open the auth store directly to verify the new password works.
	store, err := auth.Open(filepath.Join(dataDir, "auth.db"), time.Hour)
	if err != nil {
		t.Fatalf("auth.Open: %v", err)
	}
	defer store.Close()

	u, err := store.Authenticate("reset@test.com", "newpw456")
	if err != nil {
		t.Fatalf("Authenticate with new password: %v", err)
	}
	if u == nil {
		t.Error("Authenticate returned nil for new password after CLI reset")
	}

	// Old password must no longer work.
	u, err = store.Authenticate("reset@test.com", "oldpw123")
	if err != nil {
		t.Fatalf("Authenticate with old password: %v", err)
	}
	if u != nil {
		t.Error("Authenticate returned non-nil for old password after reset")
	}
}

// TestAuthCreateToken creates a user and a bearer token, and asserts the
// token is printed to stdout as a non-empty hex string.
func TestAuthCreateToken(t *testing.T) {
	cfgHome, _ := writeAuthConfig(t)
	_, _, code := runAuth(t, cfgHome, "tokenpass\n",
		"create-user", "--email", "tokenuser@test.com", "--name", "Token", "--password-stdin",
	)
	if code != 0 {
		t.Fatalf("create-user: want exit 0, got %d", code)
	}

	stdout, _, code := runAuth(t, cfgHome, "", "create-token", "--email", "tokenuser@test.com")
	if code != 0 {
		t.Fatalf("create-token: want exit 0, got %d", code)
	}
	token := strings.TrimSpace(stdout)
	if token == "" {
		t.Error("create-token: stdout is empty (expected token plaintext)")
	}
	if len(token) < 64 {
		t.Errorf("token length %d < 64, want ≥64 hex chars", len(token))
	}
}

// TestAuthHelp runs `kaos-control auth --help` and asserts all subcommands
// are listed.
func TestAuthHelp(t *testing.T) {
	cfgHome, _ := writeAuthConfig(t)
	stdout, _, code := runAuth(t, cfgHome, "", "--help")
	if code != 0 {
		t.Fatalf("auth --help: want exit 0, got %d", code)
	}
	for _, sub := range []string{"create-user", "list-users", "delete-user", "reset-password", "create-token"} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("auth --help output missing subcommand %q\ngot: %s", sub, stdout)
		}
	}
}

// TestTopLevelHelp runs `kaos-control --help` and asserts all top-level
// commands including auth are listed.
func TestTopLevelHelp(t *testing.T) {
	stdout, stderr, code := runBin(t, "--help")
	_ = stderr
	if code != 0 {
		t.Fatalf("kaos-control --help: want exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for _, cmd := range []string{"serve", "init", "auth"} {
		if !strings.Contains(stdout, cmd) {
			t.Errorf("top-level --help missing command %q\ngot: %s", cmd, stdout)
		}
	}
}
