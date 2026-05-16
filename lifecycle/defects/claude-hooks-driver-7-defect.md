---
title: hook-helper test assertions use obsolete simple JSON format
type: defect
status: done
lineage: claude-hooks-driver
created: "2026-05-16T00:00:00+10:00"
parent: lifecycle/tests/claude-hooks-driver-6-test.md
labels:
    - defect
release: KC-Release2
assignees:
    - role: test-developer
      who: agent
---

# hook-helper test assertions use obsolete simple JSON format

Two tests in `tests/integration/hook_helper_test.go` fail because they assert
that the hook-helper outputs `{"decision":"deny","reason":"..."}` (the old
simple format), but `cmd/kaos-control/hookcmd/hook.go` was updated to output
the Claude Code hooks-native format:

```json
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"..."}}
```

The `writeResponse()` function comment in `hook.go` explicitly documents this
change (the simple format "is silently ignored by Claude"). The tests were not
updated to match.

---

## Reproduction Steps

1. From the repo root, run:
   ```
   go test -tags integration ./tests/integration/ -run "TestHookHelper_ServerUnreachable|TestHookHelper_MissingSecret" -v
   ```
2. Observe failures.

---

## Expected Behaviour

Both tests pass. The assertions match the JSON actually written to stdout by
the hook-helper.

---

## Actual Behaviour

`TestHookHelper_ServerUnreachable` fails:
```
hook_helper_test.go:231: decision = "", want deny when server is unreachable
hook_helper_test.go:236: reason = "", expected to mention unreachable server
```

`TestHookHelper_MissingSecret` fails:
```
hook_helper_test.go:270: response must contain a 'decision' key
hook_helper_test.go:275: decision = "", want deny when secret is missing
```

Both tests unmarshal the hook-helper stdout into a `map[string]any` and
check `resp["decision"]`. The actual stdout is:

```
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"server unreachable"}}
```

The top-level key is `hookSpecificOutput`, not `decision`, so the assertions
produce empty strings and fail.

---

## Logs / Output

```
=== RUN   TestHookHelper_ServerUnreachable
    hook_helper_test.go:212: hook-helper stdout: {"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"server unreachable"}}
    hook_helper_test.go:212: hook-helper stderr: 2026/05/16 09:10:22 hook-helper: server unreachable: Post "http://127.0.0.1:29187/api/agent/run-unreachable/permission": dial tcp 127.0.0.1:29187: connect: connection refused
    hook_helper_test.go:231: decision = "", want deny when server is unreachable
    hook_helper_test.go:236: reason = "", expected to mention unreachable server
--- FAIL: TestHookHelper_ServerUnreachable (0.51s)

=== RUN   TestHookHelper_MissingSecret
    hook_helper_test.go:257: hook-helper stdout: {"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"KC_HOOK_SECRET not set"}}
    hook_helper_test.go:257: hook-helper stderr: hook-helper: KC_HOOK_SECRET not set
    hook_helper_test.go:270: response must contain a 'decision' key
    hook_helper_test.go:275: decision = "", want deny when secret is missing
--- FAIL: TestHookHelper_MissingSecret (0.03s)
```

---

## Fix guidance

Update the two failing tests to assert against the Claude-native output shape.
For `TestHookHelper_ServerUnreachable` and `TestHookHelper_MissingSecret`,
replace:

```go
if dec, _ := resp["decision"].(string); dec != "deny" { ... }
if _, ok := resp["decision"]; !ok { ... }
reason, _ := resp["reason"].(string)
```

with logic that drills into `resp["hookSpecificOutput"]`:

```go
inner, _ := resp["hookSpecificOutput"].(map[string]any)
if dec, _ := inner["permissionDecision"].(string); dec != "deny" { ... }
reason, _ := inner["permissionDecisionReason"].(string)
```

Also update the test plan artifact (`lifecycle/test-plans/claude-hooks-driver-5-test.md`)
to document the Claude-native output format so future tests are written correctly.
