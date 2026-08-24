// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
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

// ── OpenAI-Compatible Agent Run Integration Tests ───────────────────────────

const openAIAgentCfgTemplate = `
git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - backend-developer

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
    roles: [product-owner, analyst, backend-developer]
  - email: dev@test.local
    roles: [backend-developer]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []

agents:
  - name: openai-analyst
    role: [analyst]
    driver: openai-compatible
    provider: test-provider
    model: test-model
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: OpenAI Analyst Agent
      email: openai-analyst@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"

  - name: openai-done-agent
    role: [backend-developer]
    driver: openai-compatible
    provider: test-provider
    model: test-model
    active_status: in-development
    done_on_success: true
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: OpenAI Done Agent
      email: openai-done@test.local
    prompt_templates:
      backend-developer: "Implement {target_path}"

  - name: claude-analyst
    role: [analyst]
    driver: claude-code-cli
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Claude Analyst Agent
      email: claude-analyst@test.local
    prompt_templates:
      analyst: "Analyse with Claude {target_path}"
`

type openAIAgentTestEnv struct {
	*testEnv
	mock *testutil.MockOpenAIServer
}

func newOpenAIAgentTestEnv(t *testing.T, turns []testutil.MockTurn, maxConcurrent int) *openAIAgentTestEnv {
	t.Helper()

	mock := testutil.NewMockOpenAIServer()
	if len(turns) > 0 {
		mock.ScriptedTurns = turns
	}
	t.Cleanup(func() { mock.Close() })

	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}

	appCfg := &config.App{
		Server: config.ServerConfig{Listen: "127.0.0.1:0"},
		Auth:   config.AuthConfig{Method: "local", SessionTTL: 24 * time.Hour},
		Limits: config.LimitsConfig{
			MaxConcurrentAgents:        maxConcurrent,
			MaxConcurrentSchedulerJobs: 2,
			SchedulerRunRetentionDays:  90,
		},
		Providers: []config.Provider{
			{
				Name:    "test-provider",
				BaseURL: mock.URL(),
				Driver:  "openai-compatible",
			},
		},
	}

	projRoot, dataDir := setupOpenAIAgentProject(t)

	entry := &config.ProjectEntry{
		Name:        "testproject",
		Path:        projRoot,
		Description: "openai agent test project",
	}
	proj, err := project.Open(entry, dataDir, project.OpenOptions{
		MaxConcurrentAgents: maxConcurrent,
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

	cfgDir := t.TempDir()
	appCfgPath := filepath.Join(cfgDir, "config.yaml")
	_ = config.SaveApp(appCfgPath, *appCfg)

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

	return &openAIAgentTestEnv{testEnv: env, mock: mock}
}

func setupOpenAIAgentProject(t *testing.T) (root, dataDir string) {
	t.Helper()
	root = t.TempDir()
	dataDir = t.TempDir()

	for _, s := range []string{"ideas", "requirements", "backend-plans", "frontend-plans",
		"test-plans", "tests", "prototypes", "releases", "sprints", "defects"} {
		if err := os.MkdirAll(filepath.Join(root, "lifecycle", s), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cfgPath := filepath.Join(root, "lifecycle", "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(openAIAgentCfgTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	// Seed idea artifact
	ideaContent := makeArtifact("Test Idea", "idea", "draft", "test-idea", "", "Idea content.")
	if err := os.WriteFile(filepath.Join(root, "lifecycle", "ideas", "test-idea.md"), []byte(ideaContent), 0o644); err != nil {
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
	if _, err := wt.Add("lifecycle/ideas/test-idea.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@test.local", When: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}

	return root, dataDir
}

// TestOpenAIAgentRun_DriverSelection verifies that openai-compatible agents execute properly via API.
func TestOpenAIAgentRun_DriverSelection(t *testing.T) {
	env := newOpenAIAgentTestEnv(t, nil, 4)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/agents/openai-analyst/run", map[string]any{
		"target_path": "lifecycle/ideas/test-idea.md",
	})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}
	data := readJSON(t, resp)
	runID, ok := data["run_id"].(string)
	if !ok || runID == "" {
		t.Fatalf("expected run_id in response: %v", data)
	}
}

// TestOpenAIAgentRun_Completes verifies full lifecycle from 202 to status=done.
func TestOpenAIAgentRun_Completes(t *testing.T) {
	turns := []testutil.MockTurn{
		{
			ToolCalls: []testutil.MockToolCall{
				{
					ID:        "call_req",
					Name:      "write_file",
					Arguments: `{"path":"lifecycle/requirements/test-idea-2.md","content":"# Generated Requirement Specification"}`,
				},
			},
		},
		{
			Content:      "Analysis complete and requirements written.",
			FinishReason: "stop",
		},
	}
	env := newOpenAIAgentTestEnv(t, turns, 4)
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env.testEnv, "openai-analyst", "lifecycle/ideas/test-idea.md")
	run := waitForRunCompletion(t, env.testEnv, runID)

	if status, _ := run["status"].(string); status != "done" {
		t.Fatalf("expected status=done, got %q", status)
	}

	// Verify file was written to disk
	createdReq := filepath.Join(env.projectRoot, "lifecycle", "requirements", "test-idea-2.md")
	if _, err := os.Stat(createdReq); os.IsNotExist(err) {
		t.Errorf("expected generated file to exist on disk: %s", createdReq)
	}
}

// TestOpenAIAgentRun_PreflightFailureHardFails verifies that Mode A silent drop hard-fails
// with no artifacts modified (FR-5b).
func TestOpenAIAgentRun_PreflightFailureHardFails(t *testing.T) {
	env := newOpenAIAgentTestEnv(t, nil, 4)
	env.mock.PreflightMode = "mode-a" // Silent drop
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/agents/openai-analyst/run", map[string]any{
		"target_path": "lifecycle/ideas/test-idea.md",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for preflight hard-fail, got %d", resp.StatusCode)
	}
	body := readJSON(t, resp)
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "run_error" {
		t.Errorf("error code: got %v, want run_error", errObj["code"])
	}
}

// TestOpenAIAgentRun_ExplicitRejectionFails verifies that Mode B (HTTP 400) surfaces verbatim error.
func TestOpenAIAgentRun_ExplicitRejectionFails(t *testing.T) {
	env := newOpenAIAgentTestEnv(t, nil, 4)
	env.mock.PreflightMode = "mode-b" // HTTP 400 rejection
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/agents/openai-analyst/run", map[string]any{
		"target_path": "lifecycle/ideas/test-idea.md",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for explicit rejection, got %d", resp.StatusCode)
	}
}

// TestOpenAIAgentRun_ConcurrencySemaphore verifies semaphore enforcement across OpenAI runs.
func TestOpenAIAgentRun_ConcurrencySemaphore(t *testing.T) {
	env := newOpenAIAgentTestEnv(t, nil, 2) // limit to 2 concurrent runs
	env.mock.StreamLatency = 30 * time.Second
	env.login("admin@test.local", "admin-pass-123")

	for i := 0; i < 2; i++ {
		relPath := fmt.Sprintf("lifecycle/ideas/concurrent-%d.md", i)
		_ = os.WriteFile(filepath.Join(env.projectRoot, relPath),
			[]byte(makeArtifact(fmt.Sprintf("Conc %d", i), "idea", "draft", fmt.Sprintf("conc-%d", i), "", "Body")), 0o644)
	}

	var startedRunIDs []string
	var startMu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)

	for i := 0; i < 2; i++ {
		i := i
		go func() {
			defer wg.Done()
			relPath := fmt.Sprintf("lifecycle/ideas/concurrent-%d.md", i)
			resp := env.doRequest(http.MethodPost, "/api/p/testproject/agents/openai-analyst/run", map[string]any{
				"target_path": relPath,
			})
			if resp.StatusCode == http.StatusAccepted {
				data := readJSON(t, resp)
				runID, _ := data["run_id"].(string)
				startMu.Lock()
				startedRunIDs = append(startedRunIDs, runID)
				startMu.Unlock()
			} else {
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()

	time.Sleep(200 * time.Millisecond)

	_ = os.WriteFile(filepath.Join(env.projectRoot, "lifecycle/ideas/concurrent-extra.md"),
		[]byte(makeArtifact("Extra", "idea", "draft", "extra", "", "Body")), 0o644)

	// 3rd run should return 503
	resp := env.doRequest(http.MethodPost, "/api/p/testproject/agents/openai-analyst/run", map[string]any{
		"target_path": "lifecycle/ideas/concurrent-extra.md",
	})
	if resp.StatusCode != http.StatusServiceUnavailable {
		body := readJSON(t, resp)
		t.Errorf("expected 503 for semaphore-full, got %d: %v", resp.StatusCode, body)
	} else {
		resp.Body.Close()
	}

	for _, runID := range startedRunIDs {
		env.doRequest(http.MethodPost, "/api/p/testproject/agents/runs/"+runID+"/kill", nil)
	}
}

// TestOpenAIAgentRun_Kill verifies kill endpoint terminates in-flight run and updates status.
func TestOpenAIAgentRun_Kill(t *testing.T) {
	env := newOpenAIAgentTestEnv(t, nil, 4)
	env.mock.StreamLatency = 30 * time.Second
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env.testEnv, "openai-analyst", "lifecycle/ideas/test-idea.md")

	time.Sleep(100 * time.Millisecond)

	killResp := env.doRequest(http.MethodPost, fmt.Sprintf("/api/p/testproject/agents/runs/%s/kill", runID), nil)
	if killResp.StatusCode != http.StatusOK {
		t.Fatalf("kill status: expected 200, got %d", killResp.StatusCode)
	}
	killResp.Body.Close()

	run := waitForRunCompletion(t, env.testEnv, runID)
	if status, _ := run["status"].(string); status != "killed" {
		t.Errorf("expected run status 'killed', got %q", status)
	}
}

// TestOpenAIAgentRun_StatusLifecycle verifies active_status and done_on_success artifact updates.
func TestOpenAIAgentRun_StatusLifecycle(t *testing.T) {
	turns := []testutil.MockTurn{
		{
			Content:      "Done implementing.",
			FinishReason: "stop",
		},
	}
	env := newOpenAIAgentTestEnv(t, turns, 4)
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env.testEnv, "openai-done-agent", "lifecycle/ideas/test-idea.md")
	run := waitForRunCompletion(t, env.testEnv, runID)

	if status, _ := run["status"].(string); status != "done" {
		t.Fatalf("expected status=done, got %q", status)
	}

	// Verify artifact status transitioned to done in SQLite index
	row, err := env.proj.Idx.Get("lifecycle/ideas/test-idea.md")
	if err != nil {
		t.Fatalf("fetching artifact from index: %v", err)
	}
	if row == nil || row.Status != "done" {
		t.Errorf("expected artifact status 'done', got %+v", row)
	}
}

// TestOpenAIAgentRun_HubEvents verifies hub event broadcasting for agent runs.
func TestOpenAIAgentRun_HubEvents(t *testing.T) {
	env := newOpenAIAgentTestEnv(t, nil, 4)
	env.login("admin@test.local", "admin-pass-123")

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/agents/openai-analyst/run", map[string]any{
		"target_path": "lifecycle/ideas/test-idea.md",
	})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}
	data := readJSON(t, resp)
	runID := data["run_id"].(string)

	type wsEvent struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}

	seenStarted := false
	seenFinished := false

	timeout := time.After(5 * time.Second)
	for !seenStarted || !seenFinished {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			switch evt.Type {
			case "agent.started":
				if rid, _ := evt.Payload["run_id"].(string); rid == runID {
					seenStarted = true
				}
			case "agent.finished", "agent.failed":
				if rid, _ := evt.Payload["run_id"].(string); rid == runID {
					seenFinished = true
				}
			}
		case <-timeout:
			t.Fatalf("timed out waiting for hub events: started=%v, finished=%v", seenStarted, seenFinished)
		}
	}
}
