// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

// repoRoot returns the absolute path to the repository root by walking up from
// this source file's location. It relies on runtime.Caller which resolves the
// path at compile time — safe for both `go test` and `go test -count=1`.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// file is .../tests/integration/version_test.go → go up two levels
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// ── Milestone 1 — Backend endpoint tests ─────────────────────────────────────

// TestVersion_Returns200WithJSON verifies that GET /api/version responds with
// HTTP 200, Content-Type application/json, and a body of
// {"version": "<non-empty string>"}.
// Covers test plan Milestone 1, scenario 1.
func TestVersion_Returns200WithJSON(t *testing.T) {
	env := newTestEnv(t, nil)

	resp, err := http.Get(env.baseURL + "/api/version")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)

	ct := resp.Header.Get("Content-Type")
	if ct == "" || ct[:16] != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	data := readJSON(t, resp)
	version, ok := data["version"].(string)
	if !ok || version == "" {
		t.Errorf("expected non-empty string for 'version' key, got %v", data["version"])
	}
}

// TestVersion_Unauthenticated verifies that GET /api/version returns 200
// without any session cookie — the endpoint is intentionally public.
// Covers test plan Milestone 1, scenario 2.
func TestVersion_Unauthenticated(t *testing.T) {
	env := newTestEnv(t, nil)

	// Use a plain http.Get so no session cookies are sent.
	resp, err := http.Get(env.baseURL + "/api/version")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for unauthenticated /api/version, got %d", resp.StatusCode)
	}
}

// TestHealth_IncludesVersion verifies that GET /api/health contains a
// non-empty "version" key in its JSON response.
// Covers test plan Milestone 1, scenario 3.
func TestHealth_IncludesVersion(t *testing.T) {
	env := newTestEnv(t, nil)

	resp, err := http.Get(env.baseURL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	version, ok := data["version"].(string)
	if !ok || version == "" {
		t.Errorf("expected non-empty string for 'version' key in /api/health, got %v", data["version"])
	}
}

// TestVersion_ParityWithHealth verifies that the version string returned by
// GET /api/version and GET /api/health are identical.
// Covers test plan Milestone 1, scenario 4.
func TestVersion_ParityWithHealth(t *testing.T) {
	env := newTestEnv(t, nil)

	respV, err := http.Get(env.baseURL + "/api/version")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, respV, 200)
	versionData := readJSON(t, respV)

	respH, err := http.Get(env.baseURL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, respH, 200)
	healthData := readJSON(t, respH)

	vFromVersion, _ := versionData["version"].(string)
	vFromHealth, _ := healthData["version"].(string)

	if vFromVersion == "" {
		t.Fatal("version from /api/version is empty")
	}
	if vFromHealth == "" {
		t.Fatal("version from /api/health is empty")
	}
	if vFromVersion != vFromHealth {
		t.Errorf("/api/version reports %q but /api/health reports %q", vFromVersion, vFromHealth)
	}
}

// ── Milestone 2 — Build integration: VERSION file format ─────────────────────

// semverRe matches bare semver strings (no leading "v", no extra whitespace).
var semverRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// TestVersionFile_ExistsAndIsValidSemver asserts that a VERSION file exists at
// the repository root and that its contents match the bare semver pattern
// expected by the build system (e.g. "0.1.0", not "v0.1.0").
// Covers test plan Milestone 2, scenario 1.
func TestVersionFile_ExistsAndIsValidSemver(t *testing.T) {
	root := repoRoot(t)
	versionPath := filepath.Join(root, "VERSION")

	raw, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatalf("VERSION file not found at %s: %v", versionPath, err)
	}

	// Trim any trailing newline / carriage return before matching.
	content := string(raw)
	for len(content) > 0 && (content[len(content)-1] == '\n' || content[len(content)-1] == '\r') {
		content = content[:len(content)-1]
	}

	if !semverRe.MatchString(content) {
		t.Errorf("VERSION file contents %q do not match bare semver pattern (e.g. 0.1.0)", content)
	}
}
