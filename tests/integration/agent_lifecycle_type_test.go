// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Integration tests for the agent runner's pre-run and post-run lifecycle
// transitions for test artifacts, and the concurrent-run guard.
//
// Milestone 3: agent runner status transitions (pre-run, post-run).
// Milestone 4: concurrent run guard (status check + lineage lock).
//
// Test plan: lifecycle/test-plans/test-artifact-status-lifecycle-5-test.md §Milestone 3, §Milestone 4

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── Milestone 3 ────────────────────────────────────────────────────────────────

// TC 3.1: Pre-run transition — starting a QA agent run against an approved test
// artifact synchronously sets the artifact status to in-qa before the driver
// process is launched.
func TestAgentLifecycleType_PreRunTransitionApprovedToInQA(t *testing.T) {
	setupFakeClaude(t, 0) // quick exit so we can inspect pre-run state

	const artifactPath = "lifecycle/tests/prerun-approved.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Pre-Run Approved", "test", "approved", "prerun-approved", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	// StartRun sets the artifact status synchronously before returning the run_id.
	resp := env.doRequest("POST", "/api/p/testproject/agents/qa/run", map[string]any{
		"target_path": artifactPath,
	})
	requireStatus(t, resp, 202)
	readJSON(t, resp) // consume body

	// Check the file on disk immediately — the status change is synchronous.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: in-qa") {
		t.Errorf("expected status: in-qa on disk immediately after StartRun; got:\n%s", raw)
	}
}

// TC 3.2: Pre-run rejection — starting a QA agent run against a test artifact
// that is NOT in approved status is rejected (409) and the artifact status is
// unchanged.
func TestAgentLifecycleType_PreRunRejectionDraftStatus(t *testing.T) {
	setupFakeClaude(t, 0)

	const artifactPath = "lifecycle/tests/prerun-draft.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Pre-Run Draft", "test", "draft", "prerun-draft", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/agents/qa/run", map[string]any{
		"target_path": artifactPath,
	})
	// Workflow check fails: CanTransition("draft","in-qa",["qa"],"test") == false.
	// The HTTP handler maps all non-special errors to 409 Conflict.
	if resp.StatusCode != 409 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 409 for QA run against draft test artifact, got %d: %s", resp.StatusCode, b)
	}
	resp.Body.Close()

	// Verify the artifact status is unchanged on disk.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: draft") {
		t.Errorf("artifact status must remain 'draft' after rejected run; got:\n%s", raw)
	}
	if containsLine(string(raw), "status: in-qa") {
		t.Error("artifact status must NOT be 'in-qa' after rejected run")
	}
}

// TC 3.3: Post-run success — after the QA agent exits with code 0, the test
// artifact is reset from in-qa back to approved.
func TestAgentLifecycleType_PostRunSuccessReturnsToApproved(t *testing.T) {
	setupFakeClaude(t, 0)

	const artifactPath = "lifecycle/tests/postrun-success.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Post-Run Success", "test", "approved", "postrun-success", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	runID := startAgentRun(t, env, "qa", artifactPath)
	run := waitForRunCompletion(t, env, runID)

	if got, _ := run["status"].(string); got != "done" {
		t.Errorf("expected run record status 'done', got %q", got)
	}

	// After successful completion the supervisor sets the artifact back to approved.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: approved") {
		t.Errorf("expected status: approved after successful QA run; got:\n%s", raw)
	}
}

// TC 3.4: Post-run failure — after the QA agent exits with a non-zero code, the
// test artifact remains in in-qa (no reset to approved).
func TestAgentLifecycleType_PostRunFailureStaysInQA(t *testing.T) {
	setupFakeClaude(t, 1) // non-zero exit → "failed" run status

	const artifactPath = "lifecycle/tests/postrun-failure.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Post-Run Failure", "test", "approved", "postrun-failure", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	runID := startAgentRun(t, env, "qa", artifactPath)
	run := waitForRunCompletion(t, env, runID)

	if got, _ := run["status"].(string); got != "failed" {
		t.Errorf("expected run record status 'failed', got %q", got)
	}

	// Artifact must remain in-qa: the supervisor only resets on exit code 0.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: in-qa") {
		t.Errorf("expected status: in-qa to persist after failed QA run; got:\n%s", raw)
	}
	if containsLine(string(raw), "status: approved") {
		t.Error("status must NOT be reset to 'approved' after a failed QA run")
	}
}

// TC 3.5: Non-test artifacts use existing DoneOnSuccess behaviour — the
// stub-done-agent (done_on_success=true) marks the artifact as done on exit 0,
// not approved.  This verifies the test-artifact branch does not contaminate
// the general path.
func TestAgentLifecycleType_NonTestArtifactUsesExistingBehaviour(t *testing.T) {
	setupFakeClaude(t, 0)

	// stub-done-agent (from agentLifecycleCfgYAML) has active_status=in-development
	// and done_on_success=true.  It targets idea artifacts.
	const artifactPath = "lifecycle/ideas/non-test-done.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Non-Test Done", "idea", "draft", "non-test-done", "", "Idea body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "stub-done-agent", artifactPath)
	run := waitForRunCompletion(t, env, runID)

	if got, _ := run["status"].(string); got != "done" {
		t.Errorf("expected run record status 'done', got %q", got)
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	// Non-test artifact with done_on_success=true → status must be "done", NOT "approved".
	if !containsLine(string(raw), "status: done") {
		t.Errorf("expected status: done for non-test artifact with done_on_success; got:\n%s", raw)
	}
	if containsLine(string(raw), "status: approved") {
		t.Error("non-test artifact status must NOT be 'approved' (that is the test-artifact reset path)")
	}
}

// ── Milestone 4 ────────────────────────────────────────────────────────────────

// TC 4.1: A second QA run attempted while the test artifact is already in in-qa
// is rejected immediately with 409 (concurrent run guard).
// This test directly transitions the artifact to in-qa via the API and then
// tries to start a QA run, verifying the status-based guard fires.
func TestAgentLifecycleType_SecondRunRejectedWhenInQA(t *testing.T) {
	setupFakeClaude(t, 0)

	const artifactPath = "lifecycle/tests/concurrent-guard.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Concurrent Guard", "test", "approved", "concurrent-guard", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	// Put the artifact in in-qa via the transition API (no agent run needed).
	transResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "in-qa"})
	requireStatus(t, transResp, 200)
	transResp.Body.Close()

	// Now attempt to start a QA run — the status guard must reject it.
	runResp := env.doRequest("POST", "/api/p/testproject/agents/qa/run", map[string]any{
		"target_path": artifactPath,
	})
	if runResp.StatusCode != 409 {
		b, _ := io.ReadAll(runResp.Body)
		runResp.Body.Close()
		t.Fatalf("expected 409 for second run on in-qa artifact, got %d: %s", runResp.StatusCode, b)
	}
	data := readJSON(t, runResp)
	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code == "" {
		t.Errorf("expected error.code in 409 response, got: %v", data)
	}
	t.Logf("correctly rejected with error.code=%q", errObj["code"])
}

// TC 4.2: A new QA run is allowed after the first run completes (artifact
// returns to approved, lock released).
func TestAgentLifecycleType_RunAllowedAfterFirstCompletes(t *testing.T) {
	setupFakeClaude(t, 0)

	const artifactPath = "lifecycle/tests/run-after-complete.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Run After Complete", "test", "approved", "run-after-complete", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	// First run — start, wait for completion, artifact returns to approved.
	runID1 := startAgentRun(t, env, "qa", artifactPath)
	run1 := waitForRunCompletion(t, env, runID1)
	if got, _ := run1["status"].(string); got != "done" {
		t.Fatalf("first run expected 'done', got %q", got)
	}

	// The artifact must now be approved again (lock released, post-run reset).
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: approved") {
		t.Fatalf("expected status: approved after first run completes; got:\n%s", raw)
	}

	// Second run — must succeed (202) because artifact is approved and lock is free.
	runID2 := startAgentRun(t, env, "qa", artifactPath)
	run2 := waitForRunCompletion(t, env, runID2)
	if got, _ := run2["status"].(string); got != "done" {
		t.Errorf("second run expected 'done', got %q", got)
	}
}

// TC 4.3: A QA run on a different lineage is not blocked by a concurrent run
// on another lineage (locks are per-lineage).
func TestAgentLifecycleType_DifferentLineageNotAffected(t *testing.T) {
	// Slow fake claude holds the lineage-A lock open while we start lineage-B.
	setupSlowFakeClaude(t, 5)

	const pathA = "lifecycle/tests/lineage-a-concurrent.md"
	const pathB = "lifecycle/tests/lineage-b-concurrent.md"
	env := newQATestEnv(t, []seedArtifact{
		{
			relPath: pathA,
			content: makeArtifact("Lineage A Concurrent", "test", "approved", "lineage-a-concurrent", "", "Test body."),
		},
		{
			relPath: pathB,
			content: makeArtifact("Lineage B Concurrent", "test", "approved", "lineage-b-concurrent", "", "Test body."),
		},
	})
	env.login("qa@test.local", "qa-pass-123")

	// Start run on lineage A — it holds the lock for 5 seconds.
	respA := env.doRequest("POST", "/api/p/testproject/agents/qa/run", map[string]any{
		"target_path": pathA,
	})
	requireStatus(t, respA, 202)
	readJSON(t, respA)

	// Immediately start run on lineage B — must NOT be blocked by lineage A's lock.
	respB := env.doRequest("POST", "/api/p/testproject/agents/qa/run", map[string]any{
		"target_path": pathB,
	})
	if respB.StatusCode != 202 {
		b, _ := io.ReadAll(respB.Body)
		respB.Body.Close()
		t.Fatalf("lineage-B run expected 202 (separate lineage), got %d: %s", respB.StatusCode, b)
	}
	dataB := readJSON(t, respB)
	runIDB, _ := dataB["run_id"].(string)
	if runIDB == "" {
		t.Fatal("expected run_id for lineage-B run")
	}
	t.Logf("lineage-B run started concurrently with lineage-A: run_id=%s", runIDB)

	// Give both slow runs time to finish so cleanup doesn't race.
	time.Sleep(6 * time.Second)
}
