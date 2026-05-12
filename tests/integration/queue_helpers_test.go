// SPDX-License-Identifier: AGPL-3.0-or-later

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
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/queue"
)

// queueCfgYAML is the lifecycle/config.yaml for queue integration tests.
// It extends agentLifecycleCfgYAML with a backend-developer agent that has
// source_types set, so the API can validate artifact type matching.
const queueCfgYAML = `git:
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
  - name: requirements-analyst
    role: [analyst]
    driver: claude-code-cli
    active_status: clarifying
    source_types: [idea]
    allowed_write_paths:
      - lifecycle/requirements
      - lifecycle/ideas
    git_identity:
      name: Requirements Analyst Agent
      email: requirements-analyst@test.local
    prompt_templates:
      analyst: "Test requirements analyst prompt for {target_path}"

  - name: backend-developer
    role: [backend-developer]
    driver: claude-code-cli
    active_status: in-development
    source_types: [plan-backend]
    allowed_write_paths:
      - internal
      - cmd
    git_identity:
      name: Backend Developer Agent
      email: backend-developer@test.local
    prompt_templates:
      backend-developer: "Test backend developer prompt for {target_path}"
`

// queueTestEnv wraps testEnv with queue-specific extras so tests can
// directly manipulate the queue store when needed.
type queueTestEnv struct {
	*testEnv
	queueStore *queue.Store
	dispatcher *queue.Dispatcher
	appHub     *hub.Hub
}

// newQueueTestEnv creates a fully wired test environment with an active queue
// dispatcher. seeds is the list of lifecycle artifacts to pre-populate.
//
// If existingDataDir is non-empty it is used instead of a fresh t.TempDir()
// for the data directory; this allows tests to simulate a server restart by
// pointing a second environment at the same SQLite files.
func newQueueTestEnv(t *testing.T, seeds []seedArtifact) *queueTestEnv {
	t.Helper()
	return newQueueTestEnvFromDataDir(t, seeds, "")
}

func newQueueTestEnvFromDataDir(t *testing.T, seeds []seedArtifact, existingDataDir string) *queueTestEnv {
	t.Helper()

	root := t.TempDir()
	dataDir := existingDataDir
	if dataDir == "" {
		dataDir = t.TempDir()
	}

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
	if err := os.WriteFile(cfgPath, []byte(queueCfgYAML), 0o644); err != nil {
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

	// Ensure branch is "main".
	ref, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	if ref.Name().Short() != "main" {
		_ = repo.CreateBranch(&gitconfig.Branch{Name: "main", Remote: ""})
	}

	// Open auth store.
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
		// CreateUser may fail if user already exists (restart scenario); ignore.
		_ = authStore.CreateUser(u.email, u.name, u.pass, false)
	}

	// Open project.
	entry := &config.ProjectEntry{
		Name:        "testproject",
		Path:        root,
		Description: "queue integration test project",
	}
	proj, err := project.Open(entry, dataDir, project.OpenOptions{
		MaxConcurrentAgents: 4,
		DevopsLogDir:        dataDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { proj.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	proj.StartWatcher(ctx)
	proj.StartLockReaper(ctx)

	// Open queue database (app-level).
	queueStore, err := queue.Open(filepath.Join(dataDir, "queue.db"))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = queueStore.Close() })

	if err := queueStore.RecoverOrphans(); err != nil {
		t.Logf("queue: orphan recovery: %v", err)
	}

	// Build app hub and project lookup.
	appHub := hub.New()
	projectLookup := func(name string) (queue.ProjectAccess, bool) {
		if name != "testproject" {
			return queue.ProjectAccess{}, false
		}
		if proj.Agents == nil {
			return queue.ProjectAccess{}, false
		}
		return queue.ProjectAccess{
			StartRun: func(runCtx context.Context, agentName, targetPath string) (string, error) {
				return proj.Agents.StartRun(runCtx, agentName, targetPath, "", nil)
			},
			ArtifactStatus: func(relPath string) string {
				row, err := proj.Idx.Get(relPath)
				if err != nil || row == nil {
					return ""
				}
				return row.Status
			},
			Hub: proj.Hub,
		}, true
	}

	dispatcher := queue.New(queueStore, projectLookup, appHub, queue.Config{
		TickInterval: 50 * time.Millisecond,
	})
	dispatcher.Start(ctx)

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
		Queue:  dispatcher,
		AppHub: appHub,
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
	env.login("admin@test.local", "admin-pass-123")

	t.Cleanup(func() {
		cancel()
		select {
		case <-srvDone:
		case <-time.After(5 * time.Second):
		}
	})

	return &queueTestEnv{
		testEnv:    env,
		queueStore: queueStore,
		dispatcher: dispatcher,
		appHub:     appHub,
	}
}

// enqueueViaAPI enqueues a job via POST /api/queue and returns the response
// map or fatals the test.
func (e *queueTestEnv) enqueueViaAPI(project, artifactPath, agent string) map[string]any {
	e.t.Helper()
	resp := e.doRequest("POST", "/api/queue", map[string]any{
		"project":       project,
		"artifact_path": artifactPath,
		"agent":         agent,
	})
	return readJSON(e.t, resp)
}

// waitForJobState polls GET /api/queue until the job with the given id reaches
// one of the wanted states, then returns its record from the snapshot.
func (e *queueTestEnv) waitForJobState(id string, wantStates ...string) map[string]any {
	e.t.Helper()
	wantSet := make(map[string]bool, len(wantStates))
	for _, s := range wantStates {
		wantSet[s] = true
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		snap := e.queueSnapshot()
		// Check running.
		if running, _ := snap["running"].(map[string]any); running != nil {
			if running["id"] == id {
				if wantSet[running["state"].(string)] {
					return running
				}
			}
		}
		// Check pending.
		if pending, _ := snap["pending"].([]any); pending != nil {
			for _, raw := range pending {
				j, _ := raw.(map[string]any)
				if j["id"] == id {
					if wantSet[j["state"].(string)] {
						return j
					}
				}
			}
		}
		// Check recent.
		if recent, _ := snap["recent"].([]any); recent != nil {
			for _, raw := range recent {
				j, _ := raw.(map[string]any)
				if j["id"] == id {
					if wantSet[j["state"].(string)] {
						return j
					}
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	e.t.Fatalf("timed out waiting for job %q to reach state(s) %v", id, wantStates)
	return nil
}

// queueSnapshot calls GET /api/queue and returns the parsed JSON map.
func (e *queueTestEnv) queueSnapshot() map[string]any {
	e.t.Helper()
	resp := e.doRequest("GET", "/api/queue", nil)
	requireStatus(e.t, resp, 200)
	return readJSON(e.t, resp)
}

// setupFakeClaudeWithScript writes a shell script and puts it in PATH.
// The script should be the full body (the "#!/bin/sh\n..." part is prepended).
func setupFakeClaudeWithScript(t *testing.T, script string) {
	t.Helper()
	fakeDir := t.TempDir()
	full := "#!/bin/sh\n" + script
	fakeScript := filepath.Join(fakeDir, "claude")
	if err := os.WriteFile(fakeScript, []byte(full), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))
}

// makeApprovedArtifact builds a seed artifact in the `approved` state.
func makeApprovedArtifact(title, typ, lineage string) string {
	return fmt.Sprintf(`---
title: %s
type: %s
status: approved
lineage: %s
---

Body.
`, title, typ, lineage)
}
