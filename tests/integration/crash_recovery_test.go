// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Integration tests for crash recovery of test artifacts left in in-qa status.
//
// The agent.Manager constructor (agent.New) calls recoverOrphanedTests on
// startup.  This function resets test artifacts whose status is in-qa but
// have no corresponding running agent-run record back to approved.
//
// Recovery test strategy:
//   - Seed one or more test artifacts with status: in-qa in the initial commit.
//   - Open the project (via newAgentTestEnvWithCfg), which triggers recovery.
//   - Assert the artifact is now approved (or still in-qa for non-test types).
//
// Note: scenario "Active run not reset" (test plan TC 6.2) is not implemented
// here because index.RecoverRunningRuns() — which always runs first — marks
// ALL "running" agent-run records as "failed" before recoverOrphanedTests
// executes.  There is therefore no way to have a legitimately active run at
// the time orphan recovery fires in the normal startup sequence.
//
// Test plan: lifecycle/test-plans/test-artifact-status-lifecycle-5-test.md §Milestone 6

import (
	"os"
	"path/filepath"
	"testing"
)

// TC 6.1: A test artifact left in in-qa with no corresponding active agent run
// is reset to approved on startup (orphan recovery).
func TestCrashRecovery_OrphanedTestResetOnStartup(t *testing.T) {
	const artifactPath = "lifecycle/tests/orphaned-in-qa.md"

	// Seed the artifact with status: in-qa — simulating a crash mid-run.
	env := newAgentTestEnvWithCfg(t, qaAgentCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Orphaned In-QA", "test", "in-qa", "orphaned-in-qa", "", "Test body."),
	}})
	// No agent run was ever inserted, so the artifact is a true orphan.
	// agent.New → recoverOrphanedTests has already run by the time the env is ready.

	env.login("admin@test.local", "admin-pass-123")

	// Query the artifact via the API — it should be approved.
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+artifactPath, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)
	if status, _ := fm["status"].(string); status != "approved" {
		t.Errorf("orphaned test artifact must be reset to 'approved' on startup; got status %q", status)
	}

	// Verify on disk as well.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: approved") {
		t.Errorf("disk file must show status: approved after orphan recovery; got:\n%s", raw)
	}
}

// TC 6.3 (skipping TC 6.2 — see package-level note): A non-test artifact (e.g.
// type: requirement) left in in-qa is NOT reset by recoverOrphanedTests, because
// the recovery only applies to type: test artifacts.
func TestCrashRecovery_NonTestArtifactInQANotReset(t *testing.T) {
	const artifactPath = "lifecycle/requirements/non-test-in-qa-2.md"

	env := newAgentTestEnvWithCfg(t, qaAgentCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Non-Test In-QA", "requirement", "in-qa", "non-test-in-qa", "", "Req body."),
	}})

	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+artifactPath, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)
	if status, _ := fm["status"].(string); status != "in-qa" {
		t.Errorf("non-test artifact in-qa must NOT be reset on startup; got status %q", status)
	}
}

// TC 6.4: Multiple orphaned test artifacts are all reset to approved on startup.
func TestCrashRecovery_MultipleOrphansRecovered(t *testing.T) {
	paths := []string{
		"lifecycle/tests/orphan-multi-1.md",
		"lifecycle/tests/orphan-multi-2.md",
		"lifecycle/tests/orphan-multi-3.md",
	}

	seeds := make([]seedArtifact, len(paths))
	for i, p := range paths {
		// Each artifact gets a distinct lineage slug derived from its filename.
		slug := filepath.Base(p)
		slug = slug[:len(slug)-3] // strip .md
		seeds[i] = seedArtifact{
			relPath: p,
			content: makeArtifact("Orphan Multi "+slug, "test", "in-qa", slug, "", "Test body."),
		}
	}

	env := newAgentTestEnvWithCfg(t, qaAgentCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	for _, artifactPath := range paths {
		resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+artifactPath, nil)
		requireStatus(t, resp, 200)
		data := readJSON(t, resp)

		art, _ := data["artifact"].(map[string]any)
		fm, _ := art["frontmatter"].(map[string]any)
		if status, _ := fm["status"].(string); status != "approved" {
			t.Errorf("orphan %s must be reset to 'approved'; got %q", artifactPath, status)
		}
	}
}
