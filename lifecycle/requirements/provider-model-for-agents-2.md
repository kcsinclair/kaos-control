---
title: Provider Model for Agent API Endpoints
type: requirement
status: done
lineage: provider-model-for-agents
created: "2026-08-25T07:15:00+10:00"
priority: normal
parent: lifecycle/ideas/provider-model-for-agents.md
labels:
    - provider
    - driver
    - agent
    - config
    - backend
    - frontend
    - feature
    - open-provider-support
release: KC-Release6
assignees:
    - role: product-owner
      who: agent
---

# Provider Model for Agent API Endpoints

## Problem

kaos-control currently treats local and cloud model connectivity in a fragmented, ad-hoc manner:

1. **Tight coupling to Ollama-specific configuration:** Local inference is managed via an `ollama_instances` block in app config, dedicated `/api/ollama/instances` endpoints, and an `OllamaSettingsView` frontend. The native `ollama` driver is hard-coded to single-shot `/api/chat` without tool support, while other drivers (`claude-env`, `gemini`) duplicate endpoint and credential management across individual agent declarations.
2. **Duplicated connection identity per agent:** Each agent definition in `lifecycle/config.yaml` is forced to declare raw connection details (`base_url`, `auth_token`, `endpoint`, `ollama_instance`) rather than referencing a shared, centrally managed endpoint. Pointing agents at a different inference server or updating an API key requires tedious, error-prone edits across multiple agent definitions.
3. **Inability to uniformly target OpenAI-compatible endpoints:** Gateways (OpenRouter), remote APIs (OpenAI, Groq, Together, Azure), and local servers (llama.cpp `llama-server`, Ollama `/v1`) all speak the standard OpenAI `/v1` wire format. Without a generalized **Provider** abstraction, each target appears to require a bespoke driver and distinct GUI configuration, adding unnecessary complexity.

This requirement establishes the **Provider entity** as the architecture spine of the [[open-provider-support]] epic, decoupling agent roles from API connection details and unifying all endpoint management under a single configuration surface.

---

## Goals / Non-goals

### Goals

- Introduce a first-class **Provider** model (`{name, base_url, api_key, driver, extra_headers}`) in app-level configuration (`~/.kaos-control/config.yaml`), replacing the legacy `ollama_instances` section.
- Decouple agent definitions in project configuration (`lifecycle/config.yaml`) so that an agent is a clean **`{provider, model}`** pair referencing a registered provider by name.
- Provide a unified Provider management REST API (`/api/providers`) supporting listing, creation, modification, deletion with project reference checks, live health verification, and `/v1/models` model discovery.
- Deliver a dedicated **Provider Settings** frontend view (`ProviderSettingsView.vue`) and components to register, edit, test, and monitor providers.
- Update the Agent Editor UI (`AgentConfigForm.vue`) to select from registered providers and choose from discovered models dynamically.
- Support arbitrary HTTP headers (`extra_headers`) on provider records to enable gateway attribution (e.g. OpenRouter `HTTP-Referer` and `X-Title`) purely through configuration ([[openrouter-llm-integration]]).
- Remove the native `ollama` driver and legacy Ollama surface outright, migrating existing registrations (`Loki` $\to$ `http://leia.packsin.com:11434`) seamlessly to OpenAI-compatible provider records.
- Enforce strict secret hygiene conforming to [[standards/secrets-handling]]: provider API keys are masked (`***`) in all API responses and UI displays, and never leaked in logs or lifecycle markdown artifacts.

### Non-goals

- Implementing the tool-calling loop, SSE parser, native tool-call recovery, or preflight capability checks of the `openai-compatible` driver itself. (This is specified in [[open-provider-support-2]]).
- Dynamic provider failover or automatic error-driven provider switching. (This is specified in workstream 2: [[switch-provider]]).
- Managing external inference server lifecycles (installing, starting, or provisioning Ollama, llama.cpp, or cloud accounts).
- Modifying standalone CLI drivers (`claude-code-cli`, `codex-cli`, `gemini-cli`, `shell-stub`) that execute local CLI binaries rather than connecting to HTTP API providers.

---

## Detailed Requirements

### Architecture-Breaking Requirements

Review against `lifecycle/architecture/architecture-summary.md` and recorded architectural standards:

1. **Single self-contained binary:**
   - *Requirement:* Provider configuration, model discovery, and HTTP communication must use pure Go standard library packages (`net/http`, `encoding/json`) and embedded Vue assets.
   - *Evaluation:* **Satisfied.** No external daemons, CGO bindings, or native dependencies are introduced.
2. **Local filesystem is the source of truth:**
   - *Requirement:* Global provider definitions must be stored in app config (`~/.kaos-control/config.yaml`) and project agent bindings in `lifecycle/config.yaml`.
   - *Evaluation:* **Satisfied.** Disk remains the source of truth; SQLite index remains a rebuildable cache ([[standards/index-is-a-cache]]).
3. **Offline operation capability:**
   - *Requirement:* When configured with local inference endpoints (`http://localhost:11434`, `http://localhost:7442`), the system must function with zero internet connectivity.
   - *Evaluation:* **Satisfied.** Remote cloud APIs are optional; local endpoints operate fully offline.
4. **Direct-served, no trusted proxy hop & Secrets hygiene:**
   - *Requirement:* Server calls provider APIs directly; browser client never directly contacts provider endpoints or handles unmasked API keys ([[standards/secrets-handling]], [[decisions/adr-0001-no-header-based-client-ip-trust]]).
   - *Evaluation:* **Satisfied.** API keys are masked (`***`) at API boundaries and kept out of artifacts and logs.
5. **Agent tool mediation and sandboxing:**
   - *Requirement:* Decoupling providers from agents must not bypass the mediated driver tool-call enforcement ([[decisions/adr-0006-mediated-agent-driver-permission-model]], [[standards/filesystem-sandboxing]]).
   - *Evaluation:* **Satisfied.** Tool execution remains sandbox-mediated within `allowed_write_paths`.

**Conclusion:** No architectural constraints are violated. No new ADR is required.

---

### Functional Requirements

#### FR-1: Provider Entity Model & App Configuration

- A new `ProviderConfig` struct is defined in `internal/config/config.go`:
  - `name` (string, **required**): Unique identifier (slug: `^[a-z0-9-]+$`, 2–64 characters).
  - `base_url` (string, **required**): Server root URL (e.g. `http://leia.packsin.com:11434`, `https://openrouter.ai/api/v1`). Must parse as a valid `http` or `https` URL.
  - `driver` (string, **required**): Driver identifier, defaulting to `openai-compatible`.
  - `api_key` (string, **optional**): Secret credential sent as `Authorization: Bearer <api_key>` (or driver-specific auth header).
  - `extra_headers` (map[string]string, **optional**): Additional HTTP request headers attached to all outbound requests to this provider.
- `config.App` gains `Providers []ProviderConfig` (YAML key: `providers`).
- `config.ValidateApp` validates:
  - Each provider has a non-empty `name` conforming to slug format.
  - Provider names are unique across the `providers` list.
  - `base_url` is non-empty and is a valid `http`/`https` URL.
  - `driver` is non-empty and names a registered API driver (e.g. `openai-compatible`).
- **Seamless Startup Migration:** If `~/.kaos-control/config.yaml` contains legacy `ollama_instances` and `providers` is empty/omitted:
  - Each `OllamaInstance` is migrated in-memory to a `ProviderConfig` with `name = inst.Name`, `base_url = inst.BaseURL`, `api_key = inst.APIKey`, `driver = "openai-compatible"`.
  - The migrated configuration is persisted back to `~/.kaos-control/config.yaml`.
  - A startup info log records the migration of existing instances (e.g. `Loki`).

#### FR-2: Agent Configuration Refactoring (`{provider, model}` Pair)

- `AgentConfig` in `internal/config/config.go` gains:
  - `provider` (string, **optional for CLI drivers, required for API drivers**): Names a configured provider in app config.
  - `model` (string, **required**): The model identifier requested from the provider.
- Legacy fields on `AgentConfig` (`ollama_instance`, `ollama_endpoint`, `base_url`, `auth_token`) are deprecated and removed from validation.
- `config.ValidateProject` validates:
  - For any agent using an API driver (or specifying `provider`), the referenced `provider` exists in the registered app providers (when app config is available).
  - `model` is non-empty.
  - Standalone CLI drivers (`claude-code-cli`, `codex-cli`, `gemini-cli`, `shell-stub`) continue to validate their respective CLI settings without requiring a provider.

#### FR-3: Outright Removal of Native Ollama Driver & Surface

- The native `ollama` driver (`internal/agent/ollama.go`) is **removed outright**.
- `driver: ollama` is removed from the accepted driver vocabulary.
- The `/api/ollama/instances` HTTP route group (`internal/http/ollama.go`) is removed and replaced by `/api/providers`.
- Frontend legacy Ollama files (`OllamaSettingsView.vue`, `components/ollama/OllamaInstanceForm.vue`, `stores/ollamaInstances.ts`, `api/ollama.ts`) are replaced with the generalized Provider equivalents.

#### FR-4: Provider Management REST API (`/api/providers`)

The server mounts the following authenticated routes under `/api/providers` (requiring admin or devops roles for mutations):

- `GET /api/providers`
  - Returns `{"providers": [...]}` where each record includes `name`, `base_url`, `driver`, `extra_headers`, and `api_key` masked as `"***"` (or omitted if empty).
- `POST /api/providers`
  - Creates a new provider. Request body contains `{name, base_url, driver, api_key?, extra_headers?}`.
  - Returns `201 Created` with the masked provider record.
  - Returns `400 Bad Request` if name is duplicate or fields are invalid.
- `GET /api/providers/{name}`
  - Returns `200 OK` with the single masked provider record, or `404 Not Found`.
- `PUT /api/providers/{name}`
  - Updates an existing provider.
  - If `api_key` is `"***"` or omitted from payload, the existing stored `api_key` is preserved.
  - If `api_key` is a non-empty string other than `"***"`, it replaces the stored key.
  - If `api_key` is explicitly `""`, the stored key is cleared.
  - Persists changes atomically to `~/.kaos-control/config.yaml`.
- `DELETE /api/providers/{name}`
  - Checks whether any agent across any registered project references `provider: {name}`.
  - If referenced, rejects deletion with `409 Conflict` and a message listing referencing projects and agents (e.g. `"cannot delete provider 'Loki': in use by project 'kaos-control' (agent 'requirements-analyst')"`).
  - If unreferenced, removes the provider from app config, saves atomically, and returns `204 No Content`.
- `GET /api/providers/{name}/health`
  - Performs an active HTTP probe from the Go backend to `<base_url>/v1/models` (or `<base_url>/api/tags` / root if probing generic endpoints) using a 5-second timeout.
  - Returns `200 OK` with `{"ok": true, "latency_ms": 42}` or `{"ok": false, "error": "connection refused"}`.
- `GET /api/providers/{name}/models`
  - Queries `<base_url>/v1/models` using the provider's `api_key` and `extra_headers`.
  - Normalizes the response into a sorted list of model descriptors: `{"models": [{"id": "qwen3-coder:30b", "name": "qwen3-coder:30b", "owned_by": "ollama", "supports_tools": true}]}`.
  - If the upstream provider returns capability metadata (e.g. OpenRouter `supported_parameters`), `supports_tools` reflects whether `"tools"` is supported.

#### FR-5: Extra Headers Support (Gateways & Custom Metadata)

- `ProviderConfig.extra_headers` accepts arbitrary string key-value pairs.
- When making requests (both model discovery and agent runs via the driver), the server injects these headers into outbound HTTP requests.
- Example use case: OpenRouter attribution headers:
  ```yaml
  extra_headers:
    HTTP-Referer: "https://kaos-control.local"
    X-Title: "kaos-control"
  ```
- Keys matching sensitive patterns (`auth`, `token`, `secret`, `password`) are masked if exposed in metadata endpoints.

#### FR-6: Frontend Provider Settings View (`ProviderSettingsView.vue`)

- Replaces `OllamaSettingsView.vue` with a general `ProviderSettingsView.vue`, accessible from application settings navigation.
- Displays a table/grid of configured providers showing:
  - Provider name, driver badge (e.g. `openai-compatible`), base URL.
  - Live health status badge (Green / Red / Unknown) and round-trip latency.
  - Discovered model count badge with a click-to-view modal.
  - Actions: "Test Connection", "Edit", "Delete".
- **Provider Form Modal (`ProviderForm.vue`):**
  - Name input (slug validated, disabled on edit).
  - Driver selector dropdown (`openai-compatible`, with future extensible slots).
  - Base URL input with quick preset buttons (e.g. "Ollama Local `:11434`", "llama.cpp `:7442`", "OpenRouter", "OpenAI").
  - API Key input (password field with show/hide toggle; shows placeholder `"••••••••"` when a key is already configured).
  - Extra Headers key-value editor (add/remove rows for custom headers).
  - "Test Connection" button inside modal to verify reachability and credentials before saving.
- **Delete Confirmation:**
  - Requires explicit confirmation; displays clear error message if backend returns `409 Conflict` due to active agent bindings.

#### FR-7: Frontend Agent Editor Integration (`AgentConfigForm.vue`)

- In `AgentConfigForm.vue`:
  - When configuring an agent using an API-driven role, a **Provider** dropdown selector displays all registered providers with their health indicators.
  - Selecting a provider automatically triggers a fetch of available models via `GET /api/providers/{name}/models`.
  - The **Model** input provides an autocomplete dropdown populated with discovered models, while still allowing manual text entry (for lazy-loaded or unlisted models).
  - A direct link ("Manage Providers $\to$") allows quick navigation to Provider Settings if no providers are configured.
  - Legacy Ollama instance / endpoint radio buttons are completely removed.

#### FR-8: Backend Agent Summary API

- `GET /api/p/{project}/agents` outputs agent summaries with:
  - `provider`: name of the assigned provider.
  - `model`: configured model name.
  - `driver`: resolved driver type (from the provider or agent).
- Confirms zero leakage of raw `api_key` or `auth_token` in agent list payloads.

---

### Non-Functional Requirements

#### NFR-1: Secret Hygiene & Standard Compliance

- Conforms strictly to [[standards/secrets-handling]]:
  - Provider `api_key` values must NEVER be returned in plaintext via REST API responses; they must be masked as `"***"`.
  - API keys must never appear in server logs, progress events, WebSocket broadcasts, or markdown artifacts.
  - Frontend stores and forms must only handle masked placeholders or user-entered new credentials.

#### NFR-2: Direct Server-to-Provider Communication Boundary

- All HTTP requests to provider endpoints (`/v1/models`, `/v1/chat/completions`, health checks) MUST originate from the Go backend.
- The browser frontend MUST NOT contact provider endpoints directly, preventing CORS issues and keeping API credentials safely on the server.

#### NFR-3: Non-Blocking Network Operations

- Provider health checks and model discovery requests must be strictly bounded with a 5-second timeout and executed asynchronously or with cancellation support, never blocking the main HTTP listener or worker goroutines.

#### NFR-4: Zero-Downtime Config Upgrades

- Upgrading from previous versions containing `ollama_instances` must be completely automated, preserving existing configurations (e.g. `Loki` at `http://leia.packsin.com:11434`) as valid `openai-compatible` providers without manual intervention.

---

## Acceptance Criteria

- [ ] `config.App` parses and validates `providers: []ProviderConfig` from `~/.kaos-control/config.yaml`, rejecting invalid URLs or duplicate names.
- [ ] Existing `ollama_instances` in `~/.kaos-control/config.yaml` are automatically migrated to `providers` with `driver: "openai-compatible"` on startup and persisted.
- [ ] `AgentConfig` in `lifecycle/config.yaml` accepts `provider` and `model` fields, and `config.ValidateProject` validates that referenced providers exist.
- [ ] Native `ollama` driver (`internal/agent/ollama.go`) and legacy `/api/ollama/instances` HTTP endpoints are removed.
- [ ] `GET /api/providers` returns all registered providers with `api_key` masked as `"***"`.
- [ ] `POST /api/providers` creates a new provider and persists to `~/.kaos-control/config.yaml`.
- [ ] `PUT /api/providers/{name}` updates fields and correctly preserves existing `api_key` when `"***"` is passed.
- [ ] `DELETE /api/providers/{name}` rejects deletion with HTTP 409 Conflict if any project agent references the provider.
- [ ] `GET /api/providers/{name}/health` returns latency and reachability status without blocking or exceeding 5-second timeout.
- [ ] `GET /api/providers/{name}/models` queries the provider's `/v1/models` endpoint using configured `api_key` and `extra_headers`, returning normalized model lists.
- [ ] `extra_headers` configured on a provider are properly sent in outbound HTTP headers to the provider.
- [ ] Frontend `ProviderSettingsView.vue` allows listing, adding, editing, deleting, and testing providers.
- [ ] Frontend `AgentConfigForm.vue` allows selecting a provider, displays provider health, and populates model autocomplete from the provider's model list.
- [ ] Provider API keys never appear in logs, API responses, or `lifecycle/` markdown files.
- [ ] Integration tests verify provider CRUD operations, migration from legacy config, conflict checks on deletion, and agent execution with provider-backed models.
- [ ] Lineage artifacts correctly link with `parent: lifecycle/ideas/provider-model-for-agents.md` and reference related epic artifacts [[open-provider-support]], [[llama-cpp-driver]], [[switch-provider]], and [[openrouter-llm-integration]].

---

## Resolved Questions

1. **Handling Missing Providers in Cloned Projects:**
   - When a project is cloned to a new machine where app-level `~/.kaos-control/config.yaml` does not have a provider named in `lifecycle/config.yaml`, should the agent status indicator flag "Provider missing" in the UI, and should `config.ValidateProject` treat this as a warning or a blocking error?
   - *Recommendation:* `config.ValidateProject` should log a warning and mark the agent as non-runnable (`ready_count: 0` / status `unconfigured`), while rendering a visual warning badge in the Agent panel pointing the operator to add the missing provider in Provider Settings.

> Proceed with recommendation

2. **Per-Provider Rate Limiting & Concurrency Caps:**
   - Should `ProviderConfig` optionally specify a `max_concurrent_requests` (e.g. for local servers with limited GPU VRAM or cloud accounts with low RPM tier limits)?
   - *Recommendation:* Defer per-provider concurrency caps to a follow-up enhancement; global `limits.max_concurrent_agents` and queue dispatching provide adequate protection for v1.

> Proceed with recommendation.
