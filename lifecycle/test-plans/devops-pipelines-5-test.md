---
title: 'Test Plan: DevOps Pipeline Management'
type: plan-test
status: done
lineage: devops-pipelines
parent: lifecycle/requirements/devops-pipelines-2.md
---

## Overview

Integration and unit tests covering the DevOps pipeline management feature: YAML discovery, API endpoints (listing, execution, cancellation, log retrieval), WebSocket event streaming, role-based access control, concurrency enforcement, timeout handling, and UI interaction. Tests validate the acceptance criteria from the requirement.

Related: [[devops-pipelines]]

## Milestone 1 — Pipeline Discovery Unit Tests

### Description

Test the YAML parser and discovery logic in isolation, covering valid files, malformed files, missing fields, and timeout defaults.

### Files to change

- `internal/devops/discovery_test.go` (new) — Unit tests:
  - Valid pipeline with all fields parses correctly.
  - Pipeline with custom step timeout overrides default 60s.
  - Pipeline missing `name` field is excluded with warning.
  - Pipeline missing `steps` field is excluded with warning.
  - Step missing `command` field causes pipeline exclusion.
  - Invalid YAML syntax causes pipeline exclusion with warning.
  - Empty directory returns no pipelines and no errors.
  - Multiple valid files all discovered; malformed ones excluded.
- `tests/fixtures/devops/` (new directory) — Test fixture YAML files: `valid-build.yaml`, `valid-deploy.yaml`, `missing-name.yaml`, `bad-syntax.yaml`, `custom-timeout.yaml`.

### Acceptance criteria

- [ ] All discovery edge cases have corresponding test cases.
- [ ] Tests use fixture files rather than inline YAML strings for realism.
- [ ] `go test ./internal/devops/ -short` passes.
- [ ] Test coverage for the discovery package exceeds 90%.

## Milestone 2 — Pipeline Listing API Tests

### Description

Integration tests for the `GET /api/p/:project/devops/pipelines` endpoint verifying correct response shape, grouping, malformed file exclusion, and role gating.

### Files to change

- `tests/devops_api_test.go` (new) — Test cases:
  - Authenticated `product-owner` receives pipelines grouped by type.
  - Authenticated `devops` role user receives pipelines.
  - Unauthenticated request returns `401`.
  - User without `product-owner`/`devops` role receives `403`.
  - Response schema matches `{ pipelines: [{ slug, name, type, steps }] }`.
  - Malformed YAML files are excluded from the response.
  - Steps in the response include `name` and `description` but not `command`.
  - Response time is under 200ms with 50 fixture files (NF1).
- `tests/fixtures/devops/` — Additional fixture files to reach 50 for the performance test.

### Acceptance criteria

- [ ] Role-based access control is verified for both allowed and denied roles.
- [ ] Response schema validation is explicit (field presence, types, grouping).
- [ ] Performance assertion validates NF1 requirement.
- [ ] Tests clean up any state they create.

## Milestone 3 — Pipeline Execution & Concurrency Tests

### Description

Integration tests for the `POST .../run` endpoint covering successful execution, failure handling, concurrency control (409), and role gating.

### Files to change

- `tests/devops_run_test.go` (new) — Test cases:
  - `POST .../run` with `product-owner` role returns `run_id` and 202/200.
  - `POST .../run` with `devops` role succeeds.
  - `POST .../run` without appropriate role returns `403`.
  - `POST .../run` on non-existent pipeline returns `404`.
  - `POST .../run` on already-running pipeline returns `409`.
  - After a run completes, the same pipeline can be triggered again (no stale lock).
  - Multiple different pipelines can run concurrently.
- `tests/fixtures/devops/quick-pass.yaml` (new) — Pipeline with steps that succeed quickly (e.g. `echo ok`).
- `tests/fixtures/devops/quick-fail.yaml` (new) — Pipeline with a step that exits non-zero.
- `tests/fixtures/devops/slow-step.yaml` (new) — Pipeline with a `sleep` step for concurrency testing.

### Acceptance criteria

- [ ] Concurrency constraint (one run per pipeline) is enforced and verified.
- [ ] Cross-pipeline concurrency (multiple pipelines simultaneously) is verified.
- [ ] Role gating is tested for both positive and negative cases.
- [ ] Run ID is returned and is a valid unique identifier.

## Milestone 4 — Step Execution, Timeout & Cancellation Tests

### Description

Tests verifying step-level behaviour: sequential execution, non-zero exit handling, timeout enforcement, and cancel functionality.

### Files to change

- `tests/devops_steps_test.go` (new) — Test cases:
  - Steps execute in declared order (verify via output sequence).
  - A step with non-zero exit code stops the run; subsequent steps are not executed.
  - A step exceeding its timeout is killed; run is marked `failed`.
  - Default timeout of 60s applies when not specified (use a shorter override in test config for speed).
  - `POST .../cancel` on an active run stops execution.
  - `POST .../cancel` on a pipeline with no active run returns `404`.
  - Cancelled run's remaining steps are not executed.
- `tests/fixtures/devops/timeout-step.yaml` (new) — Pipeline with a step that has a very short timeout and a command that exceeds it (e.g. `timeout: 1` with `sleep 10`).

### Acceptance criteria

- [ ] Sequential execution order is verified via output content.
- [ ] Non-zero exit code stops the pipeline at the failing step.
- [ ] Timeout kills the step process and marks the run as failed.
- [ ] Cancel stops execution mid-run.
- [ ] No zombie processes remain after timeout or cancel.

## Milestone 5 — WebSocket Event Streaming Tests

### Description

Tests verifying that the correct WebSocket events are broadcast in the correct order during pipeline execution.

### Files to change

- `tests/devops_ws_test.go` (new) — Test cases:
  - `pipeline.run.started` is the first event after triggering a run; contains `run_id`, slug, step count.
  - `pipeline.step.started` is received for each step before its output.
  - `pipeline.step.output` events contain stdout/stderr chunks with step index.
  - `pipeline.step.completed` contains exit code and duration.
  - `pipeline.run.completed` is the last event; contains overall status and duration.
  - Events arrive in correct order: run.started → (step.started → step.output* → step.completed)+ → run.completed.
  - Output latency is < 500ms from command output to WebSocket receipt.
  - On failure, remaining steps do not emit started/completed events.
  - On cancel, a `pipeline.run.completed` event with status `cancelled` is emitted.

### Acceptance criteria

- [ ] Event ordering is strictly validated.
- [ ] All event payloads contain the documented fields.
- [ ] Latency assertion validates the < 500ms requirement.
- [ ] Failure and cancellation scenarios emit correct terminal events.

## Milestone 6 — Run Log Persistence Tests

### Description

Tests verifying that run logs are persisted to disk and readable via the API, both during and after execution.

### Files to change

- `tests/devops_logs_test.go` (new) — Test cases:
  - After a run completes, a log file exists at `~/.kaos-control/devops/<project>/<run_id>.log`.
  - Log file contains valid JSON-lines with all events.
  - `GET /api/p/:project/devops/runs/:run_id` returns the log content.
  - In-progress runs return partial log content via the API.
  - The devops log directory is auto-created on first run.
  - `GET .../runs/:run_id` for a non-existent run returns `404`.

### Acceptance criteria

- [ ] Log persistence is verified by reading the file from disk.
- [ ] API endpoint returns correct content for completed and in-progress runs.
- [ ] Directory auto-creation is verified.
- [ ] Tests clean up log files they create.

## Milestone 7 — UI Integration Tests

### Description

End-to-end tests verifying the DevOps page renders correctly, role gating works in the UI, and run interactions function as expected.

### Files to change

- `tests/devops_ui_test.go` (new) or `web/src/views/__tests__/DevOpsView.spec.ts` (new) — Test cases:
  - DevOps nav item is visible for `product-owner` role.
  - DevOps nav item is visible for `devops` role.
  - DevOps nav item is hidden for other roles.
  - Pipeline cards render grouped by type.
  - Unknown pipeline types get their own column.
  - "Run" button triggers the execution API.
  - "Run" button is disabled during an active run.
  - "Cancel" button appears during active run.
  - Step progress indicators update as WebSocket events arrive.
  - Failed steps display with red styling.
  - Terminal output pane shows streamed output.

### Acceptance criteria

- [ ] Role-based visibility is tested for both allowed and denied roles.
- [ ] Card rendering and grouping matches the requirement.
- [ ] Interactive controls (Run, Cancel) are tested.
- [ ] Real-time updates via WebSocket are verified in the test.
- [ ] Error states (failed steps) are visually distinguishable in test assertions.

## Test Artifact

- `lifecycle/tests/devops-pipelines-test.md` (new) — Test artifact documenting what the test code covers, linked to this plan.
