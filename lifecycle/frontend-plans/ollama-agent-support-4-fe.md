---
title: "Ollama Agent Support — Frontend Plan"
type: plan-frontend
status: draft
lineage: ollama-agent-support
parent: lifecycle/requirements/ollama-agent-support-2.md
---

# Ollama Agent Support — Frontend Plan

## Overview

Add frontend support for registering Ollama instances, browsing models, and creating/launching agents backed by Ollama. This builds on the [[ollama-agent-support]] backend plan's API endpoints and aligns with the existing Vue 3 + Pinia + TypeScript stack.

Cross-references: [[ollama-agent-support]] backend plan for API contracts, [[ollama-agent-support]] test plan for E2E coverage.

---

## Milestone 1 — API Client & Types

### Description

Add TypeScript types and API client functions for the new Ollama endpoints. All subsequent milestones depend on this.

### Files to change

- `web/src/types/api.ts` — Add `OllamaInstance`, `OllamaHealthResponse`, `OllamaModel` interfaces. Extend `AgentSummary` with optional `ollama_instance` and `ollama_endpoint` fields.
- `web/src/api/ollama.ts` — New file. Functions: `listInstances`, `createInstance`, `updateInstance`, `deleteInstance`, `getHealth`, `listModels`.

### Acceptance criteria

- [ ] `OllamaInstance` type has `name: string`, `base_url: string`, `api_key?: string` (always `"***"` when set, from API).
- [ ] `OllamaHealthResponse` type has `ok: boolean`, `latency_ms?: number`, `error?: string`.
- [ ] `OllamaModel` type has `name: string`, `size: number`.
- [ ] All API functions use the existing `api` client from `api/client.ts` and return typed promises.
- [ ] Error handling follows the existing `ApiError` pattern.

---

## Milestone 2 — Ollama Instances Pinia Store

### Description

Create a Pinia store to manage Ollama instance state: list, CRUD, health status, and model lists per instance.

### Files to change

- `web/src/stores/ollamaInstances.ts` — New file. Store: `useOllamaInstancesStore`.

### State shape

```typescript
{
  instances: OllamaInstance[]
  health: Map<string, OllamaHealthResponse>  // name → status
  models: Map<string, OllamaModel[]>          // name → model list
  loading: boolean
}
```

### Actions

- `fetchInstances()` — GET all instances.
- `createInstance(payload)` — POST, then refresh list.
- `updateInstance(name, payload)` — PUT, then refresh list.
- `deleteInstance(name)` — DELETE, then refresh list.
- `checkHealth(name)` — GET health, update `health` map.
- `fetchModels(name)` — GET models, update `models` map.
- `checkAllHealth()` — Check health of every instance in parallel.

### Acceptance criteria

- [ ] Store loads instances on first access.
- [ ] Health checks update reactively — components re-render when health status changes.
- [ ] Model lists are cached per instance and refreshable on demand.
- [ ] Errors from CRUD operations surface as thrown `ApiError` (callers display).

---

## Milestone 3 — Ollama Settings Panel

### Description

Add a settings panel accessible from the sidebar where users can add, edit, and remove Ollama instances. Display connection health status per instance.

### Files to change

- `web/src/views/project/OllamaSettingsView.vue` — New view. Instance list with add/edit/delete actions and health indicators.
- `web/src/components/ollama/OllamaInstanceForm.vue` — New component. Form for add/edit with fields: name, base_url, api_key. Used inside a modal or inline panel.
- `web/src/router/index.ts` — Add route `/p/:project/settings/ollama` → `OllamaSettingsView`.
- `web/src/views/project/WorkspaceView.vue` — Add sidebar navigation link to the new settings page (under existing "Config" link or a new "Settings" group).

### UI specification

- Instance list as a table/card list. Each row shows: name, base_url, health indicator (green dot / red dot / grey spinner), latency.
- "Add Instance" button opens `OllamaInstanceForm` in a modal.
- Each row has Edit (pencil icon) and Delete (trash icon) actions.
- Delete requires confirmation dialog; blocked if agents reference the instance.
- Health is checked on mount (`checkAllHealth`) and refreshable via a "Refresh" button.
- `api_key` input is a password field; shows `"***"` when editing an existing instance that has a key.

### Acceptance criteria

- [ ] User can add an Ollama instance by providing name and base_url.
- [ ] User can optionally provide an api_key (password-masked input).
- [ ] Health indicators update after the view loads and on manual refresh.
- [ ] Edit pre-fills form with current values (api_key shown as `***`).
- [ ] Delete shows a confirmation; fails with a message if agents reference the instance.
- [ ] Form validates: name required and unique, base_url required and valid URL.
- [ ] View is accessible from the sidebar and via direct URL.

---

## Milestone 4 — Agent Creation Driver Selector

### Description

Extend the agent creation/edit UI to support selecting Ollama as a driver type. When Ollama is selected, show instance and model dropdowns.

### Files to change

- `web/src/components/agent/AgentLaunchModal.vue` — Update `agentInputTypeMap` to include Ollama-backed agents (these use the same artifact type mapping based on role).
- `web/src/components/agent/RunAgentDialog.vue` — No changes needed; this dialog selects from pre-configured agents, which now may include Ollama agents.
- `web/src/components/agent/AgentPanelRow.vue` — Display driver type badge (e.g., "Ollama" vs "Claude Code") and model name for each agent.
- `web/src/views/project/AgentsRunsView.vue` — Show driver column in the agents table.

### Note on agent creation

Agents are currently defined in `lifecycle/config.yaml` and edited via the raw YAML config editor (`ProjectConfigView`). The requirement (FR-5) calls for a driver-type selector in an "agent creation/edit UI". Since no structured agent creation form exists today (agents are YAML-configured), this milestone focuses on:

1. Surfacing driver info in the existing agent listing UI.
2. Extending the YAML config editor guidance.
3. If a structured agent form is built (see Milestone 5), adding driver-aware fields there.

### Acceptance criteria

- [ ] Agent list/panel rows display the driver type (`Claude Code` or `Ollama`) as a badge or label.
- [ ] Ollama agent rows also display the instance name and model.
- [ ] Existing `claude-code-cli` agents render identically to before (no regression).

---

## Milestone 5 — Structured Agent Configuration Form (FR-5)

### Description

Build a structured agent creation/edit form as a modal or panel, replacing the need to hand-edit YAML for common agent configurations. The form is driver-aware: selecting "Ollama" reveals instance and model pickers populated from the backend.

### Files to change

- `web/src/components/agent/AgentConfigForm.vue` — New component. Fields: name, roles (multi-select), driver (radio: Claude Code | Ollama), model (text for Claude, dropdown for Ollama), ollama_instance (dropdown, shown when driver=ollama), ollama_endpoint (radio: chat | generate, shown when driver=ollama), allowed_write_paths (tag input), prompt_templates (per-role textarea), timeout_minutes, git_identity (name, email).
- `web/src/views/project/AgentsRunsView.vue` — Add "New Agent" button and "Edit" action per agent row, opening `AgentConfigForm`.
- `web/src/api/config.ts` — If needed, add helpers to PATCH agent entries within the project config.

### Acceptance criteria

- [ ] Selecting "Ollama" as driver type shows instance dropdown (populated from `useOllamaInstancesStore`).
- [ ] Instance dropdown shows health status indicator next to each instance name.
- [ ] After selecting an instance, model dropdown is populated by calling `fetchModels(instanceName)`.
- [ ] A "Refresh Models" button re-fetches the model list.
- [ ] Validation prevents saving if driver=ollama but instance or model is empty.
- [ ] Saving an agent writes to the project config via API (PUT config with updated agents list).
- [ ] Form works for both create (empty) and edit (pre-populated) modes.

---

## Milestone 6 — Progress & Run Display for Ollama Agents

### Description

Ensure the existing agent run UI (progress streaming, run detail modal, run list) works correctly with Ollama driver runs. Ollama progress events have a different shape than Claude Code stream-json events, so the `formatEvent` function needs updating.

### Files to change

- `web/src/stores/agents.ts` — Update `formatEvent()` to handle Ollama-shaped progress events (simpler: `started`, `output` with text content, `completed`, `error`).
- `web/src/components/agent/RunDetailModal.vue` — Ensure Ollama run output renders correctly (plain text output rather than tool-use JSON).

### Acceptance criteria

- [ ] Ollama run progress displays streamed text chunks in real time.
- [ ] `started` and `completed`/`error` events render with appropriate indicators (▸ prefix).
- [ ] Run detail modal shows full Ollama response text on completion.
- [ ] Claude Code runs continue to render identically (no regression in `formatEvent`).
- [ ] Error events from Ollama (timeout, connection failure) display the error message clearly.
