// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// noRolesCfgYAML is a variant of the default config where qa@test.local is
// assigned an empty roles list, so RolesFor returns [] for that user.
const noRolesCfgYAML = `git:
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
    roles: []

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// TestAllowedTargetsDraftAnalyst verifies that a user with the analyst role
// receives at minimum "clarifying" in the allowed targets for a draft artifact.
// Analyst can do draft → clarifying per the default workflow rules.
func TestAllowedTargetsDraftAnalyst(t *testing.T) {
	const artifactPath = "lifecycle/requirements/at-draft-analyst.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("AT Draft Analyst", "ticket", "draft", "at-draft-analyst", "", "Body."),
	}})

	// admin@test.local holds [product-owner, analyst, reviewer, approver]; analyst
	// can do draft → clarifying so that target must appear.
	env.login("admin@test.local", "admin-pass-123")
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

	if !targets["clarifying"] {
		t.Errorf("analyst allowed-targets for draft must include 'clarifying'; got: %v", rawTargets)
	}
}

// TestAllowedTargetsDraftProductOwner verifies that a product-owner receives
// the full superset of reachable targets for a draft artifact. The product-owner
// superuser bypass must surface all possible transitions regardless of which
// role normally permits them.
func TestAllowedTargetsDraftProductOwner(t *testing.T) {
	const artifactPath = "lifecycle/requirements/at-draft-po.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("AT Draft PO", "ticket", "draft", "at-draft-po", "", "Body."),
	}})

	env.login("admin@test.local", "admin-pass-123")
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

	// Product-owner must see at minimum all transitions reachable by any role
	// from draft, including those gated by analyst, reviewer, approver, etc.
	required := []string{"clarifying", "rejected", "abandoned", "blocked"}
	for _, want := range required {
		if !targets[want] {
			t.Errorf("product-owner allowed-targets for draft missing %q; got: %v", want, rawTargets)
		}
	}
}

// TestAllowedTargetsNoMatchingRoles verifies that an authenticated user whose
// project roles list is empty receives an empty targets array. Uses noRolesCfgYAML
// which assigns qa@test.local an empty roles list.
func TestAllowedTargetsNoMatchingRoles(t *testing.T) {
	const artifactPath = "lifecycle/requirements/at-no-roles.md"
	env := newTestEnvWithCfgYAML(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("AT No Roles", "ticket", "draft", "at-no-roles", "", "Body."),
	}}, noRolesCfgYAML)

	// qa@test.local has an empty roles list in noRolesCfgYAML.
	env.login("qa@test.local", "qa-pass-123")
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+artifactPath+"/allowed-targets", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	rawTargets, ok := data["targets"].([]any)
	if !ok {
		t.Fatalf("expected 'targets' array, got: %v", data)
	}
	if len(rawTargets) != 0 {
		t.Errorf("expected empty targets for user with no roles, got: %v", rawTargets)
	}
}

// TestAllowedTargetsNotFound verifies that requesting allowed-targets for a
// non-existent artifact path returns 404.
func TestAllowedTargetsNotFound(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts/lifecycle/requirements/does-not-exist.md/allowed-targets", nil)
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// TestAllowedTargetsUnauthenticated verifies that a request without a valid
// session cookie returns 401.
func TestAllowedTargetsUnauthenticated(t *testing.T) {
	const artifactPath = "lifecycle/requirements/at-unauth.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("AT Unauth", "ticket", "draft", "at-unauth", "", "Body."),
	}})

	// Plain GET with no cookies — unauthenticated.
	resp, err := http.Get(env.baseURL + "/api/p/testproject/artifacts/" + artifactPath + "/allowed-targets")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %d", resp.StatusCode)
	}
}
