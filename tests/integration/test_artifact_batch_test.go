//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// ── Milestone 3 — Serial batch execution ──────────────────────────────────
//
// These tests verify that multiple test artifacts can be executed serially:
// each run must complete before the next begins, a failed run must not block
// subsequent runs, and agent_runs records must reach a terminal state.
//
// All tests listen to hub WS events rather than using sleep-based polling,
// ensuring there is no timing-dependent flakiness.

// batchTestSeeds returns 3 approved test artifacts for use in batch tests.
func batchTestSeeds() []seedArtifact {
	return []seedArtifact{
		{
			relPath: "lifecycle/tests/batch-test-1.md",
			content: makeArtifact("Batch Test 1", "test", "approved", "batch-test-1", "", "Test body."),
		},
		{
			relPath: "lifecycle/tests/batch-test-2.md",
			content: makeArtifact("Batch Test 2", "test", "approved", "batch-test-2", "", "Test body."),
		},
		{
			relPath: "lifecycle/tests/batch-test-3.md",
			content: makeArtifact("Batch Test 3", "test", "approved", "batch-test-3", "", "Test body."),
		},
	}
}

// waitForWSTerminalEvent blocks until an agent.finished or agent.failed event
// for the specified runID is received from the hub channel, or until timeout.
// Returns the event type and payload.  Calls t.Fatalf on timeout.
func waitForWSTerminalEvent(t *testing.T, ch <-chan []byte, runID string, timeout time.Duration) (string, map[string]any) {
	t.Helper()
	deadline := time.After(timeout)
	terminalTypes := map[string]bool{"agent.finished": true, "agent.failed": true}
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
			if !terminalTypes[evt.Type] {
				continue
			}
			if rid, _ := evt.Payload["run_id"].(string); rid == runID {
				return evt.Type, evt.Payload
			}
		case <-deadline:
			t.Fatalf("timed out after %v waiting for terminal WS event for run %s", timeout, runID)
			return "", nil
		}
	}
}

// TestTestArtifactBatch_SerialExecution submits 3 approved test artifacts
// sequentially — waiting for the agent.finished event before starting the
// next — and verifies all 3 complete successfully.
// Covers test plan Milestone 3, scenario 1.
func TestTestArtifactBatch_SerialExecution(t *testing.T) {
	setupFakeClaude(t, 0) // exit 0 → agent.finished

	env := newQATestEnv(t, batchTestSeeds())

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("qa@test.local", "qa-pass-123")

	paths := []string{
		"lifecycle/tests/batch-test-1.md",
		"lifecycle/tests/batch-test-2.md",
		"lifecycle/tests/batch-test-3.md",
	}

	for i, path := range paths {
		t.Logf("starting run %d/%d: %s", i+1, len(paths), path)

		runID := startAgentRun(t, env, "qa", path)

		evtType, _ := waitForWSTerminalEvent(t, ch, runID, 10*time.Second)
		t.Logf("run %d completed with event %q", i+1, evtType)

		if evtType != "agent.finished" {
			t.Errorf("run %d: expected agent.finished, got %q", i+1, evtType)
		}
	}
}

// TestTestArtifactBatch_FailureDoesNotHaltBatch verifies that after a failed
// test run (agent exits non-zero), the next test in the batch can still be
// started without lock contention or server errors.
// Covers test plan Milestone 3, scenario 2.
func TestTestArtifactBatch_FailureDoesNotHaltBatch(t *testing.T) {
	setupFakeClaude(t, 1) // exit 1 → agent.failed

	seeds := []seedArtifact{
		{
			relPath: "lifecycle/tests/batch-fail-1.md",
			content: makeArtifact("Batch Fail 1", "test", "approved", "batch-fail-1", "", "Test body."),
		},
		{
			relPath: "lifecycle/tests/batch-fail-2.md",
			content: makeArtifact("Batch Fail 2", "test", "approved", "batch-fail-2", "", "Test body."),
		},
	}
	env := newQATestEnv(t, seeds)

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("qa@test.local", "qa-pass-123")

	// First run — expect failure.
	runID1 := startAgentRun(t, env, "qa", seeds[0].relPath)
	evtType1, _ := waitForWSTerminalEvent(t, ch, runID1, 10*time.Second)
	if evtType1 != "agent.failed" {
		t.Errorf("run 1: expected agent.failed, got %q", evtType1)
	}

	// Second run — must start successfully despite the previous failure.
	resp2 := env.doRequest("POST", "/api/p/testproject/agents/qa/run", map[string]any{
		"target_path": seeds[1].relPath,
	})
	requireStatus(t, resp2, 202)
	data2 := readJSON(t, resp2)
	runID2, _ := data2["run_id"].(string)
	if runID2 == "" {
		t.Fatal("expected run_id in 202 response for second batch test")
	}

	// Wait for the second run to complete as well.
	evtType2, _ := waitForWSTerminalEvent(t, ch, runID2, 10*time.Second)
	t.Logf("run 2 completed with event %q (expected agent.failed due to stub)", evtType2)
}

// TestTestArtifactBatch_LockReleaseAfterFinished verifies that immediately
// after receiving agent.finished, starting the next run for a different test
// succeeds without lock contention.
// Covers test plan Milestone 3, scenario 3.
func TestTestArtifactBatch_LockReleaseAfterFinished(t *testing.T) {
	setupFakeClaude(t, 0) // exit 0 → agent.finished

	seeds := []seedArtifact{
		{
			relPath: "lifecycle/tests/batch-lock-1.md",
			content: makeArtifact("Batch Lock 1", "test", "approved", "batch-lock-1", "", "Test body."),
		},
		{
			relPath: "lifecycle/tests/batch-lock-2.md",
			content: makeArtifact("Batch Lock 2", "test", "approved", "batch-lock-2", "", "Test body."),
		},
	}
	env := newQATestEnv(t, seeds)

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("qa@test.local", "qa-pass-123")

	// Start first run and wait for it to finish.
	runID1 := startAgentRun(t, env, "qa", seeds[0].relPath)
	waitForWSTerminalEvent(t, ch, runID1, 10*time.Second)

	// Immediately start second run — must get 202, not 409.
	resp2 := env.doRequest("POST", "/api/p/testproject/agents/qa/run", map[string]any{
		"target_path": seeds[1].relPath,
	})
	requireStatus(t, resp2, 202) // would be 409 if lock from run 1 was still held
	data2 := readJSON(t, resp2)
	runID2, _ := data2["run_id"].(string)
	if runID2 == "" {
		t.Fatal("expected run_id in 202 response for run 2")
	}

	// Allow run 2 to complete to avoid leaving a running agent at cleanup.
	waitForWSTerminalEvent(t, ch, runID2, 10*time.Second)
}

// TestTestArtifactBatch_RunRecordsHaveTerminalStatus verifies that after
// completing 3 serial runs, the agent_runs table contains one record per test
// path, each with a terminal status (done or failed, not running).
// Covers test plan Milestone 3, scenario 4.
func TestTestArtifactBatch_RunRecordsHaveTerminalStatus(t *testing.T) {
	setupFakeClaude(t, 0)

	env := newQATestEnv(t, batchTestSeeds())

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("qa@test.local", "qa-pass-123")

	paths := []string{
		"lifecycle/tests/batch-test-1.md",
		"lifecycle/tests/batch-test-2.md",
		"lifecycle/tests/batch-test-3.md",
	}
	runIDs := make([]string, 0, len(paths))

	// Execute serially, collecting run IDs.
	for _, path := range paths {
		runID := startAgentRun(t, env, "qa", path)
		runIDs = append(runIDs, runID)
		waitForWSTerminalEvent(t, ch, runID, 10*time.Second)
	}

	// Verify each run record has a terminal status.
	for i, runID := range runIDs {
		resp := env.doRequest("GET", fmt.Sprintf("/api/p/testproject/agents/runs/%s", runID), nil)
		requireStatus(t, resp, 200)
		data := readJSON(t, resp)

		run, _ := data["run"].(map[string]any)
		status, _ := run["status"].(string)
		if status == "running" || status == "" {
			t.Errorf("run %d (%s): expected terminal status, got %q", i+1, runID, status)
		}
		t.Logf("run %d (%s): status=%q", i+1, runID, status)
	}
}
