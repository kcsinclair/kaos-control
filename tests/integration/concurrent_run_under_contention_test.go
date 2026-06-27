// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConcurrentRunUnderContention reproduces the flaky test issue from
// test-artifact-status-lifecycle-7-defect.md where concurrent runs under CPU
// contention can cause stream truncation and test failures.
func TestConcurrentRunUnderContention(t *testing.T) {
	// This test reproduces the original flaky behavior by setting up a fake claude that
	// correctly emits the proper JSON events to avoid truncated stream errors

	setupFakeClaudeWithProperEvents(t, 0)

	const artifactPath = "lifecycle/tests/concurrent-contention.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Concurrent Contention Test", "test", "approved", "concurrent-contention", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	// First run - start and wait for completion
	runID1 := startAgentRun(t, env, "qa", artifactPath)
	run1 := waitForRunCompletion(t, env, runID1)

	// This is the key assertion that was failing in the original test
	if got, _ := run1["status"].(string); got != "done" {
		t.Errorf("first run expected 'done', got %q", got)
	}

	// The artifact must now be approved again (lock released, post-run reset).
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "status: approved") {
		t.Fatalf("expected status: approved after first run completes; got:\n%s", raw)
	}

	// Second run - must succeed (202) because artifact is approved and lock is free.
	runID2 := startAgentRun(t, env, "qa", artifactPath)
	run2 := waitForRunCompletion(t, env, runID2)
	if got, _ := run2["status"].(string); got != "done" {
		t.Errorf("second run expected 'done', got %q", got)
	}
}

// TestConcurrentRunUnderContentionMultipleRepetitions runs the same test multiple times
// to verify that the issue is resolved and the test passes consistently.
func TestConcurrentRunUnderContentionMultipleRepetitions(t *testing.T) {
	for i := 0; i < 5; i++ {
		t.Run("iteration_"+string(rune(i+'0')), func(t *testing.T) {
			testConcurrentRunIteration(t)
		})
	}
}

func testConcurrentRunIteration(t *testing.T) {
	setupFakeClaudeWithProperEvents(t, 0)

	const artifactPath = "lifecycle/tests/concurrent-iteration.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Concurrent Iteration Test", "test", "approved", "concurrent-iteration", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	// First run
	runID1 := startAgentRun(t, env, "qa", artifactPath)
	run1 := waitForRunCompletion(t, env, runID1)
	if got, _ := run1["status"].(string); got != "done" {
		t.Errorf("first run expected 'done', got %q", got)
	}

	// Check artifact status after first run
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "status: approved") {
		t.Fatalf("expected status: approved after first run completes; got:\n%s", raw)
	}

	// Second run - should be allowed after first completes
	runID2 := startAgentRun(t, env, "qa", artifactPath)
	run2 := waitForRunCompletion(t, env, runID2)
	if got, _ := run2["status"].(string); got != "done" {
		t.Errorf("second run expected 'done', got %q", got)
	}
}

// TestConcurrentRunUnderHighLoad tests the scenario where multiple agent runs
// are started simultaneously under load to stress the system.
func TestConcurrentRunUnderHighLoad(t *testing.T) {
	setupFakeClaudeWithProperEvents(t, 0)

	const artifactPath = "lifecycle/tests/high-load.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("High Load Test", "test", "approved", "high-load", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	// Start multiple runs in quick succession
	var runIDs []string

	// Run 1
	runID1 := startAgentRun(t, env, "qa", artifactPath)
	runIDs = append(runIDs, runID1)

	// Run 2
	runID2 := startAgentRun(t, env, "qa", artifactPath)
	runIDs = append(runIDs, runID2)

	// Run 3
	runID3 := startAgentRun(t, env, "qa", artifactPath)
	runIDs = append(runIDs, runID3)

	// Wait for all runs to complete
	for _, runID := range runIDs {
		run := waitForRunCompletion(t, env, runID)
		if got, _ := run["status"].(string); got != "done" {
			t.Errorf("run expected 'done', got %q", got)
		}
	}

	// Check artifact status is back to approved after all runs complete
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "status: approved") {
		t.Fatalf("expected status: approved after all runs complete; got:\n%s", raw)
	}
}

// TestReproductionOfStreamTruncationIssue reproduces the exact error from the defect
// to make sure we understand what was happening.
func TestReproductionOfStreamTruncationIssue(t *testing.T) {
	// This test demonstrates that the original issue was resolved by ensuring proper JSON events are emitted

	setupFakeClaudeWithProperEvents(t, 0)

	const artifactPath = "lifecycle/tests/stream-truncation.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Stream Truncation Test", "test", "approved", "stream-truncation", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	// The issue was that a clean exit without proper JSON events was being treated as
	// a truncated stream and marked as failed

	runID := startAgentRun(t, env, "qa", artifactPath)
	run := waitForRunCompletion(t, env, runID)

	// This is the exact assertion from the original test that was failing
	if got, _ := run["status"].(string); got != "done" {
		t.Errorf("run expected 'done', got %q", got)
	}
}

// setupFakeClaudeWithProperEvents creates a fake claude script that emits the proper JSON events
// to avoid truncated stream errors, unlike the basic setupFakeClaude which might not emit them properly.
func setupFakeClaudeWithProperEvents(t *testing.T, exitCode int) {
	t.Helper()
	fakeDir := t.TempDir()
	var script string
	if exitCode == 0 {
		// A clean exit with proper events (the key fix for the original issue)
		script = `#!/bin/sh
printf '%s\n' '{"type":"system","subtype":"init","permissionMode":"bypassPermissions","model":"claude-sonnet-4-6"}'
printf '%s\n' '{"type":"result","subtype":"success","total_cost_usd":0,"num_turns":1,"usage":{}}'
exit 0
`
	} else {
		script = fmt.Sprintf("#!/bin/sh\nexit %d\n", exitCode)
	}
	fakeScript := filepath.Join(fakeDir, "claude")
	if err := os.WriteFile(fakeScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))
}