---
title: "Test Plan — Make Pipelines Editable"
type: plan-test
status: draft
lineage: pipeline-editing
parent: lifecycle/requirements/pipeline-editing-2.md
created: "2026-05-15T00:00:00+10:00"
labels:
    - test
    - devops
release: KC-Release1
assignees:
    - role: test-developer
      who: agent
---

# Test Plan — Make Pipelines Editable

Integration and acceptance tests for the "Make Pipelines Editable" requirement ([[pipeline-editing]]). Validates the backend endpoints from [[pipeline-editing-3-be]] and the frontend behaviour from [[pipeline-editing-4-fe]].

All backend tests follow the patterns established in `tests/integration/create_pipeline_api_test.go` and `tests/integration/devops_run_test.go`, using the `newDevopsTestEnv` helper.

---

## Milestone 1 — GET single-pipeline endpoint tests

Test the `GET /api/p/{project}/devops/pipelines/{slug}` endpoint.

### Files to change

- `tests/integration/get_pipeline_api_test.go` — new file.

### Test cases

| # | Test name | Setup | Action | Expected |
|---|-----------|-------|--------|----------|
| 1 | `TestGetPipeline_Success` | Create `quick-pass` pipeline on disk | GET `.../pipelines/quick-pass` as admin | `200`, body is verbatim YAML content matching the file |
| 2 | `TestGetPipeline_NotFound` | No pipeline created | GET `.../pipelines/nonexistent` as admin | `404 Not Found` |
| 3 | `TestGetPipeline_InvalidSlug` | — | GET `.../pipelines/../../etc/passwd` as admin | `400 Bad Request` |
| 4 | `TestGetPipeline_Unauthorized` | Create pipeline | GET without auth cookie | `401 Unauthorized` |
| 5 | `TestGetPipeline_Forbidden` | Create pipeline | GET as `qa@test.local` (qa role only) | `403 Forbidden` |

### Acceptance criteria

- [ ] All five test cases pass.
- [ ] The response body for a successful GET matches the file on disk byte-for-byte.

---

## Milestone 2 — PUT update-pipeline endpoint tests

Test the `PUT /api/p/{project}/devops/pipelines/{slug}` endpoint.

### Files to change

- `tests/integration/update_pipeline_api_test.go` — new file.

### Test cases

| # | Test name | Setup | Action | Expected |
|---|-----------|-------|--------|----------|
| 1 | `TestUpdatePipeline_Success` | Create `quick-pass` pipeline | PUT with valid modified YAML (changed name and added a step) as admin | `200`, response has updated name and step_count; file on disk matches submitted YAML |
| 2 | `TestUpdatePipeline_NotFound` | No pipeline | PUT `.../pipelines/nonexistent` | `404 Not Found` |
| 3 | `TestUpdatePipeline_InvalidYAML` | Create pipeline | PUT with `definition: "{{{"` | `400 Bad Request` with descriptive message |
| 4 | `TestUpdatePipeline_MissingRequiredFields` | Create pipeline | PUT with YAML missing `name` field | `400 Bad Request` |
| 5 | `TestUpdatePipeline_MissingStepCommand` | Create pipeline | PUT with a step that has no `command` | `400 Bad Request` |
| 6 | `TestUpdatePipeline_InvalidTimeout` | Create pipeline | PUT with `timeout: "not-a-duration"` | `400 Bad Request` |
| 7 | `TestUpdatePipeline_ConflictWhileRunning` | Create pipeline, start a run on it | PUT as admin | `409 Conflict` |
| 8 | `TestUpdatePipeline_ConflictOtherPipelineRunning` | Create two pipelines, start a run on pipeline A | PUT to pipeline B as admin | `409 Conflict` (global lock per resolved question #2) |
| 9 | `TestUpdatePipeline_SuccessAfterRunCompletes` | Create pipeline, start & wait for run to complete | PUT with valid YAML | `200 OK` |
| 10 | `TestUpdatePipeline_Unauthorized` | Create pipeline | PUT without auth cookie | `401 Unauthorized` |
| 11 | `TestUpdatePipeline_Forbidden` | Create pipeline | PUT as `qa@test.local` | `403 Forbidden` |
| 12 | `TestUpdatePipeline_PreservesSlug` | Create `my-pipe` | PUT with changed name | `200`; filename on disk is still `my-pipe.yaml` |
| 13 | `TestUpdatePipeline_AtomicWrite` | Create pipeline | PUT with valid YAML; verify no `.tmp` files left behind in the devops directory | `200`; no temp files remain |

### Acceptance criteria

- [ ] All thirteen test cases pass.
- [ ] File on disk is verified byte-for-byte after successful updates.
- [ ] Conflict tests confirm the global active-run guard works for both same-pipeline and cross-pipeline scenarios.

---

## Milestone 3 — Atomic write integrity test

Verify that a failure during the write process does not corrupt the existing pipeline file (NF4).

### Files to change

- `tests/integration/update_pipeline_api_test.go` — add to the same file.

### Test cases

| # | Test name | Setup | Action | Expected |
|---|-----------|-------|--------|----------|
| 1 | `TestUpdatePipeline_FileIntegrity_ConcurrentRead` | Create pipeline | PUT valid YAML, then immediately GET the same slug | GET returns the new content, not a partial write |
| 2 | `TestUpdatePipeline_OriginalPreservedOnValidationFailure` | Create pipeline with known content | PUT with invalid YAML | `400`; GET still returns the original content unchanged |

### Acceptance criteria

- [ ] Concurrent read after write returns consistent, complete content.
- [ ] Failed validation does not alter the file on disk.

---

## Milestone 4 — Regression tests for pipeline creation

Ensure the new endpoints and route changes do not break existing pipeline creation or run functionality.

### Files to change

- `tests/integration/create_pipeline_api_test.go` — run existing tests (no changes needed; just confirm they still pass).
- `tests/integration/devops_run_test.go` — run existing tests.

### Test cases

| # | Test name | Action | Expected |
|---|-----------|--------|----------|
| 1 | Existing `TestCreatePipeline_*` suite | Run all create tests | All pass without modification |
| 2 | Existing `TestRunPipeline_*` suite | Run all run tests | All pass without modification |
| 3 | `TestCreateThenEditThenRun` | Create pipeline → edit it (change step command) → run it | Run executes the *updated* command, not the original |

### Acceptance criteria

- [ ] No regressions in the existing create and run test suites.
- [ ] A pipeline edited via PUT executes the updated definition when run.

---

## Milestone 5 — Frontend UI acceptance tests

Browser-level tests (or component tests if the project uses Playwright/Vitest) for the edit dialog flow.

### Files to change

- `tests/integration/pipeline_edit_ui_test.go` or equivalent Playwright/Vitest test file — new file, following whatever UI test patterns exist in the project.
- `lifecycle/tests/pipeline-editing-test.md` — test artifact describing coverage.

### Test cases

| # | Scenario | Steps | Expected |
|---|----------|-------|----------|
| 1 | Edit button visibility | Log in as admin; navigate to DevOps view | Edit (pencil) button visible on each pipeline card |
| 2 | Edit button hidden for non-devops role | Log in as qa user | No edit button visible |
| 3 | Edit button disabled during run | Start a pipeline run; observe edit buttons | All edit buttons disabled with tooltip |
| 4 | Edit dialog loads current YAML | Click edit on a pipeline | Dialog opens; YAML editor contains current file content |
| 5 | Slug shown read-only | Open edit dialog | Slug displayed as label, not editable |
| 6 | Save disabled until change | Open edit dialog without modifying | Save button is disabled |
| 7 | Client validation — invalid YAML | Type invalid YAML in editor | Save remains disabled or shows validation error |
| 8 | Successful save updates card | Edit pipeline name, save | Dialog closes; card shows new name without page reload |
| 9 | Step removal confirmation | Remove a step from YAML, click save | Confirmation prompt appears; cancelling does not save |
| 10 | Network error keeps dialog open | Simulate network failure on save | Error shown inline; dialog remains open with edits preserved |
| 11 | 409 conflict inline error | Start a run, then attempt to save edit | Inline error about active run; dialog stays open |

### Acceptance criteria

- [ ] All eleven UI scenarios pass.
- [ ] Edit and create dialogs are visually consistent (NF1) — verified by visual comparison or screenshot diff.
- [ ] No regressions to pipeline creation, run, or log streaming flows.
