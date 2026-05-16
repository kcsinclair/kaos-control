---
title: "Test Suite: Mediated Claude Driver with Permission Hooks"
type: test
status: done
lineage: claude-hooks-driver
parent: lifecycle/test-plans/claude-hooks-driver-5-test.md
created: "2026-05-15T00:00:00+10:00"
---

# Test Suite: Mediated Claude Driver with Permission Hooks

Parent: [[claude-hooks-driver]].

This artifact documents the integration tests written for the
`claude-mediated` driver and its permission-hook infrastructure. Tests cover
Milestones 3, 4, and 5 from the test plan. Unit tests for Milestones 1
(policy evaluation) and 2 (run-secret / settings-file unit tests) are tracked
separately under `internal/agent/` (outside this test-developer's write scope).
Milestones 7 (frontend) and 8 (manual) are also excluded from this suite.

---

## Scenarios Covered

### Milestone 3 — Permission Endpoint (`tests/integration/permission_endpoint_test.go`)

**Authentication (FR8, AC12)**
- Correct per-run secret → 200 with a policy decision.
- Wrong secret → 403.
- Missing Authorization header → 403.
- Unknown `run_id` (no policy stored) → 400.

**Request validation (FR8)**
- Empty request body → 400.
- Well-formed body with `tool_name` + `tool_input` → 200.

**Policy decisions (FR7, AC3)**
- `Write` to an allowed path → `{"decision":"allow"}`.
- `Write` outside allowed paths → `{"decision":"deny","reason":"..."}`.
- `Bash` command matching the default denylist → deny.
- Read-only tools (`Read`, `Glob`, `Grep`, `WebFetch`) → allow, regardless of
  path restrictions (FR13).

**Observe-only mode (FR17, AC7)**
- Agent with `observe_only: true`, write to a disallowed path → allow (policy
  decision is still `deny` internally, but HTTP response returns allow).

**Denial recording (FR14, FR15)**
- After a denied tool call, `Manager.DeniedCalls(runID)` is populated with the
  tool name and target path.

**WebSocket events (FR20)**
- Every permission decision (allow or deny) broadcasts an `agent.permission`
  event on the project hub with `run_id`, `tool_name`, `decision`, and `reason`.

### Milestone 4 — Hook Helper (`tests/integration/hook_helper_test.go`)

- **Happy path (AC2):** mock server returns allow → stdout is `{"decision":"allow"}`,
  exit code 0.
- **Deny forwarded:** mock server returns deny → stdout is `{"decision":"deny"}`,
  exit code 0.
- **Secret forwarded:** helper sends `Authorization: Bearer <secret>` header
  to the server.
- **Server unreachable (NFR2):** non-listening address → helper retries once,
  completes within 5 s, stdout is `{"decision":"deny","reason":"..."}`, exit 0.
- **Missing `KC_HOOK_SECRET`:** helper outputs deny JSON, exits 0.
- **Malformed stdin:** helper does not crash, exits 0.
- **Exit code always 0:** verified for both allow and deny server responses.

### Milestone 5 — Mediated Driver (`tests/integration/claude_mediated_driver_test.go`)

Uses the existing `fake_precheck_claude` binary (compiled by `TestMain` in
`agent_precheck_test.go`) with different `FAKE_CLAUDE_MODE` values.

- **Default mode passes (AC1):** fake claude emits `permissionMode="default"` →
  mediated precheck passes → run completes with `status=done`.
- **Missing permissionMode passes:** fake claude omits the field → precheck
  passes → `status=done`.
- **Bypass mode fails (AC13):** fake claude emits `permissionMode="bypassPermissions"`
  → mediated precheck kills the run → `status=failed`, `agent.failed` WS event
  carries `reason="precheck_mediated_bypass"`.
- **No bypass flags in CLI args (FR2):** mediated driver does NOT include
  `--dangerously-skip-permissions` or `--permission-mode bypassPermissions`;
  `--settings` flag is present (FR6).
- **Settings file lifecycle (AC11, NFR4):** `hook-settings-{runID}.json` is
  written to `os.TempDir()` before the process starts, contains valid JSON with
  a `hooks` key; file is removed when `mediatedProcess.Wait()` returns after
  the run ends.
- **Concurrent runs have distinct settings files:** two simultaneous runs
  produce different settings file paths (no collision).

### Fixture

`tests/fixtures/mock_claude_mediated.sh` — a shell-script stand-in for `claude`
that can be used in future tests requiring a configurable fake mediated binary.
It honours the same `FAKE_CLAUDE_*` env vars as `fake_precheck_claude` for
compatibility.

---

## Test Files

| File | Milestones |
|------|------------|
| `tests/integration/permission_endpoint_test.go` | 3 |
| `tests/integration/hook_helper_test.go` | 4 |
| `tests/integration/claude_mediated_driver_test.go` | 5 |
| `tests/fixtures/mock_claude_mediated.sh` | 5 (fixture) |

---

## Known Gaps

The following scenarios from the test plan require a mock claude binary that
actually invokes the hook-helper binary and POSTs to the permission endpoint.
They are not covered by the current suite and are tracked as open work:

- M5: Denied write → no auto-commit and queue paused (AC4, AC6).
- M5: Bash tool with denylist match during an active run (AC5).
- M7: Frontend store and component tests (`web/src/stores/__tests__/` etc.).
- M8: Manual verification checklist (requires a real Claude Code installation).
