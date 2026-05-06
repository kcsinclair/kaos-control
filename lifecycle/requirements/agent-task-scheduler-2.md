---
title: Agent and Task Scheduler
type: requirement
status: planning
lineage: agent-task-scheduler
created: "2026-05-05T00:00:00+10:00"
priority: high
parent: lifecycle/ideas/agent-task-scheduler.md
labels:
    - feature
    - agent
    - backend
    - workflow
assignees:
    - role: analyst
      who: agent
---

## Problem

Agent runs (e.g. nightly QA, periodic analysis) and maintenance scripts currently require manual triggering or external cron infrastructure. There is no way to defer agent work until preconditions are met (API rate limits lifted, a dependent job completed), leading to failed runs or idle waiting. Job history is lost on restart, making it difficult to audit what ran and when.

## Goals / Non-goals

### Goals

- Provide a built-in scheduler that can execute agent runs and shell scripts on a one-off, interval, or cron-based schedule.
- Support precondition-gated execution so jobs wait for external conditions before starting.
- Persist job definitions, state, last-run timestamps, and output logs across restarts.
- Expose a UI surface for creating, editing, deleting, viewing, triggering, pausing, and resuming scheduled jobs.
- Keep the feature consistent with the existing lifecycle management patterns (config-driven, project-scoped).

### Non-goals

- Distributed scheduling across multiple kaos-control instances (single-node only).
- Full workflow orchestration (DAGs with fan-out/fan-in); only simple linear preconditions are in scope.
- Real-time streaming of job output in the UI (polling or post-run log retrieval is acceptable for v1).

## Detailed Requirements

### Functional

1. **Job definition schema** — A job has: `name` (unique within project), `schedule` (cron expression, interval duration, or `once` with a datetime), `target` (agent role name or absolute/relative shell script path), `args` (optional map passed to agent or script), `preconditions` (optional list — see below), `enabled` (boolean, default true), `timeout` (duration, default from config).

2. **Schedule types**
   - Cron expression (standard 5-field or extended 6-field with seconds).
   - Interval (`every: 30m`, `every: 4h`).
   - One-off (`at: 2026-05-10T03:00:00+10:00`).

3. **Preconditions** — Each precondition is one of:
   - `after_job: <job-name>` — wait until the named job's last run succeeded.
   - `file_exists: <path>` — wait until a file appears on disk.
   - `http_ok: <url>` — wait until a GET returns 2xx.
   - Custom shell predicate: `shell: <command>` — wait until exit code 0.

   When a scheduled time fires but preconditions are unmet, the scheduler retries with exponential backoff (configurable cap) until the next scheduled time, at which point it resets.

4. **Job execution** — For agent targets, the scheduler invokes the existing agent runner with the specified role and args. For shell targets, it spawns the process in the project root with a timeout. Stdout/stderr are captured to a log entry.

5. **Persistence** — Job definitions may live in `lifecycle/config.yaml` under a `scheduler.jobs` key, or in a dedicated `lifecycle/scheduler.yaml` file. Runtime state (last run time, last status, next scheduled time, run history) is persisted in the SQLite index database.

6. **Run history** — Each execution is recorded with: job name, start time, end time, exit status (success/failure/timeout/skipped), and a reference to captured output. Retain at least the last 50 runs per job (configurable).

7. **API endpoints**
   - `GET /api/scheduler/jobs` — list all jobs with current state.
   - `GET /api/scheduler/jobs/:name` — job detail + recent runs.
   - `POST /api/scheduler/jobs` — create a job.
   - `PUT /api/scheduler/jobs/:name` — update a job.
   - `DELETE /api/scheduler/jobs/:name` — delete a job.
   - `POST /api/scheduler/jobs/:name/trigger` — manually trigger immediately.
   - `POST /api/scheduler/jobs/:name/pause` — pause scheduling.
   - `POST /api/scheduler/jobs/:name/resume` — resume scheduling.
   - `GET /api/scheduler/jobs/:name/runs` — paginated run history.
   - `GET /api/scheduler/jobs/:name/runs/:id/log` — output log for a specific run.

8. **UI** — A "Scheduler" section accessible from the main navigation, showing:
   - List of jobs with name, schedule, last run status, next run time, enabled state.
   - Detail view per job: edit form, upcoming schedule preview, run history table.
   - Actions: create, edit, delete, trigger now, pause/resume.

9. **Lifecycle integration** — When an agent job produces or modifies lifecycle artifacts, the watcher/indexer picks them up normally. The scheduler itself does not need special artifact handling beyond invoking the agent runner.

10. **Startup behaviour** — On server start, load job definitions, reconcile with persisted state, and begin scheduling. Missed one-off jobs (scheduled time in the past) are marked as `skipped`.

### Non-functional

1. **Concurrency** — At most N jobs run concurrently (configurable, default 2). Additional jobs queue in FIFO order.
2. **Reliability** — A job that panics or exceeds its timeout is killed and marked failed; it does not crash the server.
3. **Observability** — Emit structured log lines at info level for job start/complete/fail. Broadcast `scheduler.job.started` and `scheduler.job.completed` WebSocket events.
4. **Resource isolation** — Shell jobs inherit a minimal environment (PATH, HOME, project root). No access to server internals.
5. **Security** — Only authenticated users may create/modify/trigger jobs via API. Shell script paths are validated against the project sandbox (no path traversal).

## Acceptance Criteria

- [ ] A cron-scheduled agent job (e.g. `qa` role, nightly) fires at the correct time and produces expected artifacts.
- [ ] An interval-scheduled shell script runs repeatedly at the configured frequency.
- [ ] A one-off job executes at the specified datetime and does not repeat.
- [ ] A job with `after_job` precondition waits until the dependency succeeds before running.
- [ ] A job that exceeds its timeout is terminated and recorded as failed.
- [ ] Job definitions in `lifecycle/config.yaml` or `lifecycle/scheduler.yaml` are loaded on startup.
- [ ] Jobs created via API are persisted and survive server restart.
- [ ] Run history (last 50) is queryable via API with correct status and timing data.
- [ ] The UI lists jobs, shows next-run time, and allows manual trigger/pause/resume.
- [ ] Concurrent job limit is enforced; excess jobs queue and eventually execute.
- [ ] WebSocket events `scheduler.job.started` and `scheduler.job.completed` are broadcast.
- [ ] Shell script paths are rejected if they escape the project sandbox.
- [ ] Unauthenticated API requests to scheduler endpoints return 401.

## Resolved Questions

- Should job definitions be editable only via UI/API (runtime-only), only via config files (declarative-only), or both with a merge strategy? If both, which takes precedence on conflict?

> Job definitions should be in the database for persistance, using the UI/API and later CLI tools to manage.

- What is the desired log retention policy — time-based, count-based, or size-based? Should old logs be prunable from the UI?

> Lets go with time based for now, 90 days is the default, make that a configuration option.

- Should the scheduler support job priorities beyond FIFO ordering?

> Yes, jobs should have priorities so they can jump the queue, use 1-10 for priority where 1 is the lowest and 10 is the highest.  By default jobs should be created with priority 5.

- Is there a need for job-level notifications (e.g. on failure, send a message) in v1, or is log + WebSocket event sufficient?

> Log and websocket for now.  Ensure the logs contain keywords which are easily searched for.  A future enhancement should generate an event which can be sent using syslog or mqtt to another system.
