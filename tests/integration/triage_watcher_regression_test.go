// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Regression tests for:
// auto-triage-new-ideas-watcher-create-raw-7-defect.md
//
// The defect: pollForArtifactStatus was checking data["status"] at the top level
// of the GET /api/p/.../artifacts/* response, but the API nests the artifact
// (including its status field) under data["artifact"]. The function never matched
// and always timed out even when the artifact was already in the desired status.

import (
	"fmt"
	"testing"
	"time"
)

// TestArtifactAPIResponse_StatusNestedInArtifactKey verifies that the GET
// /api/p/:project/artifacts/*path endpoint returns artifact fields nested under
// the "artifact" key, not at the top level of the JSON response.
//
// If the API response structure ever changes so that "status" appears at the
// top level, pollForArtifactStatus will stop working and this test will catch it.
func TestArtifactAPIResponse_StatusNestedInArtifactKey(t *testing.T) {
	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/api-response-regression.md",
		content: makeArtifact("API Response Regression", "idea", "draft",
			"api-response-regression", "",
			"Body for API response structure regression test."),
	}}
	env := newTestEnv(t, seeds)

	relPath := "lifecycle/ideas/api-response-regression.md"
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	// "status" must NOT appear at the top level.
	if _, exists := data["status"]; exists {
		t.Errorf("'status' should NOT appear at the top level of GET /artifacts response; "+
			"pollForArtifactStatus relies on it being nested under data[\"artifact\"]. Top-level keys: %v",
			topLevelKeys(data))
	}

	// "artifact" key must exist and be a nested object.
	art, ok := data["artifact"].(map[string]any)
	if !ok {
		t.Fatalf("GET /artifacts response missing 'artifact' key or it is not an object; top-level keys: %v",
			topLevelKeys(data))
	}

	// Status must be accessible as data["artifact"]["status"].
	status, _ := art["status"].(string)
	if status != "draft" {
		t.Errorf("data[\"artifact\"][\"status\"] = %q, want %q", status, "draft")
	}
}

// TestPollForArtifactStatus_ReadsNestedArtifactField is a direct regression
// for the pollForArtifactStatus parsing bug. It seeds an artifact that is
// already in the desired status and verifies that pollForArtifactStatus detects
// it immediately via the correct nested data["artifact"]["status"] path.
//
// If pollForArtifactStatus regresses to checking data["status"] (top level),
// it will never find a match because the GET response does not have "status" at
// the top level, so this test will time out and fail.
func TestPollForArtifactStatus_ReadsNestedArtifactField(t *testing.T) {
	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/poll-nested-regression.md",
		content: makeArtifact("Poll Nested Regression", "idea", "draft",
			"poll-nested-regression", "",
			"Body for pollForArtifactStatus nested field regression test."),
	}}
	env := newTestEnv(t, seeds)

	relPath := "lifecycle/ideas/poll-nested-regression.md"

	// The artifact is seeded at startup as "draft". With a correct
	// pollForArtifactStatus implementation this should return true immediately
	// (on the first or second poll). With the broken implementation (checking
	// data["status"] at the top level) it would always time out.
	if !pollForArtifactStatus(t, env, relPath, "draft", 3*time.Second) {
		t.Errorf("pollForArtifactStatus did not detect draft status within 3s; "+
			"possible regression: helper is reading data[\"status\"] instead of "+
			"data[\"artifact\"][\"status\"]. Direct API check: %v",
			readArtifactFM(t, env.projectRoot, relPath))
	}
}

// topLevelKeys returns a formatted list of top-level keys in a map for use in
// error messages.
func topLevelKeys(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return fmt.Sprintf("%v", keys)
}
