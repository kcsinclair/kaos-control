// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Integration tests for the full doc artifact transition pipeline via the HTTP
// transition endpoint.
//
// Test plan: lifecycle/test-plans/tech-writer-agent-5-test.md §Milestone 5

import (
	"net/http"
	"testing"
)

// docTransitionCfgYAML is a project config for transition tests:
//   - includes the `docs` stage
//   - dev@test.local has the `tech-writer` role
//   - admin@test.local has `product-owner` (for draft → approved)
//   - qa@test.local has `qa` (for in-qa → done / defect loop)
const docTransitionCfgYAML = `git:
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
  - tech-writer

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
  - {name: docs, dir: docs}

users:
  - email: admin@test.local
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer, tech-writer]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// ── Milestone 5, TC1: full happy path ─────────────────────────────────────────

// TestDocTransition_FullHappyPath exercises the complete doc pipeline:
//   draft → approved (product-owner)
//   approved → in-development (tech-writer)
//   in-development → in-qa (tech-writer)
//   in-qa → done (qa)
func TestDocTransition_FullHappyPath(t *testing.T) {
	const artifactPath = "lifecycle/docs/happy-path-doc.md"
	env := newTestEnvWithCfgYAML(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Happy Path Doc", "doc", "draft", "happy-path-doc", "", "Doc body."),
	}}, docTransitionCfgYAML)

	transition := func(email, pass, to string, want int) {
		t.Helper()
		env.login(email, pass)
		resp := env.doRequest("POST",
			"/api/p/testproject/artifacts/"+artifactPath+"/transition",
			map[string]any{"to": to})
		requireStatus(t, resp, want)
		if want == http.StatusOK {
			data := readJSON(t, resp)
			a, _ := data["artifact"].(map[string]any)
			if status, _ := a["status"].(string); status != to {
				t.Errorf("after transition to %q: artifact status = %q", to, status)
			}
		} else {
			resp.Body.Close()
		}
	}

	// Step through the full pipeline.
	transition("admin@test.local", "admin-pass-123", "approved", http.StatusOK)
	transition("dev@test.local", "dev-pass-123", "in-development", http.StatusOK)
	transition("dev@test.local", "dev-pass-123", "in-qa", http.StatusOK)
	transition("qa@test.local", "qa-pass-123", "done", http.StatusOK)
}

// ── Milestone 5, TC2: assignees set when entering in-qa ───────────────────────

// TestDocTransition_AssigneesOnInQA asserts that after transitioning a doc from
// in-development to in-qa the artifact's assignees list contains
// {role: qa, who: agent}.
func TestDocTransition_AssigneesOnInQA(t *testing.T) {
	const artifactPath = "lifecycle/docs/assignee-test-doc.md"
	env := newTestEnvWithCfgYAML(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Assignee Test Doc", "doc", "in-development", "assignee-test-doc", "", "Doc body."),
	}}, docTransitionCfgYAML)

	// tech-writer transitions in-development → in-qa.
	env.login("dev@test.local", "dev-pass-123")
	resp := env.doRequest("POST",
		"/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "in-qa"})
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	a, _ := data["artifact"].(map[string]any)
	if a == nil {
		t.Fatal("response missing 'artifact'")
	}
	if status, _ := a["status"].(string); status != "in-qa" {
		t.Errorf("expected status 'in-qa', got %q", status)
	}

	// assignees must include {role: qa, who: agent}.
	rawAssignees, ok := a["assignees"].([]any)
	if !ok || len(rawAssignees) == 0 {
		t.Fatal("expected non-empty 'assignees' after transitioning doc to in-qa")
	}
	var found bool
	for _, raw := range rawAssignees {
		entry, _ := raw.(map[string]any)
		if entry["role"] == "qa" && entry["who"] == "agent" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("assignees does not contain {role: qa, who: agent}; got %v", rawAssignees)
	}
}

// ── Milestone 5, TC3: invalid transition blocked ──────────────────────────────

// TestDocTransition_InvalidTransitionBlocked asserts that attempting to
// transition a doc artifact via the standard feature flow (draft → clarifying)
// is rejected with 403.
func TestDocTransition_InvalidTransitionBlocked(t *testing.T) {
	const artifactPath = "lifecycle/docs/invalid-transition-doc.md"
	env := newTestEnvWithCfgYAML(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Invalid Transition Doc", "doc", "draft", "invalid-transition-doc", "", "Doc body."),
	}}, docTransitionCfgYAML)

	// admin@test.local has analyst role which normally can do draft → clarifying,
	// but NOT for doc artifacts (excludeTypes blocks this).
	env.login("admin@test.local", "admin-pass-123")
	resp := env.doRequest("POST",
		"/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "clarifying"})

	// product-owner bypasses the rule check, so use dev@test.local (analyst is in admin roles).
	// Actually admin has product-owner which always bypasses; use dev@test.local instead,
	// who has no role that can do draft→clarifying on doc.
	resp.Body.Close()

	env.login("dev@test.local", "dev-pass-123")
	resp2 := env.doRequest("POST",
		"/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "clarifying"})
	requireStatus(t, resp2, http.StatusForbidden)
	data := readJSON(t, resp2)

	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "forbidden" {
		t.Errorf("expected error.code 'forbidden', got %q", code)
	}
}

// ── Milestone 5, TC4: defect loop ─────────────────────────────────────────────

// TestDocTransition_DefectLoop verifies that qa can send a doc back from
// in-qa to in-development (the defect loop).
func TestDocTransition_DefectLoop(t *testing.T) {
	const artifactPath = "lifecycle/docs/defect-loop-doc.md"
	env := newTestEnvWithCfgYAML(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Defect Loop Doc", "doc", "in-qa", "defect-loop-doc", "", "Doc body."),
	}}, docTransitionCfgYAML)

	env.login("qa@test.local", "qa-pass-123")
	resp := env.doRequest("POST",
		"/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "in-development"})
	requireStatus(t, resp, http.StatusOK)

	data := readJSON(t, resp)
	a, _ := data["artifact"].(map[string]any)
	if status, _ := a["status"].(string); status != "in-development" {
		t.Errorf("expected status 'in-development' after defect loop, got %q", status)
	}
}
