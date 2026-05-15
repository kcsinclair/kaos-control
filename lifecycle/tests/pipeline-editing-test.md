---
title: "Test Suite — Make Pipelines Editable"
type: test
status: in-qa
lineage: pipeline-editing
parent: lifecycle/test-plans/pipeline-editing-5-test.md
created: "2026-05-15T00:00:00+10:00"
---

# Test Suite — Make Pipelines Editable

Integration tests for the "Make Pipelines Editable" feature. Covers the GET and PUT pipeline endpoints from the backend plan, validation logic, access control, atomic write integrity, and a create-edit-run regression scenario.

Milestone 5 (frontend UI tests) is not implemented — the project has no Playwright or browser-level test infrastructure. All coverage is at the HTTP integration level.

---

## Files

- `tests/integration/get_pipeline_api_test.go`
- `tests/integration/update_pipeline_api_test.go`

---

## Milestone 1 — GET single-pipeline endpoint

File: `tests/integration/get_pipeline_api_test.go`

| Test | Scenario |
|------|----------|
| `TestGetPipeline_Success` | 200 OK; body is verbatim YAML; Content-Type is text/yaml |
| `TestGetPipeline_NotFound` | 404 with error code `not_found` for unknown slug |
| `TestGetPipeline_InvalidSlug` | 400 with error code `bad_request` for slug containing path-traversal characters (`..etc..passwd`) |
| `TestGetPipeline_Unauthorized` | 401 for unauthenticated request (no session cookie) |
| `TestGetPipeline_Forbidden` | 403 for `qa@test.local` (qa role only) |

---

## Milestones 2 & 3 — PUT update-pipeline endpoint and atomic write integrity

File: `tests/integration/update_pipeline_api_test.go`

| Test | Scenario |
|------|----------|
| `TestUpdatePipeline_Success` | 200; response has updated name and step_count; disk file matches submitted YAML byte-for-byte |
| `TestUpdatePipeline_NotFound` | 404 for non-existent slug |
| `TestUpdatePipeline_InvalidYAML` | 400 for syntactically invalid YAML |
| `TestUpdatePipeline_MissingRequiredFields` | 400 when `name` is absent |
| `TestUpdatePipeline_MissingStepCommand` | 400 when a step has no `command` |
| `TestUpdatePipeline_InvalidTimeout` | 400 for unparseable timeout value |
| `TestUpdatePipeline_ConflictWhileRunning` | 409 when the target pipeline is running |
| `TestUpdatePipeline_ConflictOtherPipelineRunning` | 409 when a different pipeline is running (global lock) |
| `TestUpdatePipeline_SuccessAfterRunCompletes` | 200 once a run finishes (no stale lock) |
| `TestUpdatePipeline_Unauthorized` | 401 for unauthenticated PUT |
| `TestUpdatePipeline_Forbidden` | 403 for `qa@test.local` |
| `TestUpdatePipeline_PreservesSlug` | 200; slug and filename unchanged after name edit |
| `TestUpdatePipeline_AtomicWrite` | 200; no `.tmp*` files left in devops directory |
| `TestUpdatePipeline_FileIntegrity_ConcurrentRead` | GET immediately after PUT returns new content |
| `TestUpdatePipeline_OriginalPreservedOnValidationFailure` | Failed PUT leaves original file intact; GET returns original |

---

## Milestone 4 — Regression

File: `tests/integration/update_pipeline_api_test.go`

| Test | Scenario |
|------|----------|
| `TestCreateThenEditThenRun` | Create pipeline with `echo original` → edit via PUT to `echo updated` → run executes updated command; NDJSON log contains "updated" not "original" |

---

## Milestone 5 — Frontend UI (not implemented)

No browser-level or component test infrastructure exists in this project. The eleven UI scenarios from the test plan (edit button visibility, YAML editor loading, slug read-only display, etc.) require Playwright or a similar framework. These are deferred.
