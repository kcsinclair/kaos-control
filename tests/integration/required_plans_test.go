//go:build integration

package integration

import (
	"testing"
)

// TestRequiredPlansGateBlocks verifies that a ticket cannot transition from
// planning → in-development when required plan types are missing.
// Test plan §7: "Required-plans gate" scenario.
func TestRequiredPlansGateBlocks(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gatetest.md",
			content: makeArtifact("Gate Test", "idea", "draft", "gatetest", "", "Testing the plans gate."),
		},
		{
			relPath: "lifecycle/requirements/gatetest-2.md",
			content: makeArtifact("Gate Test Req", "ticket", "planning", "gatetest",
				"lifecycle/ideas/gatetest.md", "A ticket in planning stage."),
		},
		// Only one plan (backend) — missing frontend and test plans.
		{
			relPath: "lifecycle/backend-plans/gatetest-3-be.md",
			content: makeArtifact("Gate Test BE Plan", "plan-backend", "approved", "gatetest",
				"lifecycle/requirements/gatetest-2.md", "Backend plan (approved)."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123") // admin has approver role

	// Attempt planning → in-development: should fail because plan-frontend and
	// plan-test are not yet approved.
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/requirements/gatetest-2.md/transition", map[string]any{
		"to": "in-development",
	})
	requireStatus(t, resp, 409)
	data := readJSON(t, resp)

	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "gate_not_ready" {
		t.Errorf("expected error code 'gate_not_ready', got %q", code)
	}

	missing, ok := data["missing"].([]any)
	if !ok {
		t.Fatal("expected missing array in response")
	}
	if len(missing) != 2 {
		t.Errorf("expected 2 missing plan types, got %d: %v", len(missing), missing)
	}

	// Verify the missing types include plan-frontend and plan-test.
	missingSet := map[string]bool{}
	for _, m := range missing {
		missingSet[m.(string)] = true
	}
	if !missingSet["plan-frontend"] {
		t.Error("expected plan-frontend in missing list")
	}
	if !missingSet["plan-test"] {
		t.Error("expected plan-test in missing list")
	}
}

// TestRequiredPlansGateSucceeds verifies that the gate passes when all
// required plan types have an approved artifact.
func TestRequiredPlansGateSucceeds(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gatepass.md",
			content: makeArtifact("Gate Pass", "idea", "draft", "gatepass", "", "All plans approved."),
		},
		{
			relPath: "lifecycle/requirements/gatepass-2.md",
			content: makeArtifact("Gate Pass Req", "ticket", "planning", "gatepass",
				"lifecycle/ideas/gatepass.md", "A ticket ready to advance."),
		},
		{
			relPath: "lifecycle/backend-plans/gatepass-3-be.md",
			content: makeArtifact("Gate Pass BE", "plan-backend", "approved", "gatepass",
				"lifecycle/requirements/gatepass-2.md", "Backend plan."),
		},
		{
			relPath: "lifecycle/frontend-plans/gatepass-4-fe.md",
			content: makeArtifact("Gate Pass FE", "plan-frontend", "approved", "gatepass",
				"lifecycle/requirements/gatepass-2.md", "Frontend plan."),
		},
		{
			relPath: "lifecycle/test-plans/gatepass-5-test.md",
			content: makeArtifact("Gate Pass Test", "plan-test", "approved", "gatepass",
				"lifecycle/requirements/gatepass-2.md", "Test plan."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123") // approver role

	resp := env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/requirements/gatepass-2.md/transition", map[string]any{
		"to": "in-development",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	artifact, _ := data["artifact"].(map[string]any)
	if status, _ := artifact["status"].(string); status != "in-development" {
		t.Errorf("expected status 'in-development', got %q", status)
	}
}
