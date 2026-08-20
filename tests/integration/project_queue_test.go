// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestProjectQueue_EnqueueAndList verifies that jobs are associated with their
// project and that the global queue API lists them with a non-empty project
// field (the M1 verification for project-queue-view). Two distinct artifacts
// are used so the duplicate-active suppression (FR3) does not reject the second.
func TestProjectQueue_EnqueueAndList(t *testing.T) {
	setupFakeClaude(t, 0) // success exit; queue dispatch will exit immediately

	env := newQueueTestEnv(t, []seedArtifact{
		{relPath: "lifecycle/ideas/test-idea-a.md", content: makeApprovedArtifact("Test Idea A", "idea", "test-idea-a")},
		{relPath: "lifecycle/ideas/test-idea-b.md", content: makeApprovedArtifact("Test Idea B", "idea", "test-idea-b")},
	})

	resp1 := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/test-idea-a.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, resp1, 201)
	id1, _ := readJSON(t, resp1)["id"].(string)
	assert.NotEmpty(t, id1)

	resp2 := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/test-idea-b.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, resp2, 201)
	id2, _ := readJSON(t, resp2)["id"].(string)
	assert.NotEmpty(t, id2)

	// The global queue lists both jobs, each carrying a non-empty project field
	// — this is what lets the frontend filter client-side without a new endpoint.
	snap := env.queueSnapshot()
	pending, ok := snap["pending"].([]any)
	assert.True(t, ok)
	assert.Len(t, pending, 2)
	for _, raw := range pending {
		j, _ := raw.(map[string]any)
		project, ok := j["project"].(string)
		assert.True(t, ok)
		assert.Equal(t, "testproject", project)
	}
}

// TestProjectQueue_CancelPendingJob verifies that cancelling pending jobs works correctly
// and emits the correct event.
func TestProjectQueue_CancelPendingJob(t *testing.T) {
	setupFakeClaude(t, 0) // success exit; queue dispatch will exit immediately

	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/cancel-test.md",
			content: makeApprovedArtifact("Cancel Test Idea", "idea", "cancel-test"),
		},
	})

	// Enqueue a job
	resp := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/cancel-test.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)
	id, _ := data["id"].(string)
	assert.NotEmpty(t, id)

	// Cancel the job
	cancelResp := env.doRequest("DELETE", "/api/queue/"+id, nil)
	requireStatus(t, cancelResp, 204)
	cancelResp.Body.Close()

	// Verify the job is in the recent list as cancelled
	snap := env.queueSnapshot()
	recent, _ := snap["recent"].([]any)
	found := false
	for _, raw := range recent {
		j, _ := raw.(map[string]any)
		if j["id"] == id {
			found = true
			assert.Equal(t, "cancelled", j["state"])
			assert.Equal(t, "testproject", j["project"])
		}
	}
	assert.True(t, found, "cancelled job should be in recent list")
}

// TestProjectQueue_GlobalQueueUnchanged verifies that the global queue lists all
// jobs and that cancelling one job does not affect the others (NFR-1: no impact
// on global-queue behaviour). Two distinct artifacts avoid FR3 duplicate
// suppression.
func TestProjectQueue_GlobalQueueUnchanged(t *testing.T) {
	setupFakeClaude(t, 0) // success exit; queue dispatch will exit immediately

	env := newQueueTestEnv(t, []seedArtifact{
		{relPath: "lifecycle/ideas/global-test-a.md", content: makeApprovedArtifact("Global Test A", "idea", "global-test-a")},
		{relPath: "lifecycle/ideas/global-test-b.md", content: makeApprovedArtifact("Global Test B", "idea", "global-test-b")},
	})

	resp1 := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/global-test-a.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, resp1, 201)
	data1 := readJSON(t, resp1)
	id1, _ := data1["id"].(string)
	assert.NotEmpty(t, id1)

	resp2 := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/global-test-b.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, resp2, 201)
	data2 := readJSON(t, resp2)
	id2, _ := data2["id"].(string)
	assert.NotEmpty(t, id2)

	// Verify global queue contains both jobs
	snap := env.queueSnapshot()
	pending, ok := snap["pending"].([]any)
	assert.True(t, ok)
	assert.Len(t, pending, 2)

	// Both jobs should be present in the global queue
	jobIds := make(map[string]bool)
	for _, raw := range pending {
		j, _ := raw.(map[string]any)
		jobIds[j["id"].(string)] = true
	}
	assert.True(t, jobIds[id1])
	assert.True(t, jobIds[id2])

	// Test that we can cancel one without affecting the other
	cancelResp := env.doRequest("DELETE", "/api/queue/"+id1, nil)
	requireStatus(t, cancelResp, 204)
	cancelResp.Body.Close()

	// Verify only one is cancelled in global queue
	snap = env.queueSnapshot()
	recent, _ := snap["recent"].([]any)
	foundCancelled := false
	for _, raw := range recent {
		j, _ := raw.(map[string]any)
		if j["id"] == id1 {
			foundCancelled = true
			assert.Equal(t, "cancelled", j["state"])
		}
	}
	assert.True(t, foundCancelled)

	// The other should still be pending
	pending, _ = snap["pending"].([]any)
	assert.Len(t, pending, 1)
	j, _ := pending[0].(map[string]any)
	assert.Equal(t, id2, j["id"])
}
// TestProjectQueue_ArtifactListShowsQueued verifies that an artefact with a
// pending queue job is reported as active_agent_status="queued" by the
// artifacts list — queued jobs have no agent_runs row, so the handler overlays
// the live queue's pending state (fixes: queued-for-agent-status-not-shown).
func TestProjectQueue_ArtifactListShowsQueued(t *testing.T) {
	setupFakeClaude(t, 0)
	env := newQueueTestEnv(t, []seedArtifact{
		{relPath: "lifecycle/ideas/q-idea.md", content: makeApprovedArtifact("Q Idea", "idea", "q-idea")},
	})
	// Hold the queue so the job stays pending (queued) rather than running.
	env.dispatcher.Pause("test-hold")

	resp := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/q-idea.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, resp, 201)

	list := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, list, 200)
	items, _ := readJSON(t, list)["items"].([]any)
	var found bool
	for _, raw := range items {
		it, _ := raw.(map[string]any)
		if it["path"] == "lifecycle/ideas/q-idea.md" {
			found = true
			if st, _ := it["active_agent_status"].(string); st != "queued" {
				t.Errorf("expected active_agent_status=queued in the list, got %q", st)
			}
		}
	}
	if !found {
		t.Error("q-idea artifact not present in the list")
	}
}
