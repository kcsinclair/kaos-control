---
title: Claude Code Permission-Mode Precheck and Fail-Fast
type: requirement
status: done
lineage: agent-permission-precheck
parent: lifecycle/ideas/agent-permission-precheck.md
created: "2026-05-12T15:35:00+10:00"
priority: high
labels:
    - agent
    - reliability
    - backend
    - operability
release: KC-Release1
---

# Claude Code Permission-Mode Precheck and Fail-Fast

Parent: [[agent-permission-precheck]].

## Goal

Agent runs that would silently fail because Claude Code is operating in
default (interactive-approval) permission mode must instead abort at the
init event with a structured, actionable error. kaos-control passes
both the legacy and the canonical bypass-permissions flags so that the
widest range of `claude` binaries are supported without configuration.

## Functional requirements

### Invocation

- **FR1 — Dual-flag invocation.** The `ClaudeCodeDriver` invocation
  passes both flags, in this order:

  ```
  --permission-mode bypassPermissions
  --dangerously-skip-permissions
  ```

  Newer Claude Code binaries honour the first form; older binaries
  ignore it (unknown flags are dropped without error per current
  Claude Code behaviour) and fall back to the legacy flag. The legacy
  flag stays in place to support older installs until it is removed
  by upstream.

### Precheck

- **FR2 — Inspect init event.** The agent supervisor's stream-json
  reader inspects every event for `type == "system"` AND
  `subtype == "init"`. On match, it reads the `permissionMode` field.

- **FR3 — Fail-fast on default mode.** If the init event's
  `permissionMode` is anything other than `"bypassPermissions"` (e.g.
  `"default"`, `"plan"`, `"acceptEdits"`), the supervisor:

  1. Sends `SIGTERM` to the `claude` subprocess immediately. If the
     process has not exited within 2 seconds, send `SIGKILL`.
  2. Marks the agent run terminal with state `failed` and reason
     `permission_mode_default`.
  3. Records the **observed** mode (`permissionMode` value from the
     init event) and the **expected** mode (`bypassPermissions`) on
     the run row for the UI.
  4. Broadcasts the existing `agent.failed` WebSocket event with the
     additional structured payload fields defined in FR5.

- **FR4 — Timeout on missing init.** If no init event is received
  within 10 seconds of subprocess start, the supervisor treats this
  as a precheck failure: same termination + same `failed` state with
  reason `precheck_timeout`. The 10-second value lives in app config
  as `agent.init_event_timeout_seconds` (default `10`).

### Error contract

- **FR5 — Structured failure payload.** The `agent.failed` WebSocket
  event for this failure carries (in addition to existing fields):

  ```json
  {
    "reason": "permission_mode_default",
    "observed_permission_mode": "default",
    "remediation": [
      "Run `claude` interactively once and accept the bypass-permissions warning.",
      "Upgrade Claude Code: `npm install -g @anthropic-ai/claude-code`.",
      "Remove or set `permissions.defaultMode: bypassPermissions` in ~/.claude/settings.json (and any project-local .claude/settings.json)."
    ]
  }
  ```

  For `precheck_timeout` the `observed_permission_mode` is `null` and
  the remediation list reads:

  ```json
  [
    "Confirm Claude Code is installed: `claude --version`.",
    "If freshly installed, run `claude` interactively once to complete first-run setup.",
    "Check kaos-control server logs for the raw stderr from the subprocess."
  ]
  ```

- **FR6 — Run-log persistence.** The structured failure event is also
  appended to the run's on-disk log
  (`~/.kaos-control/data/<project>/runs/<run_id>.log`) as a JSON line
  with `"type":"precheck_failure"` so post-mortem inspection is
  possible from the same place users already look.

### Non-regression

- **FR7 — No effect on successful runs.** A run whose init event
  reports `permissionMode == "bypassPermissions"` proceeds exactly
  as today. No new latency on the happy path beyond a single
  string comparison on the init event.

- **FR8 — No new permission requests.** This requirement does not
  alter what the agent is allowed to do at runtime; it only fails
  faster when Claude Code refuses to accept the bypass flag.

## Non-functional requirements

- **NFR1 — Backwards compatibility.** Older Claude Code releases that
  do not emit an init event with a `permissionMode` field MUST NOT
  fail-fast on that basis alone. If the init event lacks the field,
  treat it as `"bypassPermissions"` (best-effort) and log a single
  WARN line so the inconsistency can be tracked.

- **NFR2 — Logging.** Every precheck outcome (pass, default-mode
  fail, timeout fail) writes one INFO line with the run ID, the
  observed mode, and the precheck duration in milliseconds.

- **NFR3 — Config.** New app-config keys (all optional):

  ```yaml
  agent:
    init_event_timeout_seconds: 10
    require_bypass_permissions: true   # set false to disable the precheck entirely
  ```

  `require_bypass_permissions: false` is an escape hatch for users
  who deliberately want to run agents under default permission mode
  (e.g. development with a tty-attached approval loop). Default true.

## Acceptance criteria

- **AC1 — Dual flag passed.** The `claude` invocation includes both
  `--permission-mode bypassPermissions` and
  `--dangerously-skip-permissions` in that order. Verified by
  inspecting `cmd.Args` from a unit test.

- **AC2 — Bypass mode → run proceeds.** With a fake claude binary
  that emits `init` with `permissionMode == "bypassPermissions"`,
  the run reaches the first tool call without modification.

- **AC3 — Default mode → fail-fast.** With a fake claude binary
  that emits `init` with `permissionMode == "default"`:

  - The subprocess is terminated within 2 s of the init event.
  - The agent run row is `failed` with reason
    `permission_mode_default`.
  - The on-disk run log contains a `precheck_failure` JSON line.
  - The `agent.failed` WS event carries the structured remediation
    payload defined in FR5.
  - **No** `Bash`, `Write`, or other tool-call events were emitted.

- **AC4 — Timeout → fail-fast.** With a fake claude binary that
  never emits any output, the supervisor terminates the subprocess
  after 10 s (or whatever `init_event_timeout_seconds` is set to)
  and produces the timeout-shaped remediation payload.

- **AC5 — Missing field tolerated.** With a fake claude binary that
  emits an `init` event lacking `permissionMode` entirely, the run
  proceeds (treated as bypass) and a single WARN log line is
  emitted.

- **AC6 — Escape hatch.** With `require_bypass_permissions: false`,
  AC3 instead permits the run to continue. (Used by tests as well
  as by users who want the old behaviour.)

- **AC7 — UI surfacing.** The agent-runs view renders the structured
  remediation list when a failed run carries
  `reason: permission_mode_default`. Each remediation step is
  visually distinct; the observed mode is shown alongside.

## Permission model

No changes. The precheck does not gate who can launch agents — it
only changes how runs fail when the underlying Claude Code is
mis-configured.

## Out of scope

- Automatic remediation (writing settings files, accepting prompts).
- Mid-run detection of permission revocation.
- Per-tool allow-listing (`--allowedTools`).
- Detecting non-permission init-event errors (e.g. malformed prompt).

## Open questions

None blocking. Two minor calls the developer can make:

1. **`acceptEdits` and `plan` modes.** Treat these as "not
   bypassPermissions" → fail-fast. They look superficially less
   restrictive than `default` but still require user interaction.
   The requirement encodes this; the developer should confirm by
   testing with a `--permission-mode acceptEdits` invocation if
   convenient.

2. **WARN-log frequency for missing field.** If a single user's
   binary doesn't emit `permissionMode`, every run will produce a
   WARN line. Acceptable for now; revisit if it becomes noisy.
