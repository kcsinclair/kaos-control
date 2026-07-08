---
title: Test-runner agent abandons long suites — parks on ScheduleWakeup and never files defects
type: defect
status: done
lineage: test-runner-parks-on-schedulewakeup
created: "2026-07-07T00:00:00+10:00"
priority: high
labels:
    - defect
    - agent
    - test-runner
    - reliability
assignees:
    - role: backend-developer
      who: agent
---

## Resolution (2026-07-08, fix committed `193c5ff8`)

Fixed via the test-runner prompt in [config.yaml](../config.yaml) (option 1):
run every suite **synchronously in the foreground** (make lint / test-unit /
test-integration / vitest / test-e2e), blocking on each, with an explicit ban on
backgrounding and `ScheduleWakeup`; `timeout_minutes` 30→45 to cover e2e; and a
spelled-out defect-filing format + dedup. Config validated via the Go loader.

**Activation:** config is not hot-reloaded (`handleUpdateConfig` writes without
refreshing the cached `p.Cfg`, and config.yaml isn't watched), so the running
server must be **restarted** to pick it up. End-to-end confirmation (a single
`devops run test-all` that runs every suite to completion in one agent run) is
pending that restart + re-run.

# Test-runner agent abandons long suites — parks on `ScheduleWakeup` and never files defects

## Summary

The `test-runner` agent (claude-code-cli, sonnet; prompt "Run all test suites,
parse failures, and file defect artifacts") completes the **fast** suites (Go
unit, web vitest) but for the **slow** suites (Go integration, e2e) it launches
them in the background and calls **`ScheduleWakeup`** to resume and collect the
results later. kaos-control's agent runner is **one-shot** — when the agent
process returns, the run is marked `done` and never re-invoked — so the wakeup
never fires. The test-runner therefore **abandons the integration and e2e
suites, files no defects for them, and reports only a partial result.**

The whole point of the test-runner (auto-file defects from test failures) is
defeated for exactly the suites most likely to surface real regressions.

## Reproduction Steps

1. Trigger the test-runner: `kaos-control devops run test-all` (the "Test Runner
   for Everything" pipeline `POST`s to `/api/p/{project}/agents/test-runner/run`).
2. Watch the test-runner run: it runs unit + vitest, then starts
   `go test ./... -tags=integration` in the background and calls
   `ScheduleWakeup(delaySeconds: 600, …)`.
3. The agent run is marked `done` (~180s), with a terminal result like:
   *"Unit tests … and the web vitest suite both pass … Integration tests are
   still executing in the background — I'll report back … once that (and the e2e
   suite) finishes."*
4. Observe: **no continuation run ever occurs** (checked 3 h later — the run
   stayed `done`, no new test-runner run), and **zero defects were filed** for
   integration/e2e regardless of their outcome.

## Actual vs Expected

- **Actual:** partial run; slow-suite results discarded; 0 defects filed.
- **Expected:** the test-runner runs *all* suites to completion within its
  timeout and files (deduped) defects for any failures.

## Root Cause

The test-runner offloads long suites to the background and relies on
`ScheduleWakeup` for a later re-invocation. kaos-control agent runs are not
resumable — there is no wakeup/continuation mechanism — so the scheduled resume
is silently dropped when the process exits.

## Suggested Fix

Prefer the simplest robust option:

1. **Run the suites synchronously** within the agent (block on each
   `go test … -tags=integration`, `make test-e2e`, etc. until it exits, then
   parse), rather than backgrounding + `ScheduleWakeup`. The agent already has
   `timeout_minutes: 30`, which comfortably covers unit + integration + vitest
   (~5 min total); bump it if e2e (needs `make build` + Playwright) pushes it
   over. This is a **prompt-template change** in
   [lifecycle/config.yaml](../config.yaml) (`test-runner` agent): instruct it to
   run each suite to completion in the foreground and not to use
   `ScheduleWakeup`.
2. *(Alternative / larger)* teach the kaos-control agent runner to honour
   `ScheduleWakeup` (persist the run and re-invoke on the timer). More powerful
   but a substantial change to the one-shot run model.

## Notes from this run (2026-07-07)

Everything that *was* run passed: **Go unit 20/20 packages, web vitest
1536/1536, Go integration `ok` (250s)** — so no product-bug defects were
warranted from this cycle; the only defect is this test-runner limitation.
(The e2e suite was not independently verified here.)

## Verification

- After the fix, a single `devops run test-all` should run every suite to
  completion in one agent run and file defects for any failures (with the
  existing dedup), with no `ScheduleWakeup` in the run log.
