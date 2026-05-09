// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// TestTransitionEdgeCasesConcurrent models the "idempotency guard" scenario:
// after a successful transition (draft → clarifying), a second request with the
// same destination fails with 403 because the artifact's status has already
// changed. This tests the from-status-change protection without a non-deterministic
// true race condition.
func TestTransitionEdgeCasesConcurrent(t *testing.T) {
	const artifactPath = "lifecycle/requirements/ec-concurrent.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("EC Concurrent", "ticket", "draft", "ec-concurrent", "", "Body."),
	}})

	env.login("admin@test.local", "admin-pass-123")

	// First request: draft → clarifying — must succeed.
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "clarifying"})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Second request with the same target: the artifact is now at "clarifying",
	// so clarifying → clarifying is not a valid workflow edge → 403.
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "clarifying"})
	requireStatus(t, resp, http.StatusForbidden)
	data := readJSON(t, resp)
	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "forbidden" {
		t.Errorf("expected error code 'forbidden' for repeat transition, got %q", code)
	}
}

// TestTransitionEdgeCasesDeletedArtifact verifies that transitioning an artifact
// whose file has been removed from disk between index lookup and file write
// returns an appropriate error (404 or 500) rather than panicking. The watcher
// debounce (150 ms) means the index may still hold a stale entry immediately
// after the file is deleted.
func TestTransitionEdgeCasesDeletedArtifact(t *testing.T) {
	const artifactPath = "lifecycle/requirements/ec-deleted.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("EC Deleted", "ticket", "draft", "ec-deleted", "", "Body."),
	}})

	env.login("admin@test.local", "admin-pass-123")

	// Delete the file from disk before making the request. The watcher debounce
	// means the index likely still has the old entry for a short window.
	absPath := filepath.Join(env.projectRoot, artifactPath)
	if err := os.Remove(absPath); err != nil {
		t.Fatalf("removing artifact file: %v", err)
	}

	// The handler should find the index entry (stale) then fail when it cannot
	// read the deleted file, returning 404 or 500 — not a 200 or panic.
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "clarifying"})
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Errorf("expected non-200 error when artifact file is deleted, got 200")
	}
	// Accept 404 (watcher already pruned the index) or 500 (file read failed).
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 404 or 500 for deleted artifact, got %d", resp.StatusCode)
	}
}

// TestTransitionEdgeCasesRequiredPlansGate verifies that a non-product-owner
// (approver role only) cannot advance a ticket from planning → in-development
// when the required plan types are not yet approved. The response must be 409
// with error code "gate_not_ready" and a non-empty "missing" list.
//
// Uses approverOnlyCfgYAML (defined in required_plans_test.go) so that admin
// holds only [approver], disabling the product-owner bypass.
func TestTransitionEdgeCasesRequiredPlansGate(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/ec-gate.md",
			content: makeArtifact("EC Gate Idea", "idea", "draft", "ec-gate", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/ec-gate-2.md",
			content: makeArtifact("EC Gate Ticket", "ticket", "planning", "ec-gate",
				"lifecycle/ideas/ec-gate.md", "Ticket in planning, missing plans."),
		},
		// Only backend plan present — frontend and test plans are missing.
		{
			relPath: "lifecycle/backend-plans/ec-gate-3-be.md",
			content: makeArtifact("EC Gate BE Plan", "plan-backend", "approved", "ec-gate",
				"lifecycle/requirements/ec-gate-2.md", "Backend plan."),
		},
	}
	env := newTestEnvWithCfgYAML(t, seeds, approverOnlyCfgYAML)
	env.login("admin@test.local", "admin-pass-123") // approver only — gate is NOT bypassed

	resp := env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/requirements/ec-gate-2.md/transition",
		map[string]any{"to": "in-development"})
	requireStatus(t, resp, http.StatusConflict)
	data := readJSON(t, resp)

	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "gate_not_ready" {
		t.Errorf("expected error code 'gate_not_ready', got %q", code)
	}

	missing, ok := data["missing"].([]any)
	if !ok || len(missing) == 0 {
		t.Errorf("expected non-empty 'missing' list in 409 response, got: %v", data["missing"])
	}

	missingSet := make(map[string]bool, len(missing))
	for _, m := range missing {
		if s, ok := m.(string); ok {
			missingSet[s] = true
		}
	}
	if !missingSet["plan-frontend"] {
		t.Errorf("expected 'plan-frontend' in missing list; got: %v", missing)
	}
	if !missingSet["plan-test"] {
		t.Errorf("expected 'plan-test' in missing list; got: %v", missing)
	}
}

// TestTransitionEdgeCasesProductOwnerBypassesGate verifies that a product-owner
// can advance a ticket from planning → in-development even when required plan
// types are absent. Uses the default config where admin has [product-owner, ...].
func TestTransitionEdgeCasesProductOwnerBypassesGate(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/ec-po-gate.md",
			content: makeArtifact("EC PO Gate Idea", "idea", "draft", "ec-po-gate", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/ec-po-gate-2.md",
			content: makeArtifact("EC PO Gate Ticket", "ticket", "planning", "ec-po-gate",
				"lifecycle/ideas/ec-po-gate.md", "Ticket with no approved plans."),
		},
		// No approved plans at all — gate would block a non-product-owner.
	}
	env := newTestEnv(t, seeds)           // default config: admin has product-owner
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/requirements/ec-po-gate-2.md/transition",
		map[string]any{"to": "in-development"})
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	artifact, _ := data["artifact"].(map[string]any)
	if got, _ := artifact["status"].(string); got != "in-development" {
		t.Errorf("product-owner should bypass gate; expected 'in-development', got %q", got)
	}
}
