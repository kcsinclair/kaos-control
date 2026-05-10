---
title: 'Test Plan: Init Bootstrap — Config Defaults, DevOps Scaffold, and Create Pipeline'
type: plan-test
status: approved
lineage: kaos-control-init-bootstrap
parent: lifecycle/requirements/kaos-control-init-bootstrap-2.md
---

# Test Plan: Init Bootstrap — Config Defaults, DevOps Scaffold, and Create Pipeline

This plan defines integration tests for [[kaos-control-init-bootstrap]], covering the init scaffold changes, generated config defaults, the create-pipeline API endpoint, and end-to-end UI validation. Tests are implemented in `tests/` with companion test artifacts in `lifecycle/tests/`.

---

## Milestone 1: Init scaffold — `devops/` directory creation

### Description

Test that `kaos-control init` creates the `devops/` directory and sample pipeline, and that re-running init is idempotent.

### Files to change

- `tests/integration/init_devops_scaffold_test.go` — **new file**. Test cases:
  1. **`TestInit_CreatesDevopsDir`** — run `initcmd.Run()` (or shell out to the binary) against a temp directory. Assert `devops/` exists and contains `.gitkeep`.
  2. **`TestInit_CreatesSamplePipeline`** — assert `devops/sample.yaml` exists after init. Parse it with `devops.Discover()` and verify the pipeline has `name: "Sample Pipeline"`, `type: "build"`, and one step.
  3. **`TestInit_Idempotent_DevopsDir`** — run init twice on the same temp dir. Assert no error on second run. Assert `devops/sample.yaml` is unchanged (compare content).
  4. **`TestInit_PreservesExistingDevops`** — create a temp dir, write a custom `devops/custom.yaml`, then run init. Assert `custom.yaml` is untouched and `sample.yaml` is skipped (already exists scenario) or created alongside.

- `lifecycle/tests/kaos-control-init-bootstrap-5-test.md` — companion artifact documenting the test coverage (see Milestone 5).

### Acceptance criteria

- [ ] All four test cases pass with `go test ./tests/integration/ -run TestInit_`.
- [ ] Tests use `t.TempDir()` and do not leave state on disk.
- [ ] Tests run successfully with `make test-unit` (or `-short` flag causes skip if they require the binary).

---

## Milestone 2: Init scaffold — config template defaults

### Description

Test that the generated `lifecycle/config.yaml` contains the required default blocks.

### Files to change

- `tests/integration/init_config_defaults_test.go` — **new file**. Test cases:
  1. **`TestInit_ConfigHasIdeaCaptureAgent`** — run init, read `lifecycle/config.yaml`, unmarshal YAML, assert an agent with `name: "idea-capture"` exists with `driver: "inline"` and `allowed_write_paths` containing `"lifecycle/ideas"`.
  2. **`TestInit_ConfigHasKanbanDefaults`** — assert the `kanban` block exists with at least 3 columns mapping to the expected statuses: Backlog → `[draft]`, In-Progress → contains `[in-development, in-qa]`, Done → contains `[done]`.
  3. **`TestInit_ConfigHasDashboardDefaults`** — assert the `dashboard` block exists with `tracked_types` containing `["requirement", "idea", "defect"]`.
  4. **`TestInit_ConfigParsesCleanly`** — load the generated config through the project's `config.Load()` function and assert no error.

### Acceptance criteria

- [ ] All four test cases pass.
- [ ] The tests validate structure, not exact string matching, to tolerate template formatting changes.
- [ ] `go build` passes (config loader types are compatible).

---

## Milestone 3: Create Pipeline API endpoint

### Description

Test the `POST /api/p/{project}/devops/pipelines` endpoint for success, validation failures, duplicates, and access control.

### Files to change

- `tests/integration/create_pipeline_api_test.go` — **new file**. Test cases using `httptest.Server` or the project's test harness:
  1. **`TestCreatePipeline_Success`** — POST valid JSON `{"slug":"ci","definition":"name: CI\ntype: build\nsteps:\n  - name: test\n    command: go test ./...\n"}`. Assert `201 Created`, response contains `slug: "ci"`, and `devops/ci.yaml` exists on disk.
  2. **`TestCreatePipeline_DuplicateSlug`** — create a pipeline, then POST the same slug again. Assert `409 Conflict`.
  3. **`TestCreatePipeline_InvalidYAML`** — POST `{"slug":"bad","definition":"not: valid: yaml: ["}`. Assert `400 Bad Request` with a descriptive error.
  4. **`TestCreatePipeline_MissingRequiredFields`** — POST valid YAML that lacks `name` or `steps`. Assert `400 Bad Request`.
  5. **`TestCreatePipeline_InvalidSlug`** — POST with slug `"My Pipeline"` (uppercase, spaces). Assert `400 Bad Request`.
  6. **`TestCreatePipeline_Unauthorized`** — POST without authentication. Assert `401`.
  7. **`TestCreatePipeline_Forbidden`** — POST as a user with only `analyst` role. Assert `403`.
  8. **`TestCreatePipeline_AppearsInList`** — create a pipeline, then GET `/devops/pipelines`. Assert the new pipeline appears in the response.

### Acceptance criteria

- [ ] All eight test cases pass.
- [ ] Tests clean up created files (use `t.TempDir()` for the project root).
- [ ] Tests do not depend on external services or running instances.

---

## Milestone 4: YAML editor validation (frontend unit tests)

### Description

If the project has a frontend test setup (Vitest or similar), add unit tests for the `CreatePipelineDialog` validation logic. If no frontend test framework is configured, document the manual test procedures instead.

### Files to change

- `web/src/components/devops/__tests__/CreatePipelineDialog.test.ts` — **new file** (if Vitest is available). Test cases:
  1. **Valid YAML passes validation** — provide well-formed pipeline YAML, assert no error shown.
  2. **Invalid YAML shows error** — provide malformed YAML, assert error message is displayed.
  3. **Empty slug shows error** — leave slug blank, assert validation prevents submission.
  4. **Invalid slug pattern shows error** — enter `"MY PIPELINE"`, assert validation error.

- If no frontend test framework exists, document these as manual test cases in the companion test artifact.

### Acceptance criteria

- [ ] Validation tests pass (or manual procedures are documented).
- [ ] Client-side validation prevents malformed requests from reaching the API.

---

## Milestone 5: Companion test artifact

### Description

Write the lifecycle test artifact documenting all test coverage for this feature.

### Files to change

- `lifecycle/tests/kaos-control-init-bootstrap-5-test.md` — **new file**. Frontmatter:
  ```yaml
  title: 'Tests: Init Bootstrap — Config Defaults, DevOps Scaffold, and Create Pipeline'
  type: test
  status: draft
  lineage: kaos-control-init-bootstrap
  parent: lifecycle/requirements/kaos-control-init-bootstrap-2.md
  ```
  Body sections:
  - **Coverage Summary** — list of test files and what each covers.
  - **Test Matrix** — table mapping each acceptance criterion from the requirement to the test case that verifies it.
  - **Manual Test Procedures** — for any UI flows not covered by automated tests (e.g., "Create Pipeline" dialog interaction, YAML editor syntax highlighting).

### Acceptance criteria

- [ ] The test artifact exists and links back to the requirement.
- [ ] Every acceptance criterion from the requirement is mapped to at least one test case or manual procedure.
- [ ] The artifact passes frontmatter validation.

---

## Cross-references

- Backend implementation: [[kaos-control-init-bootstrap-3-be]] — Milestones 1–4 produce the code tested here.
- Frontend implementation: [[kaos-control-init-bootstrap-4-fe]] — Milestones 2–5 produce the UI tested here.
- Requirement: [[kaos-control-init-bootstrap]] — acceptance criteria are the source of truth for test coverage.
