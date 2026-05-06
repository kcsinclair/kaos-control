---
title: "Agent & Task Scheduler — Test Plan"
type: plan-test
status: done
lineage: agent-task-scheduler
parent: lifecycle/requirements/agent-task-scheduler-2.md
created: "2026-05-06T00:00:00+10:00"
assignees:
    - role: test-developer
      who: agent
---

# Agent & Task Scheduler — Test Plan

Defines the integration and unit test strategy for the scheduler subsystem. Tests cover the store layer, schedule evaluation, precondition checking, the scheduler engine, API endpoints, and security boundaries.

Cross-references: [[agent-task-scheduler-3-be]] (backend implementation), [[agent-task-scheduler-4-fe]] (frontend implementation).

---

## Milestone 1 — Store layer tests

### Description
Test the SQLite-backed job and run CRUD operations, including pagination, cascade deletes, and retention pruning.

### Files to change
- `internal/scheduler/store_test.go` (new)

### Test cases
1. **CreateJob** — insert a job and verify all fields are persisted and retrievable.
2. **CreateJob duplicate name** — inserting a job with an existing name returns an error.
3. **GetJob not found** — returns a "not found" error for a non-existent job name.
4. **UpdateJob** — update fields (schedule, priority, enabled) and verify changes persist.
5. **DeleteJob cascades** — deleting a job also removes all its run records.
6. **ListJobs** — returns all jobs, ordered by name.
7. **InsertRun and ListRuns** — insert multiple runs, verify paginated retrieval ordered by `start_time DESC`.
8. **ListRuns pagination** — verify offset/limit returns correct pages.
9. **PruneOldRuns** — insert runs with timestamps older than retention period, prune, verify only recent runs remain.
10. **PruneOldRuns deletes log files** — verify that log files referenced by pruned runs are removed from disk.

### Acceptance criteria
- [ ] All 10 test cases pass.
- [ ] Tests use an in-memory or temp-file SQLite database (no shared state between tests).
- [ ] No test takes longer than 2 seconds.

---

## Milestone 2 — Schedule evaluation tests

### Description
Test the schedule parser and next-fire-time calculator for all three schedule types.

### Files to change
- `internal/scheduler/schedule_test.go` (new)

### Test cases
1. **Cron 5-field** — `0 2 * * *` with `now = 2026-05-06T01:00:00Z` → next fire `2026-05-06T02:00:00Z`.
2. **Cron 5-field past today** — `0 2 * * *` with `now = 2026-05-06T03:00:00Z` → next fire `2026-05-07T02:00:00Z`.
3. **Cron 6-field with seconds** — `30 0 2 * * *` fires at 02:00:30.
4. **Cron day-of-week** — `0 9 * * 1` (Monday) computes the next Monday correctly.
5. **Interval from last run** — `every: 30m`, last run ended at `T01:00:00Z`, now `T01:20:00Z` → next fire `T01:30:00Z`.
6. **Interval no prior run** — first run fires immediately (next fire = now).
7. **Once future** — `at: 2026-06-01T00:00:00Z` with now before → returns the target time.
8. **Once past** — `at: 2026-01-01T00:00:00Z` with now after → returns zero (skip).
9. **Invalid cron expression** — returns a parse error.
10. **Invalid interval** — `every: abc` returns a parse error.

### Acceptance criteria
- [ ] All 10 test cases pass.
- [ ] Edge cases around midnight, month boundaries, and DST transitions are handled (at minimum, no panic).

---

## Milestone 3 — Precondition evaluation tests

### Description
Test each precondition type in isolation using controlled fixtures.

### Files to change
- `internal/scheduler/precondition_test.go` (new)

### Test cases
1. **after_job — dependency succeeded** — last run of named job has status `success` → returns true.
2. **after_job — dependency failed** — last run has status `failure` → returns false.
3. **after_job — dependency never ran** — no runs exist → returns false.
4. **after_job — dependency job does not exist** — returns false (not an error).
5. **file_exists — file present** — create a temp file, verify returns true.
6. **file_exists — file absent** — verify returns false.
7. **file_exists — path traversal** — `../../etc/passwd` is rejected (sandbox violation).
8. **http_ok — 200 response** — use `httptest.NewServer` returning 200 → returns true.
9. **http_ok — 500 response** — returns false.
10. **http_ok — unreachable host** — returns false (with timeout, no hang).
11. **shell — exit 0** — `true` command → returns true.
12. **shell — exit 1** — `false` command → returns false.
13. **shell — timeout** — `sleep 60` with 1-second timeout → returns false.

### Acceptance criteria
- [ ] All 13 test cases pass.
- [ ] HTTP tests use `httptest.NewServer` (no real network calls).
- [ ] Shell tests use trivial commands (`true`, `false`, `sleep`) that work cross-platform on macOS/Linux.
- [ ] No test takes longer than 5 seconds.

---

## Milestone 4 — Scheduler engine tests

### Description
Test the core scheduler loop: job dispatch, concurrency limits, priority ordering, timeout enforcement, and startup reconciliation.

### Files to change
- `internal/scheduler/scheduler_test.go` (new)

### Test cases
1. **Job fires on schedule** — create a job due now, tick the scheduler, verify a run is created with status `success`.
2. **Job does not fire when paused** — create a paused job due now, tick, verify no run is created.
3. **Concurrency limit** — set max=1, trigger two jobs simultaneously, verify only one runs at a time and the second runs after the first completes.
4. **Priority ordering** — enqueue a priority-1 and a priority-10 job, verify the priority-10 job executes first.
5. **TriggerNow** — manually trigger a job that is not yet due, verify it runs immediately.
6. **Timeout enforcement** — create a shell job (`sleep 60`) with a 1-second timeout, verify the run ends with status `timeout`.
7. **Panic recovery** — create a job whose execution panics (simulated), verify the run is marked `failure` and the scheduler continues.
8. **Startup reconciliation — stale running** — insert a run with status `running` into the DB, start the scheduler, verify it is marked `failed`.
9. **Startup reconciliation — missed one-off** — insert a one-off job with a past schedule time, start the scheduler, verify a run is inserted with status `skipped`.
10. **Precondition gating** — create a job with an `after_job` precondition where the dependency has not succeeded, verify the job does not fire until the dependency succeeds.
11. **Agent target dispatch** — create an agent-type job, verify the agent manager's `StartRun` is called with the correct role and args.
12. **Shell target — output capture** — run a shell job that writes to stdout/stderr, verify the log file contains the output.

### Acceptance criteria
- [ ] All 12 test cases pass.
- [ ] Tests use a mock or stub agent manager (no real Claude Code invocations).
- [ ] Tests that involve timing use controllable clocks or short timeouts (no `time.Sleep` longer than 2 seconds).
- [ ] Each test is independent (no shared state, each creates its own scheduler instance).

---

## Milestone 5 — API endpoint tests

### Description
Test all scheduler REST API endpoints for correct behaviour, validation, and auth enforcement.

### Files to change
- `tests/scheduler_api_test.go` (new) — integration tests using `httptest.Server` with the full chi router.

### Test cases
1. **GET /scheduler/jobs** — returns an empty list initially, then populated after creating jobs.
2. **POST /scheduler/jobs — valid** — creates a job with all fields, returns 201 with the job object.
3. **POST /scheduler/jobs — duplicate name** — returns 409 Conflict.
4. **POST /scheduler/jobs — invalid cron** — returns 400 with error message.
5. **POST /scheduler/jobs — shell path traversal** — target `../../etc/passwd` returns 400.
6. **POST /scheduler/jobs — invalid agent role** — returns 400.
7. **POST /scheduler/jobs — priority out of range** — priority 0 or 11 returns 400.
8. **GET /scheduler/jobs/:name** — returns job detail with recent runs.
9. **GET /scheduler/jobs/:name — not found** — returns 404.
10. **PUT /scheduler/jobs/:name** — updates fields, returns 200.
11. **DELETE /scheduler/jobs/:name** — returns 204, job is gone.
12. **POST /scheduler/jobs/:name/trigger** — returns 202, triggers the job.
13. **POST /scheduler/jobs/:name/pause** — returns 200, job is disabled.
14. **POST /scheduler/jobs/:name/resume** — returns 200, job is enabled.
15. **GET /scheduler/jobs/:name/runs** — returns paginated runs.
16. **GET /scheduler/jobs/:name/runs/:id/log** — returns log content.
17. **GET /scheduler/jobs/:name/runs/:id/log — pruned** — returns 404 when log file is missing.
18. **Unauthenticated requests** — all endpoints return 401 without a valid session.
19. **CSRF enforcement** — mutating endpoints without CSRF token return 403.

### Acceptance criteria
- [ ] All 19 test cases pass.
- [ ] Tests start a real HTTP server with the full middleware stack (auth, CSRF, project context).
- [ ] Tests create and tear down a temp project directory and SQLite DB per suite.

---

## Milestone 6 — WebSocket event tests

### Description
Verify that scheduler lifecycle events are broadcast correctly.

### Files to change
- `tests/scheduler_ws_test.go` (new) — integration tests that connect a WebSocket client and verify event delivery.

### Test cases
1. **Job started event** — trigger a job, verify a `scheduler.job.started` event is received with the correct job name and run ID.
2. **Job completed event — success** — verify `scheduler.job.completed` with status `success` and `duration_ms > 0`.
3. **Job completed event — failure** — verify `scheduler.job.completed` with status `failure`.
4. **Job completed event — timeout** — verify `scheduler.job.completed` with status `timeout`.
5. **No event for paused job** — trigger a paused job, verify no event is broadcast.

### Acceptance criteria
- [ ] All 5 test cases pass.
- [ ] Tests use a real WebSocket connection to the test server.
- [ ] Events are received within 5 seconds of job completion (test timeout).

---

## Milestone 7 — Security and sandbox tests

### Description
Verify security boundaries: sandbox enforcement for shell targets and auth requirements.

### Files to change
- `tests/scheduler_security_test.go` (new)

### Test cases
1. **Shell path traversal — relative** — `../../../etc/passwd` is rejected at API and scheduler level.
2. **Shell path traversal — absolute** — `/etc/passwd` is rejected.
3. **Shell path traversal — symlink escape** — a symlink pointing outside the project root is rejected.
4. **Shell environment isolation** — a shell job cannot access server env vars (e.g. `DATABASE_URL` is not in the env).
5. **Shell job runs in project root** — `pwd` output matches the project root path.
6. **Agent role validation** — a job targeting a non-existent agent role fails at creation time.

### Acceptance criteria
- [ ] All 6 test cases pass.
- [ ] Tests create controlled filesystem fixtures (temp dirs, symlinks) for sandbox testing.
- [ ] No test modifies real system files.
