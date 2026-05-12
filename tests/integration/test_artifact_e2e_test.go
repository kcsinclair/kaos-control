// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"
)

// ── Milestone 5 — End-to-end Testing board workflow ───────────────────────
//
// TestTestArtifactE2E_FullWorkflow implements the API-level simulation described
// in lifecycle/tests/test-artifact-management-e2e.md. It seeds 5 test artifacts,
// verifies filtering, executes 3 approved tests serially via the QA agent,
// confirms agent run records reach terminal status, and checks that the index
// reflects status=in-qa for the executed artifacts.
//
// Exit-1 stub is used so that artifacts remain in in-qa after each run.
// The post-run reset to "approved" only fires on exit-0 (successful) runs;
// for failed runs the active_status (in-qa) set at run-start persists,
// which is what step 6 verifies.
func TestTestArtifactE2E_FullWorkflow(t *testing.T) {
	// Use a failing stub: agent exits 1 → agent.failed event, artifact stays in-qa.
	setupFakeClaude(t, 1)

	// Step 1 — Seed the project with 5 test artifacts.
	// 3 approved (a, b, c), 1 draft (d), 1 done (e).
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/tests/e2e-test-a.md",
			content: makeArtifact("E2E Test A", "test", "approved", "e2e-test-a", "", "Test body."),
		},
		{
			relPath: "lifecycle/tests/e2e-test-b.md",
			content: makeArtifact("E2E Test B", "test", "approved", "e2e-test-b", "", "Test body."),
		},
		{
			relPath: "lifecycle/tests/e2e-test-c.md",
			content: makeArtifact("E2E Test C", "test", "approved", "e2e-test-c", "", "Test body."),
		},
		{
			relPath: "lifecycle/tests/e2e-test-d.md",
			content: makeArtifact("E2E Test D", "test", "draft", "e2e-test-d", "", "Test body."),
		},
		{
			relPath: "lifecycle/tests/e2e-test-e.md",
			content: makeArtifact("E2E Test E", "test", "done", "e2e-test-e", "", "Test body."),
		},
	}

	env := newQATestEnv(t, seeds)

	// Register hub channel before triggering any runs.
	ch := make(chan []byte, 256)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("qa@test.local", "qa-pass-123")

	// Step 2 — Verify unfiltered listing returns all 5 test artifacts.
	resp := env.doRequest("GET", "/api/p/testproject/artifacts?type=test", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	total, _ := data["total"].(float64)
	if int(total) != 5 {
		t.Errorf("step 2: expected total=5 for type=test, got %d", int(total))
	}
	items, _ := data["items"].([]any)
	if len(items) != 5 {
		t.Errorf("step 2: expected 5 items for type=test, got %d", len(items))
	}
	t.Logf("step 2 passed: total=%d", int(total))

	// Step 3 — Verify approved filter returns exactly 3.
	resp = env.doRequest("GET", "/api/p/testproject/artifacts?type=test&status=approved", nil)
	requireStatus(t, resp, 200)
	data = readJSON(t, resp)

	approvedTotal, _ := data["total"].(float64)
	if int(approvedTotal) != 3 {
		t.Errorf("step 3: expected total=3 for type=test&status=approved, got %d", int(approvedTotal))
	}
	approvedItems, _ := data["items"].([]any)
	for _, raw := range approvedItems {
		item, _ := raw.(map[string]any)
		if status, _ := item["status"].(string); status != "approved" {
			t.Errorf("step 3: approved filter returned artifact with status=%q", status)
		}
	}
	t.Logf("step 3 passed: approved total=%d", int(approvedTotal))

	// Step 4 — Execute the 3 approved tests serially.
	// Wait for the terminal WS event before starting the next run.
	approvedPaths := []string{
		"lifecycle/tests/e2e-test-a.md",
		"lifecycle/tests/e2e-test-b.md",
		"lifecycle/tests/e2e-test-c.md",
	}
	runIDs := make([]string, 0, len(approvedPaths))

	for i, path := range approvedPaths {
		t.Logf("step 4: starting run %d/%d: %s", i+1, len(approvedPaths), path)

		resp := env.doRequest("POST", "/api/p/testproject/agents/qa/run", map[string]any{
			"target_path": path,
		})
		requireStatus(t, resp, 202)
		runData := readJSON(t, resp)

		runID, _ := runData["run_id"].(string)
		if runID == "" {
			t.Fatalf("step 4: run %d: expected non-empty run_id in 202 response", i+1)
		}
		runIDs = append(runIDs, runID)

		// Block until terminal event arrives — no timing-dependent sleeps.
		evtType, _ := waitForWSTerminalEvent(t, ch, runID, 10*time.Second)
		t.Logf("step 4: run %d (%s) completed with event %q", i+1, runID, evtType)
	}

	// Step 5 — Verify agent run records have a terminal status (done or failed).
	for i, runID := range runIDs {
		resp := env.doRequest("GET", fmt.Sprintf("/api/p/testproject/agents/runs/%s", runID), nil)
		requireStatus(t, resp, 200)
		runData := readJSON(t, resp)

		run, _ := runData["run"].(map[string]any)
		status, _ := run["status"].(string)
		if status == "running" || status == "" {
			t.Errorf("step 5: run %d (%s): expected terminal status, got %q", i+1, runID, status)
		} else {
			t.Logf("step 5: run %d (%s): status=%q (terminal)", i+1, runID, status)
		}
	}

	// Step 6 — Verify the index reflects in-qa status for the 3 executed artifacts.
	// Failed runs (exit 1) leave the active_status (in-qa) in place because the
	// post-run reset to "approved" only fires on successful (exit 0) completion.
	resp = env.doRequest("GET", "/api/p/testproject/artifacts?type=test&status=in-qa", nil)
	requireStatus(t, resp, 200)
	data = readJSON(t, resp)

	inQATotal, _ := data["total"].(float64)
	if int(inQATotal) != 3 {
		t.Errorf("step 6: expected total=3 for type=test&status=in-qa after runs, got %d", int(inQATotal))
	}
	t.Logf("step 6 passed: in-qa total=%d", int(inQATotal))
}
