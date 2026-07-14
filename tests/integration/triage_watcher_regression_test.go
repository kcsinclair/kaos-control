// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"strings"
	"testing"
	"time"
)

// TestPollForArtifactStatus_ParsingBugRegression ensures that pollForArtifactStatus
// correctly parses the artifact status from the nested "artifact" field in API responses.
//
// This test is a regression test for the defect:
// "TestTriageWatcher_ReRunAfterStatusReset fails due to pollForArtifactStatus parsing bug"
//
// The original bug was that pollForArtifactStatus checked data["status"] directly
// instead of looking inside the nested "artifact" field returned by GET /api/p/.../artifacts/*.
// This would cause the test to always timeout because the API response structure is:
// {
//   "artifact": {
//     "status": "draft",
//     // other fields...
//   }
// }
func TestPollForArtifactStatus_ParsingBugRegression(t *testing.T) {
	installLLMFake(t, []string{defaultProposeJSON("regression", "Regression Test Idea", nil)})
	env := newTriageTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	// Create a raw idea that will be triaged
	writeRawIdea(t, env.projectRoot, "regression", "Regression Test Idea",
		"This is a regression test idea body.")

	// Wait for initial triage to complete
	if !pollForArtifactStatus(t, env, "lifecycle/ideas/regression.md", "draft", 5*time.Second) {
		t.Fatal("initial triage did not produce draft within 5s")
	}

	// Verify that the pollForArtifactStatus function works correctly by checking it
	// can detect an artifact in the expected status (this would have failed before fix)
	if !pollForArtifactStatus(t, env, "lifecycle/ideas/regression.md", "draft", 1*time.Second) {
		t.Error("pollForArtifactStatus could not detect existing draft artifact - parsing bug still exists")
	}

	// Test the specific scenario from the defect: reset status to raw and re-triage
	absPath := "lifecycle/ideas/regression.md"
	content, err := readArtifactContent(t, env.projectRoot, absPath)
	if err != nil {
		t.Fatalf("reading triaged artifact: %v", err)
	}

	// Reset status back to raw (this is what should trigger re-triage)
	// by replacing the status field in the frontmatter
	resetContent := strings.Replace(content, "status: draft", "status: raw", 1)
	if resetContent == content {
		t.Fatal("Failed to replace status field")
	}

	if err := writeArtifactContent(t, env.projectRoot, absPath, resetContent); err != nil {
		t.Fatalf("resetting status: %v", err)
	}

	// This should trigger re-triage and eventually reach draft again
	if !pollForArtifactStatus(t, env, "lifecycle/ideas/regression.md", "draft", 5*time.Second) {
		t.Fatal("re-triage did not produce draft within 5s")
	}

	// Verify the test works properly by confirming it can detect a draft status
	// This would have failed if pollForArtifactStatus was incorrectly checking data["status"]
	if !pollForArtifactStatus(t, env, "lifecycle/ideas/regression.md", "draft", 1*time.Second) {
		t.Error("pollForArtifactStatus regression - cannot detect draft status after re-triage")
	}
}