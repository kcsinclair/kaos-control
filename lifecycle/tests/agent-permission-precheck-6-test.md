---
title: "Tests — Claude Code Permission-Mode Precheck"
type: test
status: draft
lineage: agent-permission-precheck
parent: lifecycle/test-plans/agent-permission-precheck-5-test.md
created: "2026-05-12T16:30:00+10:00"
labels:
    - agent
    - reliability
    - test
---

# Tests — Claude Code Permission-Mode Precheck

Integration and unit tests for the permission-mode precheck feature described
in `agent-permission-precheck-2` (requirements) and tested against the plan in
`agent-permission-precheck-5-test` (test plan).

---

## Unit tests

File: `internal/agent/precheck_test.go` (package `agent`)

These tests drive `runPrecheck`, `defaultProcessKiller`, and the `Manager`
directly without a real `claude` binary.

| ID  | Function | What it verifies |
|-----|----------|-----------------|
| U1  | `TestPrecheck_BypassPasses` | `bypassPermissions` init event → state `precheckPassed`, kill not called |
| U2  | `TestPrecheck_DefaultFails` | `default` init event → state `precheckFailedMode`, kill called |
| U3  | `TestPrecheck_AcceptEditsFails` | `acceptEdits` → `precheckFailedMode`, kill called |
| U4  | `TestPrecheck_PlanModeFails` | `plan` → `precheckFailedMode`, kill called |
| U5  | `TestPrecheck_MissingFieldWarnsAndPasses` | init event without `permissionMode` → `precheckPassed`, kill not called |
| U6  | `TestPrecheck_NonInitEventBeforeInit` | non-init event before init → precheck stays pending; bypass init → passes |
| U7  | `TestPrecheck_Timeout` | no events for 50 ms → `precheckFailedTimeout`, kill called |
| U8  | `TestPrecheck_SIGKILLEscalation` | `defaultProcessKiller` sends SIGTERM then SIGKILL after ~2 s (python3 process that ignores SIGTERM; skipped if python3 unavailable or -short) |
| U9  | `TestPrecheck_EscapeHatch` | `requireBypass=false` + `default` mode → `precheckPassed`, kill not called |
| U10 | `TestPrecheck_ConfigDefaultTrue` | empty `AppAgentConfig` → Manager defaults: timeout=10 s, requireBypass=true |
| U11 | `TestPrecheck_LogLineAppended` | `killAndFail` appends a `precheck_failure` JSON line with correct fields to the on-disk run log |
| —   | `TestPrecheck_ConfigRoundTrip` | `init_event_timeout_seconds: 2` in `AppAgentConfig` → timer fires at ~2 s, not 10 s |

---

## Integration tests

File: `tests/integration/agent_precheck_test.go` (build tag `integration`)

### Fake claude binary

`tests/integration/testutil/fake_precheck_claude/main.go` — compiled by
`TestMain` at suite start from the module path
`github.com/kaos-control/kaos-control/tests/integration/testutil/fake_precheck_claude`.
Reads env vars to control behaviour:

| Env var | Effect |
|---------|--------|
| `FAKE_CLAUDE_MODE` | Value for `permissionMode` field. Empty → field omitted. `omit-init` → no init event at all. |
| `FAKE_CLAUDE_DELAY_MS` | Milliseconds to sleep before emitting. |
| `FAKE_CLAUDE_HOLD_AFTER_INIT` | `"true"` → block for 24 h (lets the supervisor kill the process). |
| `FAKE_CLAUDE_ARGS_FILE` | If set, writes `os.Args` as JSON array to this path. |

### Tests

| ID | Function | What it verifies |
|----|----------|-----------------|
| I1 | `TestAgentPrecheck_BypassMode_Succeeds` | Fake claude emits `bypassPermissions` and exits 0. Agent run reaches `done`. |
| I2 | `TestAgentPrecheck_DefaultMode_FailsFast` | Fake claude emits `default` and holds. Run terminates within 3 s with `status=failed`, `reason=permission_mode_default` in `agent.failed` WS event, and `precheck_failure` line in on-disk log. |
| I3 | `TestAgentPrecheck_NoInitEvent_TimesOut` | Fake claude emits no init event and holds; `init_event_timeout_seconds=2`. Run fails with `reason=precheck_timeout` within ~2 s and log contains `precheck_failure` line. |
| I4 | `TestAgentPrecheck_DualFlagPassed` | Fake claude records `os.Args` to a temp file. Both `--permission-mode bypassPermissions` and `--dangerously-skip-permissions` appear in the correct order. |
| I5 | `TestAgentPrecheck_ConfigRoundTrip` | `init_event_timeout_seconds=5`; timeout fires at ~5 s confirming the config knob is wired end-to-end. |
| I6 | `TestAgentPrecheck_EscapeHatch_AllowsDefault` | `require_bypass_permissions=false`; fake claude emits `default` and exits. Run completes as `done`; no `precheck_failure` log line written. |

### Helpers defined in this file

- `setupFakePrecheckClaude(t, mode, holdAfterInit)` — installs the fake binary as `claude` on PATH, sets `FAKE_CLAUDE_MODE` and `FAKE_CLAUDE_HOLD_AFTER_INIT` via `t.Setenv`.
- `newAgentTestEnvCustom(t, cfgYAML, opts, seeds)` — creates a full test environment accepting `project.OpenOptions` (including `AgentCfg`).
- `precheckRunLogPath(env, runID)` — returns the expected on-disk log path.
- `waitForPrecheckRunCompletion(t, env, runID, timeout)` — polls run API until status leaves `running`.
- `collectAgentFailed(t, ch, runID, timeout)` — drains hub events until `agent.failed` arrives for the given run.
- `readPrecheckLogLine(t, logPath)` — scans the log file for a `precheck_failure` JSON object.
