---
title: "Frontend Plan: Lineage Status Checker UI"
type: plan-frontend
status: draft
lineage: status-checker-button
parent: lifecycle/requirements/status-checker-button-2.md
---

## Overview

Implement the UI for the lineage status checker: a "Check status" button on the artifact detail panel, a "Check all statuses" action in the project toolbar, and a persistent results summary panel with per-artifact advance and batch "Fix all" actions.

---

## Milestone 1: API Client Functions

### Description

Add TypeScript API client functions to call the new backend endpoints.

### Files to Change

- `web/src/api/statusCheck.ts` (new) — API client module

### Acceptance Criteria

- `checkStatus(project: string, lineage?: string): Promise<StatusCheckResponse>` — calls `GET /api/p/{project}/status-check[?lineage=slug]`.
- `advanceStatuses(project: string, paths: string[]): Promise<AdvanceResponse>` — calls `POST /api/p/{project}/status-check/advance`.
- TypeScript types defined for `StaleArtifact`, `StatusCheckResponse`, `AdvanceResult`, `AdvanceResponse` matching the backend JSON schema from [[status-checker-button]].

---

## Milestone 2: Status Check Results Panel Component

### Description

Create a persistent panel component that displays staleness check results grouped by lineage.

### Files to Change

- `web/src/components/artifact/StatusCheckPanel.vue` (new)

### Acceptance Criteria

- Displays a list of stale artifacts grouped by lineage slug.
- Each entry shows: artifact title, current status (badge), suggested status (badge), and the child artifact(s) that make it stale.
- Per-artifact "Advance" button: enabled when `can_advance` is true, disabled with tooltip showing `blocked_reason` when false.
- "Fix all" button at the top advances all advanceable artifacts in one action.
- Loading state while the check is in progress.
- Empty state: "No stale statuses found" message when results are empty.
- Panel is dismissible (close button) but persists until explicitly closed (not a toast).
- After advancing, the panel refreshes results automatically (re-runs the check).

---

## Milestone 3: "Check Status" Button on Artifact Detail Panel

### Description

Add a "Check status" button to the `ArtifactModal.vue` component that triggers a single-lineage check and opens the results panel.

### Files to Change

- `web/src/components/artifact/ArtifactModal.vue` — add button and panel integration

### Acceptance Criteria

- Button visible on the artifact detail panel for every artifact type.
- Button label: "Check status" with a suitable icon (e.g. `lucide-vue-next` `CircleCheck` or `ListChecks`).
- Clicking triggers `checkStatus(project, artifact.lineage)` and opens `StatusCheckPanel` inline within or beside the modal.
- Button shows a spinner/loading state while the API call is in flight.
- Works for artifacts that have a lineage value in frontmatter; hidden or disabled if lineage is missing.

---

## Milestone 4: "Check All Statuses" in Project Toolbar

### Description

Add a project-wide "Check all statuses" action accessible from the graph view controls or project toolbar.

### Files to Change

- `web/src/components/graph/GraphFilters.vue` or `web/src/components/layout/AppHeader.vue` — add button
- Integration with `StatusCheckPanel` (show as a slide-over or panel within the graph view)

### Acceptance Criteria

- "Check all statuses" button/action visible in graph view controls or the project-level toolbar area.
- Clicking triggers `checkStatus(project)` (no lineage param = project-wide).
- Results displayed in the same `StatusCheckPanel` component, showing stale artifacts across all lineages.
- Button shows loading state during the check.

---

## Milestone 5: Real-time Updates via WebSocket

### Description

Ensure the results panel reacts to `artifact.indexed` WebSocket events so that status changes made by other clients (or the batch advance itself) are reflected without a manual refresh.

### Files to Change

- `web/src/components/artifact/StatusCheckPanel.vue` — subscribe to WS events
- `web/src/stores/artifacts.ts` (if needed) — ensure WS handler updates relevant state

### Acceptance Criteria

- When an `artifact.indexed` event fires for an artifact currently shown in the results panel, the panel re-fetches results to show updated state.
- Debounce re-fetches (e.g. 500 ms) to avoid excessive API calls during batch advance.
- If all stale artifacts are resolved, the panel transitions to the "No stale statuses found" empty state.
