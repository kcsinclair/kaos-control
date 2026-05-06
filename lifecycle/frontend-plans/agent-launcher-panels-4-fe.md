---
title: 'Frontend Plan: Agent Launcher Panels'
type: plan-frontend
status: blocked
lineage: agent-launcher-panels
parent: lifecycle/requirements/agent-launcher-panels-2.md
assignees:
    - role: product-owner
      who: agent
---

## Overview

Add a row of agent panels above the runs table on the Agents screen. Each panel shows the agent's name, role(s), and model. Clicking a launchable panel opens a modal listing eligible artifacts filtered by the workflow predecessor of the agent's `active_status`. Inline-driver agents render as disabled. This implements FR-1 through FR-4 and FR-7, plus NFR-1/2/3.

Depends on [[agent-launcher-panels]] backend plan (Milestone 1) for the `model` and `active_status` fields in `GET /agents`.

## Milestone 1 — Update TypeScript types and API layer

### Description

Add `model` and `active_status` to the `AgentSummary` interface so the frontend can consume the new backend fields. No UI changes yet.

### Files to change

- `web/src/types/api.ts` — Add two optional fields to the `AgentSummary` interface (currently lines 75-80):
  ```typescript
  export interface AgentSummary {
    name: string
    roles: string[]
    driver: string
    model?: string
    active_status?: string
    allowed_write_paths?: string[]
  }
  ```

### Acceptance criteria

- [ ] `AgentSummary` includes `model?: string` and `active_status?: string`.
- [ ] `pnpm exec vue-tsc --noEmit` passes.
- [ ] No runtime changes — this is a type-only update.

## Milestone 2 — Create the AgentPanelRow component

### Description

Build a new `AgentPanelRow.vue` component that renders a horizontal, wrapping row of agent cards. Each card shows name, role(s), and model. Inline-driver agents are visually muted and non-interactive. This satisfies FR-1, FR-2, FR-3, FR-7 (empty state: row not rendered when no agents), NFR-1 (responsive wrap), and NFR-3 (keyboard accessibility).

### Files to change

- `web/src/components/agent/AgentPanelRow.vue` — **New file**. Component structure:
  - **Props**: `agents: AgentSummary[]`.
  - **Emits**: `select(agent: AgentSummary)` — fired when a launchable panel is clicked.
  - **Template**:
    - Outer container: `display: flex; flex-wrap: wrap; gap: 0.75rem;` to satisfy NFR-1.
    - One panel per agent. Each panel is a `<button>` (for native keyboard focus/activation per NFR-3).
    - Panel content: agent `name` as the primary label, `roles` joined with ", " below, `model` below that (hidden if empty).
    - Disabled state: if `agent.driver === 'inline'`, set `disabled` attribute and `aria-disabled="true"`. Apply muted styling (opacity, no pointer cursor). Add subtitle text "Externally driven".
    - Enabled state: pointer cursor, hover/focus ring. On click/Enter/Space, emit `select`.
  - **Styles**: Compact card appearance matching the existing design system. Scoped CSS.

### Acceptance criteria

- [ ] One panel renders per agent in the `agents` array.
- [ ] Each panel displays name, role(s), and model (model line hidden when empty).
- [ ] Inline-driver agents render with muted style, `aria-disabled="true"`, and are not clickable.
- [ ] Panels wrap on narrow viewports without horizontal overflow (flex-wrap).
- [ ] Panels are focusable via Tab and activatable via Enter/Space (native `<button>` behaviour).
- [ ] When `agents` is empty, the component renders nothing.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

## Milestone 3 — Create the AgentLaunchModal component

### Description

Build a modal that appears when a launchable agent panel is clicked. It fetches artifacts filtered by the workflow predecessor of the agent's `active_status`, displays them in a selectable list, and fires `POST /agents/:name/run` on confirmation. This satisfies FR-4, FR-6 (artifact eligibility), and FR-7 (empty artifact state).

### Files to change

- `web/src/components/agent/AgentLaunchModal.vue` — **New file**. Component structure:
  - **Props**: `agent: AgentSummary`, `project: string`.
  - **Emits**: `started(runId: string)`, `cancel`.
  - **Workflow predecessor map** (computed or constant):
    ```typescript
    const predecessorMap: Record<string, string> = {
      clarifying: 'draft',
      planning: 'clarifying',
      'in-development': 'planning',
      'in-qa': 'in-development',
      approved: 'in-qa',
    }
    ```
    Derive `inputStatus` from `predecessorMap[agent.active_status]`. If `active_status` is empty or not in the map, skip status filtering (show all artifacts per FR-4 fallback).
  - **On mount**: call `artifactsApi.listArtifacts(project, { status: inputStatus })` to populate the list. Show a loading spinner while fetching.
  - **Template**:
    - Modal overlay with click-outside-to-dismiss (FR-4.4).
    - Header: "Run {agent.name}" with a Cancel button/X.
    - Artifact list: each item shows `title`, `lineage`, `status`, and `path`. Items are selectable (radio-style or highlight-on-click).
    - Empty state: "No eligible artifacts for this agent." message (FR-7).
    - Footer: "Run" button (disabled until an artifact is selected) and "Cancel" button.
  - **On confirm**: call `agentsStore.startRun(project, agent.name, selectedArtifact.path, agent.roles[0])`. On success, emit `started` with the run ID and show a success toast. On error, show an error toast.

### Acceptance criteria

- [ ] Modal opens with a filtered artifact list based on the predecessor of `active_status`.
- [ ] When `active_status` is empty, all artifacts are listed (no status filter).
- [ ] Each artifact item shows title, lineage, status, and relative path.
- [ ] Selecting an artifact and clicking "Run" calls `POST /agents/:name/run` with the artifact's path.
- [ ] On successful run start, the modal closes, a toast appears, and the `started` event fires.
- [ ] When no artifacts match, an empty-state message is displayed.
- [ ] Click-outside and Cancel both dismiss the modal.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

## Milestone 4 — Integrate panels and modal into AgentsRunsView

### Description

Wire the new `AgentPanelRow` and `AgentLaunchModal` into the existing Agents screen. The panel row appears above the runs table. The existing `RunAgentDialog` remains as a fallback (per acceptance criteria).

### Files to change

- `web/src/views/project/AgentsRunsView.vue` — Modify the template and script:
  - Import `AgentPanelRow` and `AgentLaunchModal`.
  - Add reactive state: `selectedAgent: AgentSummary | null = null`.
  - Insert `<AgentPanelRow>` above the runs table, passing `agentsStore.agents` as props. Handle `@select` by setting `selectedAgent`.
  - Conditionally render `<AgentLaunchModal>` when `selectedAgent` is not null. Pass `selectedAgent` and the project slug. Handle `@started` to refresh runs and clear `selectedAgent`. Handle `@cancel` to clear `selectedAgent`.
  - Ensure the existing "Run Agent" button and `RunAgentDialog` remain untouched.
  - Ensure `agentsStore.fetchAgents(project)` is called on mount (it likely already is for the existing dialog — verify).

### Acceptance criteria

- [ ] Agent panels appear above the runs table on the Agents screen.
- [ ] Clicking a panel opens the launch modal for that agent.
- [ ] After a successful run, the new run appears in the runs table (via WebSocket or refresh).
- [ ] The existing "Run Agent" button and `RunAgentDialog` remain functional.
- [ ] NFR-2 satisfied: no extra API calls on page load beyond the existing `GET /agents`.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

## Open Questions

1. **Requirement Open Question 2 — Type filtering**: Should `AgentLaunchModal` also filter by artifact `type`? For example, `planning-analyst` would only list `requirement`-type artifacts. The backend supports `?type=` filtering. The current plan does **not** add type filtering because there is no explicit mapping from agent name to expected input type in the config. If type filtering is desired, a new `input_type` field on `AgentConfig` would be the cleanest approach — but that is outside the scope of this plan unless the product owner decides otherwise.

> yes, only the artifacts type which match that agent type, e.g. requirements-analyst processes ideas, analyst-planning processes requirements, backend-developer does backend-plans, frontend-developer does frontend-plans, test-developer does test-plans, qa does tests
