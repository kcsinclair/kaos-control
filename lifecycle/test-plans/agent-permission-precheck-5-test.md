---
title: "Test Plan — Claude Code Permission-Mode Precheck"
type: plan-test
status: approved
lineage: agent-permission-precheck
parent: lifecycle/requirements/agent-permission-precheck-2.md
created: "2026-05-12T15:50:00+10:00"
priority: high
labels:
    - agent
    - reliability
    - test
release: KC-Release1
---

# Test Plan — Claude Code Permission-Mode Precheck

Covers the acceptance criteria in [[agent-permission-precheck-2]].
Two test suites (Go unit + Go integration) plus the frontend vitest
cases listed in [[agent-permission-precheck-4-fe]]. A tight manual
smoke script covers the bits that can't be cheaply mechanised.

The integration suite uses a **fake claude binary** built from a tiny
Go program that emits configurable stream-json events to stdout. The
binary is compiled into the test temp dir at suite start, and the
agent runner's `ClaudeCodeDriver` is pointed at it via the existing
`PATH` override mechanism used by `setupFakeClaude` in
`tests/integration/agent_helpers_test.go`.

---

## Suite 1 — Go unit tests (`internal/agent/precheck_test.go`)

Driven against the supervisor with a mock subprocess: an
`io.Reader` for stdout and an injectable process killer interface.
No real `claude` binary needed.

| # | Test | Description |
|---|---|---|
| U1 | `TestPrecheck_BypassPasses` | Feed `{"type":"system","subtype":"init","permissionMode":"bypassPermissions"}`. Assert state transitions to `precheckPassed`. No kill issued. Run continues. |
| U2 | `TestPrecheck_DefaultFails` | Feed `{"type":"system","subtype":"init","permissionMode":"default"}`. Assert: (a) subprocess gets SIGTERM within 100 ms of init event, (b) `UpdateAgentRun` called with `state=failed, reason=permission_mode_default`, (c) `Hub.Broadcast` called with `agent.failed` and the structured payload, (d) the run log has a `precheck_failure` JSON line. |
| U3 | `TestPrecheck_AcceptEditsFails` | Same as U2 but `permissionMode=acceptEdits`. Confirms the precheck is strict-equal to `bypassPermissions`, not "anything other than default". |
| U4 | `TestPrecheck_PlanModeFails` | Same as U3 but `permissionMode=plan`. |
| U5 | `TestPrecheck_MissingFieldWarnsAndPasses` | Init event without `permissionMode`. Assert: state `precheckPassed`, single WARN log line captured (`agent: init event missing permissionMode`), no kill. |
| U6 | `TestPrecheck_NonInitEventBeforeInit` | Feed an `{"type":"assistant",...}` event BEFORE the init event. Precheck timer still ticking; init arrives second; behaves like U1. |
| U7 | `TestPrecheck_Timeout` | Feed nothing on stdout. Mock clock advances past `init_event_timeout_seconds`. Assert: subprocess gets SIGTERM, agent run failed with `reason=precheck_timeout`, remediation payload matches FR5 timeout shape, no init event ever processed. |
| U8 | `TestPrecheck_SIGKILLEscalation` | Like U2, but the mock killer ignores SIGTERM. Assert: SIGKILL is sent 2 s after SIGTERM. |
| U9 | `TestPrecheck_EscapeHatch` | Config: `RequireBypassPermissions=false`. Feed `permissionMode=default`. Assert: state `precheckPassed`, one WARN line, no kill, run continues. |
| U10 | `TestPrecheck_ConfigDefaultTrue` | Construct AppConfig from empty YAML; assert `RequireBypassPermissions` is `true` and `InitEventTimeoutSeconds` is `10`. |
| U11 | `TestPrecheck_LogLineAppended` | Like U2; after the kill, read the captured run log file and verify the appended JSON line shape (fields: `type`, `run_id`, `reason`, `observed_permission_mode`, `remediation`, `timestamp`). |

### Unit-test helper sketch

```go
type mockSubprocess struct {
    stdoutR  *io.PipeReader
    stdoutW  *io.PipeWriter
    killed   chan os.Signal
}

func newMockSubprocess() *mockSubprocess { … }
func (m *mockSubprocess) Emit(event map[string]any)  // writes JSON+newline to stdoutW
func (m *mockSubprocess) Kill(sig os.Signal)         // records signal; closes pipe on SIGTERM/SIGKILL
```

---

## Suite 2 — Go integration tests (`tests/integration/agent_precheck_test.go`)

Use a small fake-claude Go binary built at suite start that reads
its config from environment variables and emits a single init event
then exits. This exercises the real subprocess path (signals,
pipes, working directory).

### Fake-claude flags

The binary takes these env vars:

- `FAKE_CLAUDE_MODE` — value to emit in `permissionMode`. If empty,
  omits the field entirely. If literally `omit-init`, emits no init
  event at all.
- `FAKE_CLAUDE_DELAY_MS` — milliseconds to wait before emitting.
- `FAKE_CLAUDE_HOLD_AFTER_INIT` — if "true", the binary blocks
  indefinitely after emitting init (lets the precheck terminate it).

### Tests

| # | Test | Description |
|---|---|---|
| I1 | `TestAgentPrecheck_BypassMode_Succeeds` | Fake claude emits `bypassPermissions` then exits 0. Assert agent run reaches `completed`. |
| I2 | `TestAgentPrecheck_DefaultMode_FailsFast` | Fake claude emits `default` then holds. Assert: (a) agent run terminates within 3 s, (b) state `failed`, (c) reason `permission_mode_default`, (d) `agent.failed` WS event received with the structured payload, (e) the run log on disk contains the `precheck_failure` JSON line. |
| I3 | `TestAgentPrecheck_NoInitEvent_TimesOut` | Fake claude emits nothing and holds. App config has `init_event_timeout_seconds: 2`. Assert run fails with `reason=precheck_timeout` within ~2 s. |
| I4 | `TestAgentPrecheck_DualFlagPassed` | Inspect the `cmd.Args` recorded by the fake binary (it writes its argv to a known file in the test temp dir). Assert both `--permission-mode bypassPermissions` and `--dangerously-skip-permissions` appear in order. |
| I5 | `TestAgentPrecheck_ConfigRoundTrip` | Start the test server with `init_event_timeout_seconds: 5` in YAML. Run an agent against a fake claude that holds. Assert the timeout fires at ~5 s, not 10 s, confirming the config knob is wired. |
| I6 | `TestAgentPrecheck_EscapeHatch_AllowsDefault` | App config has `require_bypass_permissions: false`. Fake claude emits `default` then a `text` content event and exits. Assert run completes (not fail-fast) and one WARN log line captured. |

---

## Suite 3 — Frontend (Vitest)

Defined in detail in [[agent-permission-precheck-4-fe]]; reproduced
here for completeness.

| # | Test | Description |
|---|---|---|
| F1 | `RunFailureBanner_PermissionModeDefault` | Mount with `failureReason='permission_mode_default'`, `observedMode='default'`, three remediation strings. Assert heading, body mentions `default`, three list items rendered. |
| F2 | `RunFailureBanner_PrecheckTimeout` | Mount with `failureReason='precheck_timeout'` and the timeout remediation list. Assert correct heading and list. |
| F3 | `RunFailureBanner_UnknownReason` | Mount with `failureReason='custom_unknown'`. Assert fallback heading. |
| F4 | `AgentsRunsView_RendersBannerOnFailedPrecheck` | Mount the view with a mocked store containing one failed run carrying the precheck payload. Assert `<RunFailureBanner>` is in the DOM. |
| F5 | `AgentsRunsView_NoBannerOnRegularFailure` | Same view but the failed run has no `failure_reason`. Assert banner absent; existing failure UI unaffected. |
| F6 | `AgentsStore_PreservesFailureFieldsFromWS` | Dispatch a synthetic `agent.failed` WS event carrying the precheck fields. Assert the matching run row in the store has `failure_reason`, `observed_permission_mode`, `remediation` populated. |

---

## Manual smoke (not automated)

Run after Suites 1, 2, and 3 pass.

1. `make all && make run` against the kaos-control project itself.
2. **Happy path** — assuming your `claude` binary is in bypass mode
   already (test by running `claude --dangerously-skip-permissions
   -p "echo hi"` and confirming it doesn't prompt):
   - Approve any idea.
   - Click Queue Work or Run Agent on it.
   - Confirm the run reaches `completed` and the artefact is written.
   - No banner is shown.

3. **Trigger the precheck failure** — force default mode by
   temporarily setting `~/.claude/settings.json`:
   ```json
   { "permissions": { "defaultMode": "default" } }
   ```
   (back up the existing file first). Then:
   - Approve another idea.
   - Launch an agent run.
   - Confirm the run shows up as `failed` within a few seconds.
   - Confirm the agent-runs detail panel shows the
     `RunFailureBanner` with:
     - Heading "Claude Code is in default permission mode".
     - Body mentions `default`.
     - Three numbered remediation steps.
   - Open `~/.kaos-control/data/kaos-control/runs/<run_id>.log` and
     confirm a `precheck_failure` JSON line is present.
   - Restore the original `~/.claude/settings.json`.

4. **Escape hatch** — set
   `agent.require_bypass_permissions: false` in
   `~/.kaos-control/config.yaml`, re-create the default-mode
   settings file, restart kaos-control. Launch an agent run. Confirm
   it proceeds (now subject to whatever default-mode requires; in
   most cases the run will fail later when a tool call hits the
   approval prompt, but the precheck no longer terminates it
   up-front). Confirm a WARN line in the server log.

### Optional adversarial smoke

- Run on a fresh machine that has never run `claude` interactively.
  Confirm the precheck failure points the user at the correct first
  step ("Run `claude` interactively once and accept the
  bypass-permissions warning").
- Run with an outdated Claude Code binary that does not emit the
  `permissionMode` field. Confirm the run proceeds and a single
  WARN line is logged (does NOT fail-fast on the missing field).
