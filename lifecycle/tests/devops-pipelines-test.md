---
title: DevOps Pipeline Management — Integration & Unit Test Suite
type: test
status: approved
lineage: devops-pipelines
parent: lifecycle/test-plans/devops-pipelines-5-test.md
---

## Overview

This artifact documents the test suite built for the DevOps pipeline management feature. Tests are split across Go integration tests (Go build tag `integration`), Go unit tests, and frontend (Vitest) tests.

## Test Files

### Go Integration Tests (`tests/integration/`)

| File | Milestone | Scenarios |
|---|---|---|
| `devops_helpers_test.go` | — | Shared constants, YAML fixtures, `newDevopsTestEnv`, `waitForRunComplete`, URL helpers |
| `devops_api_test.go` | M2 | Pipeline listing API |
| `devops_run_test.go` | M3 | Pipeline execution & concurrency |
| `devops_steps_test.go` | M4 | Step ordering, failure, timeout, cancel |
| `devops_ws_test.go` | M5 | WebSocket event streaming |
| `devops_logs_test.go` | M6 | Run log persistence |

### Go Unit Tests (`internal/devops/`)

| File | Milestone | Scenarios |
|---|---|---|
| `discovery_test.go` | M1 | Pipeline YAML discovery (pre-existing) |

### Frontend Tests (`tests/web/`)

| File | Milestone | Scenarios |
|---|---|---|
| `DevOpsView.test.ts` | M7 | DevOpsView and PipelineCard component behaviour |

### Fixture Files (`tests/fixtures/devops/`)

- `valid-build.yaml` — Valid two-step build pipeline
- `valid-deploy.yaml` — Valid three-step deploy pipeline
- `missing-name.yaml` — Pipeline missing required `name` field (excluded)
- `bad-syntax.yaml` — Invalid YAML syntax (excluded)
- `custom-timeout.yaml` — Pipeline with custom per-step timeout
- `quick-pass.yaml` — Single-step pipeline that exits 0 immediately
- `quick-fail.yaml` — Single-step pipeline that exits 1
- `slow-step.yaml` — Pipeline with `sleep 30` for concurrency/cancel testing
- `timeout-step.yaml` — Pipeline with `sleep 10` and `timeout: 1s`

## Milestone 1 — Pipeline Discovery Unit Tests

**Status: Pre-existing** (`internal/devops/discovery_test.go`)

Scenarios covered:

- Valid pipeline with all required fields parses correctly
- Custom step `timeout` field overrides the 60s default
- Invalid step timeout format is rejected with a warning
- Pipeline missing `name` is excluded with a warning
- Pipeline missing `type` is excluded with a warning
- Pipeline with empty `steps` array is excluded
- Step missing `command` causes pipeline exclusion
- Invalid YAML syntax causes pipeline exclusion
- Non-existent directory returns empty results with no error
- Mixed directory (valid + invalid) returns only valid pipelines
- Non-YAML files (`.md`, `.sh`) are ignored

## Milestone 2 — Pipeline Listing API Tests

**File:** `tests/integration/devops_api_test.go`

- `TestDevopsListPipelines_ProductOwnerAccess` — `product-owner` role → 200 with pipelines array
- `TestDevopsListPipelines_DevopsRoleAccess` — `devops` role → 200
- `TestDevopsListPipelines_Unauthenticated` — no session cookie → 401
- `TestDevopsListPipelines_ForbiddenRole` — `qa` role → 403 with `error.code=forbidden`
- `TestDevopsListPipelines_ResponseSchema` — validates `slug`, `name`, `type`, `steps[].name`, `steps[].description`; confirms `command` is absent from steps
- `TestDevopsListPipelines_MalformedFilesExcluded` — invalid YAML files are omitted; valid file is returned
- `TestDevopsListPipelines_EmptyDirectory` — empty devops dir → empty pipelines array
- `TestDevopsListPipelines_Performance` — 50 pipeline files → response in < 200ms (NF1)

## Milestone 3 — Pipeline Execution & Concurrency Tests

**File:** `tests/integration/devops_run_test.go`

- `TestDevopsRun_ProductOwnerSucceeds` — `product-owner` → 202 + 16-char hex `run_id`
- `TestDevopsRun_DevopsRoleSucceeds` — `devops` role → 202
- `TestDevopsRun_ForbiddenRole` — `qa` role → 403
- `TestDevopsRun_NotFound` — non-existent slug → 404 + `error.code=not_found`
- `TestDevopsRun_AlreadyRunning` — second trigger for active pipeline → 409 + `error.code=conflict`
- `TestDevopsRun_ReTriggerAfterCompletion` — completed pipeline can be re-triggered; run IDs differ
- `TestDevopsRun_MultiplePipelinesConcurrently` — two different pipelines run simultaneously without blocking

## Milestone 4 — Step Execution, Timeout & Cancellation Tests

**File:** `tests/integration/devops_steps_test.go`

- `TestDevopsSteps_ExecuteInOrder` — `step.started` events arrive in declared step order
- `TestDevopsSteps_NonZeroExitStopsPipeline` — failing step stops pipeline; later step not started; run status=`failed`
- `TestDevopsSteps_TimeoutKillsStep` — step with `timeout: 1s` and `sleep 10` is killed; run status=`failed`
- `TestDevopsSteps_CancelActiveRun` — cancel returns `cancelled=true`; run status=`cancelled`
- `TestDevopsSteps_CancelNoActiveRun` — cancel on idle pipeline → 404 + `error.code=not_found`
- `TestDevopsSteps_CancelledRunSkipsRemainingSteps` — second step does not emit `step.started` after cancel

## Milestone 5 — WebSocket Event Streaming Tests

**File:** `tests/integration/devops_ws_test.go`

Events are captured via `Hub.Register` (in-process channel) — no HTTP WebSocket connection required.

- `TestDevopsWS_RunStartedIsFirst` — `pipeline.run.started` is first event; contains `run_id`, `pipeline`, `project`
- `TestDevopsWS_StepStartedPerStep` — one `pipeline.step.started` per step, in declared order
- `TestDevopsWS_StepOutputContainsText` — `step.output` events have `step`, `step_index`, `text`, `stream` fields
- `TestDevopsWS_StepCompletedHasFields` — `step.completed` has `exit_code`, `duration_seconds`, `status`
- `TestDevopsWS_RunCompletedIsLast` — `pipeline.run.completed` is last event; has `status` and `duration_seconds`
- `TestDevopsWS_EventOrdering` — strict ordering verified: run.started → (step.started → step.output* → step.completed)+ → run.completed
- `TestDevopsWS_FailureSkipsRemainingStepEvents` — failed run: remaining steps emit no `step.started`; status=`failed`
- `TestDevopsWS_CancelEmitsCompletedEvent` — cancelled run emits `run.completed` with status=`cancelled`
- `TestDevopsWS_OutputLatency` — first `step.output` event arrives within 500ms of trigger (NF)

## Milestone 6 — Run Log Persistence Tests

**File:** `tests/integration/devops_logs_test.go`

- `TestDevopsLogs_FileExistsAfterRun` — log file exists at `<dataDir>/devops/testproject/<run_id>.log`
- `TestDevopsLogs_ValidJSONLines` — log is valid JSON-lines; each entry has `time`, `event_type`, `payload`; `run.started` and `run.completed` entries present
- `TestDevopsLogs_APIReturnsContent` — `GET /devops/runs/{run_id}` returns log with `run.started` and `run.completed`
- `TestDevopsLogs_InProgressReturnsPartialContent` — in-progress log already contains `run.started`
- `TestDevopsLogs_DirectoryAutoCreated` — `devops/testproject/` directory is created on first run
- `TestDevopsLogs_NotFoundForUnknownRunID` — unknown `run_id` → 404 + `error.code=not_found`
- `TestDevopsLogs_ContentTypeIsNDJSON` — log API response has `Content-Type: application/x-ndjson`

## Milestone 7 — UI Integration Tests

**File:** `tests/web/DevOpsView.test.ts`

Uses Vitest + `@vue/test-utils` + happy-dom. Network calls and WebSocket are mocked.

**Access control (DevOpsView):**
- `qa` role → access-denied message shown; no pipeline columns rendered
- Empty roles → access-denied message shown
- `product-owner` role → pipeline content rendered (no access-denied)
- `devops` role → pipeline content rendered

**Pipeline column rendering:**
- Multiple pipeline types → one column per type
- Two pipelines of same type → single shared column
- Unknown type (`security`) → its own column
- No pipelines → empty-state message rendered

**PipelineCard — Run button:**
- Idle pipeline: `btn-run` exists, not disabled, text="Run"
- Active run: `btn-cancel` shown; `btn-run` hidden
- Completed run: `btn-run` shown; `btn-cancel` hidden
- Click Run → `runPipeline('testproject', 'build')` called

**PipelineCard — Cancel button:**
- Click Cancel → `cancelPipeline('testproject', 'build')` called

**Run status styling:**
- `failed` run → `pipeline-card--failed` class + `run-status--failed` badge
- `running` run → `pipeline-card--running` class
- `passed` run → `pipeline-card--passed` class

**Step progress display:**
- Active run → `.step-list` rendered
- No active run → `.step-list` absent
