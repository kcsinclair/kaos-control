// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	kaoshttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/project"
)

// testEnv holds all the resources for a single integration test.
type testEnv struct {
	t           *testing.T
	projectRoot string
	dataDir     string
	baseURL     string
	cancel      context.CancelFunc
	authStore   *auth.Store
	proj        *project.Project
	cookies     []*http.Cookie // session cookies after login
	csrfToken   string
}

// newTestEnv creates a temp project dir, inits a git repo, writes a lifecycle/config.yaml,
// seeds initial artifacts, starts the full HTTP server on a random port, and returns
// a testEnv ready for API calls. No frontend FS is provided; SPA routes will return 500.
func newTestEnv(t *testing.T, seeds []seedArtifact) *testEnv {
	t.Helper()
	return newTestEnvFull(t, seeds, nil, defaultCfgYAML)
}

// newTestEnvWithFrontend is like newTestEnv but injects a frontend fs.FS so that the
// SPA catch-all handler can serve index.html. Pass an fs.FS whose root contains a
// "dist/index.html" file (e.g. a testing/fstest.MapFS stub).
func newTestEnvWithFrontend(t *testing.T, seeds []seedArtifact, frontendFS fs.FS) *testEnv {
	t.Helper()
	return newTestEnvFull(t, seeds, frontendFS, defaultCfgYAML)
}

// defaultCfgYAML is the lifecycle/config.yaml used by newTestEnv.
// Tests that need a different project config should use newTestEnvWithCfgYAML.
const defaultCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
  - reviewer
  - approver

stages:
  - {name: ideas, dir: ideas}
  - {name: requirements, dir: requirements}
  - {name: backend-plans, dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans, dir: test-plans}
  - {name: tests, dir: tests}
  - {name: prototypes, dir: prototypes}
  - {name: releases, dir: releases}
  - {name: sprints, dir: sprints}
  - {name: defects, dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// newTestEnvWithCfgYAML is like newTestEnv but uses a custom lifecycle/config.yaml.
// Auth store users (admin@test.local, dev@test.local, qa@test.local) are still created
// so that env.login() works regardless of the project-level users configuration.
func newTestEnvWithCfgYAML(t *testing.T, seeds []seedArtifact, cfgYAML string) *testEnv {
	t.Helper()
	return newTestEnvFull(t, seeds, nil, cfgYAML)
}

// newTestEnvFull is the shared implementation for newTestEnv and newTestEnvWithFrontend.
func newTestEnvFull(t *testing.T, seeds []seedArtifact, frontendFS fs.FS, cfgYAML string) *testEnv {
	t.Helper()

	root := t.TempDir()
	dataDir := t.TempDir()

	// Create lifecycle directories.
	stages := []string{
		"ideas", "requirements", "backend-plans", "frontend-plans",
		"test-plans", "tests", "prototypes", "releases", "sprints", "defects",
	}
	for _, s := range stages {
		if err := os.MkdirAll(filepath.Join(root, "lifecycle", s), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Write lifecycle/config.yaml with the provided project config.
	if err := os.WriteFile(filepath.Join(root, "lifecycle", "config.yaml"), []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Init git repo.
	repo, err := git.PlainInit(root, false)
	if err != nil {
		t.Fatal(err)
	}
	// Configure git user for commits.
	cfg, _ := repo.Config()
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@test.local"
	if err := repo.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Create initial commit so the repo has a HEAD ref.
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	_, err = wt.Add("lifecycle/config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	// Seed artifacts.
	for _, s := range seeds {
		absPath := filepath.Join(root, s.relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absPath, []byte(s.content), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err = wt.Add(s.relPath)
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err = wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@test.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Ensure the branch is named "main".
	ref, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	if ref.Name().Short() != "main" {
		err = repo.CreateBranch(&gitconfig.Branch{Name: "main", Remote: ""})
		_ = err // might already be "main"
	}

	// Open auth store.
	authDBPath := filepath.Join(dataDir, "auth.db")
	authStore, err := auth.Open(authDBPath, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { authStore.Close() })

	// Create test users.
	for _, u := range []struct{ email, name, pass string }{
		{"admin@test.local", "Admin", "admin-pass-123"},
		{"dev@test.local", "Developer", "dev-pass-123"},
		{"qa@test.local", "QA Engineer", "qa-pass-123"},
	} {
		if err := authStore.CreateUser(u.email, u.name, u.pass, false); err != nil {
			t.Fatal(err)
		}
	}

	// Open project.
	entry := &config.ProjectEntry{
		Name:        "testproject",
		Path:        root,
		Description: "integration test project",
	}
	// In tests, dataDir is the t.TempDir() — pass it explicitly as DevopsLogDir
	// so devops run logs land inside the temp dir (not in its parent, where the
	// production default would put them when dataDir == appHome/data).
	proj, err := project.Open(entry, dataDir, project.OpenOptions{
		MaxConcurrentAgents: 2,
		DevopsLogDir:        dataDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { proj.Close() })

	// Start watcher.
	ctx, cancel := context.WithCancel(context.Background())
	proj.StartWatcher(ctx)
	proj.StartLockReaper(ctx)

	// Find a free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	addr := ln.Addr().String()

	// Start HTTP server.
	srv := kaoshttp.New(kaoshttp.ServerConfig{
		Listener: ln,
		Auth:     authStore,
		Frontend: frontendFS,
	}, map[string]*project.Project{
		"testproject": proj,
	})

	srvDone := make(chan error, 1)
	go func() {
		srvDone <- srv.ListenAndServe(ctx)
	}()

	// Wait for server to be ready.
	baseURL := "http://" + addr
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
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

	// Auto-login as admin so every test starts authenticated. Tests that need
	// to verify auth-failure behaviour can call e.logout() or e.login(...) to
	// switch identity. Tests that need anonymous access can clear cookies via
	// e.cookies = nil.
	env.login("admin@test.local", "admin-pass-123")

	return env
}

// seedArtifact defines a file to pre-populate in the project.
type seedArtifact struct {
	relPath string
	content string
}

// makeArtifact builds a markdown artifact string from frontmatter fields and body.
func makeArtifact(title, typ, status, lineage, parent, body string, labels ...string) string {
	var sb bytes.Buffer
	sb.WriteString("---\n")
	sb.WriteString("title: " + title + "\n")
	sb.WriteString("type: " + typ + "\n")
	sb.WriteString("status: " + status + "\n")
	sb.WriteString("lineage: " + lineage + "\n")
	if parent != "" {
		sb.WriteString("parent: " + parent + "\n")
	}
	if len(labels) > 0 {
		sb.WriteString("labels:\n")
		for _, l := range labels {
			sb.WriteString("    - " + l + "\n")
		}
	}
	sb.WriteString("---\n\n")
	sb.WriteString(body + "\n")
	return sb.String()
}

// makeBlockedArtifact builds a markdown artifact with status "blocked" and an
// "## Open Questions" section in its body so that the autoblock rule in
// internal/index/autoblock.go does not fire and auto-transition the artifact
// back to "draft" during indexing.
func makeBlockedArtifact(title, typ, lineage, parent string, labels ...string) string {
	body := "Body.\n\n## Open Questions\n- Why is the sky blue?"
	return makeArtifact(title, typ, "blocked", lineage, parent, body, labels...)
}

// makeArtifactWithPriority is like makeArtifact but also sets the priority field.
// Pass priority="" to omit the field entirely.
func makeArtifactWithPriority(title, typ, status, lineage, priority, body string, labels ...string) string {
	var sb bytes.Buffer
	sb.WriteString("---\n")
	sb.WriteString("title: " + title + "\n")
	sb.WriteString("type: " + typ + "\n")
	sb.WriteString("status: " + status + "\n")
	sb.WriteString("lineage: " + lineage + "\n")
	if priority != "" {
		sb.WriteString("priority: " + priority + "\n")
	}
	if len(labels) > 0 {
		sb.WriteString("labels:\n")
		for _, l := range labels {
			sb.WriteString("    - " + l + "\n")
		}
	}
	sb.WriteString("---\n\n")
	sb.WriteString(body + "\n")
	return sb.String()
}

// findNodeByID locates a graph node by its ID (path) in the nodes slice.
// Returns nil if not found.
func findNodeByID(nodes []any, id string) map[string]any {
	for _, n := range nodes {
		node, _ := n.(map[string]any)
		if node["id"] == id {
			return node
		}
	}
	return nil
}

// logout clears any session/CSRF state on the test env so subsequent
// doRequest calls go out anonymously. Useful in tests that verify the
// 401 path; pairs with the auto-login in newTestEnvFull.
func (e *testEnv) logout() {
	e.cookies = nil
	e.csrfToken = ""
}

// login authenticates against the API and saves cookies + CSRF token.
func (e *testEnv) login(email, password string) {
	e.t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	resp, err := http.Post(e.baseURL+"/api/auth/login", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		e.t.Fatalf("login failed: %d %s", resp.StatusCode, b)
	}
	e.cookies = resp.Cookies()
	for _, c := range e.cookies {
		if c.Name == "kc_csrf" {
			e.csrfToken = c.Value
		}
	}
}

// doRequest makes an HTTP request with session cookies and CSRF token.
func (e *testEnv) doRequest(method, path string, body any) *http.Response {
	e.t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.baseURL+path, bodyReader)
	if err != nil {
		e.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range e.cookies {
		req.AddCookie(c)
	}
	if method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("X-CSRF-Token", e.csrfToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	return resp
}

// readJSON reads the response body and decodes into a map.
func readJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("JSON unmarshal failed on %q: %v", string(b), err)
	}
	return m
}

// stringReader creates an io.Reader from a string for use in http.Post.
func stringReader(s string) io.Reader {
	return bytes.NewReader([]byte(s))
}

// requireStatus asserts the HTTP response has the expected status code.
// IMPORTANT: call this BEFORE readJSON, or on responses you don't need to parse.
// On mismatch it reads and closes the body, then calls t.Fatalf.
func requireStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected status %d, got %d: %s", want, resp.StatusCode, string(b))
	}
}
