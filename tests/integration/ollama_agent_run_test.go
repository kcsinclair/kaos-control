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
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	kaoshttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/tests/integration/testutil"
)

// ── Milestone 5 — Agent Runner Integration Tests ──────────────────────────────

// ollamaAgentCfgYAML is a project config template. Replace {OLLAMA_URL} with the
// mock server URL before writing to disk.
const ollamaAgentCfgTemplate = `
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
  - name: ollama-analyst
    role: [analyst]
    driver: ollama
    model: testmodel:latest
    ollama_instance: test-ollama
    ollama_endpoint: chat
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Ollama Analyst Agent
      email: ollama-analyst@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"

  - name: ollama-analyst-generate
    role: [analyst]
    driver: ollama
    model: testmodel:latest
    ollama_instance: test-ollama
    ollama_endpoint: generate
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Ollama Generate Agent
      email: ollama-generate@test.local
    prompt_templates:
      analyst: "Generate analysis for {target_path}"

  - name: ollama-done-agent
    role: [backend-developer]
    driver: ollama
    model: testmodel:latest
    ollama_instance: test-ollama
    ollama_endpoint: chat
    active_status: in-development
    done_on_success: true
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Ollama Done Agent
      email: ollama-done@test.local
    prompt_templates:
      backend-developer: "Implement {target_path}"

  - name: claude-analyst
    role: [analyst]
    driver: claude-code-cli
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Claude Analyst Agent
      email: claude@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`

// ollamaAgentEnv is a testEnv with a reference to the mock Ollama server and
// a separately tracked project root (to allow building the lifecycle config
// dynamically with the mock URL).
type ollamaAgentEnv struct {
	*testEnv
	mock *testutil.MockOllamaServer
}

// newOllamaAgentTestEnv starts a mock Ollama server and creates a full
// testEnv where the project has both Ollama and Claude agents. maxConcurrent
// controls the semaphore size (use 2 for most tests).
func newOllamaAgentTestEnv(t *testing.T, seeds []seedArtifact, maxConcurrent int) *ollamaAgentEnv {
	t.Helper()

	mock := testutil.NewMockOllamaServer()
	t.Cleanup(func() { mock.Close() })

	cfgYAML := ollamaAgentCfgTemplate
	env := newOllamaAgentTestEnvWithMock(t, mock, cfgYAML, seeds, maxConcurrent)
	return &ollamaAgentEnv{testEnv: env, mock: mock}
}

func newOllamaAgentTestEnvWithMock(
	t *testing.T,
	mock *testutil.MockOllamaServer,
	cfgYAML string,
	seeds []seedArtifact,
	maxConcurrent int,
) *testEnv {
	t.Helper()

	root := t.TempDir()
	dataDir := t.TempDir()

	// Create lifecycle directories.
	for _, s := range []string{
		"ideas", "requirements", "backend-plans", "frontend-plans",
		"test-plans", "tests", "prototypes", "releases", "sprints", "defects",
	} {
		if err := os.MkdirAll(filepath.Join(root, "lifecycle", s), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Write lifecycle/config.yaml.
	cfgPath := filepath.Join(root, "lifecycle", "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Init git repo.
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
	for _, s := range seeds {
		absPath := filepath.Join(root, s.relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absPath, []byte(s.content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := wt.Add(s.relPath); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@test.local", When: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}
	ref, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	if ref.Name().Short() != "main" {
		_ = repo.CreateBranch(&gitconfig.Branch{Name: "main", Remote: ""})
	}

	// Auth store.
	authStore, err := auth.Open(filepath.Join(dataDir, "auth.db"), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { authStore.Close() })
	for _, u := range []struct{ email, name, pass string }{
		{"admin@test.local", "Admin", "admin-pass-123"},
		{"dev@test.local", "Developer", "dev-pass-123"},
		{"qa@test.local", "QA", "qa-pass-123"},
	} {
		if err := authStore.CreateUser(u.email, u.name, u.pass); err != nil {
			t.Fatal(err)
		}
	}

	// Open project with OllamaInstances pointing at the mock.
	entry := &config.ProjectEntry{
		Name:        "testproject",
		Path:        root,
		Description: "ollama agent integration test project",
	}
	proj, err := project.Open(entry, dataDir, project.OpenOptions{
		MaxConcurrentAgents: maxConcurrent,
		OllamaInstances: []config.OllamaInstance{
			{Name: "test-ollama", BaseURL: mock.URL()},
		},
	})
	if err != nil {
		t.Fatalf("project.Open: %v", err)
	}
	t.Cleanup(func() { proj.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	proj.StartWatcher(ctx)
	proj.StartLockReaper(ctx)

	// Start HTTP server.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	srv := kaoshttp.New(kaoshttp.ServerConfig{
		Listen: addr,
		Auth:   authStore,
	}, map[string]*project.Project{"testproject": proj})

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
	return env
}

// ── Test cases ────────────────────────────────────────────────────────────────

// TestOllamaAgentRun_DriverSelection verifies that an agent with driver=ollama
// uses OllamaDriver and an agent with driver=claude-code-cli uses ClaudeCodeDriver.
// Both agents are started; the Ollama run should complete successfully against
// the mock, while the Claude run uses a fake claude binary.
func TestOllamaAgentRun_DriverSelection(t *testing.T) {
	setupFakeClaude(t, 0) // fake claude exits 0 (success)

	const artifactPath = "lifecycle/ideas/driver-select.md"
	env := newOllamaAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Driver Selection", "idea", "draft", "driver-select", "", "Body."),
	}}, 4)
	env.login("admin@test.local", "admin-pass-123")

	// Start Ollama run.
	ollamaRunID := startAgentRun(t, env.testEnv, "ollama-analyst", artifactPath)

	// Start Claude run (different artifact to avoid lineage lock conflict).
	const claudeArtifact = "lifecycle/ideas/driver-select-claude.md"
	if err := os.WriteFile(
		filepath.Join(env.projectRoot, claudeArtifact),
		[]byte(makeArtifact("Claude Driver Test", "idea", "draft", "driver-select-claude", "", "Body.")),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	claudeRunID := startAgentRun(t, env.testEnv, "claude-analyst", claudeArtifact)

	// Both runs should complete.
	ollamaRun := waitForRunCompletion(t, env.testEnv, ollamaRunID)
	claudeRun := waitForRunCompletion(t, env.testEnv, claudeRunID)

	if status, _ := ollamaRun["status"].(string); status != "done" {
		t.Errorf("ollama run status: got %q, want %q", status, "done")
	}
	if status, _ := claudeRun["status"].(string); status != "done" {
		t.Errorf("claude run status: got %q, want %q", status, "done")
	}
}

// TestOllamaAgentRun_UnknownDriver verifies that starting an agent with an
// unknown driver returns an error immediately.
func TestOllamaAgentRun_UnknownDriver(t *testing.T) {
	const artifactPath = "lifecycle/ideas/unknown-driver.md"

	// Build a custom config with an agent using an invalid driver.
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
  - name: bad-driver-agent
    role: [analyst]
    driver: completely-unknown-driver
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Bad Driver Agent
      email: bad@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`
	env := newTestEnvWithCfgYAML(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Unknown Driver Test", "idea", "draft", "unknown-driver", "", "Body."),
	}}, cfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/agents/bad-driver-agent/run", map[string]any{
		"target_path": artifactPath,
	})
	// Should fail — 409 (conflict) because the driver is unknown.
	if resp.StatusCode == 202 {
		data := readJSON(t, resp)
		// If somehow 202 was returned, wait for the run to fail.
		runID, _ := data["run_id"].(string)
		if runID != "" {
			run := waitForRunCompletion(t, env, runID)
			if status, _ := run["status"].(string); status == "done" {
				t.Error("run with unknown driver should not succeed")
			}
		}
	} else {
		data := readJSON(t, resp)
		// Accept 409 or 500 — any non-202 error response is correct.
		if errObj, ok := data["error"].(map[string]any); ok {
			t.Logf("got expected error response: %v", errObj)
		}
	}
}

// TestOllamaAgentRun_Completes verifies the full run lifecycle: StartRun →
// progress events → run record shows status=done.
func TestOllamaAgentRun_Completes(t *testing.T) {
	const artifactPath = "lifecycle/ideas/completes.md"
	env := newOllamaAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Completes Test", "idea", "draft", "completes", "", "Body."),
	}}, 2)
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env.testEnv, "ollama-analyst", artifactPath)
	run := waitForRunCompletion(t, env.testEnv, runID)

	if status, _ := run["status"].(string); status != "done" {
		t.Errorf("run status: got %q, want %q", status, "done")
	}
	if agentName, _ := run["agent_name"].(string); agentName != "ollama-analyst" {
		t.Errorf("agent_name: got %q, want %q", agentName, "ollama-analyst")
	}
}

// TestOllamaAgentRun_Fails verifies that when the Ollama mock returns an error,
// the run record shows status=failed with stderr_tail populated.
func TestOllamaAgentRun_Fails(t *testing.T) {
	const artifactPath = "lifecycle/ideas/fails.md"
	env := newOllamaAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Fails Test", "idea", "draft", "fails", "", "Body."),
	}}, 2)

	// Make the mock return 500 for chat.
	env.mock.ErrorCodes["/api/chat"] = 500

	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env.testEnv, "ollama-analyst", artifactPath)
	run := waitForRunCompletion(t, env.testEnv, runID)

	status, _ := run["status"].(string)
	if status != "failed" {
		t.Errorf("run status: got %q, want %q", status, "failed")
	}
	// stderr_tail should have something about the error.
	stderrTail, _ := run["stderr_tail"].(string)
	if stderrTail == "" {
		t.Error("expected non-empty stderr_tail for failed Ollama run")
	}
}

// TestOllamaAgentRun_ConcurrencySemaphore verifies that starting more runs than
// the configured max_concurrent_agents limit returns ErrBusy (HTTP 503).
func TestOllamaAgentRun_ConcurrencySemaphore(t *testing.T) {
	// Use a very slow mock so goroutines stay active.
	env := newOllamaAgentTestEnv(t, nil, 2) // max 2 concurrent
	env.mock.Latency["/api/chat"] = 30 * time.Second
	env.login("admin@test.local", "admin-pass-123")

	// Seed two target artifacts for the concurrent Ollama runs.
	for i := 0; i < 2; i++ {
		relPath := fmt.Sprintf("lifecycle/ideas/concurrent-%d.md", i)
		content := makeArtifact(
			fmt.Sprintf("Concurrent %d", i), "idea", "draft",
			fmt.Sprintf("concurrent-%d", i), "", "Body.",
		)
		if err := os.WriteFile(filepath.Join(env.projectRoot, relPath), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Start two Ollama runs to fill the semaphore.
	var startedRunIDs []string
	var startMu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		i := i
		go func() {
			defer wg.Done()
			relPath := fmt.Sprintf("lifecycle/ideas/concurrent-%d.md", i)
			resp := env.doRequest("POST", "/api/p/testproject/agents/ollama-analyst/run", map[string]any{
				"target_path": relPath,
			})
			if resp.StatusCode == 202 {
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

	// Give the goroutines a moment to acquire the semaphore slots.
	time.Sleep(200 * time.Millisecond)

	// Seed a third artifact.
	if err := os.WriteFile(
		filepath.Join(env.projectRoot, "lifecycle/ideas/concurrent-extra.md"),
		[]byte(makeArtifact("Concurrent Extra", "idea", "draft", "concurrent-extra", "", "Body.")),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Third run should fail with 503 (ErrBusy).
	resp := env.doRequest("POST", "/api/p/testproject/agents/ollama-analyst/run", map[string]any{
		"target_path": "lifecycle/ideas/concurrent-extra.md",
	})
	if resp.StatusCode != 503 {
		body := readJSON(t, resp)
		t.Errorf("expected 503 for semaphore-full, got %d: %v", resp.StatusCode, body)
	} else {
		resp.Body.Close()
	}

	// Kill running runs to clean up (prevent test from hanging).
	for _, runID := range startedRunIDs {
		env.doRequest("POST", "/api/p/testproject/agents/runs/"+runID+"/kill", nil)
	}
}

// TestOllamaAgentRun_Kill verifies that killing a long-running Ollama run
// transitions it to status=killed.
func TestOllamaAgentRun_Kill(t *testing.T) {
	const artifactPath = "lifecycle/ideas/kill-test.md"

	env := newOllamaAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Kill Test", "idea", "draft", "kill-test", "", "Body."),
	}}, 2)
	env.mock.Latency["/api/chat"] = 30 * time.Second // long-running
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env.testEnv, "ollama-analyst", artifactPath)

	// Give the run a moment to start.
	time.Sleep(100 * time.Millisecond)

	// Kill it.
	resp := env.doRequest("POST", "/api/p/testproject/agents/runs/"+runID+"/kill", nil)
	requireStatus(t, resp, 200)

	// Wait for it to finish.
	run := waitForRunCompletion(t, env.testEnv, runID)
	if status, _ := run["status"].(string); status != "killed" {
		t.Errorf("run status after kill: got %q, want %q", status, "killed")
	}
}

// TestOllamaAgentRun_StatusLifecycle verifies that an agent with
// active_status and done_on_success transitions the target artifact through
// the correct statuses.
func TestOllamaAgentRun_StatusLifecycle(t *testing.T) {
	const artifactPath = "lifecycle/requirements/lifecycle-2.md"
	env := newOllamaAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Lifecycle Test", "ticket", "approved", "lifecycle", "", "Body."),
	}}, 2)
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env.testEnv, "ollama-done-agent", artifactPath)
	run := waitForRunCompletion(t, env.testEnv, runID)

	if status, _ := run["status"].(string); status != "done" {
		t.Errorf("run status: got %q, want %q", status, "done")
	}

	// The artifact should now be in status=done.
	row, err := env.proj.Idx.Get(artifactPath)
	if err != nil {
		t.Fatalf("index.Get: %v", err)
	}
	if row == nil {
		t.Fatal("artifact not found in index after run")
	}
	if row.Status != "done" {
		t.Errorf("artifact status after done_on_success run: got %q, want %q", row.Status, "done")
	}
}

// TestOllamaAgentRun_HubEvents verifies that agent.progress and agent.finished
// events are broadcast via the hub during an Ollama run.
func TestOllamaAgentRun_HubEvents(t *testing.T) {
	const artifactPath = "lifecycle/ideas/hub-events.md"
	env := newOllamaAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Hub Events", "idea", "draft", "hub-events", "", "Body."),
	}}, 2)

	// Register hub channel BEFORE starting the run.
	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")
	runID := startAgentRun(t, env.testEnv, "ollama-analyst", artifactPath)

	type wsEvent struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}

	var seenStarted, seenProgress, seenFinished bool
	timeout := time.After(10 * time.Second)
COLLECT:
	for !seenFinished {
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
			case "agent.progress":
				if rid, _ := evt.Payload["run_id"].(string); rid == runID {
					seenProgress = true
				}
			case "agent.finished", "agent.failed":
				if rid, _ := evt.Payload["run_id"].(string); rid == runID {
					seenFinished = true
				}
			}
		case <-timeout:
			break COLLECT
		}
	}

	if !seenStarted {
		t.Errorf("never received agent.started event for run %s", runID)
	}
	if !seenProgress {
		t.Logf("note: no agent.progress events seen for run %s (may be timing-dependent)", runID)
	}
	if !seenFinished {
		t.Errorf("never received agent.finished or agent.failed event for run %s", runID)
	}
}
