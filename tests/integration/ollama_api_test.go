// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	kaoshttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/tests/integration/testutil"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ── Milestone 3 — Ollama API Endpoint Tests ────────────────────────────────────

// ollamaAPIEnv extends testEnv with access to the mutable app config and its path.
type ollamaAPIEnv struct {
	*testEnv
	appCfg     *config.App
	appCfgPath string
	mock       *testutil.MockOllamaServer
}

// newOllamaAPITestEnv builds a full integration environment with the Ollama
// instance management API available. It starts:
//   - A mock Ollama HTTP server for health/model proxy tests.
//   - A minimal project (for the "delete referenced" test to use).
//   - The kaos-control HTTP server with AppCfg and AppCfgPath configured.
//
// The mock server URL is registered as instance "mock-ollama" in the initial
// app config. Tests that need a blank slate should start with an env whose
// initial instances list is empty.
func newOllamaAPITestEnv(t *testing.T, initialInstances []config.OllamaInstance) *ollamaAPIEnv {
	t.Helper()

	mock := testutil.NewMockOllamaServer()
	t.Cleanup(func() { mock.Close() })

	// Merge the mock instance into the initial list.
	instances := append([]config.OllamaInstance{{
		Name:    "mock-ollama",
		BaseURL: mock.URL(),
	}}, initialInstances...)

	// Build a valid app config.
	appCfg := &config.App{
		Server: config.ServerConfig{Listen: "127.0.0.1:0"},
		Auth:   config.AuthConfig{Method: "local", SessionTTL: 24 * time.Hour},
		Limits: config.LimitsConfig{
			MaxConcurrentAgents:        4,
			MaxConcurrentSchedulerJobs: 2,
			SchedulerRunRetentionDays:  90,
		},
		OllamaInstances: instances,
	}

	// Write initial app config to a temp file.
	cfgDir := t.TempDir()
	appCfgPath := filepath.Join(cfgDir, "config.yaml")
	if err := config.SaveApp(appCfgPath, *appCfg); err != nil {
		t.Fatalf("writing initial app config: %v", err)
	}

	// Create a project that references "mock-ollama" (used by the
	// "delete referenced instance" test).
	projRoot, dataDir := setupOllamaProject(t, mock.URL())

	// Open the project.
	entry := &config.ProjectEntry{
		Name:        "testproject",
		Path:        projRoot,
		Description: "ollama api integration test project",
	}
	proj, err := project.Open(entry, dataDir, project.OpenOptions{
		MaxConcurrentAgents: 2,
		OllamaInstances:     appCfg.OllamaInstances,
	})
	if err != nil {
		t.Fatalf("project.Open: %v", err)
	}
	t.Cleanup(func() { proj.Close() })

	// Auth store.
	authDBPath := filepath.Join(dataDir, "auth.db")
	authStore, err := auth.Open(authDBPath, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { authStore.Close() })
	for _, u := range []struct{ email, name, pass string }{
		{"admin@test.local", "Admin", "admin-pass-123"},
		{"dev@test.local", "Developer", "dev-pass-123"},
	} {
		if err := authStore.CreateUser(u.email, u.name, u.pass, false); err != nil {
			t.Fatal(err)
		}
	}

	// Watcher / reaper.
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
	ln.Close()

	// Start HTTP server with AppCfg wired.
	srv := kaoshttp.New(kaoshttp.ServerConfig{
		Listen:     addr,
		Auth:       authStore,
		AppCfg:     appCfg,
		AppCfgPath: appCfgPath,
	}, map[string]*project.Project{
		"testproject": proj,
	})

	srvDone := make(chan error, 1)
	go func() { srvDone <- srv.ListenAndServe(ctx) }()

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
		projectRoot: projRoot,
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

	return &ollamaAPIEnv{testEnv: env, appCfg: appCfg, appCfgPath: appCfgPath, mock: mock}
}

// setupOllamaProject creates a minimal git project that has an ollama agent
// referencing "mock-ollama" (to test the "delete referenced" endpoint).
func setupOllamaProject(t *testing.T, mockURL string) (root, dataDir string) {
	t.Helper()
	root = t.TempDir()
	dataDir = t.TempDir()

	// Create lifecycle dirs.
	for _, s := range []string{"ideas", "requirements", "backend-plans", "frontend-plans",
		"test-plans", "tests", "prototypes", "releases", "sprints", "defects"} {
		if err := os.MkdirAll(filepath.Join(root, "lifecycle", s), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cfgYAML := `
git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles: [product-owner, analyst]

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
    roles: [product-owner, analyst]

agents:
  - name: ollama-ref-agent
    role: [analyst]
    driver: ollama
    model: testmodel:latest
    ollama_instance: mock-ollama
    ollama_endpoint: chat
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Ollama Ref Agent
      email: ollama-ref@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`
	cfgPath := filepath.Join(root, "lifecycle", "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Init git repo with initial commit.
	repo, err := git.PlainInit(root, false)
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := repo.Config()
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@test.local"
	if err := repo.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("lifecycle/config.yaml"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@test.local", When: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}
	// Ensure branch is "main".
	ref, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	if ref.Name().Short() != "main" {
		_ = repo.CreateBranch(&gitconfig.Branch{Name: "main", Remote: ""})
	}

	return root, dataDir
}

// ── Test cases ────────────────────────────────────────────────────────────────

// TestOllamaInstances_List verifies GET /api/ollama/instances returns configured
// instances with api_key masked.
func TestOllamaInstances_List(t *testing.T) {
	env := newOllamaAPITestEnv(t, []config.OllamaInstance{
		{Name: "keyed-instance", BaseURL: "http://other.local:11434", APIKey: "my-secret"},
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/ollama/instances", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	instancesRaw, ok := data["instances"].([]any)
	if !ok || len(instancesRaw) == 0 {
		t.Fatalf("expected non-empty instances list, got: %v", data["instances"])
	}

	// Build a lookup by name.
	byName := make(map[string]map[string]any)
	for _, raw := range instancesRaw {
		inst, _ := raw.(map[string]any)
		name, _ := inst["name"].(string)
		byName[name] = inst
	}

	// keyed-instance must have api_key masked as "***".
	ki, ok := byName["keyed-instance"]
	if !ok {
		t.Fatal("keyed-instance not found in list response")
	}
	if apiKey, _ := ki["api_key"].(string); apiKey != "***" {
		t.Errorf("api_key: want %q, got %q", "***", apiKey)
	}

	// mock-ollama has no api_key — field should be absent or empty.
	mo, ok := byName["mock-ollama"]
	if !ok {
		t.Fatal("mock-ollama not found in list response")
	}
	if apiKey, exists := mo["api_key"]; exists && apiKey != "" {
		t.Errorf("mock-ollama api_key should be absent/empty, got %q", apiKey)
	}
}

// TestOllamaInstances_Create verifies that POST creates a new instance and a
// subsequent GET confirms persistence.
func TestOllamaInstances_Create(t *testing.T) {
	env := newOllamaAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/ollama/instances", map[string]any{
		"name":     "new-instance",
		"base_url": "http://new.local:11434",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	inst, _ := data["instance"].(map[string]any)
	if name, _ := inst["name"].(string); name != "new-instance" {
		t.Errorf("created instance name: got %q, want %q", name, "new-instance")
	}

	// Re-fetch to confirm persistence.
	resp2 := env.doRequest("GET", "/api/ollama/instances", nil)
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)
	instancesRaw, _ := data2["instances"].([]any)
	found := false
	for _, raw := range instancesRaw {
		inst, _ := raw.(map[string]any)
		if inst["name"] == "new-instance" {
			found = true
		}
	}
	if !found {
		t.Error("new-instance not found in list after create")
	}
}

// TestOllamaInstances_CreateDuplicate verifies that POST with an existing name
// returns 409.
func TestOllamaInstances_CreateDuplicate(t *testing.T) {
	env := newOllamaAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// mock-ollama already exists in the initial config.
	resp := env.doRequest("POST", "/api/ollama/instances", map[string]any{
		"name":     "mock-ollama",
		"base_url": "http://some.local:11434",
	})
	requireStatus(t, resp, 409)
	data := readJSON(t, resp)
	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "conflict" {
		t.Errorf("error code: got %q, want %q", code, "conflict")
	}
}

// TestOllamaInstances_Update verifies that PUT updates the base_url of an instance.
func TestOllamaInstances_Update(t *testing.T) {
	env := newOllamaAPITestEnv(t, []config.OllamaInstance{
		{Name: "updatable", BaseURL: "http://old.local:11434"},
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("PUT", "/api/ollama/instances/updatable", map[string]any{
		"base_url": "http://new.local:11434",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	inst, _ := data["instance"].(map[string]any)
	if baseURL, _ := inst["base_url"].(string); baseURL != "http://new.local:11434" {
		t.Errorf("updated base_url: got %q, want %q", baseURL, "http://new.local:11434")
	}
}

// TestOllamaInstances_Delete verifies that DELETE removes the instance and a
// subsequent GET confirms it is gone.
func TestOllamaInstances_Delete(t *testing.T) {
	env := newOllamaAPITestEnv(t, []config.OllamaInstance{
		{Name: "deletable", BaseURL: "http://deletable.local:11434"},
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("DELETE", "/api/ollama/instances/deletable", nil)
	requireStatus(t, resp, 200)

	// Confirm removal.
	resp2 := env.doRequest("GET", "/api/ollama/instances", nil)
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)
	instancesRaw, _ := data2["instances"].([]any)
	for _, raw := range instancesRaw {
		inst, _ := raw.(map[string]any)
		if inst["name"] == "deletable" {
			t.Error("deletable instance still present after delete")
		}
	}
}

// TestOllamaInstances_DeleteReferenced verifies that DELETE returns 409 when a
// project agent still references the instance.
func TestOllamaInstances_DeleteReferenced(t *testing.T) {
	// "mock-ollama" is referenced by the ollama-ref-agent in the test project.
	env := newOllamaAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("DELETE", "/api/ollama/instances/mock-ollama", nil)
	requireStatus(t, resp, 409)
	data := readJSON(t, resp)
	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "conflict" {
		t.Errorf("error code: got %q, want %q", code, "conflict")
	}
}

// TestOllamaInstances_HealthHealthy verifies GET /{name}/health returns ok:true
// when the mock Ollama server is reachable.
func TestOllamaInstances_HealthHealthy(t *testing.T) {
	env := newOllamaAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/ollama/instances/mock-ollama/health", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	ok, _ := data["ok"].(bool)
	if !ok {
		t.Errorf("health check: expected ok=true, got %v", data)
	}
	if _, hasLatency := data["latency_ms"]; !hasLatency {
		t.Error("health check response missing 'latency_ms' field")
	}
}

// TestOllamaInstances_HealthUnreachable verifies that when the Ollama instance
// is unreachable, health returns ok:false with an error message.
func TestOllamaInstances_HealthUnreachable(t *testing.T) {
	env := newOllamaAPITestEnv(t, []config.OllamaInstance{
		{Name: "unreachable", BaseURL: "http://127.0.0.1:19999"}, // nothing listening
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/ollama/instances/unreachable/health", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	ok, _ := data["ok"].(bool)
	if ok {
		t.Error("expected ok=false for unreachable instance, got ok=true")
	}
	if errMsg, _ := data["error"].(string); errMsg == "" {
		t.Error("expected non-empty error message in health response")
	}
}

// TestOllamaInstances_HealthTimeout verifies that when the Ollama server delays
// beyond the 10-second timeout, health returns ok:false. We simulate this with
// a mock that delays on /api/tags.
func TestOllamaInstances_HealthTimeout(t *testing.T) {
	// Use a separate mock that adds a long latency on /api/tags.
	slowMock := testutil.NewMockOllamaServer()
	slowMock.Latency["/api/tags"] = 15 * time.Second // exceeds 10s handler timeout
	t.Cleanup(func() { slowMock.Close() })

	env := newOllamaAPITestEnv(t, []config.OllamaInstance{
		{Name: "slow-ollama", BaseURL: slowMock.URL()},
	})
	env.login("admin@test.local", "admin-pass-123")

	// The health endpoint uses a 10-second client timeout. Our mock delays 15s,
	// so the request context will be cancelled by the server's write timeout (60s)
	// but the client timeout fires first. We extend the test timeout slightly.
	done := make(chan map[string]any, 1)
	go func() {
		resp := env.doRequest("GET", "/api/ollama/instances/slow-ollama/health", nil)
		if resp.StatusCode == 200 {
			done <- readJSON(t, resp)
		} else {
			resp.Body.Close()
			done <- nil
		}
	}()

	select {
	case data := <-done:
		if data == nil {
			t.Fatal("health endpoint returned non-200 for slow instance")
		}
		ok, _ := data["ok"].(bool)
		if ok {
			t.Error("expected ok=false for timeout, got ok=true")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("test timed out waiting for health response")
	}
}

// TestOllamaInstances_ListModels verifies GET /{name}/models returns model names and sizes.
func TestOllamaInstances_ListModels(t *testing.T) {
	env := newOllamaAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/ollama/instances/mock-ollama/models", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	modelsRaw, ok := data["models"].([]any)
	if !ok || len(modelsRaw) == 0 {
		t.Fatalf("expected non-empty models list, got: %v", data["models"])
	}
	m, _ := modelsRaw[0].(map[string]any)
	if name, _ := m["name"].(string); name != "testmodel:latest" {
		t.Errorf("model name: got %q, want %q", name, "testmodel:latest")
	}
	if _, hasSize := m["size"]; !hasSize {
		t.Error("model entry missing 'size' field")
	}
}

// TestOllamaInstances_ListModels_NotFound verifies that requesting models for
// an unknown instance returns 404.
func TestOllamaInstances_ListModels_NotFound(t *testing.T) {
	env := newOllamaAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/ollama/instances/no-such-instance/models", nil)
	requireStatus(t, resp, 404)
}

// TestOllamaInstances_AuthHeaderForwarded verifies that when api_key is set,
// health-check and models requests include Authorization: Bearer <key>.
func TestOllamaInstances_AuthHeaderForwarded(t *testing.T) {
	const apiKey = "my-test-api-key"

	// Use a mock that requires auth.
	authMock := testutil.NewMockOllamaServer()
	authMock.RequireAuthToken = apiKey
	t.Cleanup(func() { authMock.Close() })

	env := newOllamaAPITestEnv(t, []config.OllamaInstance{
		{Name: "auth-instance", BaseURL: authMock.URL(), APIKey: apiKey},
	})
	env.login("admin@test.local", "admin-pass-123")

	// Health check should succeed (key forwarded by handler).
	resp := env.doRequest("GET", "/api/ollama/instances/auth-instance/health", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	if ok, _ := data["ok"].(bool); !ok {
		t.Errorf("health check with api_key: expected ok=true, got %v", data)
	}

	// Also verify the Authorization header actually reached the mock.
	reqs := authMock.RequestsForPath("/api/tags")
	if len(reqs) == 0 {
		t.Fatal("no requests recorded by auth mock")
	}
	lastReq := reqs[len(reqs)-1]
	wantAuth := "Bearer " + apiKey
	if got := lastReq.Headers.Get("Authorization"); got != wantAuth {
		t.Errorf("Authorization header forwarded: got %q, want %q", got, wantAuth)
	}
}
