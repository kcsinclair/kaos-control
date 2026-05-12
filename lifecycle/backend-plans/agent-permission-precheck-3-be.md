---
title: "Backend Plan — Claude Code Permission-Mode Precheck"
type: plan-backend
status: done
lineage: agent-permission-precheck
parent: lifecycle/requirements/agent-permission-precheck-2.md
created: "2026-05-12T15:40:00+10:00"
priority: high
labels:
    - agent
    - reliability
    - backend
    - release-blocker
release: KC-Release1
---

# Backend Plan — Claude Code Permission-Mode Precheck

Implements [[agent-permission-precheck-2]]. All changes live in
`internal/agent/` and `internal/config/`. No new packages.

## Milestone 1 — Dual-flag invocation

### Description

Add `--permission-mode bypassPermissions` to the existing argument
list, in front of the legacy `--dangerously-skip-permissions`.

### Files to change

- **Edit** `internal/agent/agent.go` (around line 134-140):

  ```go
  func (d *ClaudeCodeDriver) Start(ctx context.Context, run Run) (Process, error) {
      args := []string{
          "--permission-mode", "bypassPermissions",
          "--dangerously-skip-permissions",  // legacy alias; older binaries
          "-p", run.PromptText,
          "--output-format", "stream-json",
          "--verbose",
      }
      …
  }
  ```

  Also update the doc comment above the function from
  `// ClaudeCodeDriver spawns claude --dangerously-skip-permissions …`
  to reflect the dual-flag invocation.

### Acceptance criteria

- `go build ./...` clean; `go vet` clean.
- Unit test asserts `cmd.Args` contains the two flags in order.

---

## Milestone 2 — App-config additions

### Description

Two new keys under `agent:` in `~/.kaos-control/config.yaml`:

```yaml
agent:
  init_event_timeout_seconds: 10
  require_bypass_permissions: true
```

Both are optional; defaults match the values shown.

### Files to change

- **Edit** `internal/config/config.go`:
  - Locate the `Agent` (or equivalent) section of the app config
    struct. If no struct exists for agent-level config yet, add one:

    ```go
    type AppAgentConfig struct {
        InitEventTimeoutSeconds  int  `yaml:"init_event_timeout_seconds,omitempty"`
        RequireBypassPermissions *bool `yaml:"require_bypass_permissions,omitempty"`
    }
    ```

    Pointer-to-bool so the YAML loader can distinguish "unset"
    (default true) from "explicitly false".
  - Apply defaults in `defaultApp()` (or the equivalent
    `LoadApp` post-processing step):
    `InitEventTimeoutSeconds = 10`, `RequireBypassPermissions = &true`.

- **Edit** the `defaultApp()` template comment block (or wherever
  the persisted YAML template lives) to include both new keys with
  inline comments.

### Acceptance criteria

- `go test ./internal/config/...` passes including a new test
  `TestLoadApp_AgentPrecheckDefaults` that round-trips an app
  config with these keys.
- A second test `TestLoadApp_AgentPrecheckExplicitFalse` confirms
  the pointer-bool semantics: explicit `false` survives the load.

---

## Milestone 3 — Init-event reader and precheck state

### Description

The supervisor in `internal/agent/agent.go` already reads stream-json
events line-by-line. Add a small state machine in front of the main
read loop that watches for the init event and applies the precheck.

### Design sketch

```go
type precheckState int

const (
    precheckPending precheckState = iota
    precheckPassed
    precheckFailedMode
    precheckFailedTimeout
)

// In the supervisor goroutine:
precheckTimer := time.NewTimer(cfg.InitEventTimeout)
defer precheckTimer.Stop()
state := precheckPending

for {
    select {
    case <-precheckTimer.C:
        if state == precheckPending {
            state = precheckFailedTimeout
            killAndFail(run, "precheck_timeout", "", timeoutRemediation)
            return
        }
    case line, ok := <-stdoutLines:
        if !ok { /* normal completion */ return }
        var ev map[string]any
        if json.Unmarshal([]byte(line), &ev) != nil {
            // not JSON; forward to existing log handling
            forwardEvent(line); continue
        }
        if state == precheckPending && ev["type"] == "system" && ev["subtype"] == "init" {
            mode, _ := ev["permissionMode"].(string)
            if mode == "" {
                slog.Warn("agent: init event missing permissionMode",
                          "run_id", run.ID)
                state = precheckPassed
            } else if mode == "bypassPermissions" {
                state = precheckPassed
            } else if !cfg.RequireBypassPermissions {
                slog.Warn("agent: permissionMode is not bypass but precheck disabled",
                          "run_id", run.ID, "mode", mode)
                state = precheckPassed
            } else {
                state = precheckFailedMode
                killAndFail(run, "permission_mode_default", mode, modeRemediation)
                return
            }
            precheckTimer.Stop()
        }
        // forward to existing event handling regardless
        forwardEvent(line)
    }
}
```

### Files to change

- **Edit** `internal/agent/agent.go`:
  - Introduce the `precheckState` type and constants.
  - Wrap the existing read loop with the precheck timer / state
    handling above.
  - Add `killAndFail(run, reason, observedMode, remediation)` helper:
    - Sends `SIGTERM`. Waits up to 2 s. Sends `SIGKILL` if still alive.
    - Calls `UpdateAgentRun(state="failed", reason=…)`.
    - Builds the structured WS payload (FR5) and broadcasts via the
      existing `Hub.Broadcast("agent.failed", payload)` path.
    - Appends a `{"type":"precheck_failure", …}` line to the run log.

- **Edit** the `Run` / `RunHandle` struct if needed so the supervisor
  has access to the app-config `AgentConfig` (timeout + require flag).

### Acceptance criteria

- New unit test in `internal/agent/precheck_test.go` covers:
  - `TestPrecheck_BypassPasses` — init event with
    `permissionMode=bypassPermissions` → state transitions to
    `precheckPassed`; no kill.
  - `TestPrecheck_DefaultFails` — init event with `default` →
    `precheckFailedMode`; subprocess receives SIGTERM; agent run
    state is `failed`, reason `permission_mode_default`.
  - `TestPrecheck_AcceptEditsFails` — same as above but with
    `acceptEdits`.
  - `TestPrecheck_MissingFieldWarnsAndPasses` — init event without
    `permissionMode` → state `precheckPassed`; one WARN log line
    captured.
  - `TestPrecheck_Timeout` — fake binary that never emits init →
    after `init_event_timeout_seconds`, subprocess killed, agent
    run failed with reason `precheck_timeout`.
  - `TestPrecheck_EscapeHatch` — `require_bypass_permissions=false`
    plus `default` mode → run proceeds; one WARN line captured.

The tests use a mock subprocess (an `io.Reader` for stdout) rather
than a real `claude` binary. The kill path is tested via an
injectable `processKiller` interface so tests don't actually need
to fork.

---

## Milestone 4 — On-disk log append

### Description

Append a single JSON line to the run log when a precheck fails, in
addition to the WS broadcast.

```json
{"type":"precheck_failure","run_id":"…","reason":"permission_mode_default","observed_permission_mode":"default","remediation":[…],"timestamp":"2026-05-12T15:42:01Z"}
```

The existing run-log writer in the supervisor already takes
`map[string]any` payloads — just add a call before `killAndFail`
returns.

### Files to change

- **Edit** `internal/agent/agent.go` — `killAndFail` helper writes
  the line.

### Acceptance criteria

- Unit test `TestPrecheck_LogLineAppended` reads the captured run
  log file and verifies the JSON line shape.

---

## Milestone 5 — Wire app config through

### Description

The agent runner today reads its `MaxConcurrentAgents` and similar
limits from `AppConfig.Limits`. Thread the new `InitEventTimeout`
and `RequireBypassPermissions` through the same channels.

### Files to change

- **Edit** `cmd/kaos-control/main.go` (or wherever the agent manager
  is constructed) to pass the `AgentConfig` values into the
  `agent.Manager` constructor.

- **Edit** `internal/agent/agent.go`'s manager / driver structs to
  store these values and apply them when starting a run.

### Acceptance criteria

- An integration test in `tests/integration/agent_precheck_test.go`:
  - `TestAgentPrecheck_ConfigRoundTrip` — start a test server with
    `init_event_timeout_seconds: 2` in app config; launch a run
    against a fake claude that never emits init; assert the
    timeout fires at ~2 s, not the default 10 s.

---

## Verification (end-to-end)

1. `make lint` clean.
2. `make test-unit` clean (new tests pass).
3. `make test-integration` clean (new test file).
4. Manual smoke against a real `claude`:
   - On a machine that has accepted bypass mode: agent run completes
     as today.
   - On a fresh box (or with `~/.claude/settings.json` set to
     `permissions.defaultMode: default`): agent run fails fast with
     `permission_mode_default` and the WS event carries the
     three-item remediation list.

## Risk notes

- **`permissionMode` field rename.** Anthropic may rename or
  restructure the init event. The code reads the field defensively
  (typed map with `_` for the second return) and the WARN log on a
  missing field gives us a clear breadcrumb if the contract changes.

- **Timeout false positives.** A genuinely slow first event could
  hit the 10-second timeout. The config knob exists to extend it
  for users on slow networks or slow machines. The default value
  is deliberately generous — Claude Code emits init within ~100ms
  in normal conditions.

- **Race between init event and subprocess exit.** If the
  subprocess exits before emitting init (e.g. binary missing),
  the `stdoutLines` channel closes; the existing code path handles
  that and the precheck simply never fires (state remains
  `precheckPending` until the subprocess exit handler runs, which
  records the exit code as the failure reason). No special handling
  needed.
