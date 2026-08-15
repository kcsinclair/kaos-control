---
title: Agent runner should reject/strip ScheduleWakeup — one-shot runs silently drop deferred work
type: defect
status: done
lineage: agent-runner-strip-schedulewakeup
created: "2026-07-08T00:00:00+10:00"
priority: medium
labels:
    - defect
    - agent
    - reliability
release: KC-Release5
assignees:
    - role: backend-developer
      who: agent
parent: lifecycle/tests/new-project-init-directory-options-6-test.md
---

# Agent runner should reject/strip `ScheduleWakeup` — one-shot runs silently drop deferred work

## Summary

kaos-control agent runs are **one-shot**: when the agent process exits, the run
is marked terminal and never re-invoked. But a claude-code-cli agent can call
the `ScheduleWakeup` tool to defer work ("check back in N seconds"). That wakeup
is never honoured, so anything the agent deferred is **silently dropped** — the
run ends looking `done` while its real work never happened.

This was the root cause of `test-runner-parks-on-schedulewakeup` (the test-runner
backgrounded the slow suites and parked on a wakeup). That was worked around by
forbidding `ScheduleWakeup` in the test-runner **prompt** — but a prompt ban is
not a guarantee: on the confirming re-run the agent **still called
`ScheduleWakeup` twice** (it happened to push through that time). Any agent that
reaches for it can still park. The runner should defend against this
structurally rather than trusting every prompt.

## Reproduction Steps

1. Run any claude-code-cli agent whose task involves a long-running step.
2. Observe the run log contains `{"type":"...","name":"ScheduleWakeup",...}` tool
   calls (e.g. `test-runner` run `1fc6cac66ff8d756` — 2 calls; the earlier
   `82a23727743a` parked on one and filed 0 defects).
3. The scheduled wakeup never fires; if the agent relied on it, its deferred work
   is lost and the run is still marked `done`.

## Expected Behaviour

An agent run should not be able to silently defer work past its own lifetime.
The runner should make `ScheduleWakeup` a no-op that the agent cannot depend on —
forcing it to complete its work inline within the run's `timeout_minutes`.

## Suggested Fix

Intercept the `ScheduleWakeup` tool at the agent-runner boundary for one-shot
agent runs and **reject / hard-strip** it (return a tool result telling the
agent it is unavailable and it must finish the work in this run), rather than
letting the call succeed and schedule a wakeup nothing honours. See the
`ClaudeCodeDriver` / supervisor tool handling in
[internal/agent/agent.go](../../internal/agent/agent.go).

*(Alternative, larger: make agent runs actually resumable — persist the run and
re-invoke on the wakeup timer. More powerful but a substantial change to the
one-shot model; the reject-and-force-inline option is the low-risk fix.)*

## Verification

- An agent that attempts `ScheduleWakeup` receives a tool result indicating it is
  unavailable, and completes its task within the run.
- `test-runner` runs to completion with **zero** honoured wakeups regardless of
  whether the model attempts one.
