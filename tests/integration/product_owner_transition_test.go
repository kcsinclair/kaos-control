// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// -----------------------------------------------------------------------------
// Milestone 1 — Product owner can perform every standard transition
// -----------------------------------------------------------------------------

// TestProductOwnerFullLifecycle walks a ticket through the complete state
// sequence (draft → clarifying → planning → in-development → in-qa →
// approved → done) using only the product-owner (admin) user.
//
// This is the regression test for the defect where the product-owner role was
// absent from the in-development → in-qa rule list, causing a 403 on that
// step.  With the superuser bypass in workflow.CanTransition the product-owner
// short-circuits all role checks.
//
// Three approved plans are seeded so the test reflects a realistic lineage,
// though the product-owner also bypasses the required-plans gate.
func TestProductOwnerFullLifecycle(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/po-full.md",
			content: makeArtifact("PO Full Lifecycle Idea", "idea", "draft", "po-full", "", "Originating idea."),
		},
		{
			relPath: "lifecycle/requirements/po-full-2.md",
			content: makeArtifact("PO Full Lifecycle Ticket", "ticket", "draft", "po-full",
				"lifecycle/ideas/po-full.md", "Ticket for product-owner full lifecycle test."),
		},
		{
			relPath: "lifecycle/backend-plans/po-full-3-be.md",
			content: makeArtifact("PO Full BE Plan", "plan-backend", "approved", "po-full",
				"lifecycle/requirements/po-full-2.md", "Backend plan."),
		},
		{
			relPath: "lifecycle/frontend-plans/po-full-4-fe.md",
			content: makeArtifact("PO Full FE Plan", "plan-frontend", "approved", "po-full",
				"lifecycle/requirements/po-full-2.md", "Frontend plan."),
		},
		{
			relPath: "lifecycle/test-plans/po-full-5-test.md",
			content: makeArtifact("PO Full Test Plan", "plan-test", "approved", "po-full",
				"lifecycle/requirements/po-full-2.md", "Test plan."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	path := "lifecycle/requirements/po-full-2.md"

	// Full standard sequence — every step must return 200 for product-owner.
	transitions := []string{
		"clarifying",
		"planning",
		"in-development", // product-owner bypasses the required-plans gate
		"in-qa",          // previously broken: product-owner not in role list
		"approved",
		"done",
	}

	for _, to := range transitions {
		resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition",
			map[string]any{"to": to})
		requireStatus(t, resp, 200)
		data := readJSON(t, resp)
		artifact, _ := data["artifact"].(map[string]any)
		if got, _ := artifact["status"].(string); got != to {
			t.Fatalf("product-owner transition to %q: response shows status %q", to, got)
		}
	}

	// Verify the final status is persisted on disk.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle", "requirements", "po-full-2.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: done") {
		t.Error("final status 'done' not found on disk after full lifecycle")
	}
}

// -----------------------------------------------------------------------------
// Milestone 2 — Product owner can skip-ahead transitions
// -----------------------------------------------------------------------------

// TestProductOwnerSkipAheadDraftToDone verifies that the product-owner can
// jump an artifact from draft directly to done — a transition that has no
// entry in the default rule matrix — while a non-product-owner (dev) gets 403.
func TestProductOwnerSkipAheadDraftToDone(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/po-skip-draft-done.md",
			content: makeArtifact("PO Skip Draft→Done", "ticket", "draft", "po-skip-draft-done", "", "Skip test."),
		},
	}
	env := newTestEnv(t, seeds)
	path := "lifecycle/requirements/po-skip-draft-done.md"

	// dev user has no rule that permits draft → done.
	env.login("dev@test.local", "dev-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition",
		map[string]any{"to": "done"})
	requireStatus(t, resp, 403)

	// admin (product-owner) bypasses all rules.
	env.login("admin@test.local", "admin-pass-123")
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition",
		map[string]any{"to": "done"})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	artifact, _ := data["artifact"].(map[string]any)
	if got, _ := artifact["status"].(string); got != "done" {
		t.Errorf("expected status 'done' after skip-ahead, got %q", got)
	}
}

// TestProductOwnerSkipAheadClarifyingToInQA verifies that the product-owner
// can transition clarifying → in-qa (no rule in the matrix) while dev gets 403.
func TestProductOwnerSkipAheadClarifyingToInQA(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/po-skip-clar-inqa.md",
			content: makeArtifact("PO Skip Clarifying→InQA", "ticket", "clarifying", "po-skip-clar-inqa", "", "Skip test."),
		},
	}
	env := newTestEnv(t, seeds)
	path := "lifecycle/requirements/po-skip-clar-inqa.md"

	// dev cannot skip from clarifying to in-qa.
	env.login("dev@test.local", "dev-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition",
		map[string]any{"to": "in-qa"})
	requireStatus(t, resp, 403)

	// admin (product-owner) succeeds.
	env.login("admin@test.local", "admin-pass-123")
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition",
		map[string]any{"to": "in-qa"})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	artifact, _ := data["artifact"].(map[string]any)
	if got, _ := artifact["status"].(string); got != "in-qa" {
		t.Errorf("expected status 'in-qa' after skip-ahead, got %q", got)
	}
}

// TestProductOwnerSkipAheadPlanningToApproved verifies that the product-owner
// can transition planning → approved (no rule in the matrix) while dev gets 403.
func TestProductOwnerSkipAheadPlanningToApproved(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/po-skip-plan-approved.md",
			content: makeArtifact("PO Skip Planning→Approved", "ticket", "planning", "po-skip-plan-approved", "", "Skip test."),
		},
	}
	env := newTestEnv(t, seeds)
	path := "lifecycle/requirements/po-skip-plan-approved.md"

	// dev cannot skip from planning to approved.
	env.login("dev@test.local", "dev-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition",
		map[string]any{"to": "approved"})
	requireStatus(t, resp, 403)

	// admin (product-owner) succeeds.
	env.login("admin@test.local", "admin-pass-123")
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition",
		map[string]any{"to": "approved"})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	artifact, _ := data["artifact"].(map[string]any)
	if got, _ := artifact["status"].(string); got != "approved" {
		t.Errorf("expected status 'approved' after skip-ahead, got %q", got)
	}
}

// -----------------------------------------------------------------------------
// Milestone 3 — Allowed-targets endpoint tests
//
// Requires: GET /api/p/:project/artifacts/*path/allowed-targets endpoint,
// implemented in internal/http/transition.go (handleAllowedTargets) and
// registered in internal/http/server.go under the GET /artifacts/* dispatcher.
// Response shape: {"targets": ["clarifying", "in-qa", ...]}
// -----------------------------------------------------------------------------

// TestAllowedTargetsProductOwnerGetsSuperSet verifies that the product-owner
// sees a superset of every other role's allowed targets when the artifact is
// at in-development.  The response must include at a minimum: in-qa, done,
// approved, rejected, abandoned, blocked.
func TestAllowedTargetsProductOwnerGetsSuperSet(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/po-targets.md",
			content: makeArtifact("PO Allowed Targets", "ticket", "in-development", "po-targets", "", "Targets test."),
		},
	}
	env := newTestEnv(t, seeds)
	path := "lifecycle/requirements/po-targets.md"

	env.login("admin@test.local", "admin-pass-123")
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path+"/allowed-targets", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	rawTargets, ok := data["targets"].([]any)
	if !ok {
		t.Fatalf("expected 'targets' array in response, got: %v", data)
	}
	targets := make(map[string]bool, len(rawTargets))
	for _, v := range rawTargets {
		if s, ok := v.(string); ok {
			targets[s] = true
		}
	}

	required := []string{"in-qa", "done", "approved", "rejected", "abandoned", "blocked"}
	for _, want := range required {
		if !targets[want] {
			t.Errorf("product-owner allowed-targets missing %q; got: %v", want, rawTargets)
		}
	}
}

// TestAllowedTargetsDevUserSubset verifies that a backend-developer at
// in-development sees only their authorised targets (in-qa and blocked) and
// does NOT see done or approved.
func TestAllowedTargetsDevUserSubset(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/dev-targets.md",
			content: makeArtifact("Dev Allowed Targets", "ticket", "in-development", "dev-targets", "", "Targets test."),
		},
	}
	env := newTestEnv(t, seeds)
	path := "lifecycle/requirements/dev-targets.md"

	env.login("dev@test.local", "dev-pass-123")
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path+"/allowed-targets", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	rawTargets, ok := data["targets"].([]any)
	if !ok {
		t.Fatalf("expected 'targets' array in response, got: %v", data)
	}
	targets := make(map[string]bool, len(rawTargets))
	for _, v := range rawTargets {
		if s, ok := v.(string); ok {
			targets[s] = true
		}
	}

	// Must include authorised targets.
	for _, want := range []string{"in-qa", "blocked"} {
		if !targets[want] {
			t.Errorf("dev allowed-targets missing %q; got: %v", want, rawTargets)
		}
	}
	// Must NOT include superuser-only targets.
	for _, notWant := range []string{"done", "approved"} {
		if targets[notWant] {
			t.Errorf("dev allowed-targets must not include %q; got: %v", notWant, rawTargets)
		}
	}
}

// TestAllowedTargetsQAUserDoesNotIncludeInQA verifies that a qa user at
// in-development does NOT have in-qa in their allowed targets (qa can only
// move in-qa → approved, not in-development → in-qa).
func TestAllowedTargetsQAUserDoesNotIncludeInQA(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/qa-targets.md",
			content: makeArtifact("QA Allowed Targets", "ticket", "in-development", "qa-targets", "", "Targets test."),
		},
	}
	env := newTestEnv(t, seeds)
	path := "lifecycle/requirements/qa-targets.md"

	env.login("qa@test.local", "qa-pass-123")
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path+"/allowed-targets", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	rawTargets, ok := data["targets"].([]any)
	if !ok {
		t.Fatalf("expected 'targets' array in response, got: %v", data)
	}
	for _, v := range rawTargets {
		if s, ok := v.(string); ok && s == "in-qa" {
			t.Errorf("qa user at in-development must not have 'in-qa' in allowed-targets; got: %v", rawTargets)
		}
	}
}

// TestAllowedTargetsUnauthenticatedReturns401 verifies that a request without
// a session cookie receives a 401 Unauthorized response.
func TestAllowedTargetsUnauthenticatedReturns401(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/unauth-targets.md",
			content: makeArtifact("Unauth Targets", "ticket", "in-development", "unauth-targets", "", "Targets test."),
		},
	}
	env := newTestEnv(t, seeds)
	path := "lifecycle/requirements/unauth-targets.md"

	// Plain GET with no cookies — unauthenticated.
	resp, err := http.Get(env.baseURL + "/api/p/testproject/artifacts/" + path + "/allowed-targets")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %d", resp.StatusCode)
	}
}

// -----------------------------------------------------------------------------
// Milestone 4 — Regression: existing role gates still enforced
// -----------------------------------------------------------------------------

// TestRoleGateRegressionAfterSuperuserBypass confirms that the product-owner
// superuser bypass has not weakened enforcement for other roles.  Three
// canonical negative cases must still return 403 with the expected error shape.
func TestRoleGateRegressionAfterSuperuserBypass(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/regression-gate.md",
			content: makeArtifact("Regression Gate Test", "ticket", "draft", "regression-gate", "", "Regression test artifact."),
		},
	}
	env := newTestEnv(t, seeds)
	path := "lifecycle/requirements/regression-gate.md"

	assertForbiddenWithAllowedTargets := func(t *testing.T, resp *http.Response) {
		t.Helper()
		requireStatus(t, resp, 403)
		data := readJSON(t, resp)
		errData, _ := data["error"].(map[string]any)
		if code, _ := errData["code"].(string); code != "forbidden" {
			t.Errorf("expected error code 'forbidden', got %q", code)
		}
		allowed, _ := data["allowed_targets"].([]any)
		if len(allowed) == 0 {
			t.Error("expected non-empty allowed_targets in 403 response")
		}
	}

	// Case 1: dev (backend-developer) → clarifying: only product-owner/analyst may do this.
	env.login("dev@test.local", "dev-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition",
		map[string]any{"to": "clarifying"})
	assertForbiddenWithAllowedTargets(t, resp)

	// Case 2: qa → in-development: only approver may do this.
	env.login("qa@test.local", "qa-pass-123")
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition",
		map[string]any{"to": "in-development"})
	assertForbiddenWithAllowedTargets(t, resp)

	// Case 3: dev → done: only approver may do this.
	env.login("dev@test.local", "dev-pass-123")
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition",
		map[string]any{"to": "done"})
	assertForbiddenWithAllowedTargets(t, resp)
}
