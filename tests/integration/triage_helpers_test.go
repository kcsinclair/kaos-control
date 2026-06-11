// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/ideachat"
)

// triageCfgYAML is the lifecycle/config.yaml used for triage integration tests.
// It includes the idea-triage agent with an idea-generate template.
const triageCfgYAML = `git:
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
  - name: idea-triage
    role: [product-owner]
    driver: inline
    model: claude-sonnet-4-6
    allowed_write_paths:
      - lifecycle/ideas
    prompt_templates:
      idea-generate: "Triage system prompt (overridden in tests by LLM fake)"
`

// installLLMFake replaces ideachat.CallLLM with a function that returns
// scripted responses in order. When the scripted list is exhausted, the last
// entry is repeated. Restores the original implementation via t.Cleanup.
//
// Call installLLMFake BEFORE creating any testEnv so that the cleanup runs
// after the project is fully stopped (t.Cleanup is LIFO).
func installLLMFake(t *testing.T, scripted []string) {
	t.Helper()
	if len(scripted) == 0 {
		t.Fatal("installLLMFake: scripted list must not be empty")
	}

	orig := ideachat.CallLLM
	var mu sync.Mutex
	idx := 0

	ideachat.CallLLM = func(_ context.Context, _ ideachat.ModelConfig, _ []ideachat.LLMMessage) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		resp := scripted[idx]
		if idx < len(scripted)-1 {
			idx++
		}
		return resp, nil
	}

	t.Cleanup(func() {
		ideachat.CallLLM = orig
	})
}

// installLLMFakeError replaces ideachat.CallLLM with a function that always
// returns an error. Useful for failure tests.
func installLLMFakeError(t *testing.T, errMsg string) {
	t.Helper()
	orig := ideachat.CallLLM
	ideachat.CallLLM = func(_ context.Context, _ ideachat.ModelConfig, _ []ideachat.LLMMessage) (string, error) {
		return "", fmt.Errorf("%s", errMsg)
	}
	t.Cleanup(func() {
		ideachat.CallLLM = orig
	})
}

// defaultProposeJSON returns a valid LLM response that ideachat.parseAction
// accepts as an action="propose" without error.
func defaultProposeJSON(slug, title string, labels []string) string {
	if labels == nil {
		labels = []string{}
	}
	body := fmt.Sprintf("# %s\n\nThis is the triage-generated idea body with enough content to be processed.", title)
	data := map[string]any{
		"action": "propose",
		"reply":  "Here is my proposal.",
		"slug":   slug,
		"title":  title,
		"labels": labels,
		"body":   body,
	}
	b, _ := json.Marshal(data)
	return string(b)
}

// writeRawIdea writes a lifecycle/ideas/<slug>.md artifact with status: raw
// to the project root and returns the relative path.
func writeRawIdea(t *testing.T, projectRoot, slug, title, body string) string {
	t.Helper()
	content := fmt.Sprintf("---\ntitle: %s\ntype: idea\nstatus: raw\nlineage: %s\n---\n\n%s\n",
		title, slug, body)
	relPath := fmt.Sprintf("lifecycle/ideas/%s.md", slug)
	absPath := filepath.Join(projectRoot, relPath)
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("writeRawIdea: %v", err)
	}
	return relPath
}

// readArtifactFM reads the artifact at relPath and returns key frontmatter
// fields as a map. Returns nil if the file does not exist.
func readArtifactFM(t *testing.T, projectRoot, relPath string) map[string]any {
	t.Helper()
	absPath := filepath.Join(projectRoot, relPath)
	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("readArtifactFM: %v", err)
	}
	a := artifact.Parse(content, relPath, time.Time{})
	fm := map[string]any{
		"title":   a.FM.Title,
		"type":    a.FM.Type,
		"status":  a.FM.Status,
		"lineage": a.FM.Lineage,
	}
	if a.FM.Priority != "" {
		fm["priority"] = a.FM.Priority
	}
	if len(a.FM.Labels) > 0 {
		fm["labels"] = a.FM.Labels
	}
	return fm
}

// pollForArtifactStatus polls GET /api/p/testproject/artifacts/<relPath>
// until the artifact's status matches want or timeout elapses.
// Returns true if the status was reached.
func pollForArtifactStatus(t *testing.T, env *testEnv, relPath, want string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
		if resp.StatusCode == 200 {
			data := readJSON(t, resp)
			if status, _ := data["status"].(string); status == want {
				return true
			}
		} else {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// pollForRunStatus polls agent runs for target path until a run with the
// expected status exists. Returns the run map or nil on timeout.
func pollForRunStatus(t *testing.T, env *testEnv, targetPath, wantStatus string, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp := env.doRequest("GET",
			"/api/p/testproject/agents/runs?target_path="+targetPath, nil)
		if resp.StatusCode == 200 {
			data := readJSON(t, resp)
			runs, _ := data["runs"].([]any)
			for _, r := range runs {
				run, _ := r.(map[string]any)
				if run == nil {
					continue
				}
				if s, _ := run["status"].(string); s == wantStatus {
					return run
				}
			}
		} else {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// newTriageTestEnv creates a testEnv using triageCfgYAML.
// The LLM fake must be installed by the caller (before calling this) via installLLMFake.
func newTriageTestEnv(t *testing.T) *testEnv {
	t.Helper()
	return newTestEnvWithCfgYAML(t, nil, triageCfgYAML)
}

// newTriageTestEnvWithSeeds creates a testEnv using triageCfgYAML and pre-seeded artifacts.
func newTriageTestEnvWithSeeds(t *testing.T, seeds []seedArtifact) *testEnv {
	t.Helper()
	return newTestEnvWithCfgYAML(t, seeds, triageCfgYAML)
}
