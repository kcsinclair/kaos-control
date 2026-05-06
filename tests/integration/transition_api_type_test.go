//go:build integration

package integration

// Integration tests for the HTTP transition and allowed-targets endpoints with
// type-conditional workflow rules for test artifacts.
//
// Test plan: lifecycle/test-plans/test-artifact-status-lifecycle-5-test.md §Milestone 2

import (
	"net/http"
	"testing"
)

// TC1: Transition a test artifact from approved → in-qa via the API using the
// qa role succeeds with HTTP 200.
func TestTransitionAPIType_TestApprovedToInQA_QASucceeds(t *testing.T) {
	const artifactPath = "lifecycle/tests/api-type-tc1.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("API Type TC1 Test", "test", "approved", "api-type-tc1", "", "Test body."),
	}})

	// qa@test.local has the qa role which is permitted to do approved→in-qa on test type.
	env.login("qa@test.local", "qa-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "in-qa"})
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	artifact, _ := data["artifact"].(map[string]any)
	if status, _ := artifact["status"].(string); status != "in-qa" {
		t.Errorf("expected artifact status 'in-qa' after transition, got %q", status)
	}
}

// TC2: Transition a non-test artifact from approved → in-qa via the API using
// the qa role is rejected with HTTP 403.  The response must include
// allowed_targets showing what the user can actually do.
func TestTransitionAPIType_RequirementApprovedToInQA_QAForbidden(t *testing.T) {
	const artifactPath = "lifecycle/requirements/api-type-tc2-2.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("API Type TC2 Req", "requirement", "approved", "api-type-tc2", "", "Requirement body."),
	}})

	env.login("qa@test.local", "qa-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "in-qa"})
	requireStatus(t, resp, http.StatusForbidden)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "forbidden" {
		t.Errorf("expected error.code 'forbidden', got %q", code)
	}

	// allowed_targets must be present and must NOT include in-qa.
	rawTargets, ok := data["allowed_targets"].([]any)
	if !ok {
		t.Fatalf("expected 'allowed_targets' array in 403 response, got: %v", data)
	}
	for _, v := range rawTargets {
		if s, _ := v.(string); s == "in-qa" {
			t.Errorf("allowed_targets for qa on requirement type must not include 'in-qa'; got: %v", rawTargets)
		}
	}
}

// TC3: allowed-targets for a test artifact in approved status with qa role
// must include in-qa.
func TestTransitionAPIType_AllowedTargets_InQAForTestWithQA(t *testing.T) {
	const artifactPath = "lifecycle/tests/api-type-tc3.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("API Type TC3 Test", "test", "approved", "api-type-tc3", "", "Test body."),
	}})

	env.login("qa@test.local", "qa-pass-123")
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+artifactPath+"/allowed-targets", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	rawTargets, ok := data["targets"].([]any)
	if !ok {
		t.Fatalf("expected 'targets' array in response, got: %v", data)
	}
	for _, v := range rawTargets {
		if s, _ := v.(string); s == "in-qa" {
			return // found — test passes
		}
	}
	t.Errorf("allowed-targets for qa on approved test artifact must include 'in-qa'; got: %v", rawTargets)
}

// TC4: allowed-targets for a non-test artifact in approved status with qa role
// must NOT include in-qa (type-restricted rule excluded).
func TestTransitionAPIType_AllowedTargets_NoInQAForRequirementWithQA(t *testing.T) {
	const artifactPath = "lifecycle/requirements/api-type-tc4-2.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("API Type TC4 Req", "requirement", "approved", "api-type-tc4", "", "Requirement body."),
	}})

	env.login("qa@test.local", "qa-pass-123")
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+artifactPath+"/allowed-targets", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	rawTargets, ok := data["targets"].([]any)
	if !ok {
		t.Fatalf("expected 'targets' array in response, got: %v", data)
	}
	for _, v := range rawTargets {
		if s, _ := v.(string); s == "in-qa" {
			t.Errorf("allowed-targets for qa on approved requirement must NOT include 'in-qa'; got: %v", rawTargets)
			return
		}
	}
	// in-qa absent — test passes
}
