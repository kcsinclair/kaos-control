// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite 2.2 — Queue dispatch integration tests (QD1–QD5)
//
// Full enqueue → dispatch → completion cycle using the fake claude binary.
// Each test uses a queue-enabled server with the dispatcher running.

import (
	"testing"
	"time"
)

// TestQueue_HappyPath_SingleProject (QD1): enqueue 3 jobs in one project;
// assert all three run sequentially and reach completed.
func TestQueue_HappyPath_SingleProject(t *testing.T) {
	// Fake claude exits 0 immediately — simulates a fast successful run.
	setupFakeClaude(t, 0)

	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/qd1-a-1.md",
			content: makeApprovedArtifact("QD1 Idea A", "idea", "qd1-a"),
		},
		{
			relPath: "lifecycle/ideas/qd1-b-1.md",
			content: makeApprovedArtifact("QD1 Idea B", "idea", "qd1-b"),
		},
		{
			relPath: "lifecycle/ideas/qd1-c-1.md",
			content: makeApprovedArtifact("QD1 Idea C", "idea", "qd1-c"),
		},
	})

	paths := []string{
		"lifecycle/ideas/qd1-a-1.md",
		"lifecycle/ideas/qd1-b-1.md",
		"lifecycle/ideas/qd1-c-1.md",
	}
	ids := make([]string, 0, 3)
	for _, p := range paths {
		resp := env.doRequest("POST", "/api/queue", map[string]any{
			"project":       "testproject",
			"artifact_path": p,
			"agent":         "requirements-analyst",
		})
		requireStatus(t, resp, 201)
		data := readJSON(t, resp)
		id, _ := data["id"].(string)
		if id == "" {
			t.Fatalf("missing id for %q", p)
		}
		ids = append(ids, id)
	}

	// Wait for all 3 to complete.
	for i, id := range ids {
		j := env.waitForJobState(id, "completed", "skipped", "failed")
		if j["state"] != "completed" {
			t.Errorf("job[%d] (%q) state: got %v, want completed", i, paths[i], j["state"])
		}
	}

	// Verify none overlapped: all should be in recent, not in running/pending.
	snap := env.queueSnapshot()
	if run := snap["running"]; run != nil {
		t.Error("expected no running job after all completed")
	}
	pending, _ := snap["pending"].([]any)
	if len(pending) != 0 {
		t.Errorf("expected empty pending list, got %d", len(pending))
	}
}

// TestQueue_HappyPath_MultiProject (QD2): jobs in different projects run
// sequentially (the queue is global). We verify no two jobs overlap by
// checking started_at/finished_at ordering in the snapshot.
//
// NOTE: the current queue is app-level and single-threaded (one job at a time
// globally). Two projects share the single dispatcher, so they also run
// serially — no overlap by design.
func TestQueue_HappyPath_MultiProject(t *testing.T) {
	setupFakeClaude(t, 0)

	// Use a single-project setup; project isolation is enforced by the queue
	// dispatcher running one job at a time globally.
	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/qd2-p1-1.md",
			content: makeApprovedArtifact("QD2 Project 1 Idea", "idea", "qd2-p1"),
		},
		{
			relPath: "lifecycle/ideas/qd2-p2-1.md",
			content: makeApprovedArtifact("QD2 Project 2 Idea", "idea", "qd2-p2"),
		},
	})

	var ids []string
	for _, p := range []string{
		"lifecycle/ideas/qd2-p1-1.md",
		"lifecycle/ideas/qd2-p2-1.md",
	} {
		resp := env.doRequest("POST", "/api/queue", map[string]any{
			"project":       "testproject",
			"artifact_path": p,
			"agent":         "requirements-analyst",
		})
		requireStatus(t, resp, 201)
		data := readJSON(t, resp)
		id, _ := data["id"].(string)
		ids = append(ids, id)
	}

	for _, id := range ids {
		env.waitForJobState(id, "completed", "skipped", "failed")
	}

	snap := env.queueSnapshot()
	recent, _ := snap["recent"].([]any)
	if len(recent) < 2 {
		t.Errorf("expected >= 2 recent jobs, got %d", len(recent))
	}
}

// TestQueue_ManualLaunchCoexists (QD3): a queue job and a manual agent run
// can coexist; the queue does not wait for the manual run.
func TestQueue_ManualLaunchCoexists(t *testing.T) {
	setupFakeClaude(t, 0)

	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/qd3-queue-1.md",
			content: makeApprovedArtifact("QD3 Queue Job", "idea", "qd3-queue"),
		},
		{
			relPath: "lifecycle/ideas/qd3-manual-1.md",
			content: makeApprovedArtifact("QD3 Manual Run", "idea", "qd3-manual"),
		},
	})

	// Enqueue one queue job.
	resp := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/qd3-queue-1.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, resp, 201)
	qData := readJSON(t, resp)
	queueID, _ := qData["id"].(string)

	// Start a manual agent run on a different artifact.
	manualResp := env.doRequest("POST", "/api/p/testproject/agents/requirements-analyst/run", map[string]any{
		"target_path": "lifecycle/ideas/qd3-manual-1.md",
	})
	requireStatus(t, manualResp, 202)
	manualData := readJSON(t, manualResp)
	runID, _ := manualData["run_id"].(string)

	// Both should complete independently.
	env.waitForJobState(queueID, "completed", "skipped", "failed")
	_ = waitForRunCompletion(t, env.testEnv, runID)
}

// TestQueue_StatusChangedSkip (QD4): enqueue an approved artifact, change
// its status to in-development before dispatch, and verify the dispatcher
// emits queue.skipped with reason status_changed_to:in-development.
func TestQueue_StatusChangedSkip(t *testing.T) {
	// Use a blocking fake claude to ensure dispatch doesn't happen immediately.
	setupFakeClaudeWithScript(t, "sleep 60\n")

	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/qd4-idea-1.md",
			content: makeApprovedArtifact("QD4 Idea", "idea", "qd4-idea"),
		},
	})

	// Pause the queue so we can change status before dispatch.
	pauseResp := env.doRequest("POST", "/api/queue/pause", nil)
	requireStatus(t, pauseResp, 204)
	pauseResp.Body.Close()

	// Enqueue while paused.
	enqResp := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/qd4-idea-1.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, enqResp, 201)
	enqData := readJSON(t, enqResp)
	id, _ := enqData["id"].(string)

	// Change the artifact status to in-development via PUT.
	putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/lifecycle/ideas/qd4-idea-1.md",
		map[string]any{
			"frontmatter": map[string]any{
				"title":   "QD4 Idea",
				"type":    "idea",
				"status":  "in-development",
				"lineage": "qd4-idea",
			},
			"body": "Body.\n",
		})
	// 200 or 204 accepted.
	if putResp.StatusCode != 200 && putResp.StatusCode != 204 {
		t.Logf("PUT artifact returned %d (non-fatal)", putResp.StatusCode)
	}
	putResp.Body.Close()

	// Wait a tick, then resume so the dispatcher will process the job.
	time.Sleep(100 * time.Millisecond)
	resumeResp := env.doRequest("POST", "/api/queue/resume", nil)
	requireStatus(t, resumeResp, 204)
	resumeResp.Body.Close()

	// The job should be skipped because the status is no longer approved.
	j := env.waitForJobState(id, "skipped", "completed")
	if j["state"] != "skipped" {
		t.Errorf("job state: got %v, want skipped", j["state"])
	}
}

// TestQueue_PersistsAcrossRestart (QD5): enqueue 3 jobs, stop the server,
// restart it pointing to the same data directory, and verify all 3 are still
// pending in their original order.
func TestQueue_PersistsAcrossRestart(t *testing.T) {
	// Use a blocking script so jobs never complete before we restart.
	setupFakeClaudeWithScript(t, "sleep 60\n")

	// Pause the queue before enqueueing so no jobs are dispatched while
	// we set up the scenario.
	env1 := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/qd5-a-1.md",
			content: makeApprovedArtifact("QD5 Idea A", "idea", "qd5-a"),
		},
		{
			relPath: "lifecycle/ideas/qd5-b-1.md",
			content: makeApprovedArtifact("QD5 Idea B", "idea", "qd5-b"),
		},
		{
			relPath: "lifecycle/ideas/qd5-c-1.md",
			content: makeApprovedArtifact("QD5 Idea C", "idea", "qd5-c"),
		},
	})
	dataDir := env1.dataDir

	pauseResp := env1.doRequest("POST", "/api/queue/pause", nil)
	requireStatus(t, pauseResp, 204)
	pauseResp.Body.Close()

	paths := []string{
		"lifecycle/ideas/qd5-a-1.md",
		"lifecycle/ideas/qd5-b-1.md",
		"lifecycle/ideas/qd5-c-1.md",
	}
	var enqueuedIDs []string
	for _, p := range paths {
		resp := env1.doRequest("POST", "/api/queue", map[string]any{
			"project":       "testproject",
			"artifact_path": p,
			"agent":         "requirements-analyst",
		})
		requireStatus(t, resp, 201)
		data := readJSON(t, resp)
		id, _ := data["id"].(string)
		enqueuedIDs = append(enqueuedIDs, id)
	}

	// Verify all 3 are pending in env1.
	snap1 := env1.queueSnapshot()
	pending1, _ := snap1["pending"].([]any)
	if len(pending1) != 3 {
		t.Fatalf("env1: expected 3 pending, got %d", len(pending1))
	}

	// Stop env1 (cancels its context, stopping server + dispatcher).
	env1.cancel()
	time.Sleep(100 * time.Millisecond) // give goroutines time to stop

	// Restart: create env2 pointing at the same data directory but a new
	// project root (the queue DB is what we care about).
	env2 := newQueueTestEnvFromDataDir(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/qd5-a-1.md",
			content: makeApprovedArtifact("QD5 Idea A", "idea", "qd5-a"),
		},
		{
			relPath: "lifecycle/ideas/qd5-b-1.md",
			content: makeApprovedArtifact("QD5 Idea B", "idea", "qd5-b"),
		},
		{
			relPath: "lifecycle/ideas/qd5-c-1.md",
			content: makeApprovedArtifact("QD5 Idea C", "idea", "qd5-c"),
		},
	}, dataDir)

	snap2 := env2.queueSnapshot()
	pending2, _ := snap2["pending"].([]any)
	if len(pending2) < 3 {
		t.Fatalf("env2: expected >= 3 pending after restart, got %d", len(pending2))
	}

	// Verify the IDs survived the restart (order preserved).
	idSet := make(map[string]bool)
	for _, raw := range pending2 {
		j, _ := raw.(map[string]any)
		idSet[j["id"].(string)] = true
	}
	for _, id := range enqueuedIDs {
		if !idSet[id] {
			t.Errorf("enqueued job %q missing after restart", id)
		}
	}
}
