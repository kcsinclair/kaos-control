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
	"github.com/kaos-control/kaos-control/internal/project"
)

// ideaCaptureConfigYAML is a project config that includes the idea-capture agent,
// used in agent-config-related tests (Milestone 7).
const ideaCaptureConfigYAML = `git:
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

agents:
  - name: idea-capture
    role: [product-owner]
    driver: inline
    model: claude-sonnet-4-6
    allowed_write_paths:
      - lifecycle/ideas
    prompt_templates:
      idea-capture: |
        You are an idea-capture assistant for a software project lifecycle tool.
        Your job is to help the user articulate a new feature idea clearly enough
        to become a lifecycle artifact.

        RULES:
        1. If the user's input is vague, ask ONE short clarifying question (max 3 total).
        2. Once you have enough context, produce a proposal as structured JSON.
        3. Pick labels ONLY from the provided label vocabulary.
        4. The slug must match: ^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$

        ALWAYS respond with a JSON object in a ` + "```json" + ` code block.

        For a clarifying question:
        ` + "```json" + `
        {"action":"clarify","reply":"<your single clarifying question>","slug":"","title":"","labels":[],"body":""}
        ` + "```" + `

        For a proposal:
        ` + "```json" + `
        {"action":"propose","reply":"<short confirmation message>","slug":"<slug>","title":"<title>","labels":[],"body":"# <title>\n\n<paragraph>"}
        ` + "```" + `
`

// skipIfNoAPIKey skips the test when ANTHROPIC_API_KEY is not set. Call this at
// the top of any test that requires a live LLM call.
func skipIfNoAPIKey(t *testing.T) {
	t.Helper()
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set: skipping LLM-dependent test")
	}
}

// converseAPI posts to the /ideas/converse endpoint. When sessionID is empty
// the request omits the session_id field (creates a new session).
func converseAPI(env *testEnv, sessionID, message string) *http.Response {
	body := map[string]any{
		"message": message,
	}
	if sessionID != "" {
		body["session_id"] = sessionID
	}
	return env.doRequest("POST", "/api/p/testproject/ideas/converse", body)
}

// convergeToProposal drives a fresh conversation to the "proposed" state by
// sending firstMessage and, if needed, follow-up messages. It returns the
// session_id and the final response map. The test fails if "proposed" is not
// reached within 6 turns.
func convergeToProposal(t *testing.T, env *testEnv, firstMessage string) (string, map[string]any) {
	t.Helper()

	resp := converseAPI(env, "", firstMessage)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	sessionID, _ := data["session_id"].(string)
	if sessionID == "" {
		t.Fatal("convergeToProposal: missing session_id in first response")
	}

	for i := 0; i < 5; i++ {
		status, _ := data["status"].(string)
		if status == "proposed" {
			return sessionID, data
		}
		// Send a nudge to move things along.
		resp = converseAPI(env, sessionID, "Please go ahead and create the best proposal you can from what you know.")
		requireStatus(t, resp, 200)
		data = readJSON(t, resp)
		sid, _ := data["session_id"].(string)
		if sid != "" {
			sessionID = sid
		}
	}

	// Final status check.
	status, _ := data["status"].(string)
	if status != "proposed" {
		t.Fatalf("convergeToProposal: could not reach 'proposed' state after 6 turns; last status=%q", status)
	}
	return sessionID, data
}

// newTestEnvCustomConfig is like newTestEnv but uses a caller-supplied cfgYAML
// instead of the standard minimal config. Use this when the test needs
// specific agent configuration present at project-open time.
func newTestEnvCustomConfig(t *testing.T, cfgYAML string, seeds []seedArtifact) *testEnv {
	t.Helper()

	root := t.TempDir()
	dataDir := t.TempDir()

	stages := []string{
		"ideas", "requirements", "backend-plans", "frontend-plans",
		"test-plans", "tests", "prototypes", "releases", "sprints", "defects",
	}
	for _, s := range stages {
		if err := os.MkdirAll(filepath.Join(root, "lifecycle", s), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "lifecycle", "config.yaml"), []byte(cfgYAML), 0o644); err != nil {
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

	ref, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	if ref.Name().Short() != "main" {
		_ = repo.CreateBranch(&gitconfig.Branch{Name: "main", Remote: ""})
	}

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
		Description: "integration test project",
	}
	proj, err := project.Open(entry, dataDir, project.OpenOptions{MaxConcurrentAgents: 2})
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

// cookieHeader builds an HTTP Cookie header value from a slice of cookies.
func cookieHeader(cookies []*http.Cookie) string {
	var s string
	for i, c := range cookies {
		if i > 0 {
			s += "; "
		}
		s += fmt.Sprintf("%s=%s", c.Name, c.Value)
	}
	return s
}
