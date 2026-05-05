---
title: DevOps Pipeline Management
type: requirement
status: planning
lineage: devops-pipelines
priority: high
parent: lifecycle/ideas/devops-pipelines.md
labels:
    - feature
    - backend
    - frontend
    - workflow
    - v1
---

# DevOps Pipeline Management — Requirements

## Problem

The Product Owner currently has no way to define, trigger, or monitor build/deploy/release pipelines from within the Innovation Maker UI. Operations like building the binary, deploying a release, or running migrations require direct terminal access and offer no visibility to non-technical stakeholders. There is no standard format for capturing pipeline definitions alongside the lifecycle artifacts they relate to.

## Goals / Non-goals

### Goals

- Provide a declarative YAML format for pipeline definitions stored under `lifecycle/devops/`.
- Support an extensible pipeline type vocabulary starting with `build`, `deploy`, and `release`.
- Expose a DevOps page in the SPA that discovers, groups, and displays pipelines as actionable cards.
- Allow the Product Owner to trigger a pipeline run and observe step-by-step progress in real time.
- Stream execution output via WebSocket using the existing hub infrastructure.
- Track per-run step state (pending, running, passed, failed) so errors are surfaced inline.
- Gate pipeline execution to the `product-owner` role.

### Non-goals

- Scheduled/cron-triggered pipeline runs (future; relates to [[agent-task-scheduler]]).
- Pipeline editing or creation through the UI (files are authored on disk or by agents).
- Parameterised pipelines (environment variables, runtime inputs) — may be added later.
- Remote execution or distributed runners; pipelines execute on the host machine.
- Persistent run history beyond the current server session (no database storage of run logs in v1).

## Detailed Requirements

### Functional

#### F1 — Pipeline Definition Format

- Each pipeline is a single YAML file in `lifecycle/devops/`.
- Required top-level fields:
  - `name` (string) — human-readable pipeline name.
  - `type` (string) — one of `build`, `deploy`, `release`, or any future value (open vocabulary).
  - `steps` (ordered list) — each step contains:
    - `name` (string) — short label for the step.
    - `description` (string, optional) — explains what the step does.
    - `command` (string) — shell command to execute.
- The file's basename (without `.yaml`) serves as the pipeline's slug identifier.
- Malformed files are reported as warnings in the server log and excluded from the UI listing.

#### F2 — Discovery & Listing API

- `GET /api/projects/:id/devops/pipelines` returns all valid pipeline definitions found in `lifecycle/devops/`, grouped by type.
- The endpoint performs a fresh scan of the directory on each call (no caching required; directory is expected to be small).
- Response shape: `{ pipelines: [{ slug, name, type, steps: [{name, description}] }] }`.

#### F3 — Pipeline Execution API

- `POST /api/projects/:id/devops/pipelines/:slug/run` triggers execution of the named pipeline.
- Only users with the `product-owner` role may invoke this endpoint; others receive `403 Forbidden`.
- The server executes each step sequentially in the project's working directory.
- If a step exits non-zero the run is marked `failed` and subsequent steps are skipped.
- The endpoint returns immediately with a `run_id`; progress is delivered over WebSocket.

#### F4 — Real-time Output Streaming

- Pipeline execution broadcasts events on the project's WebSocket channel using event types:
  - `pipeline.run.started` — includes `run_id`, pipeline slug, total step count.
  - `pipeline.step.started` — includes `run_id`, step index, step name.
  - `pipeline.step.output` — includes `run_id`, step index, stdout/stderr chunk.
  - `pipeline.step.completed` — includes `run_id`, step index, exit code, duration.
  - `pipeline.run.completed` — includes `run_id`, overall status (`passed` | `failed`), duration.
- Output chunks are sent as they arrive (unbuffered streaming), not batched.

#### F5 — DevOps UI Page

- A new "DevOps" entry appears in the left navigation menu (below existing entries).
- The page displays pipelines grouped into columns by type (Build, Deploy, Release; additional types get their own column dynamically).
- Each pipeline renders as a card showing: name, step count, and a "Run" button.
- When a run is active, the card expands to show step progress: each step displays its state icon (pending/running/passed/failed) and streams command output in a scrollable terminal-style pane.
- Only one run per pipeline may be active at a time; the Run button is disabled while a run is in progress.
- Error output (non-zero exit) is highlighted visually (red border or background).

#### F6 — Concurrency

- Multiple pipelines may run concurrently (they are independent).
- A single pipeline cannot have two simultaneous runs; the API returns `409 Conflict` if re-triggered while active.

### Non-functional

- **NF1 — Latency**: Pipeline listing must respond within 200 ms for directories containing up to 50 YAML files.
- **NF2 — Security**: Commands execute with the server process's permissions. No shell injection vectors may be introduced through pipeline YAML content (commands are passed to `sh -c` as a single string; no interpolation of user-controlled request data into commands).
- **NF3 — Isolation**: A failed pipeline run must not crash the server or affect other active runs.
- **NF4 — Observability**: All pipeline runs log start/end/exit-code at INFO level with the pipeline slug and run ID.

## Acceptance Criteria

- [ ] A valid `lifecycle/devops/example-build.yaml` file with type `build` and 2+ steps is discovered by the listing endpoint.
- [ ] `GET /api/projects/:id/devops/pipelines` returns pipelines grouped by type with correct schema.
- [ ] Malformed YAML files do not appear in the listing and produce a server log warning.
- [ ] `POST .../run` with `product-owner` role starts execution and returns a `run_id`.
- [ ] `POST .../run` without `product-owner` role returns `403`.
- [ ] `POST .../run` on an already-running pipeline returns `409`.
- [ ] WebSocket receives `pipeline.run.started`, per-step events, and `pipeline.run.completed` in order.
- [ ] Stdout/stderr from each step streams to the WebSocket client in real time (< 500 ms latency per chunk).
- [ ] A step with a non-zero exit code causes the run to stop; subsequent steps show as `pending`/skipped.
- [ ] The DevOps page renders pipeline cards grouped by type.
- [ ] Clicking "Run" triggers execution and the UI reflects live step progress.
- [ ] The Run button is disabled while a pipeline is already executing.
- [ ] An unknown pipeline type (e.g. `migrate`) is rendered in its own column without code changes.
- [ ] Related: [[devops-pipelines]], [[agent-task-scheduler]]

## Questions

1. Should completed run logs be persisted to disk (e.g. `lifecycle/devops/.runs/`) for post-mortem review, or is ephemeral in-memory state sufficient for v1?

> Yes, the run logs should be stored in ~/.kaos-control/devops/<project name> e.g. ~/.kaos-control/devops/kaos-control
> It should also be possible to view the pipeline logs while they are running.

2. Should pipeline definitions support a `timeout` field per step or per pipeline to prevent hung commands?

> Yes, per step with the default being 60 seconds, if not explicitly stated in the config use the default.

3. Is there a need for a "cancel" action to abort a running pipeline mid-execution?

> Yes, this is a good idea.

4. Should the DevOps page be visible (read-only) to all roles, or hidden entirely from non-product-owner users?

> We should add a new role called devops.  This page should only be visible to the product-owner and devops roles.
