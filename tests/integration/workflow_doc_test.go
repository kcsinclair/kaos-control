// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Integration tests for the `doc` artifact type and its workflow transitions.
//
// Test plan: lifecycle/test-plans/tech-writer-agent-5-test.md §Milestone 1 and §Milestone 2

import (
	"net/http"
	"testing"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/workflow"
)

// ── Milestone 1: doc type recognition ────────────────────────────────────────

// TestDocType_KnownTypes asserts that "doc" is registered in KnownTypes.
func TestDocType_KnownTypes(t *testing.T) {
	if !artifact.KnownTypes["doc"] {
		t.Error(`artifact.KnownTypes["doc"] is false; expected true`)
	}
}

// ── Milestone 1: doc workflow transition happy path ───────────────────────────

// TestDocWorkflow_HappyPathTransitions verifies the full doc pipeline is permitted:
//   draft → approved  (product-owner)
//   approved → in-development  (tech-writer)
//   in-development → in-qa  (tech-writer)
//   in-qa → done  (qa)
func TestDocWorkflow_HappyPathTransitions(t *testing.T) {
	e := workflow.New(nil)

	cases := []struct {
		from string
		to   string
		role string
	}{
		{"draft", "approved", "product-owner"},
		{"approved", "in-development", "tech-writer"},
		{"in-development", "in-qa", "tech-writer"},
		{"in-qa", "done", "qa"},
	}
	for _, c := range cases {
		if !e.CanTransition(c.from, c.to, []string{c.role}, "doc") {
			t.Errorf("role %q should be allowed to transition doc %s → %s",
				c.role, c.from, c.to)
		}
	}
}

// TestDocWorkflow_DefectLoop verifies in-qa → in-development by qa is permitted for doc.
func TestDocWorkflow_DefectLoop(t *testing.T) {
	e := workflow.New(nil)
	if !e.CanTransition("in-qa", "in-development", []string{"qa"}, "doc") {
		t.Error("qa should be allowed to transition doc in-qa → in-development (defect loop)")
	}
}

// TestDocWorkflow_BlockedTransitions asserts that the standard feature flow is
// NOT available for doc artifacts.
func TestDocWorkflow_BlockedTransitions(t *testing.T) {
	e := workflow.New(nil)

	blocked := []struct {
		from string
		to   string
		role string
		desc string
	}{
		{"draft", "clarifying", "analyst", "analyst: draft → clarifying on doc"},
		{"clarifying", "planning", "analyst", "analyst: clarifying → planning on doc"},
		{"planning", "in-development", "approver", "approver: planning → in-development on doc"},
	}
	for _, c := range blocked {
		if e.CanTransition(c.from, c.to, []string{c.role}, "doc") {
			t.Errorf("should NOT be permitted: %s", c.desc)
		}
	}
}

// TestDocWorkflow_NoRegression verifies that standard feature transitions still
// work for other artifact types after the doc rules were added.
func TestDocWorkflow_NoRegression(t *testing.T) {
	e := workflow.New(nil)

	cases := []struct {
		from         string
		to           string
		role         string
		artifactType string
		desc         string
	}{
		{"draft", "clarifying", "analyst", "requirement", "analyst: draft → clarifying on requirement"},
		{"planning", "in-development", "approver", "plan-backend", "approver: planning → in-development on plan-backend"},
		{"approved", "in-qa", "qa", "test", "qa: approved → in-qa on test"},
	}
	for _, c := range cases {
		if !e.CanTransition(c.from, c.to, []string{c.role}, c.artifactType) {
			t.Errorf("regression: %s should be allowed (got false)", c.desc)
		}
	}
}

// TestDocWorkflow_ProductOwnerOverride verifies that product-owner can perform
// any transition on doc artifacts (existing superuser behaviour).
func TestDocWorkflow_ProductOwnerOverride(t *testing.T) {
	e := workflow.New(nil)

	// Transitions that normal roles cannot do on doc:
	cases := []struct{ from, to string }{
		{"draft", "clarifying"},
		{"clarifying", "planning"},
		{"planning", "in-development"},
		{"in-development", "done"},
	}
	for _, c := range cases {
		if !e.CanTransition(c.from, c.to, []string{"product-owner"}, "doc") {
			t.Errorf("product-owner should be allowed doc %s → %s", c.from, c.to)
		}
	}
}

// ── Milestone 2: required_plans gate exclusion ────────────────────────────────

// docCfgYAML is a project config that:
//   - adds a `docs` stage
//   - assigns `tech-writer` role to dev@test.local
//   - keeps approver-only for admin (no product-owner) so gate is enforced
const docCfgYAML = `git:
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
    roles: [approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer, tech-writer]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// TestDocGate_DocBypasses verifies that a doc artifact in `approved` status can
// transition to `in-development` even when no plan artifacts exist for the
// lineage. The required-plans gate only fires on planning → in-development.
func TestDocGate_DocBypasses(t *testing.T) {
	const artifactPath = "lifecycle/docs/install-guide.md"
	env := newTestEnvWithCfgYAML(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Install Guide", "doc", "approved", "install-guide", "", "Doc body."),
	}}, docCfgYAML)

	// dev@test.local has tech-writer role.
	env.login("dev@test.local", "dev-pass-123")
	resp := env.doRequest("POST",
		"/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "in-development"})
	requireStatus(t, resp, http.StatusOK)

	data := readJSON(t, resp)
	a, _ := data["artifact"].(map[string]any)
	if status, _ := a["status"].(string); status != "in-development" {
		t.Errorf("expected status 'in-development', got %q", status)
	}
}

// TestDocGate_RequirementStillGated confirms that the existing required-plans
// gate still blocks a requirement artifact when plans are absent.
func TestDocGate_RequirementStillGated(t *testing.T) {
	const artifactPath = "lifecycle/requirements/gate-req-2.md"
	env := newTestEnvWithCfgYAML(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Gate Req", "requirement", "planning", "gate-req", "", "Req body."),
	}}, docCfgYAML)

	// admin@test.local has approver role only (no product-owner bypass).
	env.login("admin@test.local", "admin-pass-123")
	resp := env.doRequest("POST",
		"/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "in-development"})
	requireStatus(t, resp, http.StatusConflict)

	data := readJSON(t, resp)
	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "gate_not_ready" {
		t.Errorf("expected error.code 'gate_not_ready', got %q", code)
	}
}
