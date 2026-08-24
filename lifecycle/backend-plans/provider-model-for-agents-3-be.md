---
title: "Backend Plan — Provider Entity Model & Management API"
type: plan-backend
status: approved
lineage: provider-model-for-agents
parent: lifecycle/requirements/provider-model-for-agents-2.md
created: "2026-08-25T07:35:00+10:00"
priority: normal
labels:
    - provider
    - driver
    - agent
    - config
    - backend
    - open-provider-support
release: KC-Release6
---

# Backend Plan — Provider Entity Model & Management API

This plan implements the backend foundation for [[provider-model-for-agents-2]]: establishing the **Provider entity** model in application configuration (`~/.kaos-control/config.yaml`), refactoring project agent configurations in `lifecycle/config.yaml` to reference providers via clean `{provider, model}` pairs, delivering the `/api/providers` REST API (with live health checks, model discovery, secret masking, and project reference conflict checks on deletion), supporting custom request headers (`extra_headers`), and outright removing the legacy native Ollama driver and endpoints while seamlessly migrating existing instances (e.g. `Loki`) on startup.

## Scope Boundary

- **In scope:**
  - `ProviderConfig` schema definition, parsing, validation, and atomic persistence in `internal/config`.
  - Automated startup migration from legacy `ollama_instances` to `providers` in app configuration.
  - `AgentConfig` refactoring to `{provider, model}` and updating project validation.
  - Removal of `internal/agent/ollama.go` and `/api/ollama/instances` routes.
  - New `/api/providers` route group in `internal/http/providers.go` (CRUD, `/health`, `/models`).
  - Active HTTP health probing (5-second timeout) and OpenAI-compatible `/v1/models` discovery.
  - Outbound `extra_headers` injection on provider HTTP requests.
  - Secret masking (`***`) on all provider API responses conforming to [[standards/secrets-handling]].
  - Agent summary API updates (`GET /api/p/{project}/agents`) exposing resolved provider/model pairs.

- **Out of scope:**
  - OpenAI-compatible tool-calling loop, SSE streaming parser, and native tool-call recovery (specified in [[open-provider-support-2]]).
  - Dynamic runtime provider failover and automatic switching (specified in [[switch-provider]]).
  - Frontend views and component forms (specified in [[provider-model-for-agents-4-fe]]).
  - Full integration test suite execution (specified in [[provider-model-for-agents-5-test]]).

## Architecture & Standards Conformance

- **Single self-contained binary:** Uses pure Go standard library packages (`net/http`, `encoding/json`, `time`, `context`) and existing dependencies (`gopkg.in/yaml.v3`, `github.com/go-chi/chi/v5`). No external daemons or CGO bindings.
- **Local filesystem is source of truth:** App config stored in `~/.kaos-control/config.yaml` and project configs in `lifecycle/config.yaml` ([[standards/index-is-a-cache]]).
- **Direct-served & Secrets hygiene:** Server initiates all outbound provider requests directly ([[decisions/adr-0001-no-header-based-client-ip-trust]]); provider `api_key` values are masked (`***`) on all REST API responses and kept out of logs and markdown artifacts ([[standards/secrets-handling]]).
- **Tool mediation:** Decoupling providers from agents preserves sandbox-mediated permission checks in `allowed_write_paths` ([[decisions/adr-0006-mediated-agent-driver-permission-model]], [[standards/filesystem-sandboxing]]).

## Cross-References

- [[provider-model-for-agents-2]] — Authoritative requirement artifact.
- [[provider-model-for-agents-4-fe]] — Frontend plan (ProviderSettingsView, ProviderForm, AgentConfigForm).
- [[provider-model-for-agents-5-test]] — Test plan.
- [[open-provider-support]] — Open provider support epic.
- [[openrouter-llm-integration]] — Gateway attribution headers.
- [[llama-cpp-driver]] — Local llama.cpp OpenAI-compatible endpoint integration.

---

## Milestone 1 — Provider Entity Model, App Configuration & Startup Migration

### Description

Define the `ProviderConfig` struct in `internal/config/config.go`, attach `Providers []ProviderConfig` to the `App` configuration struct, implement strict validation in `validateApp`, and implement automated startup migration from legacy `ollama_instances` to `providers`.

### Files to change

- **Edit** `internal/config/config.go`:
  - Define `ProviderConfig`:
    ```go
    type ProviderConfig struct {
        Name         string            `yaml:"name"`
        BaseURL      string            `yaml:"base_url"`
        Driver       string            `yaml:"driver"` // defaults to "openai-compatible"
        APIKey       string            `yaml:"api_key,omitempty"`
        ExtraHeaders map[string]string `yaml:"extra_headers,omitempty"`
    }
    ```
  - Update `App` struct:
    - Add `Providers []ProviderConfig `yaml:"providers,omitempty"`
    - Deprecate `OllamaInstances []OllamaInstance `yaml:"ollama_instances,omitempty"`
  - Update `validateApp`:
    - Validate each provider:
      - `name` matches slug format (`^[a-z0-9-]+$`, 2–64 characters).
      - `name` is unique across the `providers` list.
      - `base_url` is a valid, absolute `http` or `https` URL.
      - `driver` is non-empty and names a recognized API driver (e.g. `openai-compatible`).
  - Implement migration helper `migrateLegacyOllamaInstances(cfg *App, path string) error`:
    - If `len(cfg.OllamaInstances) > 0` and `len(cfg.Providers) == 0`:
      - Convert each `OllamaInstance` to a `ProviderConfig` with `Name = inst.Name`, `BaseURL = inst.BaseURL`, `APIKey = inst.APIKey`, `Driver = "openai-compatible"`.
      - Clear `cfg.OllamaInstances = nil`.
      - Save the updated config to `path` atomically using `SaveApp`.
      - Log migration via `slog.Info("app config: migrated legacy ollama instances to providers", "count", len(cfg.Providers))`.
  - Invoke `migrateLegacyOllamaInstances` inside `LoadApp` after unmarshaling.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test `internal/config/config_test.go`:
  - App config with valid `providers` passes `validateApp`.
  - App config with invalid slug names, duplicate names, or malformed URLs returns descriptive validation errors.
  - An app config file containing `ollama_instances` is automatically migrated to `providers` with `driver: "openai-compatible"` and persisted to disk.

---

## Milestone 2 — Agent Configuration Refactoring (`{provider, model}` Pair)

### Description

Refactor `AgentConfig` in `internal/config/config.go` so that agents declare a `provider` name and `model`. Remove deprecated connection fields from agent definitions and update project validation.

### Files to change

- **Edit** `internal/config/config.go`:
  - Update `AgentConfig`:
    - Add `Provider string `yaml:"provider,omitempty"`
    - Retain `Model string `yaml:"model,omitempty"`
    - Remove / deprecate `OllamaInstanceName`, `OllamaEndpoint`, `BaseURL`, `AuthToken` from agent fields.
  - Update internal `agentConfigRaw` and `AgentConfig.UnmarshalYAML` to map `provider` and `model`.
  - Update `validateProject`:
    - For agents with `driver: "openai-compatible"` or where `provider` is specified:
      - Require non-empty `model`.
      - When app configuration is available in context, check that `provider` matches a declared provider in app config. If missing, log a warning and mark the agent unconfigured (ready count 0) without failing whole-project load (per Resolved Question 1).
    - Standalone CLI drivers (`claude-code-cli`, `codex-cli`, `gemini-cli`, `shell-stub`) continue to validate their command/binary settings without requiring `provider`.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test `internal/config/config_test.go`:
  - Agent config with `provider: "openrouter"` and `model: "anthropic/claude-3.5-sonnet"` parses correctly.
  - Validation catches empty `model` on provider-backed agents.
  - CLI agents parse and validate without `provider`.

---

## Milestone 3 — Outright Removal of Native Ollama Driver & Surface

### Description

Remove the legacy native Ollama driver (`internal/agent/ollama.go`) and all references to `driver: "ollama"` in agent execution runners and project containers.

### Files to change

- **Delete** `internal/agent/ollama.go`.
- **Edit** `internal/agent/agent.go` and `internal/agent/runner.go`:
  - Remove `OllamaDriver` struct and factory dispatch.
  - Remove `OllamaInstanceName` and `OllamaEndpoint` fields from `agent.Run`.
  - Reject `driver: "ollama"` if encountered, instructing the user to configure a provider with `driver: "openai-compatible"`.
- **Edit** `internal/project/project.go`:
  - Remove any Ollama instance tracking or caching from project runtime containers.

### Acceptance criteria

- `go build ./... && go vet ./...` clean with no references to `internal/agent/ollama.go`.
- Agent runner tests compile and pass without Ollama driver dependencies.

---

## Milestone 4 — Provider Management REST API (`/api/providers`)

### Description

Replace legacy Ollama HTTP handlers with the `/api/providers` REST API supporting list, create, get, update, and delete operations with masked secret hygiene and project reference conflict checking.

### Files to change

- **New** `internal/http/providers.go` (replacing `internal/http/ollama.go`):
  - Helper `maskedProviders(providers []config.ProviderConfig) []map[string]any`:
    - Replaces non-empty `api_key` with `"***"`.
  - Helper `findProvider(providers []config.ProviderConfig, name string) int`.
  - Helper `(s *Server) findProviderReferences(name string) []string`:
    - Inspects all loaded `s.projects` and returns a list of references, e.g. `["project 'kaos-control' (agent 'requirements-analyst')"]`.
  - `handleListProviders` (`GET /api/providers`):
    - Returns `200 OK` with `{"providers": maskedProviders(s.appCfg.Providers)}`.
  - `handleCreateProvider` (`POST /api/providers`):
    - Role-gate: `requireAppRole(w, r, RolesDevopsOrAdmin...)`.
    - Unmarshals `{name, base_url, driver, api_key?, extra_headers?}`.
    - Validates slug format, base URL, and driver.
    - Checks for name uniqueness; returns `400 Bad Request` or `409 Conflict` on duplicate.
    - Appends to `s.appCfg.Providers`, saves atomically via `config.SaveApp`.
    - Returns `201 Created` with the single masked provider record.
  - `handleGetProvider` (`GET /api/providers/{name}`):
    - Looks up provider by name; returns `200 OK` with masked record or `404 Not Found`.
  - `handleUpdateProvider` (`PUT /api/providers/{name}`):
    - Role-gate: `requireAppRole(w, r, RolesDevopsOrAdmin...)`.
    - Unmarshals payload:
      - If `api_key` is `"***"` or omitted, existing stored `api_key` is preserved.
      - If `api_key` is a non-empty string, updates the key.
      - If `api_key` is explicitly `""`, clears the key.
    - Updates `base_url`, `driver`, `extra_headers`.
    - Persists atomically via `config.SaveApp`.
    - Returns `200 OK` with masked record.
  - `handleDeleteProvider` (`DELETE /api/providers/{name}`):
    - Role-gate: `requireAppRole(w, r, RolesDevopsOrAdmin...)`.
    - Calls `s.findProviderReferences(name)`.
    - If referenced, returns `409 Conflict` with `apiError("conflict", "cannot delete provider '...': in use by ...")`.
    - If unreferenced, removes from `s.appCfg.Providers`, saves via `config.SaveApp`, and returns `204 No Content`.
- **Edit** `internal/http/server.go`:
  - Mount `/api/providers` route group replacing `/api/ollama/instances`:
    ```go
    r.Route("/providers", func(r chi.Router) {
        r.Get("/", s.handleListProviders)
        r.Post("/", s.handleCreateProvider)
        r.Route("/{name}", func(r chi.Router) {
            r.Get("/", s.handleGetProvider)
            r.Put("/", s.handleUpdateProvider)
            r.Delete("/", s.handleDeleteProvider)
            r.Get("/health", s.handleProviderHealth)
            r.Get("/models", s.handleProviderModels)
        })
    })
    ```
- **Delete** `internal/http/ollama.go`.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test `internal/http/providers_test.go`:
  - `GET /api/providers` returns masked keys (`"***"`).
  - `POST /api/providers` creates and persists provider.
  - `PUT /api/providers/{name}` with `"***"` preserves existing secret; non-empty replaces; `""` clears.
  - `DELETE /api/providers/{name}` returns `409 Conflict` when referenced by an active project agent.
  - `DELETE /api/providers/{name}` removes unreferenced provider with `204 No Content`.

---

## Milestone 5 — Live Health Probing, Model Discovery & Extra Headers

### Description

Implement active HTTP health checks (`GET /api/providers/{name}/health`) and `/v1/models` discovery (`GET /api/providers/{name}/models`) with bounded 5-second timeouts, secret header injection, and `extra_headers` forwarding.

### Files to change

- **Edit** `internal/http/providers.go`:
  - `handleProviderHealth` (`GET /api/providers/{name}/health`):
    - Resolves provider from `s.appCfg.Providers`.
    - Constructs an HTTP `GET` request to `<base_url>/v1/models` (or root `<base_url>/` if probing generic base URL) with a 5-second `context.WithTimeout`.
    - Injects `Authorization: Bearer <api_key>` if configured.
    - Injects all entries from `extra_headers`.
    - Measures round-trip duration.
    - Returns `200 OK` with `{"ok": true, "latency_ms": latency}` or `{"ok": false, "error": err.Error()}`.
  - `handleProviderModels` (`GET /api/providers/{name}/models`):
    - Resolves provider from `s.appCfg.Providers`.
    - Constructs an HTTP `GET` request to `<base_url>/v1/models` with a 5-second timeout.
    - Injects `Authorization` and `extra_headers`.
    - Dispatches HTTP request; checks for 200 OK response.
    - Parses OpenAI-standard models response payload:
      ```json
      {
        "data": [
          { "id": "qwen3-coder:30b", "owned_by": "ollama", "supported_parameters": ["tools"] }
        ]
      }
      ```
    - Normalizes into sorted list:
      ```json
      {
        "models": [
          { "id": "qwen3-coder:30b", "name": "qwen3-coder:30b", "owned_by": "ollama", "supports_tools": true }
        ]
      }
      ```
    - Determines `supports_tools` based on upstream metadata (e.g. OpenRouter `supported_parameters` containing `"tools"` or default true for standard known chat-completion models).
    - Returns `200 OK` with models list, or `502 Bad Gateway` on upstream failure.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test in `internal/http/providers_test.go` using `httptest.Server`:
  - Health check accurately reports latency and reachability.
  - Health check cancels and returns `ok: false` if target server exceeds 5-second timeout.
  - Models discovery passes `api_key` in `Authorization` header and passes `extra_headers` (e.g. `HTTP-Referer`, `X-Title`).
  - Model discovery properly parses OpenAI `/v1/models` JSON responses into normalized descriptors.

---

## Milestone 6 — Agent Summary API & Project Agent Binding Integration

### Description

Update the agent summary endpoint `GET /api/p/{project}/agents` to expose the assigned `provider`, `model`, and resolved `driver`, ensuring complete secrets hygiene with zero leakage of API credentials.

### Files to change

- **Edit** `internal/http/agents.go`:
  - Update `agentSummary` struct:
    - Add `Provider string `json:"provider,omitempty"`
    - Add `Model string `json:"model,omitempty"`
    - Remove legacy fields (`OllamaInstanceName`, `OllamaEndpoint`, `AuthToken`, `BaseURL`).
  - Update `handleListAgents`:
    - Populate `Provider: ag.Provider` and `Model: ag.Model` from agent config.
    - If `ag.Driver` is empty and `ag.Provider` is set, resolve driver from the referenced provider in app config.
    - Ensure zero secret leakage in JSON output.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test `internal/http/agents_test.go`:
  - `GET /api/p/{project}/agents` outputs `provider`, `model`, and `driver`.
  - Response contains no raw `api_key` or `auth_token` fields.

---

## Risk Notes

- **Concurrent App Config Mutations:** App config modifications in `internal/http/providers.go` must acquire `s.appCfgMu.Lock()` and write to disk via `config.SaveApp` atomically (`.tmp` file + rename) to avoid race conditions and file corruption.
- **Provider Probe Timeouts:** Health probes and model queries must strictly enforce `context.WithTimeout(r.Context(), 5*time.Second)` so slow or hung remote endpoints never block Go worker goroutines.
- **Secret Redaction:** API keys must never be echoed in error messages or logs when health checks or model discovery requests fail.

## Verification (End-to-End)

1. `make lint` clean (`go vet` + `staticcheck`).
2. `make test-unit` clean (all unit tests in `internal/config` and `internal/http`).
3. `make test-integration` clean (executing provider integration tests in [[provider-model-for-agents-5-test]]).
