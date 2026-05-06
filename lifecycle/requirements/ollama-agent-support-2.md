---
title: Ollama Agent Support
type: requirement
status: planning
lineage: ollama-agent-support
priority: high
parent: lifecycle/ideas/ollama-agent-support.md
labels:
    - feature
    - agent
    - agent-runner
    - backend
    - frontend
assignees:
    - role: product-owner
      who: agent
---

# Ollama Agent Support

## Problem

The agent runner currently only supports a single driver (`claude-code-cli`). Users who want to leverage local or remote Ollama instances for agent tasks have no way to do so. This limits model choice, increases cost for simpler tasks, and prevents offline or air-gapped usage.

## Goals / Non-goals

### Goals

- Allow users to register one or more Ollama instances (local or remote) via the UI and project config.
- Implement an `ollama` driver that conforms to the existing `Driver` interface and integrates with the agent runner and supervisor.
- Expose model discovery (list available models on a registered instance) in both the API and UI.
- Allow agent definitions in `lifecycle/config.yaml` to specify `driver: ollama` with an associated instance and model.
- Keep the agent execution experience (progress events, logging, scope enforcement) consistent across drivers.

### Non-goals

- Streaming token-level output to the UI (batch response is acceptable for v1).
- Fine-tuning or model management (pull/delete) through kaos-control — users manage models via Ollama directly.
- Supporting Ollama tool-use / function-calling (out of scope unless trivially available).
- Multi-turn agent orchestration specific to Ollama — the existing single-prompt-per-run model applies.

## Detailed Requirements

### Functional

#### FR-1: Instance Registration

- Users can add, edit, and remove Ollama instances via a settings panel in the frontend.
- Each instance record contains: `name` (unique identifier), `base_url` (e.g. `http://localhost:11434`), and optional `api_key` (for reverse-proxy auth).
- Instance records are persisted in the project config (`lifecycle/config.yaml`) under a new `ollama_instances` key.

#### FR-2: Model Discovery

- The backend exposes `GET /api/ollama/instances/{name}/models` which proxies `/api/tags` on the target Ollama instance and returns a list of model names and sizes.
- The frontend model-selection dropdown is populated by calling this endpoint.
- A "Refresh Models" action re-fetches the model list on demand.

#### FR-3: Ollama Driver

- A new `OllamaDriver` struct implements the `Driver` interface (`Start(ctx, Run) (Process, error)`).
- The driver resolves the target instance from config, constructs a `/api/chat` request with the prompt from `Run.PromptText`, and streams the response.
- The driver emits `ProgressEvent` messages (at minimum: `started`, `output` with the model response, `completed` or `error`).
- `Process.Wait()` blocks until the HTTP response is fully consumed or the context is cancelled.
- `Process.Kill()` cancels the underlying HTTP request context.
- `Process.StderrTail()` returns any HTTP-level error details.

#### FR-4: Agent Configuration

- The `agents` list in `lifecycle/config.yaml` accepts `driver: ollama` alongside the existing `driver: claude-code-cli`.
- When `driver: ollama`, the agent config requires an additional `ollama_instance` field (name reference) and `model` specifies the Ollama model tag (e.g. `llama3:8b`).
- The agent runner selects the correct driver implementation based on the `driver` field at run time.

#### FR-5: Frontend Agent Creation

- The agent creation/edit UI surfaces a driver-type selector (`Claude Code` | `Ollama`).
- When Ollama is selected, the UI shows an instance dropdown (populated from registered instances) and a model dropdown (populated via FR-2).
- Validation prevents saving an Ollama agent without a valid instance and model selection.

#### FR-6: Connectivity Check

- `GET /api/ollama/instances/{name}/health` pings the Ollama instance (via `GET /` or `/api/tags`) and returns a boolean `ok` status plus latency.
- The UI displays connection status indicators (green/red) next to each registered instance.

### Non-functional

#### NFR-1: Timeout & Resilience

- Ollama HTTP calls must respect configurable timeouts (default: 5 minutes for inference, 10 seconds for health/model-list).
- If an Ollama instance is unreachable, the driver must fail fast with a clear error rather than hanging.

#### NFR-2: Credential Security

- `api_key` values must not be returned in API responses (`GET /api/ollama/instances` returns `"api_key": "***"` when set).
- Keys are stored in the project config file on disk (same trust boundary as existing config).

#### NFR-3: Concurrency

- Ollama runs participate in the existing `max_concurrent_agents` semaphore — no separate pool.

#### NFR-4: Scope Enforcement

- `allowed_write_paths` and all existing sandbox/scope rules apply identically regardless of driver. Since the Ollama driver receives a text response (not CLI execution), write-path enforcement is the responsibility of the agent runner processing the driver's output, not the driver itself.

## Acceptance Criteria

- [ ] A user can register an Ollama instance via the settings UI and it persists in project config.
- [ ] `GET /api/ollama/instances/{name}/models` returns the model list from a running Ollama instance.
- [ ] `GET /api/ollama/instances/{name}/health` returns status and latency.
- [ ] An agent configured with `driver: ollama` successfully executes a prompt and returns output via `ProgressEvent` stream.
- [ ] The agent runner correctly routes to `OllamaDriver` when `driver: ollama` is specified.
- [ ] The frontend agent creation form allows selecting Ollama as a driver with instance and model pickers.
- [ ] Connection failure to an Ollama instance produces a clear, user-visible error within the configured timeout.
- [ ] API key values are masked in all API responses.
- [ ] Ollama agent runs respect the global `max_concurrent_agents` limit.
- [ ] Existing `claude-code-cli` agents continue to function without regression.
- [ ] [[ollama-agent-support]] lineage artifacts link correctly.

## Resolved Questions

1. Should the Ollama driver support system prompts separately from user prompts, or is a single combined prompt sufficient for v1?

> They should be seperated.

2. Should instance registration live in app-level config (`~/.kaos-control/config.yaml`) rather than project config, allowing shared instances across projects?

> Yes, app level config is a good idea.

3. Is there a need to support Ollama's `/api/generate` endpoint (raw completion) in addition to `/api/chat` (chat completion), or is chat-only acceptable?

> Both should be supported.
