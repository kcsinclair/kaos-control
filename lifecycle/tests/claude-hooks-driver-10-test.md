---
title: "Test Fix: HappyPath and PassesDenyDecision assertions updated to hookSpecificOutput format"
type: test
status: draft
lineage: claude-hooks-driver
parent: lifecycle/defects/claude-hooks-driver-9-defect.md
created: "2026-05-16T00:00:00+10:00"
---

# Test Fix: HappyPath and PassesDenyDecision assertions updated to hookSpecificOutput format

Parent: [[claude-hooks-driver-9-defect]].

This artifact documents the fix applied to `tests/integration/hook_helper_test.go`
to resolve the two assertions that were still checking the obsolete flat JSON format
(`{"decision":"..."}`) instead of the Claude-native PreToolUse hook response format
(`{"hookSpecificOutput":{...}}`).

---

## Scenarios Fixed

### `TestHookHelper_HappyPath`

Previously asserted `resp["decision"].(string) == "allow"`. Updated to drill into
`resp["hookSpecificOutput"]` and check `inner["permissionDecision"] == "allow"`,
matching the output of `writeResponse()` in `cmd/kaos-control/hookcmd/hook.go`.

### `TestHookHelper_PassesDenyDecision`

Previously asserted `resp["decision"].(string) == "deny"`. Updated to drill into
`resp["hookSpecificOutput"]` and check `inner["permissionDecision"] == "deny"`,
matching the actual stdout produced when the server returns a deny decision.

---

## Expected Output Shape (post-fix)

Both tests now expect the hook-helper to emit:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow"
  }
}
```

or for deny:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "not allowed"
  }
}
```

This matches the `writeResponse()` function in `cmd/kaos-control/hookcmd/hook.go`
and is the format required by the Claude Code hooks API.

---

## Test File

| File | Change |
|------|--------|
| `tests/integration/hook_helper_test.go` | Updated `TestHookHelper_HappyPath` and `TestHookHelper_PassesDenyDecision` to assert against `hookSpecificOutput` shape |

---

## Full Suite Coverage

All seven `TestHookHelper_*` tests now use the correct `hookSpecificOutput` format:

| Test | Scenario | Decision |
|------|----------|----------|
| `TestHookHelper_HappyPath` | Server returns allow | allow |
| `TestHookHelper_PassesDenyDecision` | Server returns deny | deny |
| `TestHookHelper_ForwardsSecret` | KC_HOOK_SECRET sent as Bearer token | allow |
| `TestHookHelper_ServerUnreachable` | Connection refused, retries once | deny |
| `TestHookHelper_MissingSecret` | KC_HOOK_SECRET not set | deny |
| `TestHookHelper_MalformedStdin` | Invalid JSON on stdin | (exit 0) |
| `TestHookHelper_ExitCodeAlwaysZero` | allow and deny both exit 0 | both |

---

## Verification

Run the previously-failing tests:

```sh
go test -tags integration ./tests/integration/ \
  -run "TestHookHelper_HappyPath|TestHookHelper_PassesDenyDecision" -v
```

Run the full suite to confirm all seven pass:

```sh
go test -tags integration ./tests/integration/ -run "TestHookHelper" -v
```
