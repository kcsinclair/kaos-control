// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// End-to-end integration tests that exercise the complete approved → in-qa →
// approved lifecycle for test artifacts, including defect creation and re-run
// eligibility, and stale detection data conditions.
//
// Test plan: lifecycle/test-plans/test-artifact-status-lifecycle-5-test.md §Milestone 7

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TC 7.1: Full approved → in-qa → approved cycle, including re-run eligibility.
//
// Sequence:
//   1. Create a test artifact in approved status.
//   2. Start a QA agent run → artifact transitions to in-qa.
//   3. Wait for the run to complete → artifact returns to approved.
//   4. Start a second QA run → must succeed (re-run eligible).
//   5. Wait for the second run to complete.
func TestLifecycleE2E_FullCycleApprovedInQAApproved(t *testing.T) {
	setupFakeClaude(t, 0) // exit 0 → success both times

	const artifactPath = "lifecycle/tests/e2e-full-cycle.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("E2E Full Cycle", "test", "approved", "e2e-full-cycle", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	// ── Run 1 ──────────────────────────────────────────────────────────────────

	runID1 := startAgentRun(t, env, "qa", artifactPath)

	// Synchronous pre-run: verify artifact is in-qa immediately.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: in-qa") {
		t.Errorf("after StartRun, expected status: in-qa on disk; got:\n%s", raw)
	}

	// Wait for completion and verify post-run reset.
	run1 := waitForRunCompletion(t, env, runID1)
	if got, _ := run1["status"].(string); got != "done" {
		t.Fatalf("run 1 expected status 'done', got %q", got)
	}

	raw, err = os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: approved") {
		t.Errorf("after run 1 completion, expected status: approved; got:\n%s", raw)
	}

	// ── Run 2 (re-run eligibility) ─────────────────────────────────────────────

	runID2 := startAgentRun(t, env, "qa", artifactPath)
	run2 := waitForRunCompletion(t, env, runID2)
	if got, _ := run2["status"].(string); got != "done" {
		t.Errorf("run 2 (re-run) expected status 'done', got %q", got)
	}

	raw, err = os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: approved") {
		t.Errorf("after run 2 completion, expected status: approved; got:\n%s", raw)
	}

	t.Logf("full cycle completed: run1=%s run2=%s", runID1, runID2)
}

// TC 7.2: Full cycle where the QA agent creates a defect.
//
// Sequence:
//   1. Create a test artifact in approved status.
//   2. Start a QA run with a fake claude that writes a defect (exit 0).
//   3. Verify the defect has related_to pointing to the test artifact.
//   4. Verify the test artifact returns to approved after the run.
func TestLifecycleE2E_FullCycleWithDefect(t *testing.T) {
	const testPath = "lifecycle/tests/e2e-with-defect.md"
	const defectPath = "lifecycle/defects/e2e-defect.md"

	// Fake claude writes a defect with related_to and exits 0.
	setupFakeClaudeWritingDefect(t, defectPath, testPath)

	env := newAgentTestEnvWithCfg(t, defectTraceabilityCfgYAML, []seedArtifact{{
		relPath: testPath,
		content: makeArtifact("E2E With Defect", "test", "approved", "e2e-with-defect", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	// Register hub listener to capture agent events.
	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	runID := startAgentRun(t, env, "qa", testPath)
	run := waitForRunCompletion(t, env, runID)

	if got, _ := run["status"].(string); got != "done" {
		t.Fatalf("run expected status 'done', got %q", got)
	}

	// ── Defect verification ────────────────────────────────────────────────────

	absDefectPath := filepath.Join(env.projectRoot, defectPath)
	defectContent, err := os.ReadFile(absDefectPath)
	if err != nil {
		t.Fatalf("defect file not written at %s: %v", absDefectPath, err)
	}
	if !strings.Contains(string(defectContent), testPath) {
		t.Errorf("defect must contain related_to=%q; got:\n%s", testPath, defectContent)
	}

	// Verify defect is indexed with related_to.
	defectRow, err := env.proj.Idx.Get(defectPath)
	if err != nil {
		t.Fatal(err)
	}
	if defectRow == nil {
		t.Fatal("defect not indexed")
	}
	foundRelated := false
	for _, rel := range defectRow.FM.Related {
		if rel == testPath {
			foundRelated = true
			break
		}
	}
	if !foundRelated {
		t.Errorf("indexed defect must have related_to=%q; got: %v", testPath, defectRow.FM.Related)
	}

	// ── Test artifact post-run reset ───────────────────────────────────────────

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, testPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: approved") {
		t.Errorf("test artifact must return to 'approved' after successful run with defect; got:\n%s", raw)
	}

	// ── Hub event verification ─────────────────────────────────────────────────

	// Drain the hub channel and verify we received an agent.finished event
	// with the correct target_path and that a git.committed event was broadcast
	// (defect was committed).
	var gotFinished, gotCommitted bool
	drain := time.After(500 * time.Millisecond)
drainLoop:
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			switch evt.Type {
			case "agent.finished":
				if rid, _ := evt.Payload["run_id"].(string); rid == runID {
					gotFinished = true
				}
			case "git.committed":
				if rid, _ := evt.Payload["run_id"].(string); rid == runID {
					gotCommitted = true
				}
			}
		case <-drain:
			break drainLoop
		}
	}
	if !gotFinished {
		t.Errorf("did not receive agent.finished event for run %s", runID)
	}
	if !gotCommitted {
		t.Errorf("did not receive git.committed event for run %s (defect not committed?)", runID)
	}
}

// TC 7.3: Stale detection data-condition test.
//
// An in-qa test artifact whose file modification time is older than 60 minutes
// meets the condition that causes the lock reaper to broadcast a test.stale
// WebSocket event.  This test verifies the data condition holds after we
// manually set the file mtime, confirming the stale-check logic would fire.
//
// The mtime is verified via os.Stat (not through the index) to avoid races
// with the fsnotify watcher which re-indexes asynchronously.
//
// Note: the actual test.stale WebSocket event is broadcast by the lock reaper
// goroutine on its 60-second tick.  Full event verification would require
// waiting ~60 s and is outside the scope of a normal test run.
func TestLifecycleE2E_StaleDetection(t *testing.T) {
	const artifactPath = "lifecycle/tests/stale-e2e-test.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Stale E2E Test", "test", "approved", "stale-e2e-test", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	// Transition the artifact to in-qa via the API (simulates a mid-run state).
	transResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "in-qa"})
	requireStatus(t, transResp, 200)
	transResp.Body.Close()

	// Verify the index now shows in-qa.
	row, err := env.proj.Idx.Get(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("artifact not found in index")
	}
	if row.Status != "in-qa" {
		t.Errorf("expected status in-qa after transition, got %q", row.Status)
	}

	// Set the file's mtime to 61 minutes ago to exceed the stale threshold.
	// We verify via os.Stat directly (not the index) to avoid races with the
	// fsnotify watcher which may re-index asynchronously after the mtime change.
	staleTime := time.Now().Add(-61 * time.Minute)
	absPath := filepath.Join(env.projectRoot, artifactPath)
	if err := os.Chtimes(absPath, staleTime, staleTime); err != nil {
		t.Fatalf("failed to set stale mtime: %v", err)
	}

	// Confirm os.Stat returns the stale mtime (what the lock reaper reads).
	info, err := os.Stat(absPath)
	if err != nil {
		t.Fatal(err)
	}
	age := time.Since(info.ModTime())
	const staleThreshold = 60 * time.Minute
	if age < staleThreshold {
		t.Errorf("expected file mtime age > %v for stale detection, got %v", staleThreshold, age)
	}

	// Also confirm the file still contains status: in-qa on disk.
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: in-qa") {
		t.Errorf("expected status: in-qa in file content; got:\n%s", raw)
	}

	t.Logf("stale condition verified: path=%s age=%v (threshold=%v)",
		artifactPath, age.Round(time.Second), staleThreshold)
	t.Log("Note: test.stale WebSocket event is broadcast by the lock reaper goroutine " +
		"on its 60-second tick. Full event verification requires waiting ~60s.")
}
