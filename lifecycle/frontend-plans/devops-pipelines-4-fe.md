---
title: 'Frontend Plan: DevOps Pipeline Management'
type: plan-frontend
status: in-development
lineage: devops-pipelines
parent: lifecycle/requirements/devops-pipelines-2.md
---

## Overview

Build the DevOps SPA page: a role-gated view that discovers pipelines from the backend API, renders them as grouped cards by type, and provides run/cancel controls with real-time step progress streamed over WebSocket. The UI must handle concurrent pipeline runs, display output in a terminal-style pane, and visually differentiate step states.

Related: [[devops-pipelines]]

## Milestone 1 — DevOps Route, Navigation Entry & Role Gate

### Description

Add the DevOps page route and sidebar navigation entry, visible only to users with the `product-owner` or `devops` role.

### Files to change

- `web/src/router/index.ts` — Add route `{ path: '/devops', name: 'devops', component: () => import('../views/DevOpsView.vue'), meta: { roles: ['product-owner', 'devops'] } }`.
- `web/src/components/Sidebar.vue` (or equivalent nav component) — Add "DevOps" menu item below existing entries, conditionally rendered based on user role.
- `web/src/views/DevOpsView.vue` (new) — Skeleton view component with page title and placeholder content.

### Acceptance criteria

- [ ] "DevOps" appears in the sidebar for `product-owner` and `devops` users.
- [ ] "DevOps" is hidden from users without those roles.
- [ ] Navigating to `/devops` renders the DevOps view.
- [ ] Direct navigation to `/devops` by unauthorised users redirects or shows 403.

## Milestone 2 — Pipeline Listing & Grouped Card Layout

### Description

Fetch pipelines from the backend listing API and render them as cards grouped into columns by type (Build, Deploy, Release, plus dynamic columns for unknown types).

### Files to change

- `web/src/stores/devops.ts` (new) — Pinia store with:
  - `pipelines` state (array of pipeline objects).
  - `fetchPipelines()` action calling `GET /api/p/:project/devops/pipelines`.
  - `pipelinesByType` getter grouping pipelines by their `type` field.
- `web/src/views/DevOpsView.vue` — Call `fetchPipelines()` on mount, render grouped columns.
- `web/src/components/devops/PipelineCard.vue` (new) — Card component showing pipeline name, step count, type badge, and "Run" button.

### Acceptance criteria

- [ ] Pipelines are fetched on page load and displayed grouped by type.
- [ ] Each type gets its own column with a header (e.g. "Build", "Deploy", "Release").
- [ ] Unknown types (e.g. `migrate`) render in their own dynamically-created column.
- [ ] Each card shows the pipeline name and step count.
- [ ] Empty state is handled gracefully (message when no pipelines exist).

## Milestone 3 — Run Trigger & Cancel Controls

### Description

Wire up the "Run" button to trigger pipeline execution via the API, and add a "Cancel" button that appears during active runs.

### Files to change

- `web/src/stores/devops.ts` — Add actions:
  - `runPipeline(slug: string)` — calls `POST /api/p/:project/devops/pipelines/:slug/run`, stores the `run_id`.
  - `cancelPipeline(slug: string)` — calls `POST /api/p/:project/devops/pipelines/:slug/cancel`.
  - `activeRuns` state tracking which pipelines have active runs and their `run_id`.
- `web/src/components/devops/PipelineCard.vue` — Disable "Run" button when pipeline has an active run; show "Cancel" button instead. Handle 409 response gracefully (show toast/message).

### Acceptance criteria

- [ ] Clicking "Run" triggers the execution API and stores the `run_id`.
- [ ] The "Run" button is disabled while a run is active for that pipeline.
- [ ] A "Cancel" button appears during active runs.
- [ ] Clicking "Cancel" calls the cancel endpoint and updates UI state.
- [ ] A 409 response (already running) shows an appropriate user message.
- [ ] A 403 response is handled (should not occur if role gate works, but defensive).

## Milestone 4 — Real-time Step Progress via WebSocket

### Description

Subscribe to pipeline WebSocket events and update the card UI in real time to show step-by-step progress with state icons.

### Files to change

- `web/src/stores/devops.ts` — Add WebSocket event handlers for:
  - `pipeline.run.started` — mark pipeline as running, initialise step states.
  - `pipeline.step.started` — update step state to `running`.
  - `pipeline.step.output` — append output chunk to step's output buffer.
  - `pipeline.step.completed` — update step state to `passed`/`failed`, record duration.
  - `pipeline.run.completed` — mark pipeline run as finished, update overall status.
- `web/src/components/devops/PipelineCard.vue` — When a run is active, expand to show step list with state icons.
- `web/src/components/devops/StepProgress.vue` (new) — Component rendering a single step: state icon (pending ○, running ◉ spinning, passed ✓ green, failed ✗ red), step name, and duration when complete.

### Acceptance criteria

- [ ] Step states update in real time as WebSocket events arrive.
- [ ] State icons correctly reflect pending/running/passed/failed.
- [ ] Step durations display after completion.
- [ ] When a step fails, subsequent steps show as pending/skipped visually.
- [ ] The run completion event re-enables the "Run" button and hides "Cancel".

## Milestone 5 — Terminal Output Pane

### Description

Display streaming command output for each step in a scrollable, terminal-styled pane. Highlight errors visually.

### Files to change

- `web/src/components/devops/StepOutput.vue` (new) — Scrollable `<pre>` pane with monospace font, dark background, auto-scroll to bottom as output arrives. Configurable max-height with overflow scroll.
- `web/src/components/devops/PipelineCard.vue` — Integrate `StepOutput` below each step in the expanded view. Show/hide toggle per step.
- `web/src/assets/styles/` (or scoped styles) — Terminal pane styling: dark background, light monospace text, red border/background for failed steps.

### Acceptance criteria

- [ ] Output streams into the terminal pane in real time as `pipeline.step.output` events arrive.
- [ ] The pane auto-scrolls to the latest output.
- [ ] Failed steps have visually distinct styling (red border or background).
- [ ] Output from large commands doesn't cause performance issues (virtual scrolling or max buffer).
- [ ] Each step's output can be expanded/collapsed independently.

## Milestone 6 — Run History & Log Viewer

### Description

Allow users to view logs from past and in-progress runs via the run log API endpoint defined in [[devops-pipelines-3-be]].

### Files to change

- `web/src/stores/devops.ts` — Add action `fetchRunLog(runId: string)` calling `GET /api/p/:project/devops/runs/:run_id`.
- `web/src/components/devops/RunHistory.vue` (new) — Component showing a list of recent runs for a pipeline (if the backend exposes a list endpoint), with clickable entries to view the full log.
- `web/src/components/devops/LogViewer.vue` (new) — Full-screen or modal viewer rendering the JSON-lines log in a readable format with timestamps and step separators.

### Acceptance criteria

- [ ] Users can view logs of completed runs.
- [ ] In-progress run logs show partial content and update on refresh.
- [ ] Log viewer renders output with clear step boundaries and timestamps.
- [ ] The viewer is accessible from the pipeline card (e.g. "View last run" link).
