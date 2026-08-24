---
title: Local-Model Operability and UI Error Surfacing
type: requirement
status: draft
lineage: local-model-operability
parent: lifecycle/ideas/local-model-operability.md
created: "2026-08-25T08:26:00+10:00"
priority: normal
labels:
    - agent
    - agent-runner
    - backend
    - frontend
    - prompts
    - operability
    - open-provider-support
release: KC-Release6
assignees:
    - role: product-owner
      who: agent
---

# Local-Model Operability and UI Error Surfacing

## Scope Context

This requirement represents **Workstream 3** of the [[open-provider-support]] epic, alongside [[open-provider-support-2]] (the OpenAI-compatible tool-calling driver) and [[provider-model-for-agents-2]] (the Provider entity model).

While the OpenAI-compatible driver establishes wire-level reachability to local inference servers (llama.cpp `llama-server`, Ollama `/v1`), reachability alone does not guarantee that a local model can successfully produce **usable lifecycle artifacts**. Furthermore, when local models or endpoints fail (due to unsupported tools, missing models, lazy-load timeouts, or context overflows), operators currently experience silent hangs or opaque error states.

This requirement defines the operability layer:
1. **Prompt templates tuned for local models** (concise context, rigid single-step execution, explicit frontmatter constraints, and worked few-shot examples).
2. **Proactive model availability & loading state detection** (verifying model presence and surfacing warmup/lazy-loading status in the UI).
3. **Structured error taxonomy & rich UI surfacing** (diagnostics and actionable remediation across `RunFailureBanner.vue`, `RunDetailModal.vue`, and `AgentsRunsView.vue`).
4. **Generalization of local LLM documentation** (updating legacy Ollama feature artifacts).

---

## Problem

kaos-control was initially built and tuned against frontier cloud models (Claude 3.7 Sonnet, GPT-4o). When running against locally-hosted models (typically 8B to 30B parameters running via GGUF/llama.cpp or Ollama), three critical operability gaps prevent a seamless user experience:

1. **Frontier Prompt Templates Cause Local Model Failure:**
   The default agent prompt templates in `lifecycle/config.yaml` and `internal/initcmd/templates/` contain extensive prose, multi-phase instructions, and complex negative constraints. Frontier models handle this easily, but benchmark evidence across 13 local models in [[open-provider-support-2]] shows extreme sensitivity to prompt shape (scores ranging from 0/68 to 68/68). Small models suffer from context dilution, instruction omission, frontmatter syntax corruption, and run-away reasoning loops that hit per-turn token ceilings (e.g. 8192 tokens) without ever invoking tools.

2. **Unannounced Stalls on Lazy-Loaded Models:**
   Local inference servers like `llama-server` often lazy-load GGUF models on the first incoming request (or report `status: unloaded`). When an agent run begins, loading multi-gigabyte model weights into VRAM/RAM can take 30 to 120 seconds. During this window, the backend HTTP request blocks without streaming tokens, leaving the user with a static "running" state and zero feedback as to whether the process is dead, frozen, or loading weights.

3. **Opaque Error Surfacing in the UI:**
   When an agent run fails against a local endpoint, the UI currently renders generic messages (e.g. `Run failed: unknown`) or unformatted stderr tails. Specific failure modes unique to local LLM operations are completely obscured:
   - FR-5b capability preflight failures (where a model silently drops `tools` or rejects function calling).
   - Upstream HTTP 404 (model not found on provider) or HTTP 503 (model overloaded/unloaded).
   - Local connection refused errors (`127.0.0.1:7442` or `127.0.0.1:11434` offline).
   - Context window limit exhaustion or turn-level token ceiling starvation.
   - Max tool iteration caps reached without completing the artifact.
   Operators are left without clear explanations or step-by-step remediation instructions.

4. **Legacy Ollama Coupling in Feature Documentation:**
   Standing feature documentation (`lifecycle/features/ollama-local-llms.md`) remains tied to the deprecated Ollama-specific architecture and UI rather than documenting the unified local model capabilities enabled by the OpenAI-compatible provider architecture.

---

## Goals / Non-goals

### Goals

- Deliver **local-model tuned prompt templates** for core lifecycle roles (`analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`, `tech-writer`) structured with concise context, explicit step ordering, strict markdown schemas, and concrete few-shot examples.
- Support **local-model prompt profiles/templates** in project configuration (`lifecycle/config.yaml`) and scaffolding templates (`config.yaml.tmpl`), allowing projects to select or customize prompt shapes optimized for local model strengths.
- Implement **model availability & load state preflight** in the Go backend to check whether the configured model is available and resident on the provider endpoint before initiating an agent loop.
- Provide **live warmup/loading feedback** in the UI (`AgentRunningBanner.vue`, `AgentsRunsView.vue`) when a local server is loading model weights, differentiating memory warmup from active token generation.
- Establish a **structured error taxonomy** for agent runs, capturing distinct error reasons (`tools_unsupported`, `model_not_found`, `model_unloaded`, `endpoint_unreachable`, `context_window_exceeded`, `turn_token_ceiling`, `max_iterations_reached`, `auth_error`).
- Enhance **UI error surfacing** in `RunFailureBanner.vue`, `RunDetailModal.vue`, and `AgentsRunsView.vue` with clear diagnostic descriptions, badge statuses, and numbered remediation steps with inline commands/links.
- Update and generalize feature documentation from `ollama-local-llms.md` to reflect unified local model operability.

### Non-goals

- Re-implementing the OpenAI-compatible tool-calling loop, SSE streaming parser, or token-delta preflight probe (owned by [[open-provider-support-2]]).
- Implementing multi-provider failover or automatic provider switching on error (owned by [[switch-provider]]).
- Managing local server daemon lifecycles (downloading GGUF weights, executing `llama-server`, or configuring GPU offload layers).
- Guaranteeing identical reasoning capabilities between 8B parameter local models and frontier 1T+ parameter cloud models.

---

## Detailed Requirements

### Architecture-Breaking Requirements

Review against `lifecycle/architecture/architecture-summary.md` and recorded architectural standards:

1. **Single self-contained binary:**
   - *Requirement:* All prompt templating, availability probing, error mapping, and UI components must be implemented using pure Go standard library packages and embedded Vue 3 SPA components.
   - *Evaluation:* **Satisfied.** No CGO bindings, external daemon dependencies, or native libraries are introduced.
2. **Local filesystem is the source of truth:**
   - *Requirement:* Prompt templates and agent definitions remain stored in `lifecycle/config.yaml` and project markdown files. Run records and metrics cached in SQLite index are strictly rebuildable.
   - *Evaluation:* **Satisfied.** Disk remains authoritative; index conforms to [[standards/index-is-a-cache]].
3. **Offline operation capability:**
   - *Requirement:* Local model operability must function 100% offline without external network calls or cloud dependencies when targeting local endpoints (`localhost:11434`, `localhost:7442`).
   - *Evaluation:* **Satisfied.** Enhances offline operability and removes cloud-only prompt assumptions.
4. **Direct-served, no trusted proxy hop & Secrets hygiene:**
   - *Requirement:* Backend probes endpoints directly; browser UI never connects directly to inference servers; API keys are masked (`***`) in all error payloads and UI views ([[standards/secrets-handling]], [[decisions/adr-0001-no-header-based-client-ip-trust]]).
   - *Evaluation:* **Satisfied.** Diagnostic reporting masks tokens and operates entirely server-side.
5. **Agent tool mediation and sandboxing:**
   - *Requirement:* Local-model tuned prompts must strictly guide tool usage within `allowed_write_paths` without bypassing the mediated sandbox resolver ([[decisions/adr-0006-mediated-agent-driver-permission-model]], [[standards/filesystem-sandboxing]]).
   - *Evaluation:* **Satisfied.** Prompts reinforce tool constraints and fail closed on permission errors.

**Conclusion:** All standing architectural constraints are fully satisfied. No new ADR is required.

---

### Functional Requirements

#### FR-1: Local-Model Tuned Prompt Templates

- Define optimized prompt templates for local models (e.g. 8B–30B parameters) across standard lifecycle agent roles:
  - **`analyst` (Requirements Analyst):**
    - Focused single-step brief: Read idea at `{target_path}`, evaluate architecture constraints, and produce a single requirement artifact in `lifecycle/requirements/`.
    - Explicit step sequence: 1. `read_file("{target_path}")` $\to$ 2. `read_file("lifecycle/architecture/architecture-summary.md")` $\to$ 3. `write_file("lifecycle/requirements/<slug>-2.md", content)` $\to$ 4. Conclude with assistant summary.
    - Strict frontmatter schema with a concrete few-shot example including `title`, `type: requirement`, `status: draft`, `lineage`, `parent`, and `created`.
    - Mandatory section checklist: `## Problem`, `## Goals / Non-goals`, `## Detailed Requirements`, `## Acceptance Criteria`, `## Open Questions`.
  - **`analyst` (Planning Analyst):**
    - Ordered three-artifact sequence: Creates backend plan (`-be.md`), frontend plan (`-fe.md`), and test plan (`-test.md`).
    - Clear milestone templates with explicit criteria per milestone.
  - **`backend-developer` & `frontend-developer`:**
    - Explicit milestone-at-a-time execution instructions.
    - Rigid file scoping rules reinforcing `allowed_write_paths`.
  - **`test-developer` & `qa`:**
    - Structured defect artifact templates (`lifecycle/defects/`) with exact `## Reproduction Steps`, `## Expected Behaviour`, `## Actual Behaviour` headings.
- **Template Design Constraints for Local Models:**
  - Shorter overall context length ($< 1200$ tokens per prompt template).
  - Elimination of meta-reasoning distractions and nested hypothetical instructions.
  - Inclusion of exact YAML frontmatter syntax blocks to eliminate syntax errors.
  - Explicit instruction to immediately invoke the required tool (`read_file` / `write_file`) rather than generating conversational preambles.
- **Configuration & Scaffolding:**
  - Update `internal/initcmd/templates/config.yaml.tmpl` to include local-model prompt template presets.
  - Provide a configuration setting or template section in `lifecycle/config.yaml` to easily toggle or apply local-tuned templates for agents utilizing local providers.

#### FR-2: Proactive Model Availability & Loading State Checks

- Before dispatching an agent run (in `Manager.StartRun` or driver initialization):
  - The driver / backend queries the provider's `/v1/models` endpoint.
  - If the configured `model` is not found in the returned model list, `StartRun` aborts immediately with a `model_not_found` error, preventing lock acquisition and saving execution tokens.
  - If the upstream provider returns model status metadata (e.g. llama.cpp model status `unloaded` or `loading`):
    - The backend broadcasts an `agent.status` event with `state: "model_loading"` and `details: "Loading model weights into memory..."`.
    - A dedicated loading timeout (default 60s, configurable) is applied for weight loading before marking the run as timed out.
- For providers without explicit loading state APIs (e.g. Ollama `/v1` cold-start):
  - The driver tracks time elapsed prior to Time To First Token (TTFT).
  - If TTFT exceeds 5 seconds, the driver emits a progress update: `{"type": "status", "message": "Awaiting first token (model may be warming up)..."}`.

#### FR-3: Structured Error Taxonomy & Event Enrichment

- The agent `Manager` and `openai-compatible` driver must categorize run failures into standard structured error codes:
  1. `tools_unsupported`: Tool capability preflight failed (prompt tokens identical with/without tools per FR-5b, or upstream HTTP 400 rejection).
  2. `model_not_found`: Configured model identifier not present on provider endpoint (HTTP 404).
  3. `model_unloaded`: Model failed to load into memory / server returned HTTP 503 unloaded.
  4. `endpoint_unreachable`: Failed to connect to provider `base_url` (connection refused, DNS failure, or network timeout).
  5. `context_window_exceeded`: Model context length exceeded during prompt generation or multi-turn history accumulation.
  6. `turn_token_ceiling`: Per-turn generation token limit reached (e.g. 8192 tokens) with empty tool call output.
  7. `max_iterations_reached`: Agent loop hit `max_tool_iterations` (default 25) without returning `finish_reason: stop`.
  8. `auth_error`: Upstream returned HTTP 401 / 403 unauthorized.
  9. `timeout`: Run exceeded configured `timeout_minutes`.
- `index.AgentRunRow` and the `agent.failed` WebSocket broadcast payload are enriched with:
  - `failure_reason` (string): One of the standard error codes above.
  - `error_details` (map[string]any): Contextual details (e.g. `provider`, `model`, `base_url`, `status_code`, `raw_message`).
  - `remediation` ([]string): Ordered list of human-readable, actionable steps to fix the issue.

#### FR-4: UI Error Surfacing (`RunFailureBanner.vue`)

- Extend `web/src/components/agent/RunFailureBanner.vue` to provide dedicated, rich visual diagnosis for all local model failure codes:
  - **`tools_unsupported`:**
    - *Heading:* "Model does not support tool calling (Function Calling)"
    - *Explanation:* Explains that the selected model or its chat template silently dropped or rejected tool definitions, preventing it from executing file reads or writes.
    - *Remediation Steps:*
      1. Switch agent to a model with verified tool calling support (e.g. `gemma-4-26B`, `qwen3-coder:30b`, `gpt-oss-20b-Q8_0`).
      2. If using `llama-server`, ensure `--jinja` flag is enabled.
      3. Open **Provider Settings** to test model capabilities.
  - **`model_not_found`:**
    - *Heading:* "Model not found on provider"
    - *Explanation:* The model `<model>` is not loaded or registered on `<base_url>`.
    - *Remediation:* Verify spelling in Agent Settings or run `ollama pull <model>` / launch `llama-server -m <model.gguf>`.
  - **`endpoint_unreachable`:**
    - *Heading:* "Cannot connect to inference provider"
    - *Explanation:* Connection to `<base_url>` failed (Connection Refused).
    - *Remediation:* Check that the local inference server (Ollama or llama.cpp) is running on the specified port.
  - **`context_window_exceeded` / `turn_token_ceiling`:**
    - *Heading:* "Model context limit or token ceiling reached"
    - *Explanation:* The prompt and artifact history exceeded the model's maximum context window, or the model ran out of generation tokens before completing reasoning.
    - *Remediation:* Reduce input artifact length, increase server context size (`-c 16384`), or tune `max_tokens` / prompt template.
  - **`max_iterations_reached`:**
    - *Heading:* "Maximum tool iterations cap reached"
    - *Explanation:* The agent performed 25 sequential tool calls without completing its task.
    - *Remediation:* Check prompt clarity or increase `max_tool_iterations` in agent configuration.
- Add direct UI links inside the banner to **Provider Settings** (`/settings/providers`) and **Agent Config** (`/settings/agents`).

#### FR-5: Live Model Loading & Warmup UI Indicators

- Update `web/src/components/dashboard/AgentRunningBanner.vue` and `web/src/views/project/AgentsRunsView.vue`:
  - When an agent is in the `model_loading` phase or awaiting initial tokens ($> 5\text{ s}$), render an animated "Warming up model weights..." badge alongside the timer.
  - Ensure the user can distinguish between an active model processing/loading and a frozen agent process.

#### FR-6: Update and Generalize Feature Documentation

- Rename/migrate `lifecycle/features/ollama-local-llms.md` to `lifecycle/features/local-llm-operability.md`.
- Document:
  - Configuration of local providers (llama.cpp `llama-server` and Ollama `/v1`).
  - Supported and benchmarked model families with quant recommendations (e.g. `gemma-4-26B-A4B-it-UD-Q8_K_XL`, `gpt-oss-20b-Q8_0`, `qwen3-coder:30b`).
  - Required server flags (`--jinja` for llama-server).
  - Explanation of common error codes and troubleshooting workflows.

---

### Non-Functional Requirements

#### NFR-1: Feedback Latency & Fast Failure

- Preflight model availability checks must execute within a 3-second timeout.
- If a model is missing or tools are unsupported, the run must fail immediately within 3 seconds, releasing lineage locks and avoiding unnecessary resource consumption.

#### NFR-2: Secret Hygiene & Safe Diagnostics

- Diagnostic payloads and error banners MUST NEVER expose raw API keys, bearer tokens, or sensitive custom headers ([[standards/secrets-handling]]). All provider credentials displayed in error logs or UI banners must be masked as `"***"`.

#### NFR-3: Zero Artifact Corruption on Failure

- Any run that encounters a preflight failure (`tools_unsupported`, `model_not_found`, `endpoint_unreachable`) MUST terminate cleanly before modifying any lifecycle markdown files on disk or committing changes to git.

#### NFR-4: UI Usability & Accessibility

- All error banners, status chips, and remediation steps must meet accessibility standards:
  - Render with `role="alert"` and high-contrast styling.
  - Numbered remediation steps formatted with clear typography and inline code formatting for shell commands.
  - Interactive links keyboard-focusable and accessible via Tab navigation.

---

## Acceptance Criteria

- [ ] Local-model tuned prompt templates are authored for `analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`, and `tech-writer` roles, featuring concise instructions, explicit step ordering, few-shot YAML frontmatter examples, and strict section requirements.
- [ ] Scaffolding template (`internal/initcmd/templates/config.yaml.tmpl`) and project config include local-model prompt options.
- [ ] Proactive model availability check queries `/v1/models` prior to run launch and fails fast with `model_not_found` if the specified model is absent.
- [ ] If a local server is lazy-loading weights or takes $> 5\text{ s}$ for initial token generation, an active warmup indicator is displayed in `AgentRunningBanner.vue` and `AgentsRunsView.vue`.
- [ ] Standard structured error codes (`tools_unsupported`, `model_not_found`, `model_unloaded`, `endpoint_unreachable`, `context_window_exceeded`, `turn_token_ceiling`, `max_iterations_reached`, `auth_error`, `timeout`) are populated on failed runs.
- [ ] `RunFailureBanner.vue` displays dedicated headings, diagnostic descriptions, and numbered remediation steps for all structured error codes.
- [ ] `RunFailureBanner.vue` includes direct navigation links to Provider Settings and Agent Config where appropriate.
- [ ] `RunDetailModal.vue` and `AgentsRunsView.vue` properly render the new failure banners and error details.
- [ ] No API keys or credentials are leaked in error payloads, logs, or UI failure banners.
- [ ] `lifecycle/features/ollama-local-llms.md` is updated and generalized to reflect provider-agnostic local model operability.
- [ ] Unit and integration tests verify prompt template rendering, availability check fast-fail, error taxonomy mapping, and UI banner error formatting.
- [ ] Artifact correctly links via `parent: lifecycle/ideas/local-model-operability.md` and references related workstreams [[open-provider-support]], [[open-provider-support-2]], [[provider-model-for-agents-2]], and [[switch-provider]].

---

## Resolved Questions

1. **Local-Model Prompt Template Selection Mechanism:**
   - Should local-model prompt templates be applied via a dedicated agent profile flag (e.g. `prompt_profile: local-compact`), or should they simply be the default prompt templates provided in `prompt_templates` whenever an agent is configured with a local provider?
   - *Recommendation:* Provide the local-tuned templates as the default standard in `config.yaml.tmpl` for local provider agents, while allowing `prompt_templates` in `lifecycle/config.yaml` to override them per role as today.

> Proceed with recommendation.

2. **Warmup Pre-load Endpoint Trigger:**
   - Some local servers (e.g. Ollama `/api/generate` with `keep_alive` or llama.cpp `/props`) allow sending a zero-token warmup request to force model weight loading into VRAM before starting a run. Should Provider Settings include a "Preload / Warmup" button for registered local models?
   - *Recommendation:* Defer explicit preload buttons to a future enhancement; detecting and surfacing the live loading state during agent run initiation satisfies v1 operability.

> Proceed with recommendation.
