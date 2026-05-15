---
title: 'hook-helper outputs inconsistent JSON format: pass-through vs error paths'
type: defect
status: in-development
lineage: claude-hooks-driver
created: "2026-05-16T00:00:00+10:00"
parent: lifecycle/tests/claude-hooks-driver-6-test.md
labels:
    - defect
release: KC-Release2
assignees:
    - role: backend-developer
      who: agent
---

# hook-helper outputs inconsistent JSON format: pass-through vs error paths

`cmd/kaos-control/hookcmd/hook.go` uses two different JSON shapes depending on
the code path:

1. **Error paths** (server unreachable, missing secret, stdin read error) —
   `writeResponse()` emits the Claude Code hooks-native format:
   ```json
   {"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"..."}}
   ```

2. **Server response pass-through** — the raw response body from the
   kaos-control permission endpoint is written to stdout unchanged:
   ```json
   {"decision":"allow"}
   ```

The code comment in `writeResponse()` explains that the simple
`{"decision":"..."}` format **"is silently ignored by Claude, which falls back
to interactive prompts"**. This means that when the permission endpoint
responds (the happy path), the hook-helper passes a response that Claude Code
silently ignores, defeating the purpose of the permission hook entirely.

---

## Reproduction Steps

1. Observe `cmd/kaos-control/hookcmd/hook.go` lines 74–75:
   ```go
   // Pass the response through to stdout unchanged.
   _, _ = os.Stdout.Write(respBody)
   ```
2. Compare with the `writeResponse()` function (lines 125–136) which emits
   `hookSpecificOutput`.
3. Note that `TestHookHelper_HappyPath` and `TestHookHelper_PassesDenyDecision`
   pass — but only because they test the hook-helper in isolation with a mock
   server; they do not verify Claude Code actually reads the response.

---

## Expected Behaviour

The hook-helper always writes Claude-native `hookSpecificOutput` JSON to stdout,
regardless of whether the decision came from the server or an error path. The
permission endpoint's `{"decision":"allow/deny"}` response is translated to
the Claude-native format before writing.

---

## Actual Behaviour

When the permission endpoint responds successfully, the hook-helper passes
through `{"decision":"allow"}`. Claude Code silently ignores this format,
meaning allow/deny decisions from kaos-control have **no effect** on what
Claude Code actually permits during live runs.

---

## Logs / Output

From a passing test (happy path), hook-helper stdout written to Claude Code:
```
{"decision":"allow"}
```

From an error path (server unreachable), hook-helper stdout written to Claude Code:
```
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"server unreachable"}}
```

---

## Fix guidance

In `cmd/kaos-control/hookcmd/hook.go`, replace the pass-through write with a
translation step. After reading `respBody`, unmarshal to extract `decision` and
`reason`, then call `writeResponse(decision, reason)`:

```go
var serverResp struct {
    Decision string `json:"decision"`
    Reason   string `json:"reason"`
}
if err := json.Unmarshal(respBody, &serverResp); err != nil || serverResp.Decision == "" {
    writeDeny("malformed server response")
    return
}
writeResponse(serverResp.Decision, serverResp.Reason)
```

After this fix, update `TestHookHelper_HappyPath`, `TestHookHelper_PassesDenyDecision`,
`TestHookHelper_ForwardsSecret`, and `TestHookHelper_ExitCodeAlwaysZero` to
assert against the `hookSpecificOutput` shape (coordinated with the test-developer
defect `claude-hooks-driver-7-defect.md`).
