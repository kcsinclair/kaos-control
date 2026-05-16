---
title: "Defect: HappyPath and PassesDenyDecision tests still assert old flat JSON format"
type: defect
status: draft
lineage: claude-hooks-driver
parent: lifecycle/tests/claude-hooks-driver-8-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
created: "2026-05-16T00:00:00+10:00"
---

# Defect: HappyPath and PassesDenyDecision tests still assert old flat JSON format

Parent: [[claude-hooks-driver-8-test]].

The fix documented in `claude-hooks-driver-8-test.md` updated
`TestHookHelper_ServerUnreachable` and `TestHookHelper_MissingSecret` to use
the Claude-native `hookSpecificOutput` response shape. However,
`TestHookHelper_HappyPath` and `TestHookHelper_PassesDenyDecision` were not
updated and still assert the obsolete flat `{"decision":"..."}` top-level key.

Because `writeResponse()` in `cmd/kaos-control/hookcmd/hook.go` **always**
wraps its output in `hookSpecificOutput`, those two tests now fail on every
run.

---

## Reproduction Steps

1. Check out the `kc-dev` branch.
2. Run the full hook-helper integration suite:
   ```sh
   go test -tags integration ./tests/integration/ -run "TestHookHelper" -v
   ```
3. Observe that `TestHookHelper_HappyPath` and `TestHookHelper_PassesDenyDecision`
   fail while the other five tests pass.

---

## Expected Behaviour

All seven `TestHookHelper_*` tests pass. `TestHookHelper_HappyPath` and
`TestHookHelper_PassesDenyDecision` should drill into `hookSpecificOutput` and
check `permissionDecision`, matching the format used by the already-fixed
error-path tests.

---

## Actual Behaviour

```
--- FAIL: TestHookHelper_HappyPath (1.25s)
    hook_helper_test.go:142: decision = "", want allow

--- FAIL: TestHookHelper_PassesDenyDecision (0.01s)
    hook_helper_test.go:170: decision = "", want deny
```

The stdout emitted by the hook is:

```json
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}
```

But the test asserts `resp["decision"].(string) == "allow"`, which is always
empty because the top-level key is `hookSpecificOutput`, not `decision`.

---

## Logs / Output

```
=== RUN   TestHookHelper_HappyPath
    hook_helper_test.go:131: hook-helper stdout: {"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}
    hook_helper_test.go:131: hook-helper stderr: 
    hook_helper_test.go:142: decision = "", want allow
--- FAIL: TestHookHelper_HappyPath (1.25s)

=== RUN   TestHookHelper_PassesDenyDecision
    hook_helper_test.go:160: hook-helper stdout: {"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"not allowed"}}
    hook_helper_test.go:160: hook-helper stderr: 
    hook_helper_test.go:170: decision = "", want deny
--- FAIL: TestHookHelper_PassesDenyDecision (0.01s)
```

---

## Fix Required

In `tests/integration/hook_helper_test.go`:

**`TestHookHelper_HappyPath`** (lines ~137–143): replace

```go
if dec, _ := resp["decision"].(string); dec != "allow" {
    t.Errorf("decision = %q, want allow", dec)
}
```

with

```go
inner, _ := resp["hookSpecificOutput"].(map[string]any)
if dec, _ := inner["permissionDecision"].(string); dec != "allow" {
    t.Errorf("permissionDecision = %q, want allow", dec)
}
```

**`TestHookHelper_PassesDenyDecision`** (lines ~165–172): replace

```go
if dec, _ := resp["decision"].(string); dec != "deny" {
    t.Errorf("decision = %q, want deny", dec)
}
```

with

```go
inner, _ := resp["hookSpecificOutput"].(map[string]any)
if dec, _ := inner["permissionDecision"].(string); dec != "deny" {
    t.Errorf("permissionDecision = %q, want deny", dec)
}
```
