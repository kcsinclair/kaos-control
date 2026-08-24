---
title: "Frontend Plan — Provider Failover UI, Switching Drawer & Agent Controls"
type: plan-frontend
status: approved
lineage: switch-provider
parent: lifecycle/requirements/switch-provider-2.md
created: "2026-08-25T08:50:00+10:00"
priority: high
labels:
    - agent
    - provider
    - frontend
    - vue
    - reliability
    - open-provider-support
release: KC-Release6
---

# Frontend Plan — Provider Failover UI, Switching Drawer & Agent Controls

This plan implements the frontend user interface and reactive state management for [[switch-provider-2]], connecting to the backend APIs and WebSocket events defined in [[switch-provider-3-be]]. It delivers global visibility into runtime provider failover state, an amber navigation header badge with recovery alerts, an interactive Provider Failover modal/drawer with live primary health indicators, individual and batch "Restore Primary" actions, provider template switching presets, agent panel fallback status badges, and a "Switch Provider & Resume" affordance on the queue pause banner.

## Scope Boundary

- **In scope:**
  - TypeScript types and REST client functions in `web/src/api/providerSwitch.ts`.
  - Pinia store `useProviderSwitchStore` in `web/src/stores/providerSwitch.ts` handling REST polling, cached state, and live WebSocket event streams (`provider.switched`, `provider.restored`, `provider.primary_recovered`, `config.reloaded`).
  - Navigation header alert badge in `web/src/components/layout/AppHeader.vue`.
  - Global `ProviderFailoverModal.vue` drawer showing active failover agents, original primary targets, live primary health status, "Restore All" button, and provider template presets.
  - Quick manual switching dialog `SwitchProviderModal.vue`.
  - Fallback status badges and "Restore Primary" buttons in `web/src/components/agent/AgentPanelRow.vue`.
  - Fallback provider and model configuration fields in `web/src/components/agent/AgentConfigForm.vue`.
  - "Switch Provider & Resume" action button in `web/src/components/queue/QueuePauseBanner.vue`.
  - Vitest component and store tests under `web/src/` and `tests/web/`.

- **Out of scope:**
  - Provider CRUD forms and app-level provider configuration (implemented in [[provider-model-for-agents-4-fe]]).
  - Backend REST API routes and AST config mutations (implemented in [[switch-provider-3-be]]).
  - Full end-to-end integration test execution (defined in [[switch-provider-5-test]]).

## Architecture & Standards Conformance

- **Single self-contained binary:** Pure Vue 3.5 SFCs, TypeScript, Pinia, and Lucide icons bundled via Vite into `web/dist/` and embedded in the Go binary.
- **Direct-served & Secrets hygiene:** Zero plaintext API keys or credentials exposed in frontend state, components, or network payloads ([[standards/secrets-handling]]).
- **Realtime responsiveness:** Reactive WebSocket subscriptions update UI states immediately (<100 ms) upon backend failover decisions without requiring manual page reloads.

## Cross-References

- [[switch-provider-2]] — Authoritative requirement artifact.
- [[switch-provider-3-be]] — Backend plan defining REST endpoints and WS events.
- [[switch-provider-5-test]] — Test plan covering frontend and integration suites.
- [[provider-model-for-agents-4-fe]] — Provider settings and agent form baseline.
- [[agent-rate-limit-queue-4-fe]] — Queue view and pause banner baseline.

---

## File Summary

```
web/src/
├── api/
│   └── providerSwitch.ts             NEW — REST client wrappers
├── stores/
│   └── providerSwitch.ts             NEW — Pinia store with WS event subscriptions
├── components/
│   ├── layout/
│   │   └── AppHeader.vue              EDIT — add Failover Active badge & drawer trigger
│   ├── provider/
│   │   ├── ProviderFailoverModal.vue  NEW — drawer/modal for active failovers
│   │   └── ProviderTemplateMenu.vue   NEW — preset template selector dropdown
│   ├── agent/
│   │   ├── AgentPanelRow.vue          EDIT — render fallback badges & Restore button
│   │   ├── AgentConfigForm.vue        EDIT — add Fallback Provider & Model fields
│   │   └── SwitchProviderModal.vue    NEW — quick manual switch dialog
│   └── queue/
│       └── QueuePauseBanner.vue       EDIT — add "Switch Provider & Resume" button
└── types/
    └── providerSwitch.ts             NEW — TypeScript interfaces
```

---

## Milestone 1 — REST Client, TypeScript Interfaces & Pinia Store

### Description

Create the TypeScript interfaces and REST client in `web/src/api/providerSwitch.ts` alongside the Pinia store `useProviderSwitchStore` in `web/src/stores/providerSwitch.ts`. The store maintains project failover status, handles mutations, and reacts dynamically to WebSocket events.

### Types & Schema (`web/src/types/providerSwitch.ts`)

```ts
export interface FailoverAgent {
  agent: string
  is_failover: boolean
  primary_provider?: string
  primary_model?: string
  active_provider: string
  active_model: string
  fallback_provider?: string
  fallback_model?: string
  switched_at?: string
  reason?: string
  primary_healthy?: boolean
}

export interface FailoverStatus {
  failover_active: boolean
  agents: FailoverAgent[]
}

export interface ProviderTemplate {
  name: string
  description?: string
  agents: Record<string, { provider: string; model: string }>
}
```

### Files to change

- **New** `web/src/types/providerSwitch.ts`: Defines `FailoverAgent`, `FailoverStatus`, `ProviderTemplate`, `SwitchPayload`, `ApplyTemplatePayload`.
- **New** `web/src/api/providerSwitch.ts`:
  - `getFailoverStatus(project: string)`: `GET /api/p/{project}/provider-switch/status`
  - `switchAgentProvider(project: string, agent: string, payload: SwitchPayload)`: `POST /api/p/{project}/agents/{name}/switch-provider`
  - `restoreAgentProvider(project: string, agent: string)`: `POST /api/p/{project}/agents/{name}/restore-provider`
  - `switchAllProviders(project: string, payload: { from_provider: string; to_provider: string; model?: string })`: `POST /api/p/{project}/provider-switch/switch-all`
  - `restoreAllProviders(project: string)`: `POST /api/p/{project}/provider-switch/restore-all`
  - `listProviderTemplates(project: string)`: `GET /api/p/{project}/provider-templates`
  - `applyProviderTemplate(project: string, template: string)`: `POST /api/p/{project}/provider-templates/apply`
- **New** `web/src/stores/providerSwitch.ts`:
  - State: `status: FailoverStatus`, `templates: ProviderTemplate[]`, `loading: boolean`, `recoveredProviders: string[]`.
  - Getters:
    - `isFailoverActive`: `status.failover_active`
    - `failoverCount`: number of agents with `is_failover === true`
    - `failoverAgents`: list of agents with `is_failover === true`
    - `hasRecoveredPrimary`: boolean indicating if any primary provider has recovered
  - Actions:
    - `fetchStatus(project: string)`
    - `fetchTemplates(project: string)`
    - `switchAgent(project, agent, payload)`
    - `restoreAgent(project, agent)`
    - `restoreAll(project)`
    - `applyTemplate(project, template)`
  - WebSocket Handler:
    - `provider.switched` $\to$ updates matching agent in `status.agents`, sets `failover_active = true`.
    - `provider.restored` $\to$ clears `is_failover` on matching agent; if none left, sets `failover_active = false`.
    - `provider.primary_recovered` $\to$ adds provider name to `recoveredProviders` and updates `primary_healthy = true`.
    - `config.reloaded` $\to$ triggers `fetchStatus(project)` and `fetchTemplates(project)`.

### Acceptance criteria

- Vitest unit tests in `web/src/stores/__tests__/providerSwitch.spec.ts`:
  - `fetchStatus` correctly populates failover state.
  - WS events (`provider.switched`, `provider.restored`, `provider.primary_recovered`) update state without full refetch.
  - Action methods invoke correct API endpoints and handle errors gracefully.

---

## Milestone 2 — App Header Failover Alert Badge & Navigation Integration

### Description

Add an alert badge to `AppHeader.vue` that appears when any agent in the active project is running on fallback. The badge displays the count of failover agents, pulses green when a primary provider has recovered, and opens the `ProviderFailoverModal` on click.

### Files to change

- **Edit** `web/src/components/layout/AppHeader.vue`:
  - Import `useProviderSwitchStore`.
  - Render an amber alert badge in the header action bar next to the queue badge when `providerSwitchStore.isFailoverActive`:
    - Displays an alert triangle icon and text: e.g. `"Failover Active (2)"`.
    - If `providerSwitchStore.hasRecoveredPrimary`: displays a green indicator dot and tooltip: `"Primary provider recovered — click to restore"`.
    - Click event opens `ProviderFailoverModal`.

### Acceptance criteria

- Vitest tests in `web/src/components/layout/__tests__/AppHeader.spec.ts`:
  - Badge is hidden when `isFailoverActive === false`.
  - Badge appears with amber styling and agent count when `isFailoverActive === true`.
  - Green recovery dot appears when `hasRecoveredPrimary === true`.
  - Clicking badge emits modal open event.

---

## Milestone 3 — Provider Failover Modal & Preset Templates Drawer

### Description

Create `ProviderFailoverModal.vue` to give operators a unified control center for active failovers. It displays each agent on fallback, active vs primary targets, trigger reasons, live primary reachability badges, individual restore actions, a one-click "Restore All" button, and a template selector dropdown.

### Component Design (`ProviderFailoverModal.vue`)

- **Header:**
  - Title: "Active Provider Failovers" with active count badge.
  - Action buttons:
    - **"Restore All Primary Providers"** (primary action button, disabled if no failovers active).
    - **"Apply Preset Template"** dropdown (`ProviderTemplateMenu.vue`).
- **Body / Agent Cards List:**
  - For each agent in `failoverAgents`:
    - Agent Name & Role badge.
    - Status comparison row:
      - **Active (Fallback):** `<active_provider>` (`<active_model>`) in blue/amber.
      - **Original (Primary):** `<primary_provider>` (`<primary_model>`) in muted gray.
    - Reason & Timestamp: e.g. `"Triggered by HTTP 529 Overloaded — 14 minutes ago"`.
    - **Primary Health Badge:**
      - Green `"Recovered & Reachable"` (if `primary_healthy === true`).
      - Red `"Unavailable"` (if `primary_healthy === false`).
    - Action: **"Restore Primary"** button.
- **Empty State:**
  - If no agents are in failover: displays a clean checkmark icon and message `"All agents are operating on their primary providers."`

### Files to change

- **New** `web/src/components/provider/ProviderFailoverModal.vue`.
- **New** `web/src/components/provider/ProviderTemplateMenu.vue`:
  - Dropdown listing configured project templates (e.g. `Hybrid`, `Claude`, `Gemini`, `Local AI`).
  - Clicking a template shows a confirmation modal listing affected agents and calls `applyTemplate`.
- **Edit** `web/src/App.vue` or root project view to host the global modal component.

### Acceptance criteria

- Vitest tests in `web/src/components/provider/__tests__/ProviderFailoverModal.spec.ts`:
  - Renders all active failover agents with correct metadata.
  - "Restore All" triggers `restoreAll` action and disables during network flight.
  - Individual "Restore Primary" button calls `restoreAgent(agentName)`.
  - Template dropdown renders templates from store and applies selection.

---

## Milestone 4 — Agent Panel & Config Form Controls

### Description

Update the agent roster list (`AgentPanelRow.vue`) and form editor (`AgentConfigForm.vue`) to show fallback status, allow one-click restores directly from agent cards, enable quick manual switching via `SwitchProviderModal.vue`, and support configuring fallback provider/model settings.

### Files to change

- **Edit** `web/src/components/agent/AgentPanelRow.vue`:
  - When `agent.is_failover === true`:
    - Displays an amber badge: `"Active: Gemini 2.5 (Fallback for Claude Sonnet)"`.
    - Renders a prominent **"Restore Primary"** action button on the card.
  - Adds a **"Switch Provider"** option in the agent row context/action menu.
- **New** `web/src/components/agent/SwitchProviderModal.vue`:
  - Modal with:
    - Target Provider dropdown (populated from registered providers).
    - Target Model input / autocomplete.
    - Reason text input (optional).
    - "Switch Target" button calling `switchAgent`.
- **Edit** `web/src/components/agent/AgentConfigForm.vue`:
  - Add **Failover Configuration** section:
    - `Fallback Provider` dropdown: lists registered providers.
    - `Fallback Model` autocomplete input.
    - Helper explanation: "When the primary provider encounters HTTP 529 or rate limits, the runner can automatically failover to this provider."

### Acceptance criteria

- Vitest tests in `web/src/components/agent/__tests__/AgentPanelRow.spec.ts`:
  - Renders fallback badge and "Restore Primary" button when `is_failover === true`.
  - Clicking "Restore Primary" dispatches restore action.
  - "Switch Provider" menu item opens `SwitchProviderModal`.
- Vitest tests in `web/src/components/agent/__tests__/AgentConfigForm.spec.ts`:
  - Renders fallback provider and model form inputs.
  - Form validation preserves and saves fallback fields.

---

## Milestone 5 — Queue Pause Banner Failover & Resume Integration

### Description

Update `QueuePauseBanner.vue` so that when the queue is paused due to an upstream rate limit or HTTP 529 error, operators are given an immediate **"Switch Provider & Resume"** button alongside "Resume now".

### Files to change

- **Edit** `web/src/components/queue/QueuePauseBanner.vue`:
  - When `queueStore.snapshot.paused && queueStore.snapshot.pause_reason === 'rate_limit'`:
    - Renders **"Switch Provider & Resume"** button.
    - Clicking the button identifies the failed job's agent, opens `SwitchProviderModal` pre-populated with that agent, and on successful switch automatically triggers `queueStore.resume()`.

### Acceptance criteria

- Vitest tests in `web/src/components/queue/__tests__/QueuePauseBanner.spec.ts`:
  - "Switch Provider & Resume" button appears on rate-limit queue pause.
  - Clicking button opens switch modal with agent context.
  - Successfully switching triggers queue resumption.

---

## Verification (End-to-End)

1. `pnpm build` clean without TypeScript or bundling errors.
2. `pnpm test` passes all new and existing Vitest unit/component tests.
3. Manual UI Smoke:
   - Navigate to a project with an agent in failover mode.
   - Verify amber "Failover Active" badge in header.
   - Click badge to open `ProviderFailoverModal`.
   - Verify primary health status and click "Restore Primary".
   - Confirm header badge dismisses and agent card updates in real time via WebSocket.

## Risk Notes

- **Header Space & Clutter:** The header already hosts the queue badge and run indicator. The failover badge uses compact badge styling (icon + number) to avoid overflowing mobile and tablet viewports.
- **Optimistic State Updates:** Store updates optimistic local state on WS broadcasts while gracefully rolling back if REST mutation calls return error status.
