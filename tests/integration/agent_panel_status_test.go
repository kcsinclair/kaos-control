// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 4 — Agent Panel Status and Ready Count: Integration / E2E Smoke Tests
//
// End-to-end tests verifying the full flow:
//   - Artifact created → index updated → ready-counts endpoint reflects count
//   - WebSocket artifact.indexed event fires after artifact write → counts re-fetched
//   - Agent start/finish lifecycle: agent.started and agent.finished events broadcast
//
// Configuration: re-uses agentPanelCfgYAML from agents_api_test.go which defines:
//   - agent-with-model:       active_status=clarifying, source_types=[ticket]       (count subject)
//   - agent-no-model:         active_status=planning,   source_types=[plan-backend] (count subject)
//   - agent-no-active-status: no active_status          (must not appear)
//   - idea-capture:           driver=inline, no active_status (must not appear)
//
// Ready counts always filter on status="approved" (the ready-input status) AND
// each agent's source_types. active_status is the during-run status the agent
// transitions the artifact INTO when it picks it up, not the picking-from
// status — so counting by active_status would be the wrong column.

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

// getReadyCounts fetches GET /api/p/testproject/agents/ready-counts and returns
// the decoded "counts" map. The caller must already be logged in.
func getReadyCounts(t *testing.T, env *testEnv) map[string]any {
	t.Helper()
	resp := env.doRequest("GET", "/api/p/testproject/agents/ready-counts", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	counts, _ := data["counts"].(map[string]any)
	if counts == nil {
		t.Fatal("ready-counts response missing 'counts' object")
	}
	return counts
}

// countFor returns the integer value for agentName from a counts map.
// Returns 0 if the key is missing (absent = zero-count agent that was excluded).
func countFor(counts map[string]any, agentName string) int {
	v, ok := counts[agentName]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

// ── Milestone 4, Test Case 1 — Ready count reflects indexed artifacts ─────────

// TestReadyCounts_ReflectsIndexedArtifacts seeds approved artifacts of
// different types and verifies:
//   - agent-with-model (source_types=[ticket]) counts approved tickets
//   - agent-no-model (source_types=[plan-backend]) counts approved plan-backends
//   - agent-no-active-status is absent from the response (handler skips agents without active_status)
//   - Response shape is {"counts": {...}} with numeric values
func TestReadyCounts_ReflectsIndexedArtifacts(t *testing.T) {
	// Seed 2 approved tickets, 1 approved plan-backend, 1 draft idea (must
	// not contribute to any count because draft != approved).
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/ready-count-tkt-1-2.md",
			content: makeArtifact("RC Ticket 1", "ticket", "approved", "ready-count-tkt-1", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/ready-count-tkt-2-2.md",
			content: makeArtifact("RC Ticket 2", "ticket", "approved", "ready-count-tkt-2", "", "Body."),
		},
		{
			relPath: "lifecycle/backend-plans/ready-count-plan-1-3-be.md",
			content: makeArtifact("RC Plan-Backend 1", "plan-backend", "approved", "ready-count-plan-1", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/ready-count-draft-1.md",
			content: makeArtifact("RC Draft 1", "idea", "draft", "ready-count-draft-1", "", "Body."),
		},
	}

	env := newAgentTestEnvWithCfg(t, agentPanelCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	// agent-with-model (source_types=[ticket]) — 2 approved tickets.
	if got := countFor(counts, "agent-with-model"); got != 2 {
		t.Errorf("agent-with-model: want count 2, got %d", got)
	}

	// agent-no-model (source_types=[plan-backend]) — 1 approved plan-backend.
	if got := countFor(counts, "agent-no-model"); got != 1 {
		t.Errorf("agent-no-model: want count 1, got %d", got)
	}

	// agent-no-active-status must NOT appear in the response (handler skips it).
	if _, present := counts["agent-no-active-status"]; present {
		t.Error("agent-no-active-status should be absent from ready-counts (no active_status configured)")
	}

	// idea-capture (inline driver, no active_status) must NOT appear.
	if _, present := counts["idea-capture"]; present {
		t.Error("idea-capture should be absent from ready-counts (no active_status configured)")
	}

	// Response values must be numeric (JSON numbers decode as float64).
	for name, v := range counts {
		switch v.(type) {
		case float64, int:
			// ok
		default:
			t.Errorf("count for %q has unexpected type %T (want numeric)", name, v)
		}
	}
}

// TestReadyCounts_ZeroCountReturned verifies that an agent with active_status
// but no matching artifacts in the index appears in the response with count 0.
func TestReadyCounts_ZeroCountReturned(t *testing.T) {
	// No artifacts at all — all agents with active_status should return 0.
	env := newAgentTestEnvWithCfg(t, agentPanelCfgYAML, nil)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	// Agents with active_status configured must be present with count 0.
	for _, name := range []string{"agent-with-model", "agent-no-model"} {
		v, present := counts[name]
		if !present {
			t.Errorf("%s: expected to be present in response with count 0, was absent", name)
			continue
		}
		n, _ := v.(float64)
		if int(n) != 0 {
			t.Errorf("%s: want count 0, got %v", name, v)
		}
	}
}

// TestReadyCounts_MultipleArtifactsSameStatus verifies that when multiple
// artifacts share the same status+type, the count is correctly aggregated.
func TestReadyCounts_MultipleArtifactsSameStatus(t *testing.T) {
	seeds := make([]seedArtifact, 5)
	for i := range seeds {
		slug := "rc-multi-" + string(rune('a'+i))
		seeds[i] = seedArtifact{
			relPath: "lifecycle/requirements/" + slug + "-2.md",
			content: makeArtifact("RC Multi "+string(rune('A'+i)), "ticket", "approved", slug, "", "Body."),
		}
	}

	env := newAgentTestEnvWithCfg(t, agentPanelCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	if got := countFor(counts, "agent-with-model"); got != 5 {
		t.Errorf("agent-with-model: want count 5 for 5 approved tickets, got %d", got)
	}
}

// TestReadyCounts_NoAgentsConfigured verifies that when no agents are defined
// in config, the endpoint returns {"counts": {}} rather than an error.
func TestReadyCounts_NoAgentsConfigured(t *testing.T) {
	// Use the default config (no agents section).
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/agents/ready-counts", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	counts, ok := data["counts"]
	if !ok {
		t.Fatal("response missing 'counts' key")
	}
	// Must be an object (map), possibly empty.
	countsMap, isMap := counts.(map[string]any)
	if !isMap {
		t.Fatalf("'counts' must be an object, got %T", counts)
	}
	if len(countsMap) != 0 {
		t.Errorf("expected empty counts when no agents configured, got %v", countsMap)
	}
}

// ── Milestone 4, Test Case 2 — Real-time update via WebSocket ────────────────

// TestReadyCounts_RealtimeUpdateAfterArtifactIndexed verifies:
//  1. Initial ready-counts for agent-with-model is 0 (no approved tickets).
//  2. After transitioning an artifact to "approved", an artifact.indexed
//     WebSocket event is received on the hub channel.
//  3. Re-fetching ready-counts returns an incremented count.
func TestReadyCounts_RealtimeUpdateAfterArtifactIndexed(t *testing.T) {
	const artifactPath = "lifecycle/requirements/rc-ws-update-2.md"
	// Seed the artifact in "planning" so the product-owner can transition it
	// to "approved" without going through the required-plans gate.
	env := newAgentTestEnvWithCfg(t, agentPanelCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("RC WS Update", "ticket", "planning", "rc-ws-update", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	// Verify initial count is 0 — no approved tickets yet.
	initialCounts := getReadyCounts(t, env)
	if got := countFor(initialCounts, "agent-with-model"); got != 0 {
		t.Fatalf("agent-with-model: want initial count 0, got %d", got)
	}

	// Register hub channel before triggering the transition.
	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	// Transition artifact from planning → approved — this triggers artifact.indexed broadcast.
	resp := env.doRequest("POST",
		"/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]string{"to": "approved"},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Wait for the artifact.indexed event (timeout 5 s per test plan).
	var gotIndexed bool
	timeout := time.After(5 * time.Second)
COLLECT:
	for !gotIndexed {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type != "artifact.indexed" {
				continue
			}
			path, _ := evt.Payload["path"].(string)
			if path == artifactPath {
				gotIndexed = true
			}
		case <-timeout:
			break COLLECT
		}
	}
	if !gotIndexed {
		t.Fatal("timed out waiting for artifact.indexed event after transition to approved")
	}

	// Re-fetch counts — agent-with-model should now be 1.
	updatedCounts := getReadyCounts(t, env)
	if got := countFor(updatedCounts, "agent-with-model"); got != 1 {
		t.Errorf("agent-with-model: want count 1 after transition, got %d", got)
	}
}

// TestReadyCounts_StatusChangeReflected seeds an artifact at "clarifying",
// writes a status update to the file directly, waits for the watcher to
// re-index, then confirms the count has decremented.
func TestReadyCounts_StatusChangeReflected(t *testing.T) {
	const artifactPath = "lifecycle/requirements/rc-status-change-2.md"
	env := newAgentTestEnvWithCfg(t, agentPanelCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("RC Status Change", "ticket", "approved", "rc-status-change", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	// Initial count must be 1 — one approved ticket matches
	// agent-with-model.source_types=[ticket].
	initial := getReadyCounts(t, env)
	if got := countFor(initial, "agent-with-model"); got != 1 {
		t.Fatalf("agent-with-model: want initial count 1, got %d", got)
	}

	// Register hub to detect re-index.
	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	// Transition via API: approved → in-development (moves the artifact out
	// of the ready-input pool for both agents).
	resp := env.doRequest("POST",
		"/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]string{"to": "in-development"},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Wait for artifact.indexed for our path.
	timeout := time.After(5 * time.Second)
	var reindexed bool
WAIT:
	for !reindexed {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "artifact.indexed" {
				if p, _ := evt.Payload["path"].(string); p == artifactPath {
					reindexed = true
				}
			}
		case <-timeout:
			break WAIT
		}
	}
	if !reindexed {
		t.Fatal("timed out waiting for artifact.indexed after status change")
	}

	// Count for agent-with-model should now be 0 — the ticket is no longer
	// approved, so it doesn't appear in either agent's ready set.
	updated := getReadyCounts(t, env)
	if got := countFor(updated, "agent-with-model"); got != 0 {
		t.Errorf("agent-with-model: want count 0 after status change, got %d", got)
	}
	if got := countFor(updated, "agent-no-model"); got != 0 {
		t.Errorf("agent-no-model: want count 0 (transitioned ticket is wrong type for plan-backend filter anyway), got %d", got)
	}
}

// ── Milestone 4, Test Case 3 — Agent start/finish lifecycle ──────────────────

// TestAgentPanel_StartFinishLifecycle starts an agent run against a real artifact,
// verifies the agent.started WebSocket event is broadcast with the correct payload,
// waits for run completion, then verifies the agent.finished (or agent.failed)
// terminal event is broadcast.
func TestAgentPanel_StartFinishLifecycle(t *testing.T) {
	setupFakeClaude(t, 0) // fake claude exits 0 → agent.finished

	const artifactPath = "lifecycle/ideas/panel-lifecycle-test.md"
	env := newAgentTestEnvWithCfg(t, agentPanelCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Panel Lifecycle Test", "idea", "draft", "panel-lifecycle-test", "", "Idea body."),
	}})

	// Register hub channel before triggering the run.
	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")

	// Start agent run. agent-with-model is claude-code-cli; fake claude will exit 0.
	resp := env.doRequest("POST", "/api/p/testproject/agents/agent-with-model/run", map[string]any{
		"target_path": artifactPath,
	})
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)
	runID, _ := data["run_id"].(string)
	if runID == "" {
		t.Fatal("expected non-empty run_id in 202 response")
	}

	// Collect events until both agent.started and a terminal event are received,
	// or until the 10-second deadline.
	type wsEvt struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}
	var startedPayload, terminalPayload map[string]any
	terminalTypes := map[string]bool{"agent.finished": true, "agent.failed": true}
	timeout := time.After(10 * time.Second)

COLLECT:
	for startedPayload == nil || terminalPayload == nil {
		select {
		case raw := <-ch:
			var evt wsEvt
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			switch {
			case evt.Type == "agent.started" && startedPayload == nil:
				if rid, _ := evt.Payload["run_id"].(string); rid == runID {
					startedPayload = evt.Payload
				}
			case terminalTypes[evt.Type] && terminalPayload == nil:
				if rid, _ := evt.Payload["run_id"].(string); rid == runID {
					terminalPayload = evt.Payload
				}
			}
		case <-timeout:
			break COLLECT
		}
	}

	// Verify agent.started event.
	if startedPayload == nil {
		t.Fatalf("never received agent.started event for run_id=%s", runID)
	}
	if rid, _ := startedPayload["run_id"].(string); rid != runID {
		t.Errorf("agent.started run_id: got %q, want %q", rid, runID)
	}
	if name, _ := startedPayload["agent"].(string); name != "agent-with-model" {
		t.Errorf("agent.started agent name: got %q, want %q", name, "agent-with-model")
	}

	// Verify terminal event (agent.finished since fake claude exits 0).
	if terminalPayload == nil {
		t.Fatalf("never received agent.finished/agent.failed event for run_id=%s", runID)
	}
	if rid, _ := terminalPayload["run_id"].(string); rid != runID {
		t.Errorf("terminal event run_id: got %q, want %q", rid, runID)
	}

	// Verify final run status via API is not "running".
	runResp := env.doRequest("GET", "/api/p/testproject/agents/runs/"+runID, nil)
	requireStatus(t, runResp, http.StatusOK)
	runData := readJSON(t, runResp)
	run, _ := runData["run"].(map[string]any)
	if status, _ := run["status"].(string); status == "running" {
		t.Errorf("run status still 'running' after terminal event received")
	}
}

// TestAgentPanel_AgentStartedEventWhileRunning verifies that after the
// agent.started event is received, the run record is visible in the
// GET /agents/runs list with status "running".
func TestAgentPanel_AgentStartedEventWhileRunning(t *testing.T) {
	// Use a fake claude that sleeps briefly so we can observe the running state.
	fakeDir := t.TempDir()
	// Write a fake claude that sleeps 0.5s then exits 0.
	script := "#!/bin/sh\nsleep 0.5\nexit 0\n"
	fakeScript := filepath.Join(fakeDir, "claude")
	if err := os.WriteFile(fakeScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))

	const artifactPath = "lifecycle/ideas/panel-running-test.md"
	env := newAgentTestEnvWithCfg(t, agentPanelCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Panel Running Test", "idea", "draft", "panel-running-test", "", "Idea body."),
	}})

	// Register hub channel.
	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/agents/agent-with-model/run", map[string]any{
		"target_path": artifactPath,
	})
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)
	runID, _ := data["run_id"].(string)

	// Wait for agent.started.
	timeout := time.After(5 * time.Second)
	var gotStarted bool
STARTED:
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
			if evt.Type == "agent.started" {
				if rid, _ := evt.Payload["run_id"].(string); rid == runID {
					gotStarted = true
				}
			}
		case <-timeout:
			break STARTED
		}
	}
	if !gotStarted {
		t.Fatal("timed out waiting for agent.started")
	}

	// The run must appear in GET /agents/runs with status "running".
	runsResp := env.doRequest("GET", "/api/p/testproject/agents/runs", nil)
	requireStatus(t, runsResp, http.StatusOK)
	runsData := readJSON(t, runsResp)
	runsRaw, _ := runsData["runs"].([]any)

	var foundRunning bool
	for _, r := range runsRaw {
		run, _ := r.(map[string]any)
		if rid, _ := run["run_id"].(string); rid == runID {
			if status, _ := run["status"].(string); status == "running" {
				foundRunning = true
			}
			break
		}
	}
	if !foundRunning {
		t.Errorf("run %s not found with status 'running' in GET /agents/runs immediately after agent.started", runID)
	}

	// Wait for completion so the test cleans up before t.Cleanup fires.
	waitForRunCompletion(t, env, runID)
}
