---
title: "Frontend Plan: Local-Model Operability and UI Error Surfacing"
type: plan-frontend
status: approved
lineage: local-model-operability
parent: lifecycle/requirements/local-model-operability-2.md
created: "2026-08-25T08:49:06+10:00"
---

# Frontend Plan: Local-Model Operability and UI Error Surfacing

Parent: [[local-model-operability-2]].

This plan covers the Vue 3 / TypeScript frontend implementation for FR-3, FR-4, FR-5, and NFR-2, NFR-4 of the [[local-model-operability-2]] requirement (Workstream 3 of the [[open-provider-support]] epic). Backend implementation is defined in [[local-model-operability-3-be]] and test coverage in [[local-model-operability-5-test]].

---

## Milestone 1 — TypeScript Types & Store State for Error Taxonomy and Warmup

### Description

Update the frontend data model and Pinia store modules to support the structured error taxonomy, contextual error details, remediation step lists, and live model warmup/loading states (FR-3, FR-5).

### Files to change

- **`web/src/types/api.ts`**:
  - Extend `failure_reason` on `AgentRunRow` and `AgentFailedPayload`:
    ```typescript
    export type FailureReason =
      | 'tools_unsupported'
      | 'model_not_found'
      | 'model_unloaded'
      | 'endpoint_unreachable'
      | 'context_window_exceeded'
      | 'turn_token_ceiling'
      | 'max_iterations_reached'
      | 'auth_error'
      | 'timeout'
      | 'permission_mode_default'
      | 'precheck_timeout'
      | 'truncated_stream'
      | string
    ```
  - Add `error_details?: Record<string, any>` and `remediation?: string[]` to `AgentRunRow`.
  - Add warmup fields to active run / progress types:
    ```typescript
    export interface AgentStatusEvent {
      run_id: string
      state: 'idle' | 'running' | 'model_loading' | 'warming_up' | 'generating'
      details?: string
    }
    ```
- **`web/src/stores/agents.ts`**:
  - In `handleEvent`:
    - Listen for `agent.status` and `agent.progress` warmup events (`stage === 'warming_up'` or `state === 'model_loading'`) and record live loading state on active run objects.
    - On `agent.failed`, capture `failure_reason`, `remediation`, and `error_details` onto the stored `AgentRunRow`.
- **`web/src/stores/queue.ts`**:
  - Track `is_warming_up` boolean and `warmup_message` on active running jobs.

### Acceptance criteria

- [ ] TypeScript interfaces represent all failure codes, error details, and warmup states without type errors.
- [ ] `agentsStore` and `queueStore` track live warmup and failure state from WebSocket broadcasts.
- [ ] `pnpm exec vue-tsc --noEmit` passes without type errors.

---

## Milestone 2 — Rich Diagnostic Error Surfacing in `RunFailureBanner.vue`

### Description

Enhance `web/src/components/agent/RunFailureBanner.vue` to render dedicated, rich visual diagnostics and actionable remediation instructions for all local-model and provider failure codes (FR-4, NFR-2, NFR-4):

1. **`tools_unsupported`:**
   - *Heading:* "Model does not support tool calling (Function Calling)"
   - *Explanation:* Explains that the selected model or chat template silently dropped or rejected tool schemas.
   - *Remediation Steps:*
     1. Switch agent to a model with verified tool-calling support (e.g. `gemma-4-26B`, `qwen3-coder:30b`, `gpt-oss-20b`).
     2. If using `llama-server`, ensure the `--jinja` flag is enabled.
     3. Open **Provider Settings** to test model capabilities.
2. **`model_not_found`:**
   - *Heading:* "Model not found on provider"
   - *Explanation:* The configured model identifier is not loaded or registered on the endpoint.
   - *Remediation Steps:* Verify spelling in Agent Config or run `ollama pull <model>` / launch `llama-server -m <model.gguf>`.
3. **`model_unloaded`:**
   - *Heading:* "Model failed to load into memory"
   - *Explanation:* Upstream server returned HTTP 503 or ran out of VRAM/RAM while loading model weights.
   - *Remediation Steps:* Check GPU memory availability, reduce context size, or select a smaller quantization.
4. **`endpoint_unreachable`:**
   - *Heading:* "Cannot connect to inference provider"
   - *Explanation:* Connection to the provider `base_url` failed (Connection Refused or DNS lookup failure).
   - *Remediation Steps:* Check that the local inference server (Ollama or llama.cpp) is running on the expected port.
5. **`context_window_exceeded` / `turn_token_ceiling`:**
   - *Heading:* "Model context limit or token ceiling reached"
   - *Explanation:* The prompt and artifact history exceeded the model's context window, or generation hit the per-turn token ceiling.
   - *Remediation Steps:* Reduce input artifact length, increase server context size (`-c 16384`), or tune prompt templates.
6. **`max_iterations_reached`:**
   - *Heading:* "Maximum tool iterations cap reached"
   - *Explanation:* The agent reached the maximum allowed tool iterations (default 25) without completing the artifact.
   - *Remediation Steps:* Check prompt instructions or increase `max_tool_iterations` in agent settings.
7. **`auth_error`:**
   - *Heading:* "Authentication failed with provider"
   - *Explanation:* Upstream provider rejected the API key or credentials (HTTP 401/403).
   - *Remediation Steps:* Verify the configured API key in Provider Settings.
8. **`timeout`:**
   - *Heading:* "Agent run timed out"
   - *Explanation:* The run exceeded its configured `timeout_minutes`.

Include inline action links directing operators to **Provider Settings** (`/settings/providers`) and **Agent Config** (`/settings/agents`). Ensure all credentials remain strictly masked as `"***"` (NFR-2).

### Files to change

- **`web/src/components/agent/RunFailureBanner.vue`**:
  - Add props: `errorDetails?: Record<string, any>`, `providerName?: string`.
  - Expand heading, body text, and remediation step resolution logic for all structured error codes.
  - Render router links to settings views with high-contrast, keyboard-focusable anchors.
  - Retain existing Claude Code permission-precheck cases for backward compatibility.

### Acceptance criteria

- [ ] Dedicated visual headings, descriptions, and numbered remediation steps display for all failure codes.
- [ ] Numbered remediation steps format inline `<code>` commands clearly.
- [ ] Navigation links to Provider Settings and Agent Config render correctly.
- [ ] Banner complies with accessibility guidelines (`role="alert"`, keyboard focusable links, high contrast).
- [ ] No API keys or credentials are leaked in rendered text (`[[standards/secrets-handling]]`).

---

## Milestone 3 — Run Detail Modal & Runs View Failure Surfacing Integration

### Description

Wire the enriched failure diagnostics into `RunDetailModal.vue` and `AgentsRunsView.vue`, ensuring operators have instant visibility into why an agent run failed and what to do next (FR-3, FR-4).

### Files to change

- **`web/src/components/agent/RunDetailModal.vue`**:
  - Pass `failure_reason`, `remediation`, and `error_details` from `run` or `runResult` to `RunFailureBanner.vue`.
  - When `error_details` are present, render a collapsible "Diagnostic Info" block (showing provider name, endpoint URL, HTTP status code, and sanitised error message).
- **`web/src/views/project/AgentsRunsView.vue`**:
  - Render `RunFailureBanner.vue` inside the expanded row details for failed runs.
  - Display failure badges with tooltip descriptions in the main run history table.

### Acceptance criteria

- [ ] Opening `RunDetailModal.vue` on a failed run displays `RunFailureBanner.vue` with full remediation steps.
- [ ] Diagnostic metadata (provider, endpoint, status code) is cleanly rendered.
- [ ] Expanded rows in `AgentsRunsView.vue` render the failure banner.
- [ ] UI remains fully functional and responsive across mobile and desktop viewports.

---

## Milestone 4 — Live Warmup & Weight Loading Visual Indicators

### Description

Provide real-time visual feedback when a local inference server is lazy-loading weights or warming up (FR-5, NFR-4), preventing the perception that the application is frozen during multi-gigabyte GGUF load times.

### Files to change

- **`web/src/components/dashboard/AgentRunningBanner.vue`**:
  - When `runningJob.is_warming_up` is true or state is `model_loading`:
    - Display an animated badge: "Warming up model weights..." alongside the running timer.
    - Use distinct amber/cyan styling to differentiate weight loading from active token generation.
- **`web/src/views/project/AgentsRunsView.vue`**:
  - Display a pulsing "Warming up..." status indicator on active run rows during the pre-TTFT warmup phase.
- **`web/src/components/agent/RunDetailModal.vue`**:
  - Show a "Warming up model weights in memory..." entry in the turn timeline before token generation starts.

### Acceptance criteria

- [ ] `AgentRunningBanner.vue` shows an animated warmup badge when a run is loading model weights.
- [ ] Active run rows in `AgentsRunsView.vue` indicate warmup status during pre-TTFT loading.
- [ ] Visual indicators automatically transition to active generation once the first token is received.
- [ ] All indicators meet accessibility contrast requirements.
