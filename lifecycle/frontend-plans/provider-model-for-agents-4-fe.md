---
title: "Frontend Plan — Provider Settings View & Agent Editor Integration"
type: plan-frontend
status: in-development
lineage: provider-model-for-agents
parent: lifecycle/requirements/provider-model-for-agents-2.md
created: "2026-08-25T07:35:00+10:00"
priority: normal
labels:
    - provider
    - driver
    - agent
    - config
    - frontend
    - open-provider-support
release: KC-Release6
---

# Frontend Plan — Provider Settings View & Agent Editor Integration

This plan implements the frontend UI and state management for [[provider-model-for-agents-2]]: delivering the **Provider Settings** interface (`ProviderSettingsView.vue` and `ProviderForm.vue`) to configure, inspect, test, and manage OpenAI-compatible API providers, updating Pinia stores and API clients to replace legacy Ollama surface, integrating provider selection and model autocomplete into the **Agent Editor** (`AgentConfigForm.vue`), and ensuring secret hygiene by handling masked keys.

## Scope Boundary

- **In scope:**
  - TypeScript types in `web/src/types/api.ts` for providers, health status, and discovered models.
  - API client module `web/src/api/providers.ts` replacing `web/src/api/ollama.ts`.
  - Pinia store `web/src/stores/providers.ts` replacing `web/src/stores/ollamaInstances.ts`.
  - View `web/src/views/project/ProviderSettingsView.vue` replacing `OllamaSettingsView.vue`.
  - Form component `web/src/components/provider/ProviderForm.vue` replacing `OllamaInstanceForm.vue`, supporting presets (Ollama, llama.cpp, OpenRouter, OpenAI), secret show/hide toggle, and `extra_headers` key-value table.
  - Agent Editor updates in `web/src/components/agent/AgentConfigForm.vue` with provider dropdown, live health indicator, "Manage Providers →" quick link, and discovered model autocomplete datalist.
  - Agent card indicators in `AgentPanelRow.vue` and `AgentsRunsView.vue`.
  - Navigation routing in `web/src/router/index.ts` and sidebar link in `web/src/components/layout/AppSidebar.vue`.

- **Out of scope:**
  - Backend provider endpoints, configuration persistence, and proxy health checks (specified in [[provider-model-for-agents-3-be]]).
  - OpenAI-compatible tool calling or SSE streaming handling (specified in [[open-provider-support-2]]).
  - Runtime failover status and automatic provider switching controls (specified in [[switch-provider]]).

## Architecture & Standards Conformance

- **Single self-contained binary & Embedded SPA:** Vue 3.5 SFCs, TypeScript, Pinia store architecture, and standard CSS conforming to [[decisions/adr-0004-embedded-spa-single-binary]].
- **Direct server-to-provider boundary:** Browser client only communicates with the Go backend API (`/api/providers`); never initiates direct network requests to third-party or local provider URLs.
- **Secrets handling:** API keys are never exposed in UI text or logs; forms handle masked placeholders (`••••••••` or `***`) when editing existing providers ([[standards/secrets-handling]]).

## Cross-References

- [[provider-model-for-agents-2]] — Requirement artifact.
- [[provider-model-for-agents-3-be]] — Backend plan (REST endpoints `/api/providers`).
- [[provider-model-for-agents-5-test]] — Test plan (Vitest component tests and E2E validation).
- [[open-provider-support]] — Open provider support epic.
- [[openrouter-llm-integration]] — Gateway attribution header configuration.

---

## Milestone 1 — TypeScript Types & Provider API Client Module

### Description

Define the TypeScript interfaces in `web/src/types/api.ts` and implement the API client functions in `web/src/api/providers.ts` to communicate with the backend `/api/providers` endpoints.

### Files to change

- **Edit** `web/src/types/api.ts`:
  - Define `ProviderConfig`:
    ```ts
    export interface ProviderConfig {
      name: string
      base_url: string
      driver: string
      api_key?: string
      extra_headers?: Record<string, string>
    }
    ```
  - Define `ProviderHealth`:
    ```ts
    export interface ProviderHealth {
      ok: boolean
      latency_ms?: number
      error?: string
    }
    ```
  - Define `DiscoveredModel`:
    ```ts
    export interface DiscoveredModel {
      id: string
      name: string
      owned_by?: string
      supports_tools?: boolean
    }
    ```
  - Update `AgentSummary` and `AgentConfig`:
    - Add `provider?: string`
    - Retain `model: string`
    - Remove `ollama_instance`, `ollama_endpoint`, `auth_token`, `base_url`.
  - Remove deprecated `OllamaInstance` interface.
- **New** `web/src/api/providers.ts` (replacing `web/src/api/ollama.ts`):
  - `listProviders()`: `GET /api/providers` -> returns `{ providers: ProviderConfig[] }`.
  - `getProvider(name: string)`: `GET /api/providers/${encodeURIComponent(name)}` -> returns `{ provider: ProviderConfig }`.
  - `createProvider(data: ProviderConfig)`: `POST /api/providers` -> returns `{ provider: ProviderConfig }`.
  - `updateProvider(name: string, data: Partial<ProviderConfig>)`: `PUT /api/providers/${encodeURIComponent(name)}` -> returns `{ provider: ProviderConfig }`.
  - `deleteProvider(name: string)`: `DELETE /api/providers/${encodeURIComponent(name)}` -> returns `{ ok: boolean, deleted: string }`.
  - `getProviderHealth(name: string)`: `GET /api/providers/${encodeURIComponent(name)}/health` -> returns `ProviderHealth`.
  - `getProviderModels(name: string)`: `GET /api/providers/${encodeURIComponent(name)}/models` -> returns `{ models: DiscoveredModel[] }`.
- **Delete** `web/src/api/ollama.ts`.

### Acceptance criteria

- `pnpm build` + `pnpm test` clean.
- Unit test `web/src/api/providers.spec.ts`:
  - Validates API client URL paths, HTTP methods, headers, and error handling.

---

## Milestone 2 — Pinia State Store (`stores/providers.ts`)

### Description

Create the Pinia `providers` store to manage reactive state for providers, live health status, cached model discovery results, loading states, and error notifications.

### Files to change

- **New** `web/src/stores/providers.ts` (replacing `web/src/stores/ollamaInstances.ts`):
  - State:
    - `providers = ref<ProviderConfig[]>([])`
    - `health = ref<Map<string, ProviderHealth>>(new Map())`
    - `models = ref<Map<string, DiscoveredModel[]>>(new Map())`
    - `loading = ref(false)`
    - `error = ref<string | null>(null)`
  - Actions:
    - `fetchProviders()`: calls `api.listProviders()`, updates `providers`.
    - `createProvider(data)`: calls `api.createProvider()`, appends to `providers`.
    - `updateProvider(name, data)`: calls `api.updateProvider()`, replaces provider in `providers`.
    - `deleteProvider(name)`: calls `api.deleteProvider()`, removes provider from `providers`, cleans health/models maps.
    - `checkHealth(name)`: calls `api.getProviderHealth()`, updates `health` map entry.
    - `checkAllHealth()`: concurrently triggers `checkHealth` for all providers in `providers`.
    - `fetchModels(name)`: calls `api.getProviderModels()`, updates `models` map entry.
- **Delete** `web/src/stores/ollamaInstances.ts`.

### Acceptance criteria

- `pnpm build` + `pnpm test` clean.
- Unit test `web/src/stores/providers.spec.ts`:
  - Store initializes with empty state.
  - `fetchProviders` loads provider list.
  - `checkHealth` updates health map.
  - `deleteProvider` cleans state and handles errors.

---

## Milestone 3 — Provider Settings View & Form Components

### Description

Create the Provider Settings view (`ProviderSettingsView.vue`) and form modal component (`ProviderForm.vue`) with preset buttons, extra headers editor, connection testing, and deletion conflict modal.

### Files to change

- **New** `web/src/views/project/ProviderSettingsView.vue` (replacing `OllamaSettingsView.vue`):
  - View layout with header: Title "Providers", "Refresh" button, "Add Provider" button.
  - Table / Card grid displaying:
    - Name (slug).
    - Driver badge (e.g. `openai-compatible`).
    - Base URL (monospace).
    - Health badge dot (Green = OK, Red = Error, Grey = Unknown) and latency string (e.g. `42 ms`).
    - Discovered models count badge (e.g. `12 models`) with click-to-view modal displaying model list with tool-support tags.
    - Action buttons: "Test Connection", "Edit", "Delete".
  - Empty state when no providers configured: "No providers configured. Click Add Provider to register one."
  - Delete modal:
    - Confirmation prompt: "Delete <name>? This cannot be undone."
    - Error display rendering 409 Conflict rejection messages (e.g. in use by agent).
- **New** `web/src/components/provider/ProviderForm.vue` (replacing `components/ollama/OllamaInstanceForm.vue`):
  - Name input (slug validated `^[a-z0-9-]+$`, disabled when editing).
  - Driver selector dropdown (`openai-compatible`).
  - Base URL input with quick preset buttons:
    - "Ollama Local (`http://localhost:11434`)"
    - "llama.cpp (`http://localhost:7442`)"
    - "OpenRouter (`https://openrouter.ai/api/v1`)"
    - "OpenAI (`https://api.openai.com/v1`)"
  - API Key input (password field with show/hide toggle; displays `"••••••••"` placeholder when editing a provider that has a configured key).
  - Extra Headers key-value editor:
    - Add/remove dynamic rows for custom headers (e.g. `HTTP-Referer`, `X-Title`).
  - "Test Connection" button inside form modal with live result indicator.
- **Delete** `web/src/components/ollama/OllamaInstanceForm.vue`.
- **Edit** `web/src/router/index.ts` and `web/src/components/layout/AppSidebar.vue`:
  - Replace `/settings/ollama` route and "Ollama" navigation link with `/settings/providers` and "Providers".

### Acceptance criteria

- `pnpm build` + `pnpm test` clean.
- Component test `web/src/views/project/ProviderSettingsView.spec.ts`:
  - Renders provider table and health dots correctly.
  - Clicking "Add Provider" opens `ProviderForm`.
  - Delete confirmation handles 409 Conflict error with visible explanation.
- Component test `web/src/components/provider/ProviderForm.spec.ts`:
  - Quick preset buttons populate base URL.
  - Adding/removing extra header rows updates form payload.
  - "Test Connection" calls health endpoint and displays result.

---

## Milestone 4 — Agent Editor Integration (`AgentConfigForm.vue`) & Agent Cards

### Description

Refactor `AgentConfigForm.vue` to select from registered providers, trigger automatic model fetching, provide discovered model autocomplete, and update agent panel indicators.

### Files to change

- **Edit** `web/src/components/agent/AgentConfigForm.vue`:
  - Replace `useOllamaInstancesStore` with `useProvidersStore`.
  - Remove legacy Ollama radio buttons and endpoint selectors (`ollama_instance`, `ollama_endpoint`).
  - For API drivers (or agents requiring providers):
    - **Provider selector dropdown:** lists all registered providers from `providersStore.providers` showing provider name and health dot indicator.
    - When no providers are registered: display an info banner with a link **"Manage Providers →"** navigating to `/settings/providers`.
    - On provider selection change: automatically trigger `providersStore.fetchModels(selectedProvider)`.
    - **Model input:** combobox / autocomplete input populated from `providersStore.models.get(selectedProvider)`:
      - Display model ID and tool support indicator (`tools: true`).
      - Support free-text entry for lazy-loaded or unlisted models.
- **Edit** `web/src/components/agent/AgentPanelRow.vue` and `web/src/views/project/AgentsRunsView.vue`:
  - Replace Ollama instance badges with Provider badge: `{provider} / {model}`.
  - If the assigned provider is not found in `providersStore`, render a warning badge ("Missing Provider").

### Acceptance criteria

- `pnpm build` + `pnpm test` clean.
- Component test `web/src/components/agent/AgentConfigForm.spec.ts`:
  - Selecting a provider fetches models and updates autocomplete list.
  - Submitting form emits `{ provider, model }` in agent payload.
  - Warning banner displays when no providers are configured.
- Component test `web/src/components/agent/AgentPanelRow.spec.ts`:
  - Displays provider and model badges.

---

## Risk Notes

- **Secrets in Frontend State:** Provider `api_key` values must never be stored in plain text if loaded from the server (`***` placeholder used). Forms must only send `api_key` when the user explicitly enters a new secret.
- **Asynchronous Model Discovery:** If a provider is slow or offline, model discovery must fail gracefully without locking the Agent Editor form; manual model input must remain enabled at all times.

## Verification (End-to-End)

1. `pnpm lint` clean.
2. `pnpm build` clean.
3. `pnpm test` (Vitest suite) clean.
4. Manual smoke verification against running backend:
   - Navigate to `/settings/providers` -> add a provider -> click Test Connection -> view discovered models.
   - Open Agent Editor -> select provider -> autocomplete model -> save agent.
