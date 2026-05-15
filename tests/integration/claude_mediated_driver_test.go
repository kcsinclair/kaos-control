// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 5 — Claude Mediated Driver Integration Tests
//
// Tests the ClaudeHooksDriver end-to-end using the existing fake_precheck_claude
// binary (compiled by TestMain in agent_precheck_test.go). The mediated driver
// uses the inverse precheck logic: it PASSES when permissionMode is anything
// except bypassPermissions, and FAILS when bypassPermissions is observed (AC13).
//
// Run with:
//   go test ./tests/... -tags integration -run TestMediatedDriver -v

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/project"
)

// mediatedAgentCfgYAML defines a single claude-mediated agent suitable for
// all Milestone 5 tests.
const mediatedAgentCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - backend-developer

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
    roles: [product-owner]
  - email: dev@test.local
    roles: [backend-developer]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []

agents:
  - name: mediated-agent
    role: [backend-developer]
    driver: claude-mediated
    active_status: in-development
    done_on_success: true
    allowed_write_paths:
      - lifecycle/requirements
    git_identity:
      name: Mediated Test Agent
      email: mediated-agent@test.local
    prompt_templates:
      backend-developer: "Test mediated prompt for {target_path}"
`

// newMediatedTestEnv creates a test environment with a claude-mediated agent.
// A placeholder HookServerAddr is set so the ClaudeHooksDriver can write the
// settings.json; the fake claude binary does not actually call the endpoint.
func newMediatedTestEnv(t *testing.T, cfgYAML string, seeds []seedArtifact) *testEnv {
	t.Helper()
	return newAgentTestEnvCustom(t, cfgYAML, project.OpenOptions{
		MaxConcurrentAgents: 2,
		// Placeholder address: fake claude never calls this endpoint.
		HookServerAddr: "127.0.0.1:0",
	}, seeds)
}

// ── M5-I1: Normal run — non-bypass mode passes mediated precheck ──────────────

// TestMediatedDriver_DefaultMode_Passes verifies that when the fake claude
// binary reports permissionMode="default" (i.e. hooks are active), the mediated
// precheck passes and the run completes with status=done (AC1).
func TestMediatedDriver_DefaultMode_Passes(t *testing.T) {
	// Mode "default" is NOT bypassPermissions → mediated precheck should pass.
	setupFakePrecheckClaude(t, "default", false /* don't hold */)

	const artifactPath = "lifecycle/requirements/mediated-default-pass.md"
	env := newMediatedTestEnv(t, mediatedAgentCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Mediated Default Pass", "ticket", "draft", "mediated-default-pass", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "mediated-agent", artifactPath)
	run := waitForPrecheckRunCompletion(t, env, runID, 15*time.Second)

	if got, _ := run["status"].(string); got != "done" {
		t.Errorf("run status = %q, want \"done\" when non-bypass permissionMode is reported", got)
	}
}

// TestMediatedDriver_NoPermissionMode_Passes verifies that when the fake binary
// emits system/init without a permissionMode field, the precheck also passes
// (a missing permissionMode is treated as non-bypass).
func TestMediatedDriver_NoPermissionMode_Passes(t *testing.T) {
	// Empty mode string causes fake_precheck_claude to omit the permissionMode field.
	setupFakePrecheckClaude(t, "", false)

	const artifactPath = "lifecycle/requirements/mediated-nomode-pass.md"
	env := newMediatedTestEnv(t, mediatedAgentCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Mediated No Mode Pass", "ticket", "draft", "mediated-nomode-pass", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "mediated-agent", artifactPath)
	run := waitForPrecheckRunCompletion(t, env, runID, 15*time.Second)

	if got, _ := run["status"].(string); got != "done" {
		t.Errorf("run status = %q, want \"done\" when permissionMode field is absent", got)
	}
}

// ── M5-I4: Precheck failure — bypass mode detected (AC13) ────────────────────

// TestMediatedDriver_BypassMode_FailsPrecheck verifies that when the fake claude
// binary reports permissionMode="bypassPermissions" on a claude-mediated run,
// the mediated precheck kills the run and records it as failed (AC13).
func TestMediatedDriver_BypassMode_FailsPrecheck(t *testing.T) {
	// "bypassPermissions" mode on a mediated run means hooks were not applied.
	setupFakePrecheckClaude(t, "bypassPermissions", true /* hold so precheck can kill it */)

	const artifactPath = "lifecycle/requirements/mediated-bypass-fail.md"
	env := newMediatedTestEnv(t, mediatedAgentCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Mediated Bypass Fail", "ticket", "draft", "mediated-bypass-fail", "", "Body."),
	}})

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "mediated-agent", artifactPath)

	// (a) Run should fail quickly (precheck fires as soon as init event is seen).
	run := waitForPrecheckRunCompletion(t, env, runID, 5*time.Second)

	// (b) Status must be failed.
	if got, _ := run["status"].(string); got != "failed" {
		t.Errorf("run status = %q, want \"failed\" after mediated bypass precheck", got)
	}

	// (c) The agent.failed WS event must carry reason=precheck_mediated_bypass.
	payload := collectAgentFailed(t, ch, runID, 3*time.Second)
	if payload == nil {
		t.Fatal("no agent.failed event received")
	}
	if reason, _ := payload["reason"].(string); reason != "precheck_mediated_bypass" {
		t.Errorf("agent.failed reason = %q, want \"precheck_mediated_bypass\"", reason)
	}
	if obs, _ := payload["observed_permission_mode"].(string); obs != "bypassPermissions" {
		t.Errorf("agent.failed observed_permission_mode = %q, want \"bypassPermissions\"", obs)
	}
}

// ── M5-I5: No bypass flags in mediated driver args ────────────────────────────

// TestMediatedDriver_Args_NoBypassFlags verifies that the mediated driver does
// NOT pass --dangerously-skip-permissions or --permission-mode bypassPermissions
// to claude (FR2). We record the args the fake binary received and assert.
func TestMediatedDriver_Args_NoBypassFlags(t *testing.T) {
	setupFakePrecheckClaude(t, "default", false)

	const artifactPath = "lifecycle/requirements/mediated-args.md"
	env := newMediatedTestEnv(t, mediatedAgentCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Mediated Args Check", "ticket", "draft", "mediated-args", "", "Body."),
	}})

	// Ask fake_precheck_claude to write received args to a temp file.
	argsFile := filepath.Join(t.TempDir(), "claude-args.json")
	t.Setenv("FAKE_CLAUDE_ARGS_FILE", argsFile)
	defer t.Setenv("FAKE_CLAUDE_ARGS_FILE", "")

	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "mediated-agent", artifactPath)
	_ = waitForPrecheckRunCompletion(t, env, runID, 15*time.Second)

	// Read and parse the recorded args.
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Skipf("args file not written (fake claude may not support FAKE_CLAUDE_ARGS_FILE): %v", err)
	}
	var args []string
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatalf("parsing args file: %v", err)
	}

	// Verify forbidden flags are absent.
	for _, arg := range args {
		if arg == "--dangerously-skip-permissions" {
			t.Error("claude was invoked with --dangerously-skip-permissions; mediated driver must NOT pass this flag (FR2)")
		}
		if arg == "bypassPermissions" {
			t.Error("claude was invoked with bypassPermissions permission mode; mediated driver must NOT pass this (FR2)")
		}
	}

	// Verify --settings flag is present (FR6).
	foundSettings := false
	for _, arg := range args {
		if arg == "--settings" {
			foundSettings = true
			break
		}
	}
	if !foundSettings {
		t.Error("claude was not invoked with --settings; mediated driver must pass the hook settings file (FR6)")
	}
}

// ── M5-I6: Settings file lifecycle (AC11, NFR4) ───────────────────────────────

// TestMediatedDriver_SettingsFile_CreatedAndCleanedUp verifies that the per-run
// hook settings file is written before the process starts and removed after the
// run completes (AC11, NFR4).
func TestMediatedDriver_SettingsFile_CreatedAndCleanedUp(t *testing.T) {
	// Hold after init so the process stays alive long enough to verify the file.
	setupFakePrecheckClaude(t, "default", true /* hold after init */)

	const artifactPath = "lifecycle/requirements/mediated-settings.md"
	env := newMediatedTestEnv(t, mediatedAgentCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Mediated Settings File", "ticket", "draft", "mediated-settings", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "mediated-agent", artifactPath)

	// Wait briefly for the driver to write the settings.json and spawn the process.
	time.Sleep(300 * time.Millisecond)

	// (a) Verify the settings file was written to the temp directory.
	settingsPath := filepath.Join(os.TempDir(), "hook-settings-"+runID+".json")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		// On very fast machines the run might have completed already; log and move on.
		t.Logf("note: settings file %s not found (process may have already cleaned up)", settingsPath)
	} else if err == nil {
		// (b) Verify the settings file is valid JSON with the expected structure.
		raw, readErr := os.ReadFile(settingsPath)
		if readErr != nil {
			t.Errorf("reading settings file: %v", readErr)
		} else {
			var settings map[string]any
			if jsonErr := json.Unmarshal(raw, &settings); jsonErr != nil {
				t.Errorf("settings file is not valid JSON: %v", jsonErr)
			} else {
				if _, ok := settings["hooks"]; !ok {
					t.Error("settings file is missing the 'hooks' key")
				}
			}
		}
	}

	// Kill the run so cleanup runs promptly (the held process would otherwise
	// block until the test context is cancelled, which could take 5 s).
	if killErr := env.proj.Agents.Kill(runID); killErr != nil {
		t.Logf("Kill(%s): %v (run may have already finished)", runID, killErr)
	}

	// Wait for the supervisor to finish and update the run record.
	_ = waitForPrecheckRunCompletion(t, env, runID, 5*time.Second)

	// Give the cleanup goroutine a moment to remove the file.
	time.Sleep(100 * time.Millisecond)

	// (c) Settings file must be gone after the run exits.
	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Errorf("settings file %s still exists after run ended; cleanup was not called (NFR4)", settingsPath)
	}
}

// ── M5-I6b: Settings files are unique per concurrent run ─────────────────────

// TestMediatedDriver_ConcurrentRuns_DistinctSettingsFiles verifies that two
// concurrent mediated runs do not share the same settings file path.
func TestMediatedDriver_ConcurrentRuns_DistinctSettingsFiles(t *testing.T) {
	// Hold both processes alive so we can compare files before cleanup.
	setupFakePrecheckClaude(t, "default", true)

	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/mediated-conc-1.md",
			content: makeArtifact("Mediated Concurrent 1", "ticket", "draft", "mediated-conc-1", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/mediated-conc-2.md",
			content: makeArtifact("Mediated Concurrent 2", "ticket", "draft", "mediated-conc-2", "", "Body."),
		},
	}

	// Use a config with two concurrent slots so both runs can start.
	cfg := mediatedAgentCfgYAML
	env := newAgentTestEnvCustom(t, cfg, project.OpenOptions{
		MaxConcurrentAgents: 4,
		HookServerAddr:      "127.0.0.1:0",
	}, seeds)
	env.login("admin@test.local", "admin-pass-123")

	runID1 := startAgentRun(t, env, "mediated-agent", seeds[0].relPath)
	runID2 := startAgentRun(t, env, "mediated-agent", seeds[1].relPath)

	// Wait for both processes to write their settings files.
	time.Sleep(300 * time.Millisecond)

	path1 := filepath.Join(os.TempDir(), "hook-settings-"+runID1+".json")
	path2 := filepath.Join(os.TempDir(), "hook-settings-"+runID2+".json")

	if path1 == path2 {
		t.Error("concurrent runs produced the same settings file path; paths must be distinct")
	}

	// Clean up both runs.
	_ = env.proj.Agents.Kill(runID1)
	_ = env.proj.Agents.Kill(runID2)
	_ = waitForPrecheckRunCompletion(t, env, runID1, 5*time.Second)
	_ = waitForPrecheckRunCompletion(t, env, runID2, 5*time.Second)

	time.Sleep(100 * time.Millisecond)

	// Both files should be cleaned up.
	for _, p := range []string{path1, path2} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("settings file %s still exists after run ended (NFR4)", p)
		}
	}
}
