---
title: "Test Plan: Local-Model Operability and UI Error Surfacing"
type: plan-test
status: draft
lineage: local-model-operability
parent: lifecycle/requirements/local-model-operability-2.md
created: "2026-08-25T08:49:06+10:00"
---

# Test Plan: Local-Model Operability and UI Error Surfacing

Parent: [[local-model-operability-2]].

This plan defines unit tests, integration tests, and UI verification targets for all acceptance criteria and non-functional requirements of the [[local-model-operability-2]] requirement (Workstream 3 of the [[open-provider-support]] epic). Backend implementation is in [[local-model-operability-3-be]] and frontend implementation is in [[local-model-operability-4-fe]].

---

## Milestone 1 — Availability Check & Error Classification Unit Tests

### Description

Implement unit tests verifying model availability preflight probing, structured error taxonomy classification, remediation step generation, and strict secret masking (FR-2, FR-3, NFR-1, NFR-2).

### Files to change

- **`internal/agent/openai_preflight_test.go`**:
  - `TestCheckModelAvailability_Success`: verifies that when `/v1/models` returns a model list containing the requested model, the check succeeds immediately.
  - `TestCheckModelAvailability_NotFound`: verifies that when `/v1/models` does not contain the requested model, `ErrModelNotFound` is returned.
  - `TestCheckModelAvailability_EndpointUnreachable`: verifies that a connection failure or offline port returns `ErrEndpointUnreachable`.
  - `TestCheckModelAvailability_Timeout`: verifies that a hanging endpoint is aborted strictly within 3 seconds by the context deadline (NFR-1).
- **`internal/agent/errors_test.go`** (new file):
  - `TestClassifyRunError_Taxonomy`: tests accurate classification of all structured error reasons:
    - `tools_unsupported`
    - `model_not_found`
    - `model_unloaded`
    - `endpoint_unreachable`
    - `context_window_exceeded`
    - `turn_token_ceiling`
    - `max_iterations_reached`
    - `auth_error`
    - `timeout`
  - `TestClassifyRunError_Remediation`: asserts each classified error generates ordered, actionable remediation steps.
  - `TestClassifyRunError_SecretsMasking`: asserts that API keys, bearer tokens, and custom auth headers are sanitized and masked as `"***"` in `error_details` and `remediation` outputs (NFR-2, `[[standards/secrets-handling]]`).

### Acceptance criteria

- [ ] All preflight availability check cases (success, missing model, offline endpoint, timeout) pass.
- [ ] Error classification accurately maps all 9 failure reasons and generates actionable remediation steps.
- [ ] Secret credentials never appear in plain text in any diagnostic output or error details.
- [ ] `go test ./internal/agent/ -run "TestCheckModelAvailability|TestClassifyRunError"` passes.

---

## Milestone 2 — Manager Fast-Fail & Lineage Lock Release Integration Tests

### Description

Implement integration tests ensuring that preflight failures abort runs fast (< 3 seconds) without holding lineage locks or mutating lifecycle artifacts on disk or in git (FR-2, NFR-1, NFR-3).

### Files to change

- **`internal/agent/manager_test.go`**:
  - `TestManager_StartRun_ModelNotFound_FastFail`:
    - Setup an agent configured with an `openai-compatible` provider and a nonexistent model.
    - Call `Manager.StartRun`.
    - Assert that `StartRun` returns an error in < 3 seconds.
    - Assert that the lineage lock is immediately free and can be acquired by another run.
    - Assert that the target artifact status is unchanged and no git commits were generated (NFR-3).
    - Assert that the database `agent_runs` table records the failure with `failure_reason: model_not_found`.
  - `TestManager_StartRun_EndpointUnreachable_FastFail`:
    - Setup an agent pointing to an unreachable localhost port (e.g. `127.0.0.1:59999`).
    - Assert immediate failure with `failure_reason: endpoint_unreachable` and clean lock release.

### Acceptance criteria

- [ ] Fast-fail executes within 3 seconds on invalid model or unreachable server.
- [ ] Lineage lock is released immediately on preflight failure.
- [ ] Zero artifact corruption or uncommitted git changes occur (NFR-3).
- [ ] `go test ./internal/agent/ -run TestManager_StartRun` passes.

---

## Milestone 3 — Warmup State & TTFT Progress Broadcast Integration Tests

### Description

Implement integration tests simulating lazy-loaded local inference servers to verify warmup event broadcasting and TTFT latency tracking (FR-2, FR-5).

### Files to change

- **`internal/agent/openai_compatible_test.go`**:
  - `TestOpenAIDriver_WarmupProgress_Emitted`:
    - Mock HTTP server delays the first SSE token chunk by 6 seconds.
    - Assert that at 5 seconds elapsed, the driver emits a progress event with `stage: "warming_up"` and `message: "Awaiting first token (model may be warming up)..."`.
    - When the first chunk arrives, assert that `Run.OnTTFT` is invoked and stage transitions to `"generating"`.
  - `TestOpenAIDriver_LazyLoadTimeout`:
    - Mock HTTP server stalls indefinitely before streaming tokens.
    - Assert that the driver terminates cleanly upon hitting the loading timeout with `model_unloaded` / `timeout`.

### Acceptance criteria

- [ ] Driver emits a `warming_up` progress event when TTFT exceeds 5 seconds.
- [ ] TTFT is recorded when the first content token arrives.
- [ ] Run terminates cleanly if loading timeout is exceeded.
- [ ] `go test ./internal/agent/ -run "TestOpenAIDriver_Warmup|TestOpenAIDriver_LazyLoadTimeout"` passes.

---

## Milestone 4 — Prompt Template Rendering & Scaffolding Tests

### Description

Verify that local-model tuned prompt templates and scaffolding files render correctly and conform to token length and structural constraints (FR-1).

### Files to change

- **`internal/config/config_test.go`**:
  - Test validation of local prompt configurations in `lifecycle/config.yaml`.
- **`internal/initcmd/init_test.go`**:
  - `TestInit_ConfigTemplate_LocalModelPrompts`:
    - Render `config.yaml.tmpl` with test project metadata.
    - Assert generated YAML parses successfully.
    - Assert all agent roles (`analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`, `tech-writer`) have valid prompt templates.
    - Assert each prompt template is concise (< 1200 tokens) and contains explicit few-shot frontmatter examples and mandatory section checklists.

### Acceptance criteria

- [ ] `config.yaml.tmpl` renders valid YAML without syntax errors.
- [ ] Prompt templates for all 6 roles contain required few-shot schemas and section checklists.
- [ ] Token count per prompt template is bounded under 1200 tokens.
- [ ] `go test ./internal/config/... ./internal/initcmd/...` passes.

---

## Milestone 5 — Frontend Store & Component Tests

### Description

Implement Vitest unit tests for the updated Pinia stores, `RunFailureBanner.vue`, and warmup visual indicators in `AgentRunningBanner.vue` and `AgentsRunsView.vue` (FR-3, FR-4, FR-5, NFR-4).

### Files to change

- **`tests/web/RunFailureBanner.localModels.test.ts`** (new file):
  - Test rendering each failure reason (`tools_unsupported`, `model_not_found`, `model_unloaded`, `endpoint_unreachable`, `context_window_exceeded`, `turn_token_ceiling`, `max_iterations_reached`, `auth_error`, `timeout`).
  - Assert appropriate heading and explanatory body text are displayed.
  - Assert numbered remediation steps with inline `<code>` blocks are rendered.
  - Assert direct navigation links to `/settings/providers` and `/settings/agents` are present for relevant errors.
  - Assert `role="alert"` is present.
  - Assert no masked secret values are leaked in rendered text.
- **`tests/web/AgentRunningBanner.warmup.test.ts`** (new file):
  - Test that when active job is in `model_loading` state or `is_warming_up` is true, the animated "Warming up model weights..." badge renders.
  - Test that when job transitions to active generating, the warmup badge disappears.
- **`tests/web/agentsStore.localModelFailure.test.ts`** (new file):
  - Test that `agent.failed` WebSocket events containing `failure_reason`, `remediation`, and `error_details` correctly update the stored `AgentRunRow`.

### Acceptance criteria

- [ ] All 9 failure codes are verified in `RunFailureBanner.vue` tests.
- [ ] Remediation steps and settings navigation links render properly.
- [ ] Warmup visual badge displays conditionally in `AgentRunningBanner.vue`.
- [ ] Store state correctly updates on failure and warmup events.
- [ ] `cd web && pnpm test` passes.

---

## Milestone 6 — End-to-End Regression & Secret Hygiene Verification

### Description

Run the full verification suite across Go backend packages and Vue frontend modules, ensuring complete backwards compatibility and zero regression on existing cloud-model workflows.

### Files to verify

- `internal/agent/...`
- `internal/config/...`
- `internal/http/...`
- `internal/index/...`
- `web/src/...`

### Acceptance criteria

- [ ] `make lint` passes (go vet + staticcheck).
- [ ] `make test-unit` passes.
- [ ] `make build-web` passes without TypeScript or build errors.
- [ ] No API keys, credentials, or secret headers are exposed anywhere in test output or logs (`[[standards/secrets-handling]]`).
