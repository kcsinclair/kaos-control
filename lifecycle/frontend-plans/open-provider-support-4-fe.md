---
title: "Frontend Plan: OpenAI-Compatible Agent Driver (Tool-Calling)"
type: plan-frontend
status: approved
lineage: open-provider-support
parent: lifecycle/requirements/open-provider-support-2.md
created: "2026-08-25T07:10:33+10:00"
---

# Frontend Plan: OpenAI-Compatible Agent Driver (Tool-Calling)

Parent: [[open-provider-support-2]].

This plan covers the Vue 3 / TypeScript frontend implementation for the
[[llama-cpp-driver]] requirement and the provider settings surface delivered by
the [[open-provider-support]] epic. Backend implementation is in
[[open-provider-support-3-be]] and test coverage in [[open-provider-support-5-test]].

---

## Milestone 1 — TypeScript Types & Provider API Client

### Description

Define TypeScript interfaces for Provider records, model discovery, and agent
configuration extensions. Implement the API client module for provider management
and probing (replacing the Ollama API client).

### Files to change

- **`web/src/types/api.ts`**:
  - Define `Provider` interface:
    ```typescript
    export interface Provider {
      name: string
      base_url: string
      driver: string
      api_key?: string
      extra_headers?: Record<string, string>
    }
    ```
  - Define `ProviderModel` and `ProviderProbeResult`:
    ```typescript
    export interface ProviderModel {
      id: string
      name?: string
      supports_tools?: boolean
    }

    export interface ProviderProbeResult {
      ok: boolean
      error?: string
      models: ProviderModel[]
    }
    ```
  - Update `AgentConfig` and `AgentSummary` types to include `provider?: string`, `model?: string`, and `max_tool_iterations?: number`.
  - Add `'openai-compatible'` to supported driver types.

- **`web/src/api/providers.ts`** (new file):
  - Implement API client functions:
    - `getProviders(): Promise<Provider[]>`
    - `saveProvider(provider: Provider): Promise<Provider>`
    - `deleteProvider(name: string): Promise<void>`
    - `testProvider(name: string): Promise<ProviderProbeResult>`
  - Deprecate/remove `web/src/api/ollama.ts`.

### Acceptance criteria

- [ ] All TypeScript types for providers and agent configurations are defined and exported.
- [ ] `web/src/api/providers.ts` implements CRUD and probe API calls against `/api/providers`.
- [ ] `pnpm exec vue-tsc --noEmit` compiles without type errors.

---

## Milestone 2 — Provider Settings View & Pinia Store

### Description

Create a centralized Provider Settings view (replacing `OllamaSettingsView.vue`)
allowing operators to register and configure local servers (llama.cpp, Ollama)
and cloud providers (OpenAI, OpenRouter, Groq, Together). Implement a Pinia store
to manage provider state and test connections (FR-2, Resolved Question 6).

### Files to change

- **`web/src/stores/providers.ts`** (new file):
  - State: `providers: Ref<Provider[]>`, `loading: Ref<boolean>`, `probeResults: Ref<Map<string, ProviderProbeResult>>`.
  - Actions: `fetchProviders()`, `saveProvider(provider)`, `deleteProvider(name)`, `probeProvider(name)`.

- **`web/src/views/project/ProviderSettingsView.vue`** (new file, replaces `OllamaSettingsView.vue`):
  - List configured providers with status indicators, base URLs, and configured drivers.
  - Modal/Form to add and edit providers:
    - `Name`: required unique identifier.
    - `Base URL`: required endpoint URL (e.g. `http://leia.packsin.com:7442`).
    - `API Key`: optional masked input field with visibility toggle.
    - `Extra Headers`: key-value map editor (e.g. for `HTTP-Referer`, `X-Title` on OpenRouter).
  - "Test Connection" button: invokes probe endpoint, displays reachable status, latency, and available models.
  - Quick-preset buttons for common targets (llama.cpp local/remote, Ollama, OpenRouter, OpenAI).

- **`web/src/router/index.ts`**:
  - Update route definition to load `ProviderSettingsView.vue` at `/settings/providers` (aliasing/replacing `/settings/ollama`).

- **`web/src/views/project/OllamaSettingsView.vue`**:
  - Remove or replace with redirect to `ProviderSettingsView.vue`.

### Acceptance criteria

- [ ] Providers can be created, edited, deleted, and listed in the UI.
- [ ] API keys are displayed masked with an option to reveal while typing.
- [ ] Test Connection button queries the provider and renders detected models.
- [ ] Navigation and router paths correctly route to the new Provider Settings view.

---

## Milestone 3 — Agent Configuration & Launcher Updates

### Description

Update the agent editor and launch dialogues to support the `openai-compatible`
driver, provider selection, dynamic model picker, and iteration cap settings.

### Files to change

- **`web/src/components/agent/AgentConfigForm.vue`**:
  - Add `openai-compatible` option in the Driver dropdown.
  - When `driver == 'openai-compatible'` (or any driver using providers):
    - Render Provider dropdown populated from `providersStore.providers`.
    - Render Model selector with autocomplete from the selected provider's probed models list.
    - Render `Max Tool Iterations` numeric input (with placeholder `25`).
  - Validate required fields before submission.

- **`web/src/components/agent/AgentPanelRow.vue`**:
  - Add badge styling for `openai-compatible`:
    ```css
    .driver-badge[data-driver="openai-compatible"] {
      background: #d1fae5; /* emerald-100 */
      color: #065f46;      /* emerald-800 */
    }
    ```

- **`web/src/views/project/AgentsRunsView.vue`**:
  - Ensure driver badges render cleanly for `openai-compatible` runs.

### Acceptance criteria

- [ ] AgentConfigForm enables selecting `openai-compatible` driver, choosing a provider, and picking a model.
- [ ] `max_tool_iterations` can be configured per agent.
- [ ] Emerald driver badge displays in agent roster and run history for `openai-compatible` agents.

---

## Milestone 4 — Multi-turn Live Progress & Tool-Call Timeline

### Description

Enhance the live run stream view and run detail modal to render multi-turn
tool-calling progress events, tool arguments, tool outputs, and native recovery
notices in real time (FR-5a, FR-6).

### Files to change

- **`web/src/stores/agents.ts`**:
  - Update event parsing logic to recognize multi-turn tool execution deltas (`tool_calls`, `tool` results, `finish_reason`).
  - Accumulate turn-by-turn history on active run records.

- **`web/src/components/agent/RunDetailModal.vue`**:
  - Render an interactive turn timeline:
    - User/System prompt turn.
    - Assistant reasoning / tool invocation turns (`read_file`, `write_file`, `list_dir`, `grep`).
    - Tool result payloads (with collapsible code blocks).
    - Recovered tool-call notice badge (highlighting when FR-5a recovery was triggered).
    - Final assistant completion message.
  - Display TTFT (Time to First Token) metric in the run header metadata.

- **`web/src/views/project/AgentsRunsView.vue`**:
  - Format tool call progress events in the expanded run row.

### Acceptance criteria

- [ ] Live and historical runs show step-by-step tool execution turns and outputs.
- [ ] Recovered native tool calls (FR-5a) are visually noted in the run log.
- [ ] TTFT is displayed accurately in run metadata.
- [ ] No regression in existing Claude or Ollama run log views.
