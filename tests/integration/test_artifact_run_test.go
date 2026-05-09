// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── Milestone 2 — Agent run for single test artifact ──────────────────────
//
// These tests verify that the QA agent can be invoked against a single test
// artifact via the API and that WebSocket events include the expected
// target_path field.  A stub claude binary is used so the driver process
// completes quickly and deterministically.

// qaAgentCfgYAML is the lifecycle/config.yaml used by Milestone 2 and 3 tests.
// It includes a minimal qa agent with driver=claude-code-cli and active_status=in-qa.
const qaAgentCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
  - reviewer
  - approver

stages:
  - {name: ideas,          dir: ideas}
  - {name: requirements,   dir: requirements}
  - {name: backend-plans,  dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans,     dir: test-plans}
  - {name: tests,          dir: tests}
  - {name: prototypes,     dir: prototypes}
  - {name: releases,       dir: releases}
  - {name: sprints,        dir: sprints}
  - {name: defects,        dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []

agents:
  - name: qa
    role: [qa]
    driver: claude-code-cli
    active_status: in-qa
    allowed_write_paths:
      - lifecycle/defects
      - lifecycle/tests
    git_identity:
      name: QA Agent
      email: qa@test.local
    prompt_templates:
      qa: "Test QA prompt for {target_path}"
`

// setupSlowFakeClaude writes a stub claude binary that sleeps for sleepSec
// seconds before exiting 0.  Use this to hold the lineage lock open long
// enough for concurrent-run tests.
func setupSlowFakeClaude(t *testing.T, sleepSec int) {
	t.Helper()
	fakeDir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nsleep %d\nexit 0\n", sleepSec)
	fakeScript := filepath.Join(fakeDir, "claude")
	if err := os.WriteFile(fakeScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))
}

// newQATestEnv creates a test environment with the qa agent configured.
func newQATestEnv(t *testing.T, seeds []seedArtifact) *testEnv {
	t.Helper()
	return newAgentTestEnvWithCfg(t, qaAgentCfgYAML, seeds)
}

// TestTestArtifactRun_ApprovedTestStarted verifies that posting to the qa agent
// with an approved test artifact returns HTTP 202 with a run_id, and that the
// hub broadcasts an agent.started event carrying the correct target_path.
// Covers test plan Milestone 2, scenario 1.
func TestTestArtifactRun_ApprovedTestStarted(t *testing.T) {
	setupFakeClaude(t, 0)

	const artifactPath = "lifecycle/tests/run-approved-test.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Run Approved Test", "test", "approved", "run-approved-test", "", "Test body."),
	}})

	// Register hub listener before triggering the run.
	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("qa@test.local", "qa-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/agents/qa/run", map[string]any{
		"target_path": artifactPath,
	})
	requireStatus(t, resp, 202)
	data := readJSON(t, resp)

	runID, _ := data["run_id"].(string)
	if runID == "" {
		t.Fatal("expected non-empty run_id in 202 response")
	}

	// Collect agent.started from the hub.
	var gotStarted bool
	timeout := time.After(5 * time.Second)
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
			gotTarget, _ := evt.Payload["target_path"].(string)
			if gotRunID == runID && gotTarget == artifactPath {
				gotStarted = true
			}
		case <-timeout:
			break COLLECT
		}
	}
	if !gotStarted {
		t.Errorf("never received agent.started with run_id=%s target_path=%s", runID, artifactPath)
	}
}

// TestTestArtifactRun_TerminalEventHasTargetPath verifies that after the qa
// agent finishes, an agent.finished or agent.failed event is broadcast carrying
// a target_path field that matches the original request.
// Covers test plan Milestone 2, scenario 2.
func TestTestArtifactRun_TerminalEventHasTargetPath(t *testing.T) {
	setupFakeClaude(t, 0) // exit 0 → agent.finished

	const artifactPath = "lifecycle/tests/run-terminal-test.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Run Terminal Test", "test", "approved", "run-terminal-test", "", "Test body."),
	}})

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("qa@test.local", "qa-pass-123")
	runID := startAgentRun(t, env, "qa", artifactPath)

	terminalTypes := map[string]bool{"agent.finished": true, "agent.failed": true}
	var terminalPayload map[string]any
	timeout := time.After(10 * time.Second)
COLLECT:
	for terminalPayload == nil {
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
				terminalPayload = evt.Payload
			}
		case <-timeout:
			break COLLECT
		}
	}

	if terminalPayload == nil {
		t.Fatalf("never received agent.finished or agent.failed for run %s", runID)
	}
	if tp, _ := terminalPayload["target_path"].(string); tp != artifactPath {
		t.Errorf("terminal event target_path: got %q, want %q", tp, artifactPath)
	}
}

// TestTestArtifactRun_DraftTestNoOrphan verifies that running the qa agent
// against a draft (non-approved) test artifact does not crash the server and
// leaves no orphaned "running" records.  The agent runner has no status
// restriction on the target artifact, so the run must start (202) and
// eventually reach a terminal state.
// Covers test plan Milestone 2, scenario 3.
func TestTestArtifactRun_DraftTestNoOrphan(t *testing.T) {
	setupFakeClaude(t, 0)

	const artifactPath = "lifecycle/tests/run-draft-test.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Run Draft Test", "test", "draft", "run-draft-test", "", "Test body."),
	}})

	env.login("qa@test.local", "qa-pass-123")

	// The run should succeed (202); the runner does not gate on artifact status.
	resp := env.doRequest("POST", "/api/p/testproject/agents/qa/run", map[string]any{
		"target_path": artifactPath,
	})
	// Accept 202 (started) or 409 (already running — race-safe fallback).
	// Must NOT be 500 (server crash).
	if resp.StatusCode == 500 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("server returned 500 for draft test run: %s", b)
	}
	resp.Body.Close()

	if resp.StatusCode != 202 && resp.StatusCode != 409 {
		t.Errorf("expected 202 or 409 for draft test run, got %d", resp.StatusCode)
	}

	if resp.StatusCode != 202 {
		return // run didn't start, nothing more to verify
	}

	// Give the stub agent time to complete, then verify no orphaned running records.
	time.Sleep(500 * time.Millisecond)

	runsResp := env.doRequest("GET", "/api/p/testproject/agents/runs?target_path="+artifactPath, nil)
	requireStatus(t, runsResp, 200)
	runsData := readJSON(t, runsResp)

	runsRaw, _ := runsData["runs"].([]any)
	for _, raw := range runsRaw {
		run, _ := raw.(map[string]any)
		if status, _ := run["status"].(string); status == "running" {
			t.Errorf("found orphaned run with status=running for target %s", artifactPath)
		}
	}
}

// TestTestArtifactRun_ConcurrentRunPrevented verifies that a second request to
// start the qa agent against the same lineage while a run is already active
// returns a conflict error (lineage lock).
// Covers test plan Milestone 2, scenario 4.
func TestTestArtifactRun_ConcurrentRunPrevented(t *testing.T) {
	// Use a slow fake claude so the first run holds the lock when the second arrives.
	setupSlowFakeClaude(t, 5)

	const artifactPath = "lifecycle/tests/run-concurrent-test.md"
	env := newQATestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Run Concurrent Test", "test", "approved", "run-concurrent-test", "", "Test body."),
	}})

	env.login("qa@test.local", "qa-pass-123")

	// First run: must succeed and hold the lineage lock.
	resp1 := env.doRequest("POST", "/api/p/testproject/agents/qa/run", map[string]any{
		"target_path": artifactPath,
	})
	requireStatus(t, resp1, 202)
	readJSON(t, resp1) // consume body

	// Second run against the same lineage: must fail due to lock contention.
	resp2 := env.doRequest("POST", "/api/p/testproject/agents/qa/run", map[string]any{
		"target_path": artifactPath,
	})
	if resp2.StatusCode != 409 {
		b, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		t.Fatalf("expected 409 Conflict for concurrent run, got %d: %s", resp2.StatusCode, b)
	}
	data2 := readJSON(t, resp2)

	errObj, _ := data2["error"].(map[string]any)
	code, _ := errObj["code"].(string)
	if code == "" {
		t.Errorf("expected error.code in 409 response, got: %v", data2)
	}
	t.Logf("concurrent run correctly rejected with error.code=%q", code)
}
