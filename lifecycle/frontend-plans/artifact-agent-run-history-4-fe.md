---
title: "Frontend Plan: Artifact Agent Run History"
type: plan-frontend
status: in-development
lineage: artifact-agent-run-history
parent: lifecycle/requirements/artifact-agent-run-history-2.md
created: "2026-04-28"
---

# Frontend Plan: Artifact Agent Run History

relates-to: [[artifact-agent-run-history]]

## Overview

Add an "Agent Runs" section to the artifact detail modal and a run-detail modal for inspecting individual runs. Depends on the API and WebSocket changes from [[artifact-agent-run-history-3-be]]. The [[artifact-agent-run-history-5-test]] test plan will verify the UI behaviour.

---

## Milestone 1 — API function: fetch runs by target path

### Description

Add a new API helper that calls the extended `GET /api/p/{project}/agents/runs?target_path={path}` endpoint delivered in the backend plan.

### Files to change

- `web/src/api/agents.ts` — add a new exported function:
  ```ts
  export async function listRunsByTargetPath(project: string, targetPath: string): Promise<AgentRunRow[]>
  ```
  Calls `GET /api/p/${project}/agents/runs?target_path=${encodeURIComponent(targetPath)}` and returns `response.runs`.

### Acceptance criteria

- Function is exported and callable from stores/components.
- Encodes the target path to handle special characters in filenames.
- Returns `AgentRunRow[]` (empty array when no runs exist).
- `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 2 — Agents store: target-path run state

### Description

Extend the agents Pinia store with state and an action for per-artifact run lists, keeping them separate from the global `runs` array used by the agents view.

### Files to change

- `web/src/stores/agents.ts`:
  1. Add state: `artifactRuns: ref<AgentRunRow[]>([])` and `artifactRunsPath: ref<string>('')`.
  2. Add action `fetchRunsByTargetPath(project: string, targetPath: string)` that calls the new API function and sets `artifactRuns` / `artifactRunsPath`.
  3. In `onWsEvent`, when an `agent.started`, `agent.finished`, or `agent.failed` event is received: if the event payload's `target_path` matches `artifactRunsPath`, call `fetchRunsByTargetPath` to refresh the list.

### Acceptance criteria

- `artifactRuns` contains runs only for the currently viewed artifact's target path.
- Changing artifact clears/replaces the previous list.
- WS events for unrelated target paths do not trigger a re-fetch.
- `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 3 — ArtifactRunHistory component

### Description

Create a collapsible "Agent Runs" section component to embed in the artifact detail modal. It displays a chronological list of runs targeted at the artifact.

### Files to change

- `web/src/components/artifact/ArtifactRunHistory.vue` — new SFC:
  - **Props**: `project: string`, `targetPath: string`.
  - On mount (or on expand, implementer's choice), calls `agentsStore.fetchRunsByTargetPath(project, targetPath)`.
  - Renders a collapsible `<details>` block titled "Agent Runs" with a count badge.
  - Each run row shows:
    - Truncated run ID (first 8 characters).
    - Agent name.
    - `started_at` — relative time for recent runs, absolute for older ones (use the same formatting as `AgentsRunsView`).
    - Status badge (running / done / failed / killed) with accessible text label, matching the chip styles in `AgentsRunsView.vue`.
  - Shows a spinner while loading.
  - Shows "No agent runs for this artifact" empty state.
  - Emits `select-run(runId: string)` when a row is clicked.

### Acceptance criteria

- Section renders inside the artifact detail modal.
- Lazy-loads run data (no fetch until the section mounts or expands).
- Loading, empty, and populated states all render correctly.
- Status badges use text labels, not colour alone (NFR-3 accessibility).
- `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 4 — RunDetailModal component

### Description

Create a modal overlay for viewing full details of a single agent run, following the existing modal pattern used by `TransitionDialog` and `RunAgentDialog`.

### Files to change

- `web/src/components/agent/RunDetailModal.vue` — new SFC:
  - **Props**: `project: string`, `runId: string`.
  - On mount, fetches the run via `agentsApi.getRun(project, runId)`.
  - Displays:
    - Run ID (full).
    - Agent name and role.
    - Target path.
    - Started at / Finished at (formatted).
    - Status and exit code.
    - Stderr tail in a `<pre>` code block with scrollable overflow, using the dark-themed code style from `AgentsRunsView.vue`.
    - Artifacts produced as a list of paths.
  - Dismissible via close button, Escape key, and backdrop click.
  - Focus trapping while open; restores focus on close (NFR-3).
  - Uses `<Teleport to="body">` and z-index 300 (consistent with other dialogs).

### Acceptance criteria

- Modal opens with the correct run data.
- All `AgentRunRow` fields are displayed.
- Stderr tail is rendered in a monospace, scrollable code block.
- Modal dismisses via all three methods (button, Escape, backdrop).
- Focus is trapped inside the modal while open.
- `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 5 — Wire into ArtifactModal

### Description

Integrate the `ArtifactRunHistory` section and `RunDetailModal` into the existing artifact detail modal.

### Files to change

- `web/src/components/artifact/ArtifactModal.vue`:
  1. Import and render `<ArtifactRunHistory>` below the existing content sections (after the markdown preview, before footer edges). Pass `project` and the artifact's file path as `targetPath`.
  2. Add local state `selectedRunId: ref<string | null>(null)`.
  3. On `ArtifactRunHistory`'s `select-run` event, set `selectedRunId`.
  4. Conditionally render `<RunDetailModal>` when `selectedRunId` is set, passing `project` and `selectedRunId`. On close, reset `selectedRunId` to null.

### Acceptance criteria

- Opening an artifact detail modal shows the "Agent Runs" section.
- Clicking a run row opens the run detail modal on top of the artifact modal.
- Closing the run detail modal returns to the artifact modal without closing it.
- `pnpm build` passes with no errors.

---

## Milestone 6 — Live updates via WebSocket

### Description

Ensure the run list in `ArtifactRunHistory` updates automatically when a relevant agent event arrives, without requiring a manual page refresh.

### Files to change

- `web/src/stores/agents.ts` — the `onWsEvent` handler (updated in Milestone 2) already triggers a re-fetch when `target_path` matches. This milestone verifies end-to-end behaviour.
- `web/src/components/artifact/ArtifactRunHistory.vue` — ensure the component reacts to `agentsStore.artifactRuns` changes (it should, since Pinia state is reactive).

### Acceptance criteria

- Starting an agent run targeted at the currently viewed artifact causes the run to appear in the list within one WS event cycle.
- A run completing (done/failed/killed) updates the status badge in the list without manual refresh.
- Events for other artifacts' target paths do not affect the displayed list.
