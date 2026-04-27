//go:build integration

package integration

import (
	"encoding/json"
	"testing"
	"time"
)

// TestAnalystRunBroadcastsStatusChange verifies that when an analyst agent run
// starts, the WebSocket hub broadcasts an "agent.started" event containing the
// correct run_id and lineage.  The hub channel is registered directly (no HTTP
// WebSocket connection required) so the test remains fast and deterministic.
//
// The status change on disk is also verified via the index to confirm that
// setArtifactStatus → idx.IndexFile updated the stored record.
//
// Covers test plan Milestone 4.
func TestAnalystRunBroadcastsStatusChange(t *testing.T) {
	setupFakeClaude(t, 0)

	const artifactPath = "lifecycle/ideas/ws-test.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("WS Test Idea", "idea", "draft", "ws-test", "", "Idea body."),
	}})

	// Register a hub channel before triggering the run so no events are missed.
	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")
	runID := startAgentRun(t, env, "analyst-requirements", artifactPath)

	// agent.started is broadcast synchronously inside StartRun, before the HTTP
	// response is written — the event is already in the channel by the time
	// startAgentRun returns.
	timeout := time.After(5 * time.Second)
	var gotStarted bool
COLLECT:
	for !gotStarted {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type != "agent.started" {
				continue
			}
			gotRunID, _ := evt.Payload["run_id"].(string)
			gotLineage, _ := evt.Payload["lineage"].(string)
			if gotRunID == runID && gotLineage == "ws-test" {
				gotStarted = true
			}
		case <-timeout:
			break COLLECT
		}
	}

	if !gotStarted {
		t.Errorf("never received agent.started event with run_id=%s lineage=ws-test", runID)
	}

	// Verify the re-indexed artifact reflects the new status in the SQLite index.
	row, err := env.proj.Idx.Get(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("artifact not found in index after StartRun")
	}
	if row.Status != "clarifying" {
		t.Errorf("expected index status 'clarifying', got %q", row.Status)
	}
}
