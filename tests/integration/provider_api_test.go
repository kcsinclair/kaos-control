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

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	kaoshttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/tests/integration/testutil"
)

// ── Provider REST API Endpoint Tests ─────────────────────────────────────────

type providerAPIEnv struct {
	*testEnv
	appCfg     *config.App
	appCfgPath string
	mock       *testutil.MockOpenAIServer
}

func newProviderAPITestEnv(t *testing.T, initialProviders []config.Provider) *providerAPIEnv {
	t.Helper()

	mock := testutil.NewMockOpenAIServer()
	t.Cleanup(func() { mock.Close() })

	providers := append([]config.Provider{
		{
			Name:    "mock-provider",
			BaseURL: mock.URL(),
			Driver:  "openai-compatible",
			APIKey:  "secret-key-123",
		},
	}, initialProviders...)

	appCfg := &config.App{
		Server: config.ServerConfig{Listen: "127.0.0.1:0"},
		Auth:   config.AuthConfig{Method: "local", SessionTTL: 24 * time.Hour},
		Limits: config.LimitsConfig{
			MaxConcurrentAgents:        4,
			MaxConcurrentSchedulerJobs: 2,
			SchedulerRunRetentionDays:  90,
		},
		Providers: providers,
	}

	cfgDir := t.TempDir()
	appCfgPath := filepath.Join(cfgDir, "config.yaml")
	if err := config.SaveApp(appCfgPath, *appCfg); err != nil {
		t.Fatalf("writing initial app config: %v", err)
	}

	projRoot, dataDir := setupProviderProject(t, "mock-provider")

	entry := &config.ProjectEntry{
		Name:        "testproject",
		Path:        projRoot,
		Description: "provider api integration test project",
	}
	proj, err := project.Open(entry, dataDir, project.OpenOptions{
		MaxConcurrentAgents: 2,
		Providers:           appCfg.Providers,
	})
	if err != nil {
		t.Fatalf("project.Open: %v", err)
	}
	t.Cleanup(func() { proj.Close() })

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
		Listener:   ln,
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

	return &providerAPIEnv{testEnv: env, appCfg: appCfg, appCfgPath: appCfgPath, mock: mock}
}

func setupProviderProject(t *testing.T, providerName string) (root, dataDir string) {
	t.Helper()
	root = t.TempDir()
	dataDir = t.TempDir()

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
  - name: provider-ref-agent
    role: [analyst]
    driver: openai-compatible
    model: test-model
    provider: ` + providerName + `
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Ref Agent
      email: ref@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`
	cfgPath := filepath.Join(root, "lifecycle", "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}

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

	return root, dataDir
}

// TestProviderAPI_List verifies GET /api/providers returns providers with api_key masked.
func TestProviderAPI_List(t *testing.T) {
	env := newProviderAPITestEnv(t, []config.Provider{
		{Name: "keyless", BaseURL: "http://localhost:1234", Driver: "openai-compatible"},
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, "/api/providers", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	dataBytes := readJSON(t, resp)
	dataSlice, ok := dataBytes["providers"].([]any)
	if !ok {
		t.Fatalf("expected 'providers' array, got %v", dataBytes)
	}
	if len(dataSlice) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(dataSlice))
	}

	for _, item := range dataSlice {
		p := item.(map[string]any)
		if p["name"] == "mock-provider" {
			if p["api_key"] != "***" {
				t.Errorf("mock-provider api_key: got %v, want '***'", p["api_key"])
			}
			if p["has_api_key"] != true {
				t.Errorf("mock-provider has_api_key should be true")
			}
		}
		if p["name"] == "keyless" {
			if p["api_key"] != "" && p["api_key"] != nil {
				t.Errorf("keyless api_key should be empty or omitted, got %v", p["api_key"])
			}
			if p["has_api_key"] == true {
				t.Errorf("keyless has_api_key should be false")
			}
		}
	}
}

// TestProviderAPI_Create verifies POST /api/providers creates a provider and persists it.
func TestProviderAPI_Create(t *testing.T) {
	env := newProviderAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	newProvider := map[string]any{
		"name":     "new-prov",
		"base_url": "http://localhost:9999",
		"driver":   "openai-compatible",
		"api_key":  "super-secret",
	}

	resp := env.doRequest(http.MethodPost, "/api/providers", newProvider)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	// Re-fetch to confirm
	listResp := env.doRequest(http.MethodGet, "/api/providers", nil)
	data := readJSON(t, listResp)
	provs := data["providers"].([]any)
	if len(provs) != 2 {
		t.Fatalf("expected 2 providers after create, got %d", len(provs))
	}
}

// TestProviderAPI_CreateDuplicate verifies POST duplicate provider name returns 409 Conflict.
func TestProviderAPI_CreateDuplicate(t *testing.T) {
	env := newProviderAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	dup := map[string]any{
		"name":     "mock-provider",
		"base_url": "http://localhost:9999",
		"driver":   "openai-compatible",
	}

	resp := env.doRequest(http.MethodPost, "/api/providers", dup)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", resp.StatusCode)
	}
}

// TestProviderAPI_Update verifies PUT /api/providers/{name} updates provider config.
func TestProviderAPI_Update(t *testing.T) {
	env := newProviderAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	update := map[string]any{
		"base_url": "http://localhost:8888",
		"driver":   "openai-compatible",
	}

	resp := env.doRequest(http.MethodPut, "/api/providers/mock-provider", update)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	// Verify update in on-disk app config
	savedCfg, err := config.LoadApp(env.appCfgPath)
	if err != nil {
		t.Fatal(err)
	}
	var found *config.Provider
	for i := range savedCfg.Providers {
		if savedCfg.Providers[i].Name == "mock-provider" {
			found = &savedCfg.Providers[i]
			break
		}
	}
	if found == nil || found.BaseURL != "http://localhost:8888" {
		t.Errorf("expected updated BaseURL http://localhost:8888, got %+v", found)
	}
	// APIKey should remain intact
	if found.APIKey != "secret-key-123" {
		t.Errorf("APIKey lost during update: %s", found.APIKey)
	}
}

// TestProviderAPI_Delete verifies DELETE /api/providers/{name} deletes an unreferenced provider.
func TestProviderAPI_Delete(t *testing.T) {
	env := newProviderAPITestEnv(t, []config.Provider{
		{Name: "deletable-prov", BaseURL: "http://localhost:1111", Driver: "openai-compatible"},
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodDelete, "/api/providers/deletable-prov", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	savedCfg, _ := config.LoadApp(env.appCfgPath)
	for _, p := range savedCfg.Providers {
		if p.Name == "deletable-prov" {
			t.Errorf("deleted provider still present in config")
		}
	}
}

// TestProviderAPI_DeleteReferenced verifies DELETE returns 409 when provider is in use by an agent.
func TestProviderAPI_DeleteReferenced(t *testing.T) {
	env := newProviderAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodDelete, "/api/providers/mock-provider", nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", resp.StatusCode)
	}
}

// TestProviderAPI_TestProbe verifies POST /api/providers/test probes capability cleanly.
func TestProviderAPI_TestProbe(t *testing.T) {
	env := newProviderAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	testReq := map[string]any{
		"name":  "mock-provider",
		"model": "test-model",
	}

	resp := env.doRequest(http.MethodPost, "/api/providers/test", testReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	body := readJSON(t, resp)
	if body["ok"] != true {
		t.Errorf("expected ok: true, got %+v", body)
	}
}

// TestProviderAPI_ModelsList verifies GET /api/providers/{name}/models proxies to mock /v1/models.
func TestProviderAPI_ModelsList(t *testing.T) {
	env := newProviderAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, "/api/providers/mock-provider/models", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	body := readJSON(t, resp)
	modelsList, ok := body["models"].([]any)
	if !ok || len(modelsList) == 0 {
		t.Fatalf("expected models list, got %+v", body)
	}
	firstModel := modelsList[0].(map[string]any)
	if firstModel["id"] != "test-model" {
		t.Errorf("model id: got %v, want test-model", firstModel["id"])
	}
}
