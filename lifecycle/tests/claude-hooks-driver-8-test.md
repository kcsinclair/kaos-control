---
title: "Test Fix: hook-helper assertions updated to Claude-native output format"
type: test
status: approved
lineage: claude-hooks-driver
parent: lifecycle/defects/claude-hooks-driver-7-defect.md
created: "2026-05-16T00:00:00+10:00"
---

# Test Fix: hook-helper assertions updated to Claude-native output format

Parent: [[claude-hooks-driver-7-defect]].

This artifact documents the fix applied to `tests/integration/hook_helper_test.go`
to resolve the two assertions that were checking the obsolete simple JSON format
(`{"decision":"...","reason":"..."}`) instead of the Claude-native PreToolUse
hook response format (`{"hookSpecificOutput":{...}}`).

---

## Scenarios Fixed

### `TestHookHelper_ServerUnreachable`

Previously asserted `resp["decision"]` and `resp["reason"]`. Updated to drill
into `resp["hookSpecificOutput"]` and check `permissionDecision` and
`permissionDecisionReason` instead, matching the output of `writeResponse()` in
`cmd/kaos-control/hookcmd/hook.go`.

### `TestHookHelper_MissingSecret`

Previously asserted `resp["decision"]` and checked for a top-level `"decision"`
key. Updated to check for `resp["hookSpecificOutput"]` being non-nil and
`inner["permissionDecision"] == "deny"`, matching the actual stdout produced
when `KC_HOOK_SECRET` is not set.

---

## Expected Output Shape (post-fix)

Both tests now expect the hook-helper to emit:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "<human-readable reason>"
  }
}
```

This matches the `writeResponse()` function in `cmd/kaos-control/hookcmd/hook.go`
and is the format required by the Claude Code hooks API.

---

## Test File

| File | Change |
|------|--------|
| `tests/integration/hook_helper_test.go` | Updated `TestHookHelper_ServerUnreachable` and `TestHookHelper_MissingSecret` to assert against `hookSpecificOutput` shape |

---

## Verification

Run the two previously-failing tests:

```sh
go test -tags integration ./tests/integration/ \
  -run "TestHookHelper_ServerUnreachable|TestHookHelper_MissingSecret" -v
```

Both should now pass.
