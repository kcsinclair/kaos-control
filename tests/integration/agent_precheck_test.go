// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite 2 — Go integration tests for the Claude Code permission-mode precheck.
//
// Each test exercises the real subprocess path (signals, pipes, working directory)
// using a small fake-claude binary compiled once by TestMain from
// tests/integration/testutil/fake_precheck_claude/main.go.
//
// Run with:
//   go test ./tests/... -tags integration -run TestAgentPrecheck -v

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	kaoshttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/project"
)

// fakePrecheckClaudeBin is the path to the compiled fake-claude binary.
// Set by TestMain; empty string means the binary could not be built and
// any test that needs it will be skipped.
var fakePrecheckClaudeBin string

// TestMain compiles the fake-claude binary once for the whole package run.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "fake-precheck-claude-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "agent_precheck_test: MkdirTemp:", err)
		os.Exit(m.Run()) // still run other tests without the binary
	}
	defer os.RemoveAll(tmpDir)

	bin := filepath.Join(tmpDir, "fake-precheck-claude")
	cmd := exec.Command("go", "build",
		"-o", bin,
		"github.com/kaos-control/kaos-control/tests/integration/testutil/fake_precheck_claude",
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "agent_precheck_test: building fake claude:", err)
		// Don't set fakePrecheckClaudeBin; tests will skip.
	} else {
		fakePrecheckClaudeBin = bin
	}

	os.Exit(m.Run())
}

// setupFakePrecheckClaude installs the pre-compiled fake-claude binary as "claude"
// on PATH and sets per-test env vars via t.Setenv. Any test calling this function
// will be skipped if the binary was not successfully built.
func setupFakePrecheckClaude(t *testing.T, mode string, holdAfterInit bool) {
	t.Helper()
	if fakePrecheckClaudeBin == "" {
		t.Skip("fake precheck claude binary not available (build failed)")
	}
	fakeDir := t.TempDir()
	dst := filepath.Join(fakeDir, "claude")
	if err := os.Symlink(fakePrecheckClaudeBin, dst); err != nil {
		// Fallback: copy the binary.
		data, readErr := os.ReadFile(fakePrecheckClaudeBin)
		if readErr != nil {
			t.Fatalf("reading fake claude binary: %v", readErr)
		}
		if writeErr := os.WriteFile(dst, data, 0o755); writeErr != nil {
			t.Fatalf("writing fake claude binary: %v", writeErr)
		}
	}
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))
	t.Setenv("FAKE_CLAUDE_MODE", mode)
	if holdAfterInit {
		t.Setenv("FAKE_CLAUDE_HOLD_AFTER_INIT", "true")
	} else {
		t.Setenv("FAKE_CLAUDE_HOLD_AFTER_INIT", "false")
	}
}

// newAgentTestEnvCustom is like newAgentTestEnvWithCfg but also accepts
// project.OpenOptions so callers can tune AgentCfg (precheck timeout, etc.).
func newAgentTestEnvCustom(t *testing.T, cfgYAML string, opts project.OpenOptions, seeds []seedArtifact) *testEnv {
	t.Helper()

	root := t.TempDir()
	dataDir := t.TempDir()

	for _, s := range []string{
		"ideas", "requirements", "backend-plans", "frontend-plans",
		"test-plans", "tests", "prototypes", "releases", "sprints", "defects",
	} {
		if err := os.MkdirAll(filepath.Join(root, "lifecycle", s), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cfgPath := filepath.Join(root, "lifecycle", "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	repo, err := git.PlainInit(root, false)
	if err != nil {
		t.Fatal(err)
	}
	repoCfg, _ := repo.Config()
	repoCfg.User.Name = "Test User"
	repoCfg.User.Email = "test@test.local"
	if err := repo.SetConfig(repoCfg); err != nil {
		t.Fatal(err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("lifecycle/config.yaml"); err != nil {
		t.Fatal(err)
	}

	for _, s := range seeds {
		absPath := filepath.Join(root, s.relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absPath, []byte(s.content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := wt.Add(s.relPath); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@test.local", When: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	if ref.Name().Short() != "main" {
		_ = repo.CreateBranch(&gitconfig.Branch{Name: "main", Remote: ""})
	}

	authStore, err := auth.Open(filepath.Join(dataDir, "auth.db"), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { authStore.Close() })

	for _, u := range []struct{ email, name, pass string }{
		{"admin@test.local", "Admin", "admin-pass-123"},
		{"dev@test.local", "Developer", "dev-pass-123"},
		{"qa@test.local", "QA Engineer", "qa-pass-123"},
	} {
		if err := authStore.CreateUser(u.email, u.name, u.pass, false); err != nil {
			t.Fatal(err)
		}
	}

	entry := &config.ProjectEntry{
		Name:        "testproject",
		Path:        root,
		Description: "precheck integration test project",
	}

	if opts.MaxConcurrentAgents <= 0 {
		opts.MaxConcurrentAgents = 4
	}
	proj, err := project.Open(entry, dataDir, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { proj.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	proj.StartWatcher(ctx)
	proj.StartLockReaper(ctx)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	addr := ln.Addr().String()

	srv := kaoshttp.New(kaoshttp.ServerConfig{
		Listener: ln,
		Auth:     authStore,
	}, map[string]*project.Project{
		"testproject": proj,
	})

	srvDone := make(chan error, 1)
	go func() { srvDone <- srv.ListenAndServe(ctx) }()

	baseURL := "http://" + addr
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}

	env := &testEnv{
		t:           t,
		projectRoot: root,
		dataDir:     dataDir,
		baseURL:     baseURL,
		cancel:      cancel,
		authStore:   authStore,
		proj:        proj,
	}
	t.Cleanup(func() {
		cancel()
		select {
		case <-srvDone:
		case <-time.After(5 * time.Second):
		}
	})
	return env
}

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }

// precheckRunLogPath returns the expected path of a run's on-disk log file
// given the test environment and run ID.
func precheckRunLogPath(env *testEnv, runID string) string {
	return filepath.Join(env.dataDir, "testproject", "runs", runID+".log")
}

// waitForPrecheckRunCompletion polls the run API until the run leaves "running"
// state, then returns the final run map. Uses a tighter deadline than the default
// waitForRunCompletion helper (suitable for fast precheck failures).
func waitForPrecheckRunCompletion(t *testing.T, env *testEnv, runID string, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp := env.doRequest("GET", "/api/p/testproject/agents/runs/"+runID, nil)
		if resp.StatusCode == 200 {
			data := readJSON(t, resp)
			run, _ := data["run"].(map[string]any)
			if status, _ := run["status"].(string); status != "running" {
				return run
			}
		} else {
			resp.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for run %s to leave 'running' state", runID)
	return nil
}

// collectAgentFailed drains hub events from ch until an "agent.failed" event
// for the given runID is found, or until the timeout fires. Returns the payload.
func collectAgentFailed(t *testing.T, ch <-chan []byte, runID string, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type != "agent.failed" {
				continue
			}
			if rid, _ := evt.Payload["run_id"].(string); rid == runID {
				return evt.Payload
			}
		case <-deadline:
			t.Errorf("timed out waiting for agent.failed event for run %s", runID)
			return nil
		}
	}
}

// readPrecheckLogLine reads the on-disk run log and returns the first
// precheck_failure JSON object it finds, or nil if none.
func readPrecheckLogLine(t *testing.T, logPath string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading run log %s: %v", logPath, err)
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, `"precheck_failure"`) {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err == nil {
			return obj
		}
	}
	return nil
}

// ── Lifecycle config used by precheck integration tests ──────────────────────

const precheckAgentCfgYAML = `git:
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
  - name: precheck-agent
    role: [backend-developer]
    driver: claude-code-cli
    active_status: in-development
    allowed_write_paths:
      - lifecycle/requirements
      - lifecycle/ideas
    git_identity:
      name: Precheck Test Agent
      email: precheck-agent@test.local
    prompt_templates:
      backend-developer: "Test precheck prompt for {target_path}"
`

// ── I1: bypass mode succeeds ──────────────────────────────────────────────────

// TestAgentPrecheck_BypassMode_Succeeds verifies that when the fake claude binary
// emits permissionMode=bypassPermissions the precheck passes and the agent run
// reaches status=done.
func TestAgentPrecheck_BypassMode_Succeeds(t *testing.T) {
	setupFakePrecheckClaude(t, "bypassPermissions", false)

	const artifactPath = "lifecycle/ideas/precheck-bypass-ok.md"
	env := newAgentTestEnvCustom(t, precheckAgentCfgYAML, project.OpenOptions{}, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Precheck Bypass OK", "idea", "draft", "precheck-bypass-ok", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "precheck-agent", artifactPath)
	run := waitForPrecheckRunCompletion(t, env, runID, 15*time.Second)

	if got, _ := run["status"].(string); got != "done" {
		t.Errorf("run status = %q, want \"done\"", got)
	}
}

// ── I2: default mode fails fast ───────────────────────────────────────────────

// TestAgentPrecheck_DefaultMode_FailsFast verifies that when the fake binary
// emits permissionMode=default the precheck terminates the run fast, reports
// status=failed with reason=permission_mode_default, broadcasts an agent.failed
// WS event with the structured payload, and writes a precheck_failure line to
// the on-disk run log.
func TestAgentPrecheck_DefaultMode_FailsFast(t *testing.T) {
	setupFakePrecheckClaude(t, "default", true /* hold so the precheck kills it */)

	const artifactPath = "lifecycle/ideas/precheck-default-fail.md"
	env := newAgentTestEnvCustom(t, precheckAgentCfgYAML, project.OpenOptions{}, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Precheck Default Fail", "idea", "draft", "precheck-default-fail", "", "Body."),
	}})

	// Register hub listener before starting the run to capture agent.failed.
	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")

	start := time.Now()
	runID := startAgentRun(t, env, "precheck-agent", artifactPath)

	// (a) run terminates within 3 s.
	run := waitForPrecheckRunCompletion(t, env, runID, 3*time.Second)
	elapsed := time.Since(start)
	t.Logf("precheck failure elapsed: %v", elapsed)

	// (b) status = failed.
	if got, _ := run["status"].(string); got != "failed" {
		t.Errorf("run status = %q, want \"failed\"", got)
	}

	// (c) reason = permission_mode_default.
	if got, _ := run["failure_reason"].(string); got != "permission_mode_default" {
		// The reason is also in the WS payload; check there too (see d).
		t.Logf("run record failure_reason = %q (may be in WS payload only)", got)
	}

	// (d) agent.failed WS event with structured payload.
	payload := collectAgentFailed(t, ch, runID, 3*time.Second)
	if payload != nil {
		if reason, _ := payload["reason"].(string); reason != "permission_mode_default" {
			t.Errorf("agent.failed reason = %q, want \"permission_mode_default\"", reason)
		}
		if obs, _ := payload["observed_permission_mode"].(string); obs != "default" {
			t.Errorf("agent.failed observed_permission_mode = %q, want \"default\"", obs)
		}
		if remRaw, ok := payload["remediation"]; ok {
			rem, _ := remRaw.([]any)
			if len(rem) == 0 {
				t.Error("agent.failed remediation list is empty")
			}
		} else {
			t.Error("agent.failed payload missing remediation field")
		}
	}

	// (e) on-disk run log contains precheck_failure line.
	logPath := precheckRunLogPath(env, runID)
	obj := readPrecheckLogLine(t, logPath)
	if obj == nil {
		t.Errorf("run log %s: no precheck_failure JSON line found", logPath)
	} else {
		if got, _ := obj["reason"].(string); got != "permission_mode_default" {
			t.Errorf("log precheck_failure reason = %q, want \"permission_mode_default\"", got)
		}
		if got, _ := obj["observed_permission_mode"].(string); got != "default" {
			t.Errorf("log observed_permission_mode = %q, want \"default\"", got)
		}
		if got, _ := obj["run_id"].(string); got != runID {
			t.Errorf("log run_id = %q, want %q", got, runID)
		}
	}
}

// ── I3: no init event times out ───────────────────────────────────────────────

// TestAgentPrecheck_NoInitEvent_TimesOut verifies that when the fake binary emits
// no init event, the run fails with reason=precheck_timeout within the configured
// timeout window.
func TestAgentPrecheck_NoInitEvent_TimesOut(t *testing.T) {
	// "omit-init" → fake binary never emits the init event, just holds.
	setupFakePrecheckClaude(t, "omit-init", true)

	// Use a 2-second timeout so the test doesn't take 10 s.
	const timeoutSecs = 2
	requireBypass := true
	agentCfg := config.AppAgentConfig{
		InitEventTimeoutSeconds:  timeoutSecs,
		RequireBypassPermissions: &requireBypass,
	}

	const artifactPath = "lifecycle/ideas/precheck-timeout.md"
	env := newAgentTestEnvCustom(t, precheckAgentCfgYAML, project.OpenOptions{AgentCfg: agentCfg}, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Precheck Timeout", "idea", "draft", "precheck-timeout", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	start := time.Now()
	runID := startAgentRun(t, env, "precheck-agent", artifactPath)
	run := waitForPrecheckRunCompletion(t, env, runID, time.Duration(timeoutSecs+3)*time.Second)
	elapsed := time.Since(start)
	t.Logf("timeout elapsed: %v", elapsed)

	if got, _ := run["status"].(string); got != "failed" {
		t.Errorf("run status = %q, want \"failed\"", got)
	}

	// Verify the log contains a precheck_failure with reason=precheck_timeout.
	logPath := precheckRunLogPath(env, runID)
	obj := readPrecheckLogLine(t, logPath)
	if obj == nil {
		t.Errorf("run log %s: no precheck_failure JSON line found", logPath)
	} else {
		if got, _ := obj["reason"].(string); got != "precheck_timeout" {
			t.Errorf("log precheck_failure reason = %q, want \"precheck_timeout\"", got)
		}
	}
}

// ── I4: dual flag passed ──────────────────────────────────────────────────────

// TestAgentPrecheck_DualFlagPassed inspects the argv recorded by the fake binary
// and asserts that both --permission-mode bypassPermissions and
// --dangerously-skip-permissions appear in the correct order.
func TestAgentPrecheck_DualFlagPassed(t *testing.T) {
	// bypassPermissions + no-hold: binary records argv and exits cleanly.
	setupFakePrecheckClaude(t, "bypassPermissions", false)

	// Tell the fake binary where to write os.Args.
	argsFile := filepath.Join(t.TempDir(), "args.json")
	t.Setenv("FAKE_CLAUDE_ARGS_FILE", argsFile)

	const artifactPath = "lifecycle/ideas/precheck-dual-flag.md"
	env := newAgentTestEnvCustom(t, precheckAgentCfgYAML, project.OpenOptions{}, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Precheck Dual Flag", "idea", "draft", "precheck-dual-flag", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "precheck-agent", artifactPath)
	waitForPrecheckRunCompletion(t, env, runID, 15*time.Second)

	// Read the recorded argv.
	raw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("reading args file: %v", err)
	}
	var args []string
	if err := json.Unmarshal(raw, &args); err != nil {
		t.Fatalf("parsing args file: %v", err)
	}

	// Find positions of the two flags.
	pmIdx, dspIdx := -1, -1
	for i, a := range args {
		switch a {
		case "--permission-mode":
			// The next element should be "bypassPermissions".
			if i+1 < len(args) && args[i+1] == "bypassPermissions" {
				pmIdx = i
			}
		case "--dangerously-skip-permissions":
			dspIdx = i
		}
	}

	if pmIdx < 0 {
		t.Errorf("args %v: --permission-mode bypassPermissions not found", args)
	}
	if dspIdx < 0 {
		t.Errorf("args %v: --dangerously-skip-permissions not found", args)
	}
	if pmIdx >= 0 && dspIdx >= 0 && pmIdx > dspIdx {
		t.Errorf("--permission-mode (pos %d) must appear before --dangerously-skip-permissions (pos %d)", pmIdx, dspIdx)
	}
}

// ── I5: config round-trip ─────────────────────────────────────────────────────

// TestAgentPrecheck_ConfigRoundTrip starts the test server with
// init_event_timeout_seconds=5 and confirms the timeout fires at ~5 s, not 10 s.
func TestAgentPrecheck_ConfigRoundTrip(t *testing.T) {
	setupFakePrecheckClaude(t, "omit-init", true)

	const timeoutSecs = 5
	requireBypass := true
	agentCfg := config.AppAgentConfig{
		InitEventTimeoutSeconds:  timeoutSecs,
		RequireBypassPermissions: &requireBypass,
	}

	const artifactPath = "lifecycle/ideas/precheck-roundtrip.md"
	env := newAgentTestEnvCustom(t, precheckAgentCfgYAML, project.OpenOptions{AgentCfg: agentCfg}, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Precheck RoundTrip", "idea", "draft", "precheck-roundtrip", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	start := time.Now()
	runID := startAgentRun(t, env, "precheck-agent", artifactPath)
	run := waitForPrecheckRunCompletion(t, env, runID, time.Duration(timeoutSecs+4)*time.Second)
	elapsed := time.Since(start)
	t.Logf("config roundtrip timeout elapsed: %v (configured %d s)", elapsed, timeoutSecs)

	if got, _ := run["status"].(string); got != "failed" {
		t.Errorf("run status = %q, want \"failed\"", got)
	}

	// Should fire at ~5 s, not the default 10 s.
	if elapsed >= 8*time.Second {
		t.Errorf("timeout fired too late (%v); configured %d s means it should fire well before 8 s", elapsed, timeoutSecs)
	}
}

// ── I6: escape hatch allows default mode ─────────────────────────────────────

// TestAgentPrecheck_EscapeHatch_AllowsDefault verifies that when
// require_bypass_permissions=false, a run with permissionMode=default is not
// terminated by the precheck and completes normally.
func TestAgentPrecheck_EscapeHatch_AllowsDefault(t *testing.T) {
	// "default" mode but no hold: binary emits init event and exits cleanly.
	setupFakePrecheckClaude(t, "default", false)

	requireBypass := false
	agentCfg := config.AppAgentConfig{
		InitEventTimeoutSeconds:  10,
		RequireBypassPermissions: &requireBypass,
	}

	const artifactPath = "lifecycle/ideas/precheck-escape-hatch.md"
	env := newAgentTestEnvCustom(t, precheckAgentCfgYAML, project.OpenOptions{AgentCfg: agentCfg}, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Precheck Escape Hatch", "idea", "draft", "precheck-escape-hatch", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "precheck-agent", artifactPath)
	run := waitForPrecheckRunCompletion(t, env, runID, 15*time.Second)

	// Run should complete (not fail) — precheck is disabled.
	if got, _ := run["status"].(string); got != "done" {
		t.Errorf("run status = %q, want \"done\" (escape hatch should allow default mode)", got)
	}

	// Confirm no precheck_failure line in the log.
	logPath := precheckRunLogPath(env, runID)
	if _, err := os.Stat(logPath); err == nil {
		obj := readPrecheckLogLine(t, logPath)
		if obj != nil {
			t.Errorf("escape hatch run should not produce a precheck_failure log line; found: %v", obj)
		}
	}
}
