---
title: 'Backend Plan: DevOps Pipeline Management'
type: plan-backend
status: done
lineage: devops-pipelines
parent: lifecycle/requirements/devops-pipelines-2.md
---

## Overview

Implement the backend infrastructure for DevOps pipeline management: YAML pipeline discovery, execution engine with step-level streaming, run log persistence, cancellation support, concurrency control, and role-gated API endpoints. Pipelines are defined as YAML files in `lifecycle/devops/` and executed on the host machine with real-time output broadcast via WebSocket.

Related: [[devops-pipelines]]

## Milestone 1 — Pipeline Definition Parser & Discovery

### Description

Create an `internal/devops/` package responsible for parsing pipeline YAML files from `lifecycle/devops/` and returning structured pipeline definitions. Malformed files are logged as warnings and excluded.

### Files to change

- `internal/devops/pipeline.go` (new) — Define types:
  ```go
  type Pipeline struct {
      Slug        string
      Name        string
      Type        string
      Steps       []Step
  }
  type Step struct {
      Name        string
      Description string
      Command     string
      Timeout     time.Duration // default 60s
  }
  ```
- `internal/devops/discovery.go` (new) — `Discover(dir string) ([]Pipeline, []error)` reads all `*.yaml` files in the directory, parses them, validates required fields, applies default timeout of 60s per step, and returns valid pipelines plus a list of parse warnings.
- `internal/devops/discovery_test.go` (new) — Unit tests for valid files, malformed files, missing fields, and custom timeout parsing.

### Acceptance criteria

- [ ] Valid YAML files with `name`, `type`, and `steps` (each step having `name` and `command`) parse successfully.
- [ ] A `timeout` field on a step overrides the 60s default.
- [ ] Files missing required fields are excluded and produce a warning error.
- [ ] Files with invalid YAML syntax are excluded and produce a warning error.
- [ ] `go build ./...` and `go vet ./...` pass.

## Milestone 2 — Pipeline Listing API Endpoint

### Description

Expose `GET /api/projects/:id/devops/pipelines` that scans `lifecycle/devops/` and returns all valid pipelines grouped by type. Access is restricted to users with the `product-owner` or `devops` role.

### Files to change

- `internal/http/devops.go` (new) — Handler `handleListPipelines` calls `devops.Discover()`, groups by type, returns JSON response: `{ "pipelines": [{ "slug", "name", "type", "steps": [{"name", "description"}] }] }`.
- `internal/http/router.go` — Register `GET /api/p/{project}/devops/pipelines` route with role middleware restricting to `product-owner` and `devops`.

### Acceptance criteria

- [ ] `GET /api/p/:project/devops/pipelines` returns pipelines grouped by type with correct schema.
- [ ] Response includes step count and step metadata (name, description) but not commands.
- [ ] Malformed files do not appear in the listing.
- [ ] Users without `product-owner` or `devops` role receive `403 Forbidden`.
- [ ] Responds within 200ms for up to 50 YAML files (NF1).

## Milestone 3 — Pipeline Execution Engine

### Description

Build the execution engine that runs pipeline steps sequentially, tracks state, enforces per-step timeouts, and broadcasts progress via WebSocket. Runs are identified by a unique `run_id`.

### Files to change

- `internal/devops/runner.go` (new) — `Runner` struct managing active runs. Methods:
  - `Start(pipeline Pipeline, projectDir string, hub *hub.Hub, projectID string) (string, error)` — returns `run_id`, launches goroutine.
  - `Cancel(runID string) error` — sends cancel signal to an active run.
  - `IsRunning(slug string) bool` — checks if a pipeline is currently executing.
  - Internal: execute steps via `exec.CommandContext` with `sh -c`, enforce timeout via `context.WithTimeout`, stream stdout/stderr line-by-line to the hub.
- `internal/devops/run_state.go` (new) — `RunState` struct tracking per-step status (`pending`, `running`, `passed`, `failed`, `cancelled`), start/end times, exit codes.
- `internal/devops/events.go` (new) — Event type constants and payload structs for `pipeline.run.started`, `pipeline.step.started`, `pipeline.step.output`, `pipeline.step.completed`, `pipeline.run.completed`.

### Acceptance criteria

- [ ] Steps execute sequentially in the project's working directory.
- [ ] A step exceeding its timeout is killed and marked `failed`; subsequent steps are skipped.
- [ ] A step with non-zero exit code stops the run; subsequent steps remain `pending`.
- [ ] `pipeline.step.output` events stream stdout/stderr chunks with < 500ms latency.
- [ ] All five WebSocket event types are broadcast in the correct order.
- [ ] A failed or cancelled run does not crash the server or affect other active runs (NF3).
- [ ] All runs log start/end/exit-code at INFO level (NF4).
- [ ] No shell injection vectors: commands come from YAML on disk, no request data is interpolated (NF2).

## Milestone 4 — Run Trigger & Cancel API Endpoints

### Description

Expose endpoints to trigger and cancel pipeline runs with role gating and concurrency control.

### Files to change

- `internal/http/devops.go` — Add handlers:
  - `handleRunPipeline` for `POST /api/p/{project}/devops/pipelines/{slug}/run` — validates role, checks not already running (409), calls `Runner.Start()`, returns `{ "run_id": "..." }`.
  - `handleCancelPipeline` for `POST /api/p/{project}/devops/pipelines/{slug}/cancel` — validates role, calls `Runner.Cancel()`, returns 200 or 404 if no active run.
- `internal/http/router.go` — Register both routes with `product-owner`/`devops` role middleware.

### Acceptance criteria

- [ ] `POST .../run` with `product-owner` or `devops` role starts execution and returns a `run_id`.
- [ ] `POST .../run` without appropriate role returns `403`.
- [ ] `POST .../run` on an already-running pipeline returns `409 Conflict`.
- [ ] `POST .../cancel` stops the active run; steps in progress are killed, state is `cancelled`.
- [ ] `POST .../cancel` on a pipeline with no active run returns `404`.

## Milestone 5 — Run Log Persistence

### Description

Persist pipeline run logs to `~/.kaos-control/devops/<project-name>/` so they survive server restarts and can be reviewed post-mortem. Logs are also readable while the run is in progress.

### Files to change

- `internal/devops/logger.go` (new) — `RunLogger` writes run output to a file at `~/.kaos-control/devops/<project-name>/<run_id>.log`. Provides:
  - `WriteEvent(event)` — appends structured JSON-lines for each event.
  - `ReadLog(runID string) ([]byte, error)` — reads the full log file.
  - `ListRuns(projectName string) ([]RunSummary, error)` — lists past runs with metadata.
- `internal/devops/runner.go` — Integrate `RunLogger`: write events as they occur.
- `internal/http/devops.go` — Add handler `handleGetRunLog` for `GET /api/p/{project}/devops/runs/{run_id}` returning the log contents (works for active and completed runs).
- `internal/http/router.go` — Register the run log route.

### Acceptance criteria

- [ ] Run logs are written to `~/.kaos-control/devops/<project-name>/<run_id>.log` as JSON-lines.
- [ ] Logs for in-progress runs can be read via the API (partial content returned).
- [ ] Completed run logs persist across server restarts.
- [ ] `GET /api/p/:project/devops/runs/:run_id` returns the log with correct content.
- [ ] The devops directory is created automatically if it doesn't exist.

## Milestone 6 — Role Configuration Update

### Description

Add the `devops` role to the project configuration system so it can be assigned to users and checked by middleware.

### Files to change

- `internal/config/config.go` — Add `devops` to the known roles list/constants.
- `lifecycle/config.yaml` — Add `devops` role definition with appropriate description.

### Acceptance criteria

- [ ] The `devops` role is recognised by the auth/role middleware.
- [ ] Users assigned the `devops` role can access DevOps endpoints.
- [ ] Existing roles (`product-owner`, `analyst`, etc.) are unaffected.
