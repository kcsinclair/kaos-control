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
	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)

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

// ── Milestone 3 — WebSocket events include target_path ────────────────────

// TestAgentWSEvents_IncludeTargetPath verifies that both agent.started and
// the terminal event (agent.finished or agent.failed) carry a target_path
// field matching the run's target artifact path.
func TestAgentWSEvents_IncludeTargetPath(t *testing.T) {
	setupFakeClaude(t, 0) // exit 0 → agent.finished

	const artifactPath = "lifecycle/ideas/ws-targetpath-test.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("WS TargetPath Test", "idea", "draft", "ws-targetpath-test", "", "Idea body."),
	}})

	// Register a hub channel before triggering the run so no events are missed.
	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")
	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)

	type wsEvent struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}

	// Collect agent events until we have both a started and a terminal event,
	// or until the timeout fires.
	var startedPayload, terminalPayload map[string]any
	terminalTypes := map[string]bool{"agent.finished": true, "agent.failed": true}
	timeout := time.After(10 * time.Second)
COLLECT:
	for startedPayload == nil || terminalPayload == nil {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			switch {
			case evt.Type == "agent.started":
				startedPayload = evt.Payload
			case terminalTypes[evt.Type]:
				terminalPayload = evt.Payload
			}
		case <-timeout:
			break COLLECT
		}
	}

	// agent.started must have target_path.
	if startedPayload == nil {
		t.Fatalf("never received agent.started event for run %s", runID)
	}
	if tp, _ := startedPayload["target_path"].(string); tp != artifactPath {
		t.Errorf("agent.started target_path: got %q, want %q", tp, artifactPath)
	}
	if rid, _ := startedPayload["run_id"].(string); rid != runID {
		t.Errorf("agent.started run_id: got %q, want %q", rid, runID)
	}

	// Terminal event (agent.finished or agent.failed) must also carry target_path.
	if terminalPayload == nil {
		t.Fatalf("never received agent.finished or agent.failed for run %s", runID)
	}
	if tp, _ := terminalPayload["target_path"].(string); tp != artifactPath {
		t.Errorf("terminal event target_path: got %q, want %q", tp, artifactPath)
	}
}
