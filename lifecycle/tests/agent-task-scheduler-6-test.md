---
title: "Agent & Task Scheduler — Test Suite"
type: test
status: draft
lineage: agent-task-scheduler
parent: lifecycle/test-plans/agent-task-scheduler-5-test.md
created: "2026-05-06T00:00:00+10:00"
assignees:
    - role: test-developer
      who: agent
---

# Agent & Task Scheduler — Test Suite

Covers all seven milestones from the test plan at
`lifecycle/test-plans/agent-task-scheduler-5-test.md`.

Two production bugs were discovered and fixed during implementation:

1. **`scheduler.Stop()` deadlock** (`internal/scheduler/scheduler.go`) — `Stop()`
   blocked forever on `<-sc.doneCh` when `Start()` had never been called. Fixed by
   adding an `atomic.Bool started` flag; `Stop()` now returns immediately if
   the scheduler was never started. This unblocked all integration tests whose
   cleanup calls `project.Close()` → `scheduler.Stop()`.

2. **Queued jobs never dispatched** (`internal/scheduler/scheduler.go`) — when
   all worker slots were occupied and a job was enqueued, no goroutine called
   `tryDispatch()` after a running job finished. Fixed by adding
   `sc.tryDispatch()` to the deferred cleanup inside `execute()`. This was
   needed for the concurrency-limit and priority-ordering tests to pass.

---

## Milestone 1 — Store layer

**File:** `internal/scheduler/store_test.go`

10 unit tests exercising `scheduler.Store` against a temporary file-backed
SQLite database with fresh scheduler tables per test:

| Test | Scenario |
|---|---|
| `TestCreateJob` | Insert a job; verify all fields round-trip |
| `TestCreateJobDuplicateName` | Duplicate name → error |
| `TestGetJobNotFound` | Unknown name → nil, no error |
| `TestUpdateJob` | Update priority, enabled, schedule; verify persistence |
| `TestDeleteJobCascades` | Delete job; verify runs are cascade-deleted |
| `TestListJobs` | Three jobs returned ordered by name |
| `TestInsertRunAndListRuns` | Three runs; verify ordered start_time DESC |
| `TestListRunsPagination` | Correct page boundaries across 5 runs |
| `TestPruneOldRuns` | Old runs removed; recent run retained |
| `TestPruneOldRunsDeletesLogFiles` | Pruner removes referenced log file from disk |

---

## Milestone 2 — Schedule evaluation

**File:** `internal/scheduler/schedule_test.go`

10 unit tests of `NextFireTime` and `ValidateScheduleSpec`:

| Test | Scenario |
|---|---|
| `TestCron5FieldNextFire` | `0 2 * * *` before target hour → same day |
| `TestCron5FieldPastToday` | `0 2 * * *` after target hour → next day |
| `TestCron6FieldWithSeconds` | 6-field `30 0 2 * * *` → 02:00:30 |
| `TestCronDayOfWeek` | `0 9 * * 1` (Monday) from Wednesday → next Monday |
| `TestIntervalFromLastRun` | 30 m interval, last run 20 min ago → fires in 10 min |
| `TestIntervalNoPriorRun` | No prior run → fires immediately (now) |
| `TestOnceFuture` | Future one-off → returns that time |
| `TestOncePast` | Past one-off → returns zero (skip) |
| `TestInvalidCronExpression` | Garbage string → parse error |
| `TestInvalidInterval` | Zero / negative interval → validation error |

---

## Milestone 3 — Precondition evaluation

**File:** `internal/scheduler/precondition_test.go`

13 unit tests using live SQLite, `httptest.NewServer`, temp dirs, and real
shell commands. No external network calls.

| Test | Scenario |
|---|---|
| `TestAfterJobDependencySucceeded` | Last run=success → true |
| `TestAfterJobDependencyFailed` | Last run=failure → false |
| `TestAfterJobDependencyNeverRan` | No runs → false |
| `TestAfterJobDependencyDoesNotExist` | Job absent → false (not error) |
| `TestFileExistsPresent` | File inside sandbox → true |
| `TestFileExistsAbsent` | File missing → false |
| `TestFileExistsPathTraversal` | `../../etc/passwd` → sandbox error |
| `TestHTTPOk200` | httptest server 200 → true |
| `TestHTTPOk500` | httptest server 500 → false |
| `TestHTTPOkUnreachable` | Dead port → false, no hang |
| `TestShellExitZero` | `true` → true |
| `TestShellExitNonZero` | `false` → false |
| `TestShellTimeout` | `sleep 60` with 2 s ctx → false |

---

## Milestone 4 — Scheduler engine

**File:** `internal/scheduler/scheduler_test.go`

12 unit tests using a real started `*Scheduler` with a temp SQLite DB and
`hub.Hub`. No real Claude Code invocations — `agents` is always `nil`.

| Test | Scenario |
|---|---|
| `TestJobFiresOnSchedule` | TriggerNow → run status=success |
| `TestJobDoesNotFireWhenPaused` | Pause → enabled=false, no spontaneous runs |
| `TestConcurrencyLimit` | maxWorkers=1, two jobs → B starts after A ends |
| `TestPriorityOrdering` | hi-pri(10) starts before lo-pri(1) when queued together |
| `TestTriggerNow` | Future-scheduled job dispatched immediately |
| `TestTimeoutEnforcement` | `sleep 60` with TimeoutSec=1 → status=timeout |
| `TestJobFailureRecovery` | Failure + subsequent success → scheduler alive |
| `TestStartupReconciliationStaleRunning` | Pre-existing running run → failure on Start() |
| `TestStartupReconciliationMissedOneOff` | Past one-off with no runs → skipped on Start() |
| `TestPreconditionGating` | after_job unsatisfied → no dispatch; satisfied → runs |
| `TestAgentTargetDispatch` | agent job with agents=nil → failure (dispatch code exercised) |
| `TestShellTargetOutputCapture` | `echo HELLO_MARKER` → marker in log file |
| `TestHubEventsFired` | started + completed events broadcast to hub |

---

## Milestone 5 — REST API endpoints

**File:** `tests/integration/scheduler_api_test.go`

19 integration tests using a full HTTP test server with the chi router,
auth middleware, and CSRF middleware.

| Test | Scenario |
|---|---|
| `TestSchedulerListJobsEmpty` | GET /jobs → empty list |
| `TestSchedulerCreateJobValid` | POST /jobs → 201 + job object |
| `TestSchedulerCreateJobDuplicate` | Duplicate name → 409 |
| `TestSchedulerCreateJobInvalidCron` | Bad cron → 400 |
| `TestSchedulerCreateJobShellPathTraversal` | `../../etc/passwd` target → 400 |
| `TestSchedulerCreateJobInvalidAgentRole` | Unknown agent → 400 |
| `TestSchedulerCreateJobPriorityOutOfRange` | Priority 11 / -1 → 400 |
| `TestSchedulerGetJobDetail` | GET /jobs/:name → job + runs key |
| `TestSchedulerGetJobNotFound` | GET /jobs/ghost → 404 |
| `TestSchedulerUpdateJob` | PUT /jobs/:name → updated fields |
| `TestSchedulerDeleteJob` | DELETE /jobs/:name → 204, then 404 |
| `TestSchedulerTriggerJob` | POST /trigger → 200, run status=success |
| `TestSchedulerPauseJob` | POST /pause → enabled=false |
| `TestSchedulerResumeJob` | POST /resume → enabled=true |
| `TestSchedulerListRuns` | GET /runs paginated |
| `TestSchedulerGetRunLog` | GET /runs/:id/log → text/plain |
| `TestSchedulerGetRunLogPruned` | Missing log file → 404 |
| `TestSchedulerUnauthenticated` | No session → 401 (GET) or 403 (mutating, CSRF fires first) |
| `TestSchedulerCSRFEnforcement` | No X-CSRF-Token → 403 |

---

## Milestone 6 — WebSocket events

**File:** `tests/integration/scheduler_ws_test.go`

5 integration tests. Events are collected via a registered hub channel — no
real WebSocket connection required.

| Test | Scenario |
|---|---|
| `TestSchedulerWSJobStarted` | Trigger → `scheduler.job.started` with non-zero run_id |
| `TestSchedulerWSJobCompletedSuccess` | `true` → `scheduler.job.completed` status=success, duration_ms>0 |
| `TestSchedulerWSJobCompletedFailure` | `false` → status=failure |
| `TestSchedulerWSJobCompletedTimeout` | `sleep 60` + TimeoutSec=1 → status=timeout |
| `TestSchedulerWSNoEventForPausedJob` | Paused job → no started event during 200 ms window |

---

## Milestone 7 — Security and sandbox

**File:** `tests/integration/scheduler_security_test.go`

6 integration tests verifying sandbox enforcement and input validation.

| Test | Scenario |
|---|---|
| `TestSchedulerShellPathTraversalRelative` | `../../../etc/passwd` → 400 |
| `TestSchedulerShellPathTraversalAbsolute` | `/etc/passwd` → 400 |
| `TestSchedulerShellPathTraversalSymlink` | Symlink outside root → 400 |
| `TestSchedulerShellEnvironmentIsolation` | `KAOS_TEST_SECRET` not in job env |
| `TestSchedulerShellRunsInProjectRoot` | `pwd` output matches project root |
| `TestSchedulerAgentRoleValidation` | Unknown agent → 400 at creation; job not persisted |
