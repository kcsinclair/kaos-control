---
title: Agent run wall-clock far exceeds the CLI's self-reported duration (16m unexplained)
type: defect
status: draft
lineage: agent-run-wall-clock-exceeds-reported-duration
created: "2026-08-25T12:10:00+10:00"
priority: normal
labels:
    - defect
    - agent
    - agent-runner
    - observability
release: KC-Release6
assignees:
    - role: backend-developer
      who: agent
---

# Agent run wall-clock far exceeds the CLI's self-reported duration

## Reproduction Steps

Observed on run `e698997b315e5ca8` (agent `frontend-developer`, driver
`claude-code-cli`, target `lifecycle/frontend-plans/switch-provider-*`):

1. Inspect the run record:
   `sqlite3 ~/.kaos-control/data/kaos-control/index.db "SELECT status, exit_code, finished_at - started_at FROM agent_runs WHERE run_id='e698997b315e5ca8';"`
   → `failed | -1 | 1128` (18m 48s wall-clock)
2. Inspect the terminal result line in
   `~/.kaos-control/data/kaos-control/runs/e698997b315e5ca8.log`
   → `"duration_ms":144290, "duration_api_ms":142993` (2m 24s / 2m 22s)

## Expected Behaviour

A run's recorded wall-clock duration should be broadly explainable by the work
the agent actually did. Where it is not — because the process lingered after
finishing, or was blocked — that should be visible, bounded, and attributable,
rather than silently inflating the run's duration.

## Actual Behaviour

**~16 minutes of the 18m 48s run are unaccounted for.** The Claude Code CLI
reports its own total as 2m 24s, yet `cmd.Wait()` did not return for another
~16 minutes. The run then ended with `exit_code: -1` and an **empty
`stderr_tail`**, so nothing on the record explains the gap.

The terminal result line was written — 49 turns, full token usage, and an
`is_error:true` API-error message (that message was being discarded by the UI
until the `is_error` display fix, commit `7caa1536`). So the agent's work
completed; the process simply did not exit for a further ~16 minutes.

Notably `frontend-developer` has **`timeout_minutes: 0`** — no timeout — so
nothing would have bounded this. The run happened to end at ~18m; a genuinely
hung process would linger indefinitely, holding a `max_concurrent_agents` slot
and a lineage lock.

## What has been ruled out

- **Not the "stdout readers did not drain" warning.** That warning fires 5 s
  into *every* run longer than `readerDrainGrace` (5 s) — it was logged for
  seven **successful** runs (`exit=0`, 149 s–2532 s) as well as this one. It is
  spurious noise, tracked separately as
  [[agent-runner-spurious-drain-warning]], and is **not** evidence of a detached
  grandchild here.
- **Not a server restart.** No `kaos-control started` entry appears between the
  run's start (10:41:44) and finish (11:00:32).

## What is not yet known

The cause of the gap is **undetermined**. The run log carries **no per-event
timestamps**, so it is not possible to tell from the log whether the ~16 minutes
elapsed *before* the terminal result event (e.g. a long stall mid-run that the
CLI does not count in `duration_ms`) or *after* it (the process failing to exit
once its work was done). Distinguishing those two is the first task.

## Fix guidance

1. **Make the gap diagnosable.** Add per-event timestamps to the run log (or at
   minimum record the wall-clock time at which the terminal `result` event was
   received). This alone determines whether the stall is pre- or post-result and
   would have answered the question here.
2. **Bound the post-result wait.** If the process has emitted its terminal
   result event, waiting indefinitely for it to exit is never correct: apply a
   short grace, then terminate and record that this happened as a distinct
   reason rather than a bare `exit_code: -1`.
3. **Record a reason for `exit -1`.** A killed/abandoned process currently
   leaves `stderr_tail` empty, so the run record explains nothing. Populate a
   failure reason (the codebase already has `failure_reason` on `AgentRunRow`).
4. **Reconsider `timeout_minutes: 0` as a default** for long-running developer
   agents, or enforce a separate hard ceiling, so a hung process cannot hold a
   concurrency slot and lineage lock forever.

## Notes

Found while investigating why the run summary card displayed "Success" for this
failed run. That display bug is fixed in commit `7caa1536`; this duration gap is
the separate, unresolved half.
