---
title: "Backend Plan: Local-Model Operability and UI Error Surfacing"
type: plan-backend
status: approved
lineage: local-model-operability
parent: lifecycle/requirements/local-model-operability-2.md
created: "2026-08-25T08:49:06+10:00"
---

# Backend Plan: Local-Model Operability and UI Error Surfacing

Parent: [[local-model-operability-2]].

This plan covers the Go backend implementation for FR-1, FR-2, FR-3, FR-6, and NFR-1 through NFR-3 of the [[local-model-operability-2]] requirement (Workstream 3 of the [[open-provider-support]] epic). Frontend changes are defined in [[local-model-operability-4-fe]] and test coverage in [[local-model-operability-5-test]].

---

## Milestone 1 — Local-Model Tuned Prompt Templates & Scaffolding Presets

### Description

Author concise, deterministic prompt templates optimized for local models (e.g. 8B–30B parameters running via GGUF/llama.cpp or Ollama). Local models exhibit high failure rates when subjected to frontier multi-phase prose instructions. This milestone introduces compact prompt templates (< 1200 tokens) with explicit single-step execution ordering, rigid frontmatter schemas, concrete few-shot examples, and strict section headings across standard agent roles (FR-1):

- **`analyst` (Requirements Analyst):**
  - Single-step brief: read idea at `{target_path}` -> read `lifecycle/architecture/architecture-summary.md` -> write `lifecycle/requirements/<slug>-2.md` -> output completion summary.
  - Strict frontmatter schema with concrete few-shot example (`title`, `type: requirement`, `status: draft`, `lineage`, `parent`, `created`).
  - Mandatory section headings: `## Problem`, `## Goals / Non-goals`, `## Detailed Requirements`, `## Acceptance Criteria`, `## Open Questions`.
- **`analyst` (Planning Analyst):**
  - Ordered three-artifact sequence: backend plan (`-be.md`), frontend plan (`-fe.md`), and test plan (`-test.md`).
  - Strict milestone templates with description, files to change, and explicit acceptance criteria.
- **`backend-developer` & `frontend-developer`:**
  - Milestone-by-milestone execution instructions with explicit tool invocation sequence.
  - Rigid scoping rules reinforcing `allowed_write_paths`.
- **`test-developer` & `qa`:**
  - Standardized defect artifact templates (`lifecycle/defects/`) with exact `## Reproduction Steps`, `## Expected Behaviour`, `## Actual Behaviour` headings.
- **`tech-writer`:**
  - Concise documentation and feature artifact (`lifecycle/features/`) generation.

Update the project scaffolding template (`internal/initcmd/templates/config.yaml.tmpl`) and project configuration (`lifecycle/config.yaml`) to provide local-tuned template presets and documentation.

### Files to change

- **`internal/initcmd/templates/config.yaml.tmpl`**:
  - Add tuned local-model prompt template variants and instructions for configuring local provider agents.
  - Update default prompt templates with streamlined instructions and few-shot frontmatter examples.
- **`lifecycle/config.yaml`**:
  - Provide local-tuned prompt templates for agents configured with local providers.
- **`internal/agent/prompt_defaults.go`** (new file):
  - Define package-level constants for role-specific fallback prompts optimized for local models.

### Acceptance criteria

- [ ] Local-model prompt templates for `analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`, and `tech-writer` are authored with concise token footprints (< 1200 tokens).
- [ ] Every prompt template contains explicit step sequences and concrete YAML frontmatter blocks.
- [ ] Scaffolding template (`config.yaml.tmpl`) and `lifecycle/config.yaml` include local-tuned template configurations.
- [ ] `go test ./internal/config/... ./internal/initcmd/...` passes.

---

## Milestone 2 — Proactive Model Availability Probing & Fast-Fail Preflight

### Description

Implement proactive model availability checking in the Go backend before starting an agent run (FR-2, NFR-1, NFR-3). When targeting local inference servers or cloud gateways, dispatching a run to a nonexistent model or an unreachable endpoint currently causes hung states or unhandled crashes after acquiring lineage locks.

This milestone introduces a fast-fail preflight probe that queries the provider's `/v1/models` endpoint with a strict 3-second timeout before lock acquisition in `Manager.StartRun`. If the model is not found or the endpoint is unreachable, the run aborts immediately with a structured error, avoiding lock contention and saving compute tokens.

### Files to change

- **`internal/agent/openai_preflight.go`**:
  - Implement `CheckModelAvailability(ctx context.Context, client *http.Client, provider config.Provider, model string) (bool, error)`:
    - Issues a `GET <base_url>/v1/models` request with a 3-second timeout.
    - Parses model listings and verifies whether `model` exists in the provider's returned models list.
    - Returns `ErrModelNotFound` if absent, or `ErrEndpointUnreachable` if connection fails.
- **`internal/agent/agent.go`**:
  - In `Manager.StartRun`:
    - For agents using `driver: openai-compatible` (or any provider-backed driver), invoke `CheckModelAvailability` prior to acquiring lineage locks (`m.locks.Acquire`).
    - If the check fails:
      - Do not acquire locks and do not mutate artifact status on disk/git (NFR-3).
      - Record the run row with `status: failed`, `failure_reason: model_not_found` or `endpoint_unreachable`.
      - Broadcast `agent.failed` event with structured remediation details.
      - Return an informative error immediately within 3 seconds (NFR-1).

### Acceptance criteria

- [ ] `CheckModelAvailability` verifies model existence against `/v1/models` in < 3 seconds.
- [ ] Missing models fail immediately with `model_not_found` without acquiring lineage locks.
- [ ] Offline or unreachable providers fail fast with `endpoint_unreachable`.
- [ ] No lifecycle markdown files or git histories are mutated on preflight failure.
- [ ] `go test ./internal/agent/ -run "TestCheckModelAvailability|TestStartRun_Preflight"` passes.

---

## Milestone 3 — Model Warmup & Lazy-Loading State Detection

### Description

Local inference servers (such as `llama-server` or Ollama) often lazy-load multi-gigabyte GGUF weights on the initial request, causing an unannounced pause of 30–120 seconds before token generation begins.

Implement live warmup detection and status broadcasting (FR-2, FR-5):
1. **Resident Status Detection:** If the provider `/v1/models` endpoint returns model loading metadata (e.g. llama.cpp model state `unloaded` or `loading`), `Manager.StartRun` broadcasts an `agent.status` event with `state: "model_loading"` and `details: "Loading model weights into memory..."`.
2. **TTFT Progress Emission:** In `OpenAICompatibleDriver`, track elapsed time prior to Time To First Token (TTFT). If TTFT exceeds 5 seconds, the driver emits a progress update: `{"type": "status", "stage": "warming_up", "message": "Awaiting first token (model may be warming up)..."}`.
3. **Dedicated Loading Timeout:** Apply a dedicated configurable loading timeout (default 60s) for the initial token response before marking the run as timed out.

### Files to change

- **`internal/agent/openai_compatible.go`**:
  - Add timer ticker in `openAIProcess` worker goroutine tracking time before first SSE token.
  - If 5 seconds elapse without tokens, emit `ProgressEvent` with `stage: "warming_up"`.
  - On first stream chunk, record TTFT via `run.OnTTFT(ms)` and transition stage to `"generating"`.
- **`internal/agent/openai_preflight.go`**:
  - Extract optional provider load state from `/v1/models` or `/props` responses if available.
- **`internal/agent/agent.go`**:
  - Forward `model_loading` and `warming_up` events via `m.hub.Broadcast` to connected clients.

### Acceptance criteria

- [ ] When TTFT exceeds 5 seconds, a `warming_up` status progress event is emitted.
- [ ] TTFT is accurately recorded when the first content token arrives.
- [ ] Model loading state transitions cleanly to active generation once tokens stream.
- [ ] `go test ./internal/agent/ -run "TestOpenAI_Warmup"` passes.

---

## Milestone 4 — Structured Error Taxonomy & Event Enrichment

### Description

Establish a comprehensive structured error taxonomy for all agent runs (FR-3, NFR-2) and enrich the `agent.failed` WebSocket broadcast and SQLite `agent_runs` table with actionable remediation instructions.

Categorize run failures into standard codes:
1. `tools_unsupported`: Model silently dropped or explicitly rejected tool definitions (FR-5b preflight failure).
2. `model_not_found`: Model missing from provider `/v1/models` (HTTP 404 or preflight missing).
3. `model_unloaded`: Upstream returned HTTP 503 / memory allocation failure during load.
4. `endpoint_unreachable`: TCP connection refused, DNS failure, or network timeout to `base_url`.
5. `context_window_exceeded`: Model context length exceeded during prompt generation or multi-turn loop.
6. `turn_token_ceiling`: Per-turn generation token limit reached with empty tool calls.
7. `max_iterations_reached`: Agent hit `max_tool_iterations` (default 25) without completing task.
8. `auth_error`: HTTP 401/403 unauthorized.
9. `timeout`: Run exceeded `timeout_minutes`.

Enrich `index.AgentRunRow` and the `agent.failed` WebSocket broadcast with `failure_reason`, `error_details`, and `remediation` steps. Mask all secrets and API tokens as `"***"` to strictly uphold `[[standards/secrets-handling]]`.

### Files to change

- **`internal/agent/errors.go`** (new file):
  - Define failure reason constants:
    ```go
    const (
        FailureReasonToolsUnsupported      = "tools_unsupported"
        FailureReasonModelNotFound         = "model_not_found"
        FailureReasonModelUnloaded         = "model_unloaded"
        FailureReasonEndpointUnreachable   = "endpoint_unreachable"
        FailureReasonContextWindowExceeded = "context_window_exceeded"
        FailureReasonTurnTokenCeiling      = "turn_token_ceiling"
        FailureReasonMaxIterationsReached  = "max_iterations_reached"
        FailureReasonAuthError             = "auth_error"
        FailureReasonTimeout               = "timeout"
    )
    ```
  - Implement `ClassifyRunError(err error, stderr string, details map[string]any) (reason string, remediation []string, errorDetails map[string]any)`:
    - Matches errors against known sentinel errors, HTTP status codes, and stderr patterns.
    - Generates human-readable, numbered remediation steps.
    - Sanitizes `errorDetails` to ensure API keys and bearer tokens are masked as `"***"`.
- **`internal/index/index.go`**:
  - Update `AgentRunRow` struct:
    - Add `FailureReason *string` (`json:"failure_reason,omitempty"`)
    - Add `Remediation []string` (`json:"remediation,omitempty"`)
    - Add `ErrorDetails map[string]any` (`json:"error_details,omitempty"`)
  - Update `ensureAgentRunsTable()` migrations:
    - `ALTER TABLE agent_runs ADD COLUMN failure_reason TEXT`
    - `ALTER TABLE agent_runs ADD COLUMN remediation_json TEXT`
    - `ALTER TABLE agent_runs ADD COLUMN error_details_json TEXT`
  - Update `InsertAgentRun`, `UpdateAgentRun`, and query scanners to serialize and deserialize the new fields.
- **`internal/agent/agent.go`**:
  - In `supervise` and `killAndFail`:
    - Classify error using `ClassifyRunError`.
    - Populate `row.FailureReason`, `row.Remediation`, and `row.ErrorDetails`.
    - Include `failure_reason`, `remediation`, and `error_details` in `failedPayload` sent to `m.hub.Broadcast`.

### Acceptance criteria

- [ ] All failure codes (`tools_unsupported`, `model_not_found`, `model_unloaded`, `endpoint_unreachable`, `context_window_exceeded`, `turn_token_ceiling`, `max_iterations_reached`, `auth_error`, `timeout`) are correctly classified.
- [ ] `AgentRunRow` persists `failure_reason`, `remediation_json`, and `error_details_json` in SQLite.
- [ ] `agent.failed` WebSocket broadcast includes `failure_reason`, `remediation`, and `error_details`.
- [ ] No API keys, credentials, or sensitive headers are exposed in payloads or logs (`[[standards/secrets-handling]]`).
- [ ] `go test ./internal/agent/... ./internal/index/...` passes.

---

## Milestone 5 — Local LLM Feature Documentation Generalization

### Description

Update and generalize the standing feature documentation in `lifecycle/features/` (FR-6). Migrate `lifecycle/features/ollama-local-llms.md` to `lifecycle/features/local-llm-operability.md` to document provider-agnostic local model operability across llama.cpp (`llama-server`) and Ollama (`/v1`).

Document:
1. Provider configuration for local inference servers.
2. Benchmarked model families and recommended quantizations (`gemma-4-26B`, `qwen3-coder:30b`, `gpt-oss-20b`).
3. Required server flags (`--jinja` for llama-server).
4. Error taxonomy and step-by-step troubleshooting workflows.

### Files to change

- **`lifecycle/features/local-llm-operability.md`** (replaces `lifecycle/features/ollama-local-llms.md`):
  - Rewrite feature artifact with updated title, lineage, summary, and capabilities.
  - Document local provider setup, model recommendations, error diagnostics, and prompt tuning.

### Acceptance criteria

- [ ] `lifecycle/features/local-llm-operability.md` is authored with clean frontmatter (`type: feature`, `status: approved`).
- [ ] Legacy `ollama-local-llms.md` is replaced and updated.
- [ ] Documentation covers llama.cpp, Ollama, model baselines, quant guidance, and error troubleshooting.
