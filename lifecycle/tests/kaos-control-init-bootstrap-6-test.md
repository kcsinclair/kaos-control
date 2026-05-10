---
title: 'Tests: Init Bootstrap — Config Defaults, DevOps Scaffold, and Create Pipeline'
type: test
status: approved
lineage: kaos-control-init-bootstrap
parent: lifecycle/test-plans/kaos-control-init-bootstrap-5-test.md
---

# Tests: Init Bootstrap — Config Defaults, DevOps Scaffold, and Create Pipeline

Companion artifact documenting the integration and unit test coverage added for
[[kaos-control-init-bootstrap]]. Tests implement the scenarios defined in the
test plan at `lifecycle/test-plans/kaos-control-init-bootstrap-5-test.md`.

---

## Coverage Summary

### Backend integration tests (`tests/integration/`)

| File | What it covers |
|---|---|
| `init_devops_scaffold_test.go` | `kaos-control init` creates `devops/` with `.gitkeep`; `devops/sample.yaml` is valid; re-running init is idempotent; pre-existing files are preserved. |
| `init_config_defaults_test.go` | Generated `lifecycle/config.yaml` contains the `idea-capture` agent (driver: inline, write path: lifecycle/ideas), kanban columns (Backlog/In-Progress/Done), dashboard tracked_types, and parses cleanly via `config.LoadProject`. |
| `create_pipeline_api_test.go` | `POST /api/p/{project}/devops/pipelines` — 201 success, 409 duplicate, 400 invalid YAML, 400 missing fields, 400 invalid slug, 401 unauthenticated, 403 forbidden role, created pipeline appears in GET list. |

### Frontend unit tests (`tests/web/`)

| File | What it covers |
|---|---|
| `CreatePipelineDialog.test.ts` | Client-side validation in `CreatePipelineDialog.vue` — valid input calls API; invalid YAML shows error; empty slug blocked; invalid slug pattern blocked. |

---

## Test Matrix

| Acceptance criterion | Test case | File |
|---|---|---|
| `devops/` directory created by init | `TestInit_CreatesDevopsDir` | `init_devops_scaffold_test.go` |
| `devops/.gitkeep` present | `TestInit_CreatesDevopsDir` | `init_devops_scaffold_test.go` |
| `devops/sample.yaml` created with correct name/type/steps | `TestInit_CreatesSamplePipeline` | `init_devops_scaffold_test.go` |
| Init is idempotent (no error, sample unchanged on re-run) | `TestInit_Idempotent_DevopsDir` | `init_devops_scaffold_test.go` |
| Pre-existing devops files preserved | `TestInit_PreservesExistingDevops` | `init_devops_scaffold_test.go` |
| `idea-capture` agent with driver=inline and lifecycle/ideas write path | `TestInit_ConfigHasIdeaCaptureAgent` | `init_config_defaults_test.go` |
| Kanban Backlog → draft, In-Progress → in-development+in-qa, Done → done | `TestInit_ConfigHasKanbanDefaults` | `init_config_defaults_test.go` |
| Dashboard tracked_types includes requirement, idea, defect | `TestInit_ConfigHasDashboardDefaults` | `init_config_defaults_test.go` |
| Generated config loads cleanly | `TestInit_ConfigParsesCleanly` | `init_config_defaults_test.go` |
| POST /devops/pipelines — 201 Created, file on disk | `TestCreatePipeline_Success` | `create_pipeline_api_test.go` |
| Duplicate slug returns 409 Conflict | `TestCreatePipeline_DuplicateSlug` | `create_pipeline_api_test.go` |
| Invalid YAML returns 400 | `TestCreatePipeline_InvalidYAML` | `create_pipeline_api_test.go` |
| Missing name or steps returns 400 | `TestCreatePipeline_MissingRequiredFields` | `create_pipeline_api_test.go` |
| Invalid slug (uppercase, spaces, hyphens at boundaries) returns 400 | `TestCreatePipeline_InvalidSlug` | `create_pipeline_api_test.go` |
| Unauthenticated POST returns 401 | `TestCreatePipeline_Unauthorized` | `create_pipeline_api_test.go` |
| Insufficient role returns 403 | `TestCreatePipeline_Forbidden` | `create_pipeline_api_test.go` |
| Created pipeline appears in GET list | `TestCreatePipeline_AppearsInList` | `create_pipeline_api_test.go` |
| Valid YAML + slug → API called, no error shown | `submits successfully when slug and YAML are both valid` | `CreatePipelineDialog.test.ts` |
| Invalid YAML → error shown, no API call | `shows a YAML parse error and does not call the API for invalid YAML` | `CreatePipelineDialog.test.ts` |
| Empty slug → error shown, no API call | `shows a validation error and does not call the API when the slug is empty` | `CreatePipelineDialog.test.ts` |
| Invalid slug pattern → error shown, no API call | `shows a validation error and does not call the API for an invalid slug pattern` | `CreatePipelineDialog.test.ts` |

---

## Known Issues / Expected Failures

### `TestCreatePipeline_AppearsInList` (expected to fail against current implementation)

The `handleCreatePipeline` handler writes the new pipeline file to
`<project-root>/devops/<slug>.yaml`, while `handleListPipelines` reads from
`<project-root>/lifecycle/devops/`. These are different directories, so a
pipeline created via the API will not appear in the listing response until the
handler's write path is corrected to `lifecycle/devops/<slug>.yaml`.

This test is intentionally written to match the specified behaviour (create →
list shows the result). It will fail until the implementation bug is fixed.

---

## Manual Test Procedures

These scenarios require a running server and browser; automated coverage is
limited to the validation logic.

### MT-1: Create Pipeline dialog — full round-trip

1. Start the server (`make run`) and open the DevOps view in a browser.
2. Log in as a user with `product-owner` or `devops` role.
3. Click "New Pipeline".
4. Enter a valid lowercase slug (e.g. `smoke-test`) and replace the YAML
   skeleton with the minimal valid pipeline below, then click **Create**:
   ```yaml
   name: Smoke Test
   type: build
   steps:
     - name: Echo
       command: echo "smoke test passed"
   ```
5. **Expected**: dialog closes; the new pipeline card appears in the view.
6. Verify `devops/smoke-test.yaml` exists on disk.

### MT-2: Create Pipeline dialog — YAML syntax highlighting

1. Open the Create Pipeline dialog.
2. Deliberately introduce a YAML syntax error (e.g. unclosed bracket `[`).
3. **Expected**: the editor highlights the error in red before the form is
   submitted; clicking **Create** shows the inline error message.

### MT-3: Create Pipeline dialog — duplicate slug

1. Create a pipeline with slug `dup-test`.
2. Open the dialog again and enter the same slug `dup-test` with valid YAML.
3. Click **Create**.
4. **Expected**: the dialog shows "A pipeline with this slug already exists."
   without closing.
