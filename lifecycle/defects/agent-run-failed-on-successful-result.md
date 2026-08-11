---
title: Agent run shows status "failed" despite a successful result (non-zero exit overrides success)
type: defect
status: done
lineage: agent-run-failed-on-successful-result
created: "2026-08-11T00:00:00+10:00"
priority: high
labels:
    - defect
    - agent
    - reliability
    - status
assignees:
    - role: backend-developer
      who: agent
---

# Agent run shows status "failed" despite a successful result

## Resolution (2026-08-11)

Fixed in [internal/agent/agent.go](../../internal/agent/agent.go): the
`broadcast` closure now records `resultEventSuccess` alongside `resultEventSeen`
(via a new `resultEventIsError` helper), and a **success-reconciliation** step
after the exit-code branch upgrades a plain `failed` (non-zero exit that was not
a user kill or timeout) back to `done` when a terminal result event with
`is_error:false` was observed. Scoped so explicit kills, timeouts, and error
results (`is_error:true`, incl. auth) are unaffected.

Regression tests in
[internal/agent/precheck_test.go](../../internal/agent/precheck_test.go):
`TestSupervise_NonZeroExitWithSuccessResultMarkedDone` (success result + non-zero
exit → done) and `TestSupervise_NonZeroExitWithErrorResultStaysFailed` (error
result + non-zero exit → stays failed). Full agent package green. Live
confirmation pending a server restart.

## Source

GitHub issue [#14](https://github.com/kcsinclair/kaos-control/issues/14) —
reported by **aburow**, 2026-06-21.

## Summary

An agent run's **status column reads `failed`** while its **Run Summary reads
`Success`** for the same run. In the reported case (a `frontend-developer` run,
Claude Code driver): status `failed`, but summary `Success`, cost $0.9211,
duration 5m 31s, 22 turns, full token metrics captured, 86.8% cache hit — and
"No output recorded" in the run log. The only place the failure is visible is
this screen; the run otherwise completed successfully.

## Likely root cause

Run status is derived from the process exit code alone, independently of the
terminal result event:

```go
// internal/agent/agent.go ~L876-889
exitCode := 0
status := "done"
if waitErr != nil {
    status = "failed"          // ← non-zero exit → failed, unconditionally
    ...
}
```

A Claude run can emit a clean terminal `{"type":"result","is_error":false,...}`
(which is what populates the "Success" badge, cost, turns, and token metrics —
so `resultEventSeen` is true and the truncated-stream check at
[L908](../../internal/agent/agent.go#L908) does **not** fire) and *still* exit
non-zero (a non-fatal error on shutdown, broken pipe, signal, etc.). Because
[L879-880](../../internal/agent/agent.go#L879) sets `failed` purely on
`waitErr`, the run is recorded as failed even though the agent's task succeeded.

This is a **distinct** path from the truncated-stream `cmd.Wait` race fixed
earlier (that one was `resultEventSeen == false`; here the result event *was*
seen and parsed).

## Expected

When a terminal result event with `is_error:false` was observed, the run should
be recorded as `done` (or at least the status must reconcile with the result
event) rather than being overridden to `failed` by a non-zero exit code.
Status and Run Summary must not disagree.

## Notes

Verified still present on `kc-dev` (2026-08-11): the exit-code-only status
assignment is unchanged.
