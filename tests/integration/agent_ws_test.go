// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// ── Milestone 3 — WebSocket events include result payload ─────────────────────

// setupFakeClaudeWithOutput creates a fake `claude` shell script that prints
// outputLines to stdout (one per line) and then exits with exitCode.  It
// prepends its directory to PATH so the agent driver picks it up.
func setupFakeClaudeWithOutput(t *testing.T, outputLines []string, exitCode int) {
	t.Helper()
	fakeDir := t.TempDir()

	// Write the output to a file so the script can cat it, avoiding shell-quoting
	// issues with JSON double-quotes inside the heredoc/printf.
	outputFile := filepath.Join(fakeDir, "output.txt")
	content := ""
	for _, line := range outputLines {
		content += line + "\n"
	}
	if err := os.WriteFile(outputFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	script := fmt.Sprintf("#!/bin/sh\ncat '%s'\nexit %d\n", outputFile, exitCode)
	fakeScript := filepath.Join(fakeDir, "claude")
	if err := os.WriteFile(fakeScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))
}

// TestAgentWSFinished_IncludesResult verifies that when a Claude Code run
// produces a type:result line in its stdout, the agent.finished WebSocket event
// carries a non-null "result" field with the parsed summary fields.
func TestAgentWSFinished_IncludesResult(t *testing.T) {
	const resultLine = `{"type":"result","subtype":"success","total_cost_usd":0.0234,"duration_ms":12345,"duration_api_ms":9800,"num_turns":3,"usage":{"input_tokens":1500,"cache_creation_input_tokens":200,"cache_read_input_tokens":50,"output_tokens":400},"permission_denials":[],"session_id":"ses_ws_result_test"}`
	setupFakeClaudeWithOutput(t, []string{resultLine}, 0)

	const artifactPath = "lifecycle/ideas/ws-result-test.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("WS Result Test", "idea", "draft", "ws-result-test", "", "Idea body."),
	}})

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")
	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)

	type wsEvent struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}

	var terminalPayload map[string]any
	terminalTypes := map[string]bool{"agent.finished": true, "agent.failed": true}
	timeout := time.After(15 * time.Second)
COLLECT:
	for terminalPayload == nil {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if terminalTypes[evt.Type] {
				terminalPayload = evt.Payload
			}
		case <-timeout:
			break COLLECT
		}
	}

	if terminalPayload == nil {
		t.Fatalf("never received agent.finished/agent.failed for run %s", runID)
	}

	result, hasResult := terminalPayload["result"]
	if !hasResult {
		t.Fatal("terminal event payload missing 'result' key")
	}
	if result == nil {
		t.Fatal("expected non-null 'result' in terminal event — fake claude wrote a result line")
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("'result' in terminal event is not an object: %T", result)
	}
	if cost, _ := resultMap["total_cost_usd"].(float64); cost != 0.0234 {
		t.Errorf("result.total_cost_usd: got %v, want 0.0234", cost)
	}
	if subtype, _ := resultMap["subtype"].(string); subtype != "success" {
		t.Errorf("result.subtype: got %q, want %q", subtype, "success")
	}
}

// TestAgentWSFinished_NoResultLine_ResultNull verifies that when a run produces
// no type:result line (e.g. Ollama driver, or a zero-output fake), the
// agent.finished event carries "result": null — not an error event.
func TestAgentWSFinished_NoResultLine_ResultNull(t *testing.T) {
	setupFakeClaude(t, 0) // exits 0 with no output → no result line in log

	const artifactPath = "lifecycle/ideas/ws-null-result-test.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("WS Null Result Test", "idea", "draft", "ws-null-result-test", "", "Idea body."),
	}})

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")
	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)

	type wsEvent struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}

	var terminalPayload map[string]any
	terminalTypes := map[string]bool{"agent.finished": true, "agent.failed": true}
	timeout := time.After(15 * time.Second)
COLLECT2:
	for terminalPayload == nil {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if terminalTypes[evt.Type] {
				terminalPayload = evt.Payload
			}
		case <-timeout:
			break COLLECT2
		}
	}

	if terminalPayload == nil {
		t.Fatalf("never received terminal event for run %s", runID)
	}

	// "result" key must be present in the payload (just null, not absent).
	result, hasResult := terminalPayload["result"]
	if !hasResult {
		t.Fatal("terminal event payload missing 'result' key — key should always be present")
	}
	if result != nil {
		t.Errorf("expected null 'result' for run with no result line, got %v", result)
	}
}

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
