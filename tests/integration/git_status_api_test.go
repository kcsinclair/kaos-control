// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// TestGitStatusHappyPath, TestGitStatusDirtyTree, TestGitStatusNonGitProject,
// TestGitStatusPerformance — REST endpoint tests for GET /api/p/{project}/git/status.
//
// Covers test-plan milestones:
//   M1-TC1  happy path: git-backed project → 200, all fields present and valid
//   M1-TC2  dirty tree: untracked file → dirty=true
//   M1-TC3  non-git project → 200, body is exactly {"available":false}
//   M1-TC4  performance: endpoint responds in < 100 ms (NFR1)

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	kaoshttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/project"
)

// TestGitStatusHappyPath verifies the full response shape for a clean git-backed project.
func TestGitStatusHappyPath(t *testing.T) {
	env := newTestEnv(t, nil)

	resp := env.doRequest("GET", "/api/p/testproject/git/status", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	// available must be true.
	avail, ok := body["available"].(bool)
	if !ok || !avail {
		t.Fatalf("expected available=true, got %v", body["available"])
	}

	// branch must be a non-empty string.
	branch, ok := body["branch"].(string)
	if !ok || branch == "" {
		t.Errorf("expected non-empty branch, got %v", body["branch"])
	}

	// dirty must be false for a freshly-committed repo.
	dirty, ok := body["dirty"].(bool)
	if !ok {
		t.Errorf("expected bool dirty field, got %T %v", body["dirty"], body["dirty"])
	} else if dirty {
		t.Errorf("expected dirty=false for clean working tree, got true")
	}

	// head_sha must be exactly 7 lowercase hex characters.
	headSHA, ok := body["head_sha"].(string)
	if !ok || len(headSHA) != 7 {
		t.Errorf("expected 7-char head_sha, got %q", headSHA)
	}
	if matched, _ := regexp.MatchString(`^[0-9a-f]{7}$`, headSHA); !matched {
		t.Errorf("head_sha %q is not 7 lowercase hex chars", headSHA)
	}

	// head_message must be non-empty and contain no newlines (first line only).
	headMsg, ok := body["head_message"].(string)
	if !ok || headMsg == "" {
		t.Errorf("expected non-empty head_message, got %v", body["head_message"])
	}
	for _, c := range headMsg {
		if c == '\n' || c == '\r' {
			t.Errorf("head_message must be single line, got %q", headMsg)
			break
		}
	}

	// head_author must be non-empty.
	headAuthor, ok := body["head_author"].(string)
	if !ok || headAuthor == "" {
		t.Errorf("expected non-empty head_author, got %v", body["head_author"])
	}

	// head_when must be a valid ISO 8601 / RFC 3339 timestamp.
	headWhen, ok := body["head_when"].(string)
	if !ok || headWhen == "" {
		t.Errorf("expected non-empty head_when, got %v", body["head_when"])
	}
	if _, err := time.Parse(time.RFC3339, headWhen); err != nil {
		t.Errorf("head_when %q is not a valid RFC3339/ISO 8601 timestamp: %v", headWhen, err)
	}
}

// TestGitStatusDirtyTree verifies that dirty=true is returned when the working tree
// has an untracked file (not yet committed).
func TestGitStatusDirtyTree(t *testing.T) {
	env := newTestEnv(t, nil)

	// Write an untracked file directly into the lifecycle directory.
	dirtyFile := filepath.Join(env.projectRoot, "lifecycle", "ideas", "dirty-tree-probe.md")
	if err := os.WriteFile(dirtyFile, []byte("untracked content for dirty-tree test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("GET", "/api/p/testproject/git/status", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	dirty, ok := body["dirty"].(bool)
	if !ok || !dirty {
		t.Errorf("expected dirty=true after writing untracked file, got %v", body["dirty"])
	}
}

// TestGitStatusNonGitProject verifies that a plain (non-git) project directory returns
// HTTP 200 with exactly {"available":false} and no additional fields.
func TestGitStatusNonGitProject(t *testing.T) {
	env := newNonGitTestEnv(t)

	resp := env.doRequest("GET", "/api/p/testproject/git/status", nil)
	requireStatus(t, resp, http.StatusOK)

	defer resp.Body.Close()
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal(rawBody, &m); err != nil {
		t.Fatalf("JSON unmarshal failed on %q: %v", string(rawBody), err)
	}

	avail, ok := m["available"].(bool)
	if !ok || avail {
		t.Errorf("expected available=false for non-git project, got %v", m["available"])
	}

	if len(m) != 1 {
		t.Errorf("expected response with only {\"available\":false}, got %d fields: %v", len(m), m)
	}
}

// TestGitStatusPerformance validates NFR1: the git status endpoint responds in under 100 ms.
// The implementation uses O(1) commit operations (no history walk), so this should hold
// regardless of repository history depth.
func TestGitStatusPerformance(t *testing.T) {
	env := newTestEnv(t, nil)

	start := time.Now()
	resp := env.doRequest("GET", "/api/p/testproject/git/status", nil)
	elapsed := time.Since(start)

	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	const maxLatency = 100 * time.Millisecond
	if elapsed > maxLatency {
		t.Errorf("git status endpoint took %v, want < %v (NFR1)", elapsed, maxLatency)
	}
}

// newNonGitTestEnv creates a minimal integration test environment backed by a plain
// (non-git) project directory. The resulting project.Git field will be nil. This
// helper is intentionally self-contained — it does not call newTestEnvFull — so it
// never runs git.PlainInit.
func newNonGitTestEnv(t *testing.T) *testEnv {
	t.Helper()

	root := t.TempDir()
	dataDir := t.TempDir()

	// Create lifecycle directories (mirrors newTestEnvFull).
	stages := []string{
		"ideas", "requirements", "backend-plans", "frontend-plans",
		"test-plans", "tests", "prototypes", "releases", "sprints", "defects",
	}
	for _, s := range stages {
		if err := os.MkdirAll(filepath.Join(root, "lifecycle", s), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "lifecycle", "config.yaml"), []byte(defaultCfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Deliberately skip git.PlainInit — project.Git will be nil.

	authDBPath := filepath.Join(dataDir, "auth.db")
	authStore, err := auth.Open(authDBPath, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { authStore.Close() })

	for _, u := range []struct{ email, name, pass string }{
		{"admin@test.local", "Admin", "admin-pass-123"},
		{"dev@test.local", "Developer", "dev-pass-123"},
		{"qa@test.local", "QA Engineer", "qa-pass-123"},
	} {
		if err := authStore.CreateUser(u.email, u.name, u.pass, false); err != nil {
			t.Fatal(err)
		}
	}

	entry := &config.ProjectEntry{
		Name:        "testproject",
		Path:        root,
		Description: "non-git integration test project",
	}
	proj, err := project.Open(entry, dataDir, project.OpenOptions{
		MaxConcurrentAgents: 2,
		DevopsLogDir:        dataDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { proj.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	proj.StartWatcher(ctx)
	proj.StartLockReaper(ctx)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	addr := ln.Addr().String()

	srv := kaoshttp.New(kaoshttp.ServerConfig{
		Listener: ln,
		Auth:     authStore,
		Frontend: nil,
	}, map[string]*project.Project{
		"testproject": proj,
	})

	srvDone := make(chan error, 1)
	go func() {
		srvDone <- srv.ListenAndServe(ctx)
	}()

	baseURL := "http://" + addr
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		r, err := http.Get(baseURL + "/api/health")
		if err == nil {
			r.Body.Close()
			if r.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}

	env := &testEnv{
		t:           t,
		projectRoot: root,
		dataDir:     dataDir,
		baseURL:     baseURL,
		cancel:      cancel,
		authStore:   authStore,
		proj:        proj,
	}

	t.Cleanup(func() {
		cancel()
		select {
		case <-srvDone:
		case <-time.After(5 * time.Second):
		}
	})

	env.login("admin@test.local", "admin-pass-123")
	return env
}
