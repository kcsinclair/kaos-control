---
title: "Agent & Task Scheduler — Backend Plan"
type: plan-backend
status: done
lineage: agent-task-scheduler
parent: lifecycle/requirements/agent-task-scheduler-2.md
created: "2026-05-06T00:00:00+10:00"
assignees:
    - role: backend-developer
      who: agent
---

# Agent & Task Scheduler — Backend Plan

Implements the scheduler subsystem: job persistence in SQLite, a tick-based scheduler goroutine, precondition evaluation, concurrency control with priority queuing, and a full REST API surface. Integrates with the existing agent runner for agent-type targets and spawns sandboxed processes for shell targets.

Cross-references: [[agent-task-scheduler-4-fe]] (frontend consumes the API and WS events), [[agent-task-scheduler-5-test]] (integration tests).

---

## Milestone 1 — Domain types and SQLite schema

### Description
Define the core Go types for jobs, runs, and preconditions. Extend the SQLite index with `scheduler_jobs` and `scheduler_runs` tables. These tables use `CREATE TABLE IF NOT EXISTS` (like `agent_runs`) so they survive schema-version rebuilds of cache tables.

### Files to change
- `internal/scheduler/types.go` (new) — `Job`, `Run`, `Precondition`, `ScheduleSpec`, `RunStatus` types.
- `internal/index/index.go` — add `CREATE TABLE IF NOT EXISTS scheduler_jobs (...)` and `scheduler_runs (...)` to the init block that runs alongside `agent_runs` and `events`.

### Schema

```sql
CREATE TABLE IF NOT EXISTS scheduler_jobs (
    name        TEXT PRIMARY KEY,
    target_type TEXT NOT NULL,          -- 'agent' | 'shell'
    target      TEXT NOT NULL,          -- agent role name or script path
    args_json   TEXT,                   -- JSON map
    schedule    TEXT NOT NULL,          -- serialised ScheduleSpec
    preconditions_json TEXT,            -- JSON array of Precondition
    enabled     INTEGER NOT NULL DEFAULT 1,
    priority    INTEGER NOT NULL DEFAULT 5,
    timeout_sec INTEGER NOT NULL,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS scheduler_runs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    job_name    TEXT NOT NULL REFERENCES scheduler_jobs(name) ON DELETE CASCADE,
    start_time  TEXT NOT NULL,
    end_time    TEXT,
    status      TEXT NOT NULL,          -- 'running','success','failure','timeout','skipped'
    log_path    TEXT,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_runs_job_start ON scheduler_runs(job_name, start_time DESC);
```

### Acceptance criteria
- [ ] `Job` and `Run` structs compile and round-trip through JSON.
- [ ] Tables are created on fresh DB open and survive schema-version cache rebuild.
- [ ] `scheduler_runs` rows cascade-delete when a job is removed.

---

## Milestone 2 — Job CRUD repository

### Description
Implement a `Store` that wraps SQLite access for job and run records, including time-based log retention pruning.

### Files to change
- `internal/scheduler/store.go` (new) — `Store` struct with methods: `ListJobs`, `GetJob`, `CreateJob`, `UpdateJob`, `DeleteJob`, `InsertRun`, `UpdateRun`, `ListRuns` (paginated), `GetRun`, `PruneOldRuns(retentionDays int)`.

### Acceptance criteria
- [ ] All CRUD operations work correctly with SQLite.
- [ ] `ListRuns` supports offset/limit pagination ordered by `start_time DESC`.
- [ ] `PruneOldRuns` deletes runs older than the configured retention (default 90 days) and their associated log files.
- [ ] Concurrent calls do not corrupt state (single-writer SQLite + WAL).

---

## Milestone 3 — Schedule evaluation and preconditions

### Description
Implement schedule parsing (cron, interval, one-off) and a `NextFireTime` calculator. Implement precondition evaluators with exponential backoff.

### Files to change
- `internal/scheduler/schedule.go` (new) — `ScheduleSpec` parsing, `NextFireTime(now, lastRun) time.Time`, cron via a minimal 5/6-field parser or a small dependency.
- `internal/scheduler/precondition.go` (new) — `Evaluate(ctx, Precondition, Store) (bool, error)` for each type: `after_job`, `file_exists`, `http_ok`, `shell`.

### Acceptance criteria
- [ ] Cron expressions (5-field and 6-field) produce correct next-fire times.
- [ ] Interval schedules (`every: 30m`) compute next fire from last run end time.
- [ ] One-off schedules return the target time if in the future, or zero if past.
- [ ] `after_job` returns true only when the named job's most recent run has status `success`.
- [ ] `file_exists` checks the path through the sandbox resolver.
- [ ] `http_ok` performs a GET with a 10-second timeout; returns true on 2xx.
- [ ] `shell` runs the command in the project root with a 30-second timeout; returns true on exit 0.

---

## Milestone 4 — Scheduler engine

### Description
Implement the core scheduler goroutine that ticks every 15 seconds, evaluates which jobs are due, checks preconditions, and dispatches execution through a priority-aware work queue with configurable concurrency.

### Files to change
- `internal/scheduler/scheduler.go` (new) — `Scheduler` struct with `Start(ctx)`, `Stop()`, `TriggerNow(jobName)`, `Pause(jobName)`, `Resume(jobName)`.
- `internal/scheduler/queue.go` (new) — priority queue (heap-based) that orders pending jobs by priority (10 highest) then FIFO within the same priority.

### Design
- On each tick: iterate enabled jobs, compute next fire time, if `<= now` and job not already running/queued → evaluate preconditions. If met, enqueue. If unmet, schedule a backoff retry (capped at configurable max, e.g. 5 min). If next scheduled time arrives before preconditions are met, reset backoff.
- Worker pool: `N` goroutines (default 2, from `App.MaxConcurrentSchedulerJobs` config) pull from the priority queue.
- On startup: load all jobs, reconcile — mark stale `running` entries as `failed`, mark missed one-off jobs as `skipped`.
- Each worker: inserts a `Run` record, broadcasts `scheduler.job.started`, invokes the target (agent or shell), captures output to a log file under `<dataDir>/<project>/scheduler-runs/<jobName>/<runID>.log`, updates the run record, broadcasts `scheduler.job.completed`.
- Agent targets: call `agentManager.StartRun()` with the configured role and args (reusing the existing agent runner pipeline).
- Shell targets: spawn via `exec.CommandContext` in the project root, with a minimal env (`PATH`, `HOME`, `PROJECT_ROOT`), timeout from job config, stdout/stderr captured to the log file.

### Acceptance criteria
- [ ] Scheduler starts on project open and stops on project close.
- [ ] Jobs fire within one tick (15 s) of their scheduled time.
- [ ] Concurrency limit is enforced; excess jobs wait in the priority queue.
- [ ] Higher-priority jobs (10) execute before lower-priority (1) when queued simultaneously.
- [ ] `TriggerNow` bypasses schedule and preconditions, respects concurrency limit.
- [ ] `Pause`/`Resume` toggle the `enabled` flag and persist it.
- [ ] Missed one-off jobs on startup are marked `skipped`.
- [ ] Stale `running` records on startup are marked `failed`.
- [ ] A job exceeding its timeout is killed (process killed / context cancelled) and recorded as `timeout`.
- [ ] Panics in job execution are recovered and recorded as `failure`.

---

## Milestone 5 — Configuration

### Description
Add scheduler-related config fields to app and project config.

### Files to change
- `internal/config/config.go` — add to `App`: `MaxConcurrentSchedulerJobs int` (default 2), `SchedulerRunRetentionDays int` (default 90). Add to `Project` or as a recognised top-level key: `scheduler.default_timeout` (duration, default 30m).

### Acceptance criteria
- [ ] Config fields load from YAML with correct defaults when omitted.
- [ ] `MaxConcurrentSchedulerJobs` controls the worker pool size.
- [ ] `SchedulerRunRetentionDays` is used by `PruneOldRuns` on startup.

---

## Milestone 6 — REST API endpoints

### Description
Add the scheduler API routes under `/api/p/{project}/scheduler/`.

### Files to change
- `internal/http/scheduler.go` (new) — handler functions for all endpoints.
- `internal/http/server.go` — register the scheduler route group inside the project sub-router, behind `requireAuth`.

### Endpoints
| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | `/scheduler/jobs` | `listJobs` | Returns jobs with computed `next_run_at` |
| GET | `/scheduler/jobs/{name}` | `getJob` | Job detail + last 10 runs |
| POST | `/scheduler/jobs` | `createJob` | Validates name uniqueness, schedule, sandbox for shell paths |
| PUT | `/scheduler/jobs/{name}` | `updateJob` | Partial update; re-evaluates schedule |
| DELETE | `/scheduler/jobs/{name}` | `deleteJob` | Cascade-deletes runs and logs |
| POST | `/scheduler/jobs/{name}/trigger` | `triggerJob` | Calls `scheduler.TriggerNow` |
| POST | `/scheduler/jobs/{name}/pause` | `pauseJob` | Sets `enabled=false` |
| POST | `/scheduler/jobs/{name}/resume` | `resumeJob` | Sets `enabled=true` |
| GET | `/scheduler/jobs/{name}/runs` | `listRuns` | Paginated: `?page=1&per_page=20` |
| GET | `/scheduler/jobs/{name}/runs/{id}/log` | `getRunLog` | Streams log file content |

### Validation
- Job `name`: alphanumeric + hyphens, 1–64 chars.
- Shell `target`: must pass `sandbox.Resolve(projectRoot, target)`.
- Agent `target`: must match a configured agent role name.
- `schedule`: must parse as a valid cron/interval/one-off.
- `priority`: integer 1–10, default 5.

### Acceptance criteria
- [ ] All 10 endpoints return correct status codes (200, 201, 204, 400, 401, 404).
- [ ] Unauthenticated requests return 401.
- [ ] Shell paths that escape the sandbox return 400.
- [ ] Invalid cron expressions return 400 with a descriptive error message.
- [ ] `listJobs` includes computed `next_run_at` for each enabled job.
- [ ] `getRunLog` returns `404` if the log file has been pruned.

---

## Milestone 7 — WebSocket events

### Description
Broadcast scheduler lifecycle events through the existing project hub.

### Files to change
- `internal/scheduler/scheduler.go` — emit events via `hub.Broadcast`.
- `internal/hub/hub.go` — no changes needed (generic event dispatch), but document new event types.

### Event payloads
```json
{ "type": "scheduler.job.started",   "payload": { "job": "<name>", "run_id": 42 } }
{ "type": "scheduler.job.completed", "payload": { "job": "<name>", "run_id": 42, "status": "success", "duration_ms": 12340 } }
```

### Acceptance criteria
- [ ] `scheduler.job.started` is broadcast when a job begins execution.
- [ ] `scheduler.job.completed` is broadcast when a job finishes (any terminal status).
- [ ] Events contain the job name, run ID, and (for completed) status and duration.

---

## Milestone 8 — Project wiring and startup

### Description
Wire the scheduler into the project lifecycle: create it in `project.Open()`, start it alongside the watcher and lock reaper, stop it in `project.Close()`.

### Files to change
- `internal/project/project.go` — add `Scheduler *scheduler.Scheduler` field. In `Open()`: create scheduler (passing store, agent manager, hub, config), call `scheduler.Start()`. In `Close()`: call `scheduler.Stop()`. Run `store.PruneOldRuns()` on startup.

### Acceptance criteria
- [ ] Scheduler starts automatically when a project is opened.
- [ ] Scheduler stops gracefully (in-flight jobs finish or are cancelled after timeout) on project close.
- [ ] Old runs beyond retention are pruned on startup.
- [ ] Log lines at info level: `scheduler started`, `scheduler stopped`, and per-job `job started`, `job completed`, `job failed`, `job timed out` with keywords `SCHEDULER`, `JOB_START`, `JOB_SUCCESS`, `JOB_FAIL`, `JOB_TIMEOUT` for searchability.
