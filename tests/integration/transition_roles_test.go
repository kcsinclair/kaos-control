// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// multiRoleCfgYAML remaps qa@test.local to [analyst, backend-developer] so
// that multi-role union behaviour can be tested with the existing test credentials.
// admin keeps [product-owner, analyst, reviewer, approver]; dev keeps
// [backend-developer, frontend-developer, test-developer].
const multiRoleCfgYAML = `git:
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
    roles: [analyst, backend-developer]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// TestTransitionRolesProductOwnerDraftToApproved verifies that a product-owner
// can skip the normal workflow sequence and transition an artifact directly from
// draft to approved — a combination not present in the rule matrix. A non-product-
// owner (dev) must receive 403 for the same attempt.
func TestTransitionRolesProductOwnerDraftToApproved(t *testing.T) {
	const artifactPath = "lifecycle/requirements/tr-po-draft-approved.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("TR PO Draft Approved", "ticket", "draft", "tr-po-draft-approved", "", "Body."),
	}})

	// dev (no product-owner) must be forbidden.
	env.login("dev@test.local", "dev-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "approved"})
	requireStatus(t, resp, http.StatusForbidden)
	resp.Body.Close()

	// admin (product-owner) bypasses all role checks.
	env.login("admin@test.local", "admin-pass-123")
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "approved"})
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	artifact, _ := data["artifact"].(map[string]any)
	if got, _ := artifact["status"].(string); got != "approved" {
		t.Errorf("expected status 'approved' after product-owner skip-ahead, got %q", got)
	}
}

// TestTransitionRolesMultiRoleUnionSuccess verifies that a user holding
// [analyst, backend-developer] can perform transitions permitted by analyst
// (draft → clarifying) AND transitions permitted by backend-developer
// (in-development → in-qa). Uses multiRoleCfgYAML where qa@test.local has
// those two roles.
func TestTransitionRolesMultiRoleUnionSuccess(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/tr-union-draft.md",
			content: makeArtifact("TR Union Draft", "ticket", "draft", "tr-union-draft", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/tr-union-indev.md",
			content: makeArtifact("TR Union InDev", "ticket", "in-development", "tr-union-indev", "", "Body."),
		},
	}
	env := newTestEnvWithCfgYAML(t, seeds, multiRoleCfgYAML)

	// qa@test.local holds [analyst, backend-developer] in multiRoleCfgYAML.
	env.login("qa@test.local", "qa-pass-123")

	// analyst can do draft → clarifying.
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/requirements/tr-union-draft.md/transition",
		map[string]any{"to": "clarifying"})
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	artifact, _ := data["artifact"].(map[string]any)
	if got, _ := artifact["status"].(string); got != "clarifying" {
		t.Errorf("multi-role union: expected 'clarifying', got %q", got)
	}

	// backend-developer can do in-development → in-qa.
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/requirements/tr-union-indev.md/transition",
		map[string]any{"to": "in-qa"})
	requireStatus(t, resp, http.StatusOK)
	data = readJSON(t, resp)
	artifact, _ = data["artifact"].(map[string]any)
	if got, _ := artifact["status"].(string); got != "in-qa" {
		t.Errorf("multi-role union: expected 'in-qa', got %q", got)
	}
}

// TestTransitionRolesMultiRoleCannotDoNeither verifies that a user holding
// [analyst, backend-developer] cannot perform a transition that neither role
// permits. approved → done requires [approver]; both analyst and backend-developer
// are excluded.
func TestTransitionRolesMultiRoleCannotDoNeither(t *testing.T) {
	const artifactPath = "lifecycle/requirements/tr-union-neither.md"
	env := newTestEnvWithCfgYAML(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("TR Union Neither", "ticket", "approved", "tr-union-neither", "", "Body."),
	}}, multiRoleCfgYAML)

	// qa@test.local holds [analyst, backend-developer]; neither may do approved → done.
	env.login("qa@test.local", "qa-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "done"})
	requireStatus(t, resp, http.StatusForbidden)
	data := readJSON(t, resp)
	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "forbidden" {
		t.Errorf("expected error code 'forbidden', got %q", code)
	}
}

// TestTransitionRolesAllowedTargetsMultiRole verifies that the allowed-targets
// endpoint returns the union of targets reachable by any of the user's roles.
// For a draft artifact, [analyst, backend-developer] should yield at least
// "clarifying" (from analyst) and "blocked" (shared by both).
func TestTransitionRolesAllowedTargetsMultiRole(t *testing.T) {
	const artifactPath = "lifecycle/requirements/tr-union-targets.md"
	env := newTestEnvWithCfgYAML(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("TR Union Targets", "ticket", "draft", "tr-union-targets", "", "Body."),
	}}, multiRoleCfgYAML)

	// qa@test.local holds [analyst, backend-developer] in multiRoleCfgYAML.
	env.login("qa@test.local", "qa-pass-123")
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+artifactPath+"/allowed-targets", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	rawTargets, ok := data["targets"].([]any)
	if !ok {
		t.Fatalf("expected 'targets' array, got: %v", data)
	}
	targets := make(map[string]bool, len(rawTargets))
	for _, v := range rawTargets {
		if s, ok := v.(string); ok {
			targets[s] = true
		}
	}

	// analyst contributes "clarifying" from draft.
	if !targets["clarifying"] {
		t.Errorf("multi-role allowed-targets for draft must include 'clarifying' (analyst); got: %v", rawTargets)
	}
	// both analyst and backend-developer contribute "blocked".
	if !targets["blocked"] {
		t.Errorf("multi-role allowed-targets for draft must include 'blocked'; got: %v", rawTargets)
	}
	// approver-only targets must NOT appear.
	if targets["done"] {
		t.Errorf("multi-role allowed-targets must not include 'done' (approver only); got: %v", rawTargets)
	}
}
