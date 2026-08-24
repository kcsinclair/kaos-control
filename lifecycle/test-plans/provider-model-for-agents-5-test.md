---
title: "Test Plan — Provider Entity Model, Management API & Agent Integration"
type: plan-test
status: done
lineage: provider-model-for-agents
parent: lifecycle/requirements/provider-model-for-agents-2.md
created: "2026-08-25T07:35:00+10:00"
priority: normal
labels:
    - provider
    - driver
    - agent
    - config
    - test
    - open-provider-support
release: KC-Release6
---

# Test Plan — Provider Entity Model, Management API & Agent Integration

This plan defines the automated test suite to verify [[provider-model-for-agents-2]] against the backend implementation [[provider-model-for-agents-3-be]] and frontend implementation [[provider-model-for-agents-4-fe]]. It covers provider entity configuration in `~/.kaos-control/config.yaml`, startup migration from legacy `ollama_instances`, `{provider, model}` agent configuration, the `/api/providers` REST API (CRUD, health checks, `/v1/models` discovery, secret masking, extra headers, and deletion conflict checks), removal of legacy Ollama endpoints and drivers, and frontend component tests.

## Scope Boundary

- **In scope:**
  - Unit tests for app configuration validation, provider schema, and legacy Ollama migration in `internal/config`.
  - Unit tests for agent configuration parsing and project validation with `{provider, model}` pairs.
  - Integration tests in `tests/` for `/api/providers` REST API (CRUD, role-gating, conflict on deletion).
  - Integration tests with mock upstream HTTP servers for `/health` probe (latency, timeout) and `/models` discovery (parsing, headers injection).
  - Secret masking assertions across all API responses, logs, and artifacts ([[standards/secrets-handling]]).
  - Deprecation and removal tests verifying legacy Ollama endpoints and drivers are completely absent.
  - Frontend Vitest suite for `ProviderSettingsView.vue`, `ProviderForm.vue`, and `AgentConfigForm.vue`.

- **Out of scope:**
  - OpenAI-compatible tool-calling streaming loop and tool recovery tests (specified in [[open-provider-support-2]]).
  - Dynamic runtime provider failover tests (specified in [[switch-provider]]).

## Architecture & Standards Conformance

- **Single self-contained binary:** Tests run via `go test` and `vitest` with zero external process dependencies or CGO requirements.
- **Index is a cache & Local filesystem source of truth:** Verifies that providers persist to `~/.kaos-control/config.yaml` and agents persist to `lifecycle/config.yaml` ([[standards/index-is-a-cache]]).
- **Secrets hygiene standard:** Tests explicitly assert that API keys are masked (`***`) on all API endpoints and never leaked in error logs ([[standards/secrets-handling]]).
- **Direct-served & Sandbox bounds:** Verifies all provider calls originate from Go server and agent tool mediation remains active ([[standards/filesystem-sandboxing]]).

## Cross-References

- [[provider-model-for-agents-2]] — Authoritative requirement artifact.
- [[provider-model-for-agents-3-be]] — Backend plan.
- [[provider-model-for-agents-4-fe]] — Frontend plan.
- [[open-provider-support]] — Open provider support epic.

---

## Milestone 1 — App Configuration Parsing, Provider Validation & Startup Migration Tests (FR-1, NFR-4)

### Description

Verify that `config.App` parses and validates `providers: []ProviderConfig`, rejects invalid formats or duplicate names, and automatically migrates legacy `ollama_instances` to `providers` with `driver: "openai-compatible"`.

### Files to change

- **New** `internal/config/provider_config_test.go`:
  - `TestAppConfig_ValidProviders`:
    - Tests parsing valid YAML with multiple providers (local Ollama, OpenRouter with `extra_headers`, OpenAI with `api_key`).
    - Asserts fields parse correctly into `ProviderConfig` structs.
  - `TestAppConfig_ValidationErrors`:
    - Table-driven tests asserting errors on:
      - Invalid slug names (e.g. `"My Provider"`, `"provider_1"`, `""`).
      - Duplicate provider names (e.g. two providers named `"ollama-local"`).
      - Invalid base URLs (e.g. `"ftp://example.com"`, `"/relative/path"`).
      - Empty driver strings.
  - `TestAppConfig_LegacyOllamaMigration`:
    - Writes a temporary `config.yaml` containing legacy `ollama_instances: [{name: "Loki", base_url: "http://leia.packsin.com:11434", api_key: "secret-123"}]` and no `providers`.
    - Calls `config.LoadApp`.
    - Asserts `app.Providers` contains 1 provider with `Name == "Loki"`, `BaseURL == "http://leia.packsin.com:11434"`, `APIKey == "secret-123"`, and `Driver == "openai-compatible"`.
    - Asserts `app.OllamaInstances` is empty.
    - Reads the persisted file from disk and asserts `providers:` is written and `ollama_instances:` is absent.

### Acceptance criteria

- `go test -v ./internal/config/...` passes.
- Legacy configuration migration is completely automatic and idempotent on subsequent loads.

---

## Milestone 2 — Agent Configuration `{provider, model}` & Project Validation Tests (FR-2, FR-8)

### Description

Verify that project agent definitions parse `{provider, model}` pairs, that validation handles valid and invalid provider references, and that the agent summary endpoint exposes resolved pairs with zero credential leakage.

### Files to change

- **New** `internal/config/agent_provider_test.go`:
  - `TestAgentConfig_ProviderModelPair`:
    - Unmarshals `AgentConfig` from YAML with `provider: "openrouter"` and `model: "anthropic/claude-3.5-sonnet"`.
    - Asserts `a.Provider == "openrouter"` and `a.Model == "anthropic/claude-3.5-sonnet"`.
  - `TestProjectConfig_ValidateAgents`:
    - Agent with `provider` and non-empty `model` passes validation.
    - Agent with missing `model` fails validation with descriptive error.
    - CLI agents (`claude-code-cli`, `shell-stub`) pass without `provider`.
    - Standalone project validation on missing provider logs a warning and marks the agent unconfigured without panicking or halting load.
- **New** `tests/agent_summary_test.go`:
  - Calls `GET /api/p/{project}/agents`.
  - Asserts response contains `provider`, `model`, and resolved `driver`.
  - Asserts response payload contains no `api_key` or `auth_token` fields.

### Acceptance criteria

- `go test -v ./internal/config/...` passes.
- `go test -v ./tests -run TestAgentSummary` passes.

---

## Milestone 3 — Provider Management REST API CRUD & Deletion Conflict Tests (FR-4, NFR-1)

### Description

Verify all `/api/providers` REST operations: listing with masked keys, creating new providers, updating fields with key preservation, role-based authorization, and deleting with active agent reference checks.

### Files to change

- **New** `tests/provider_api_test.go`:
  - `TestProviderAPI_ListMasked`:
    - Seeds app config with a provider having `api_key: "real-secret-key"`.
    - Calls `GET /api/providers`.
    - Asserts response status is `200 OK` and `api_key` is `"***"`.
  - `TestProviderAPI_Create`:
    - Calls `POST /api/providers` with `{name: "local-llama", base_url: "http://localhost:7442", driver: "openai-compatible", api_key: "sk-123"}`.
    - Asserts `201 Created` with masked key.
    - Asserts duplicate name returns `400 Bad Request` or `409 Conflict`.
    - Re-reads app config from disk; asserts plain key is persisted.
  - `TestProviderAPI_Update_KeyPreservation`:
    - Calls `PUT /api/providers/local-llama` with `api_key: "***"` and new `base_url`.
    - Asserts `200 OK`; verifies on disk that original `api_key` was preserved.
    - Calls `PUT /api/providers/local-llama` with `api_key: "new-secret"`; verifies key was updated.
    - Calls `PUT /api/providers/local-llama` with `api_key: ""`; verifies key was cleared.
  - `TestProviderAPI_Delete_ConflictCheck`:
    - Configures project agent `backend-developer` with `provider: "local-llama"`.
    - Calls `DELETE /api/providers/local-llama`.
    - Asserts `409 Conflict` and error body contains `"in use by project ... (agent 'backend-developer')"`.
    - Reconfigures agent to another provider.
    - Calls `DELETE /api/providers/local-llama`.
    - Asserts `204 No Content`; verifies provider is removed from app config.
  - `TestProviderAPI_AuthRoles`:
    - Non-admin/non-devops user attempting `POST`, `PUT`, `DELETE` receives `403 Forbidden`.

### Acceptance criteria

- `go test -v ./tests -run TestProviderAPI` passes.
- Secret masking and key preservation rules strictly verified.

---

## Milestone 4 — Live Health Probe & Model Discovery Tests (FR-4, FR-5, NFR-2, NFR-3)

### Description

Verify `/api/providers/{name}/health` and `/api/providers/{name}/models` using a mock upstream HTTP server, asserting latency calculation, 5-second timeout bounding, auth header injection, `extra_headers` forwarding, and model list normalization.

### Files to change

- **New** `tests/provider_probe_test.go`:
  - `TestProviderHealth_Success`:
    - Creates `httptest.Server` returning `200 OK` on `/v1/models`.
    - Registers provider pointing to test server.
    - Calls `GET /api/providers/{name}/health`.
    - Asserts `200 OK` with `{"ok": true, "latency_ms": N}` where `N >= 0`.
  - `TestProviderHealth_Timeout`:
    - Creates `httptest.Server` with a 6-second sleep.
    - Calls `GET /api/providers/{name}/health`.
    - Asserts response finishes in ~5 seconds with `{"ok": false, "error": "..."}`.
  - `TestProviderModels_DiscoveryAndHeaders`:
    - Creates `httptest.Server` that inspects request headers and returns OpenAI `/v1/models` JSON:
      ```json
      {
        "data": [
          { "id": "qwen-coder", "owned_by": "ollama", "supported_parameters": ["tools"] },
          { "id": "llama-3-8b", "owned_by": "meta" }
        ]
      }
      ```
    - Registers provider with `api_key: "secret-abc"` and `extra_headers: {"HTTP-Referer": "https://test.local", "X-Title": "kaos-test"}`.
    - Calls `GET /api/providers/{name}/models`.
    - Asserts test server received:
      - `Authorization: Bearer secret-abc`
      - `HTTP-Referer: https://test.local`
      - `X-Title: kaos-test`
    - Asserts response JSON:
      ```json
      {
        "models": [
          { "id": "llama-3-8b", "name": "llama-3-8b", "owned_by": "meta", "supports_tools": true },
          { "id": "qwen-coder", "name": "qwen-coder", "owned_by": "ollama", "supports_tools": true }
        ]
      }
      ```

### Acceptance criteria

- `go test -v ./tests -run TestProviderProbe` passes.
- Timeout bounding at 5 seconds confirmed.
- Request header forwarding and response normalization verified.

---

## Milestone 5 — Removal Verification & Clean Deprecation Sweep (FR-3)

### Description

Assert that all legacy Ollama routes, drivers, and stores are completely deleted and that references to `driver: "ollama"` are rejected.

### Files to change

- **New** `tests/ollama_removal_test.go`:
  - Calls `GET /api/ollama/instances` -> asserts `404 Not Found`.
  - Loads project with agent having `driver: "ollama"` -> asserts validation error instructing user to use `driver: "openai-compatible"`.
  - Greps backend and frontend source directories to ensure no lingering imports of `internal/agent/ollama.go` or `api/ollama.ts`.

### Acceptance criteria

- `go test -v ./tests -run TestOllamaRemoval` passes.
- No dead code or orphaned Ollama endpoints remain.

---

## Milestone 6 — Frontend Vitest Component Tests (FR-6, FR-7)

### Description

Implement Vitest unit and component tests for `ProviderSettingsView.vue`, `ProviderForm.vue`, `AgentConfigForm.vue`, and the Pinia `providers` store.

### Files to change

- **New** `web/src/stores/providers.spec.ts`:
  - Tests `fetchProviders`, `checkHealth`, `fetchModels`, and `deleteProvider` actions.
- **New** `web/src/views/project/ProviderSettingsView.spec.ts`:
  - Tests rendering provider table, health badges, latency display, and modal trigger.
  - Tests delete confirmation and rendering 409 Conflict error message.
- **New** `web/src/components/provider/ProviderForm.spec.ts`:
  - Tests form preset buttons (Ollama, llama.cpp, OpenRouter, OpenAI).
  - Tests adding and removing extra header rows.
  - Tests secret field mask placeholder behavior.
  - Tests "Test Connection" button invocation.
- **New** `web/src/components/agent/AgentConfigForm.spec.ts`:
  - Tests provider dropdown population.
  - Tests automatic model list fetching on provider change.
  - Tests model autocomplete input and free-text entry.
  - Tests "Manage Providers →" shortcut link when no providers exist.

### Acceptance criteria

- `pnpm test` passes all tests cleanly with zero regressions.

---

## Test Data & Fixtures

- **Legacy App Config Fixture:** `config.yaml` with `ollama_instances` block for migration testing.
- **OpenAI-Compatible Mock Server:** `httptest.Server` responding to `/v1/models` with OpenAI standard schemas.
- **Gateway Attribution Headers:** `extra_headers` mapping `{"HTTP-Referer": "https://kaos-control.local", "X-Title": "kaos-control"}`.
- **Test Users:** devops/admin user fixtures for mutation tests, and standard viewer user fixture for 403 Forbidden authorization tests.

## Verification Sweep (End-to-End)

1. `make test-unit` clean (all unit tests in `internal/config`, `internal/http`, `internal/agent`).
2. `make test-integration` clean (all `tests/provider_*_test.go`).
3. `pnpm test` clean (all frontend Vitest suites).
4. All acceptance criteria in [[provider-model-for-agents-2]] verified.
