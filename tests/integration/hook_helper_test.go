// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 4 — Hook Helper Integration Tests
//
// End-to-end tests for the `kaos-control hook-helper` subcommand. The tests
// build the binary once (lazily, via sync.Once) and exercise the full
// stdin → HTTP → stdout flow without a real Claude Code installation.
//
// Run with:
//   go test ./tests/... -tags integration -run TestHookHelper -v

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// hookBinOnce ensures the kaos-control binary is built only once per test-suite
// run, shared across all TestHookHelper_* tests.
var (
	hookBinOnce sync.Once
	hookBinPath string
	hookBinErr  error
)

// requireHookBin returns the path to the kaos-control binary, building it if
// necessary. Skips the test if the binary could not be built.
func requireHookBin(t *testing.T) string {
	t.Helper()
	hookBinOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "kc-hook-bin-*")
		if err != nil {
			hookBinErr = err
			return
		}
		bin := filepath.Join(tmpDir, "kaos-control")
		cmd := exec.Command("go", "build",
			"-o", bin,
			"github.com/kaos-control/kaos-control/cmd/kaos-control",
		)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			hookBinErr = err
			return
		}
		hookBinPath = bin
	})
	if hookBinErr != nil {
		t.Skipf("skipping: could not build kaos-control binary: %v", hookBinErr)
	}
	return hookBinPath
}

// runHookHelper executes `<bin> hook-helper --server <addr> --run-id <id>` with
// the given secret injected as KC_HOOK_SECRET and the given bytes piped to stdin.
// It returns (stdout output, exit code).
func runHookHelper(t *testing.T, bin, serverAddr, runID, secret string, stdinData []byte) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, "hook-helper",
		"--server", serverAddr,
		"--run-id", runID,
	)
	cmd.Stdin = bytes.NewReader(stdinData)

	// Inherit environment but inject (or override) KC_HOOK_SECRET.
	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "KC_HOOK_SECRET=") {
			env = append(env, e)
		}
	}
	if secret != "" {
		env = append(env, "KC_HOOK_SECRET="+secret)
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
	}
	t.Logf("hook-helper stdout: %s", stdout.String())
	t.Logf("hook-helper stderr: %s", stderr.String())
	return stdout.String(), code
}

// toolCallJSON is a helper that marshals a tool-call object to JSON bytes.
func toolCallJSON(toolName string, toolInput map[string]any) []byte {
	b, _ := json.Marshal(map[string]any{
		"tool_name":  toolName,
		"tool_input": toolInput,
	})
	return b
}

// ── Happy path ────────────────────────────────────────────────────────────────

// TestHookHelper_HappyPath verifies the full stdin → HTTP → stdout flow when
// the server returns allow (AC2).
func TestHookHelper_HappyPath(t *testing.T) {
	bin := requireHookBin(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}`))
	}))
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")
	stdin := toolCallJSON("Write", map[string]any{"file_path": "lifecycle/requirements/foo.md"})

	stdout, exitCode := runHookHelper(t, bin, addr, "test-run-id", "my-secret", stdin)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("stdout is not valid JSON: %v (stdout=%q)", err, stdout)
	}
	inner, _ := resp["hookSpecificOutput"].(map[string]any)
	if dec, _ := inner["permissionDecision"].(string); dec != "allow" {
		t.Errorf("permissionDecision = %q, want allow", dec)
	}
}

// TestHookHelper_PassesDenyDecision verifies that a deny response from the
// server is translated into the Claude-native hookSpecificOutput shape and
// exit code is still 0.
func TestHookHelper_PassesDenyDecision(t *testing.T) {
	bin := requireHookBin(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"not allowed"}}`))
	}))
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")
	stdin := toolCallJSON("Write", map[string]any{"file_path": "web/src/App.vue"})

	stdout, exitCode := runHookHelper(t, bin, addr, "run-deny", "my-secret", stdin)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 even on deny decision", exitCode)
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("stdout is not valid JSON: %v (stdout=%q)", err, stdout)
	}
	inner, _ := resp["hookSpecificOutput"].(map[string]any)
	if dec, _ := inner["permissionDecision"].(string); dec != "deny" {
		t.Errorf("permissionDecision = %q, want deny", dec)
	}
}

// TestHookHelper_ForwardsSecret verifies that the hook-helper includes the
// KC_HOOK_SECRET in the Authorization header sent to the server.
func TestHookHelper_ForwardsSecret(t *testing.T) {
	bin := requireHookBin(t)

	const wantSecret = "supersecret-123"
	gotSecret := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}`))
	}))
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")
	stdin := toolCallJSON("Read", map[string]any{})

	runHookHelper(t, bin, addr, "run-secret-check", wantSecret, stdin)

	if gotSecret != wantSecret {
		t.Errorf("server received Authorization Bearer %q, want %q", gotSecret, wantSecret)
	}
}

// ── Server unreachable ────────────────────────────────────────────────────────

// TestHookHelper_ServerUnreachable verifies that when the permission server is
// not listening, the helper retries once and then returns deny (NFR2).
func TestHookHelper_ServerUnreachable(t *testing.T) {
	bin := requireHookBin(t)

	// Use a port that is not listening (connection-refused).
	addr := "127.0.0.1:29187"

	stdin := toolCallJSON("Write", map[string]any{"file_path": "lifecycle/requirements/foo.md"})

	start := time.Now()
	stdout, exitCode := runHookHelper(t, bin, addr, "run-unreachable", "my-secret", stdin)
	elapsed := time.Since(start)

	// Exit code must always be 0 regardless of failure mode.
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 for unreachable server", exitCode)
	}

	// Should finish within 5 s (initial attempt + 500 ms retry + overhead).
	if elapsed > 5*time.Second {
		t.Errorf("took %v, want < 5s for unreachable server", elapsed)
	}

	// Response must be JSON in the Claude-native hookSpecificOutput shape.
	var resp map[string]any
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("stdout is not valid JSON: %v (stdout=%q)", err, stdout)
	}
	inner, _ := resp["hookSpecificOutput"].(map[string]any)
	if dec, _ := inner["permissionDecision"].(string); dec != "deny" {
		t.Errorf("decision = %q, want deny when server is unreachable", dec)
	}
	reason, _ := inner["permissionDecisionReason"].(string)
	if !strings.Contains(strings.ToLower(reason), "unreachable") &&
		!strings.Contains(strings.ToLower(reason), "server") {
		t.Errorf("reason = %q, expected to mention unreachable server", reason)
	}
}

// ── Missing secret ────────────────────────────────────────────────────────────

// TestHookHelper_MissingSecret verifies that when KC_HOOK_SECRET is not set,
// the helper writes a deny response and exits 0.
func TestHookHelper_MissingSecret(t *testing.T) {
	bin := requireHookBin(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not be reached when the secret is missing.
		_, _ = w.Write([]byte(`{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}`))
	}))
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")
	stdin := toolCallJSON("Read", map[string]any{})

	// Run without KC_HOOK_SECRET by passing empty string to runHookHelper.
	stdout, exitCode := runHookHelper(t, bin, addr, "run-nosecret", "" /* empty = not set */, stdin)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 even without secret", exitCode)
	}
	if stdout == "" {
		t.Fatal("expected non-empty stdout when secret is missing")
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("stdout is not valid JSON: %v (stdout=%q)", err, stdout)
	}
	inner, _ := resp["hookSpecificOutput"].(map[string]any)
	if inner == nil {
		t.Error("response must contain a 'hookSpecificOutput' key")
	}
	// With no secret, the hook-helper should deny (not allow), since it cannot
	// authenticate the request.
	if dec, _ := inner["permissionDecision"].(string); dec != "deny" {
		t.Errorf("decision = %q, want deny when secret is missing", dec)
	}
}

// ── Malformed stdin ───────────────────────────────────────────────────────────

// TestHookHelper_MalformedStdin verifies that piping invalid JSON to stdin does
// not crash the helper (exit code 0) and produces output (valid JSON or
// forwarded server response).
func TestHookHelper_MalformedStdin(t *testing.T) {
	bin := requireHookBin(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The hook-helper forwards raw stdin bytes; the server may return any JSON.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"bad_request"}}`))
	}))
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")

	// Not valid JSON.
	stdin := []byte("not { valid } json!!!")

	_, exitCode := runHookHelper(t, bin, addr, "run-malformed", "my-secret", stdin)

	// The helper must not crash — exit code must be 0.
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 for malformed stdin", exitCode)
	}
}

// TestHookHelper_ExitCodeAlwaysZero verifies exit code is 0 regardless of the
// server's decision (required by the Claude Code hook contract).
func TestHookHelper_ExitCodeAlwaysZero(t *testing.T) {
	bin := requireHookBin(t)

	for _, decision := range []string{"allow", "deny"} {
		t.Run(decision, func(t *testing.T) {
			dec := decision
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"` + dec + `"}}`))
			}))
			defer srv.Close()

			addr := strings.TrimPrefix(srv.URL, "http://")
			stdin := toolCallJSON("Write", map[string]any{"file_path": "lifecycle/requirements/foo.md"})

			_, exitCode := runHookHelper(t, bin, addr, "run-exit-"+dec, "my-secret", stdin)
			if exitCode != 0 {
				t.Errorf("exit code = %d, want 0 for decision=%q", exitCode, dec)
			}
		})
	}
}
