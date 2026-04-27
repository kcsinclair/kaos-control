//go:build integration

package integration

import (
	"context"
	"fmt"
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

// agentLifecycleCfgYAML is the lifecycle/config.yaml written by newAgentTestEnv.
// It extends the base config with three test agents:
//   - analyst-requirements: active_status=clarifying (no done_on_success)
//   - analyst-planner:      active_status=planning   (no done_on_success)
//   - stub-done-agent:      active_status=in-development, done_on_success=true
const agentLifecycleCfgYAML = `git:
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
  - {name: ideas,          dir: ideas}
  - {name: requirements,   dir: requirements}
  - {name: backend-plans,  dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans,     dir: test-plans}
  - {name: tests,          dir: tests}
  - {name: prototypes,     dir: prototypes}
  - {name: releases,       dir: releases}
  - {name: sprints,        dir: sprints}
  - {name: defects,        dir: defects}

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

agents:
  - name: analyst-requirements
    role: [analyst]
    driver: claude-code-cli
    active_status: clarifying
    allowed_write_paths:
      - lifecycle/requirements
      - lifecycle/ideas
    git_identity:
      name: Analyst Requirements Agent
      email: analyst-requirements@test.local
    prompt_templates:
      analyst: "Test analyst requirements prompt for {target_path}"

  - name: analyst-planner
    role: [analyst]
    driver: claude-code-cli
    active_status: planning
    allowed_write_paths:
      - lifecycle/backend-plans
      - lifecycle/frontend-plans
      - lifecycle/test-plans
      - lifecycle/requirements
    git_identity:
      name: Analyst Planner Agent
      email: analyst-planner@test.local
    prompt_templates:
      analyst: "Test analyst planner prompt for {target_path}"

  - name: stub-done-agent
    role: [backend-developer]
    driver: claude-code-cli
    active_status: in-development
    done_on_success: true
    allowed_write_paths:
      - lifecycle/requirements
      - lifecycle/ideas
    git_identity:
      name: Stub Done Agent
      email: stub-done@test.local
    prompt_templates:
      backend-developer: "Test stub done prompt for {target_path}"
`

// newAgentTestEnv creates a fully wired test environment whose lifecycle/config.yaml
// includes agent definitions (analyst-requirements, analyst-planner, stub-done-agent).
// It mirrors newTestEnv but uses agentLifecycleCfgYAML instead of the minimal config.
func newAgentTestEnv(t *testing.T, seeds []seedArtifact) *testEnv {
	t.Helper()
	return newAgentTestEnvWithCfg(t, agentLifecycleCfgYAML, seeds)
}

// newAgentTestEnvWithCfg is like newAgentTestEnv but accepts an arbitrary
// lifecycle/config.yaml string.  Use this when a test needs a custom agent
// configuration (e.g. agents with specific model or driver values).
func newAgentTestEnvWithCfg(t *testing.T, cfgYAML string, seeds []seedArtifact) *testEnv {
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

	// Write lifecycle/config.yaml with agents configured.
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

	// Seed artifacts.
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
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@test.local",
			When:  time.Now(),
		},
	}); err != nil {
		t.Fatal(err)
	}

	// Ensure the branch is named "main".
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
		{"qa@test.local", "QA Engineer", "qa-pass-123"},
	} {
		if err := authStore.CreateUser(u.email, u.name, u.pass); err != nil {
			t.Fatal(err)
		}
	}

	// Open project.
	entry := &config.ProjectEntry{
		Name:        "testproject",
		Path:        root,
		Description: "agent integration test project",
	}
	proj, err := project.Open(entry, dataDir, project.OpenOptions{MaxConcurrentAgents: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { proj.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	proj.StartWatcher(ctx)
	proj.StartLockReaper(ctx)

	// Find a free port and start the HTTP server.
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

// setupFakeClaude writes a stub `claude` shell script that exits with exitCode
// and prepends its directory to PATH.  The original PATH is restored when the
// test ends via t.Setenv.
func setupFakeClaude(t *testing.T, exitCode int) {
	t.Helper()
	fakeDir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nexit %d\n", exitCode)
	fakeScript := filepath.Join(fakeDir, "claude")
	if err := os.WriteFile(fakeScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))
}

// startAgentRun calls POST /api/p/testproject/agents/{name}/run and returns
// the run_id.  The env must already be logged in before calling this.
func startAgentRun(t *testing.T, env *testEnv, agentName, targetPath string) string {
	t.Helper()
	resp := env.doRequest("POST", "/api/p/testproject/agents/"+agentName+"/run", map[string]any{
		"target_path": targetPath,
	})
	requireStatus(t, resp, 202)
	data := readJSON(t, resp)
	runID, _ := data["run_id"].(string)
	if runID == "" {
		t.Fatal("expected run_id in 202 response")
	}
	return runID
}

// waitForRunCompletion polls GET /api/p/testproject/agents/runs/{run_id} until
// the run leaves "running" state, then returns the final run record map.
func waitForRunCompletion(t *testing.T, env *testEnv, runID string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp := env.doRequest("GET", "/api/p/testproject/agents/runs/"+runID, nil)
		if resp.StatusCode == 200 {
			data := readJSON(t, resp)
			run, _ := data["run"].(map[string]any)
			if status, _ := run["status"].(string); status != "running" {
				return run
			}
		} else {
			resp.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("timed out waiting for run to complete")
	return nil
}
