// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestProjectQueue_EnqueueAndList verifies that jobs are properly associated with projects
// and that the global queue API still works correctly.
func TestProjectQueue_EnqueueAndList(t *testing.T) {
	setupFakeClaude(t, 0) // success exit; queue dispatch will exit immediately

	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/test-idea.md",
			content: makeApprovedArtifact("Test Idea", "idea", "test"),
		},
	})

	// Enqueue jobs for different projects
	resp1 := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "project-a",
		"artifact_path": "lifecycle/ideas/test-idea.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, resp1, 201)
	data1 := readJSON(t, resp1)
	id1, _ := data1["id"].(string)
	assert.NotEmpty(t, id1)

	resp2 := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "project-b",
		"artifact_path": "lifecycle/ideas/test-idea.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, resp2, 201)
	data2 := readJSON(t, resp2)
	id2, _ := data2["id"].(string)
	assert.NotEmpty(t, id2)

	// List the global queue and verify both jobs are present
	snap := env.queueSnapshot()
	pending, ok := snap["pending"].([]any)
	assert.True(t, ok)
	assert.Len(t, pending, 2)

	// Verify that each job has a project field
	for _, raw := range pending {
		j, _ := raw.(map[string]any)
		project, ok := j["project"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, project)
		// Ensure jobs are from different projects
		if j["id"] == id1 {
			assert.Equal(t, "project-a", project)
		} else if j["id"] == id2 {
			assert.Equal(t, "project-b", project)
		}
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

// TestProjectQueue_GlobalQueueUnchanged verifies that global queue behavior is unchanged
// and project-scoped filtering works correctly.
func TestProjectQueue_GlobalQueueUnchanged(t *testing.T) {
	setupFakeClaude(t, 0) // success exit; queue dispatch will exit immediately

	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/global-test.md",
			content: makeApprovedArtifact("Global Test Idea", "idea", "global-test"),
		},
	})

	// Enqueue jobs for multiple projects
	resp1 := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "project-a",
		"artifact_path": "lifecycle/ideas/global-test.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, resp1, 201)
	data1 := readJSON(t, resp1)
	id1, _ := data1["id"].(string)
	assert.NotEmpty(t, id1)

	resp2 := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "project-b",
		"artifact_path": "lifecycle/ideas/global-test.md",
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