---
title: "Frontend Plan — Releases and Roadmaps"
type: plan-frontend
status: done
lineage: releases-and-roadmaps
parent: lifecycle/requirements/releases-and-roadmaps-2.md
---

# Frontend Plan — Releases and Roadmaps

This plan covers all Vue 3 / TypeScript frontend changes: the Roadmap page with Gantt chart and graph sub-views, release CRUD modals, artifact editor release dropdown, and kanban release filter.

Cross-references: [[releases-and-roadmaps]] (backend plan for REST API and data model), [[releases-and-roadmaps]] (test plan for integration tests).

---

## Milestone 1 — Release API client and Pinia store

### Description

Create a typed API client module for release endpoints and a Pinia store to manage release state across components. This is the foundation all subsequent milestones depend on.

### Files to change

- `web/src/api/releases.ts` (new) — typed functions matching the backend REST API:
  - `listReleases(project: string): Promise<Release[]>`
  - `createRelease(project: string, data: CreateReleasePayload): Promise<Release>`
  - `getRelease(project: string, id: number): Promise<ReleaseDetail>`
  - `updateRelease(project: string, id: number, data: UpdateReleasePayload): Promise<Release>`
  - `deleteRelease(project: string, id: number, reassignTo?: number): Promise<{ orphaned_artifact_count: number }>`
  - `listReleaseArtifacts(project: string, id: number): Promise<Artifact[]>`
  - `getRoadmapGraph(project: string): Promise<GraphData>`
- `web/src/types/release.ts` (new) — define `Release`, `ReleaseDetail`, `CreateReleasePayload`, `UpdateReleasePayload` interfaces. `Release` has: `id`, `name`, `status` (`'planned' | 'active' | 'shipped'`), `start_date` (nullable), `end_date` (nullable), `created_at`, `updated_at`. `ReleaseDetail` extends `Release` with `idea_count` and `defect_count`.
- `web/src/stores/releases.ts` (new) — Pinia store with:
  - State: `releases: Release[]`, `loading: boolean`.
  - Actions: `fetch(project)`, `create(project, data)`, `update(project, id, data)`, `remove(project, id, reassignTo?)`.
  - Getters: `scheduled` (releases with dates, sorted by start_date), `unscheduled` (releases without dates), `byId(id)`, `byName(name)`.
  - WebSocket integration: listen for `release.created`, `release.updated`, `release.deleted` events and update state reactively.

### Acceptance criteria

- [ ] All API functions are typed and call the correct endpoints.
- [ ] The Pinia store fetches releases on first access and stays in sync via WebSocket events.
- [ ] `scheduled` getter returns releases ordered by `start_date`; `unscheduled` getter returns releases with null dates.
- [ ] TypeScript compiles with no errors (`pnpm type-check`).

---

## Milestone 2 — Release CRUD modals

### Description

Build reusable modal components for creating, editing, and deleting releases. These modals are shared across the Gantt chart, artifact editor, and potentially other views.

### Files to change

- `web/src/components/releases/ReleaseFormModal.vue` (new) — a modal dialog with fields:
  - Name (text input, required, 1–120 chars).
  - Status (select: planned / active / shipped).
  - Schedule toggle: "Scheduled" (default) vs "Unscheduled". When unscheduled, date fields are hidden.
  - Start date (date picker, required when scheduled).
  - Duration (number input + unit select: days/weeks) OR explicit end date (date picker). Auto-calculate the other when one changes. End date must be ≥ start date.
  - Props: `release?: Release` (if provided, edit mode; otherwise create mode), `project: string`.
  - Emits: `saved(release: Release)`, `close`.
  - Calls `store.create` or `store.update` on submit.
- `web/src/components/releases/ReleaseDeleteModal.vue` (new) — confirmation modal showing:
  - Release name and the count of assigned artifacts.
  - Option to reassign artifacts to another release (select dropdown of other releases) or leave orphaned.
  - Props: `release: Release`, `project: string`, `artifactCount: number`.
  - Emits: `confirmed(reassignTo?: number)`, `close`.

### Acceptance criteria

- [ ] Create modal validates all fields and shows inline errors.
- [ ] Duration ↔ end date auto-calculation works correctly.
- [ ] Unscheduled toggle hides date fields and submits null dates.
- [ ] Edit modal pre-fills all fields from the existing release.
- [ ] Delete modal shows artifact count and reassignment option.
- [ ] Modals emit correct events on save/cancel.
- [ ] `pnpm type-check` passes.

---

## Milestone 3 — Roadmap page with Gantt chart view

### Description

Add the Roadmap page as a new route with a Gantt chart as the default sub-view. The chart renders releases as horizontal bars on a time axis with configurable granularity.

### Files to change

- `web/src/views/project/RoadmapView.vue` (new) — page container with:
  - Toolbar: "Create Release" button, granularity toggle (week / month / quarter / half-year / year, default month), view toggle (Gantt / Graph).
  - Renders either `GanttChart` or the graph sub-view based on the toggle.
- `web/src/components/releases/GanttChart.vue` (new) — the core Gantt component:
  - **Time axis**: compute column boundaries based on selected granularity. Render column headers (e.g., "Jan 2026", "Q1 2026").
  - **Rows**: one row per scheduled release, ordered by `start_date`. Below scheduled rows, an "Unscheduled" column section on the right containing unscheduled releases as cards.
  - **Bars**: each release is a `<div>` absolutely positioned using CSS `left`/`width` calculated from `start_date`/`end_date` relative to the time axis. Bar colour reflects release status (planned=grey, active=blue, shipped=green). Bar displays release name and a summary badge ("3 ideas · 1 defect").
  - **Today marker**: a vertical red line at the current date position.
  - **Click interaction**: clicking a bar opens `ReleaseDetailModal`.
  - **Empty state**: when no releases exist, show a centred prompt with "Create your first release" button.
  - Props: `releases: Release[]`, `granularity: string`, `project: string`.
- `web/src/components/releases/ReleaseDetailModal.vue` (new) — modal opened when clicking a release bar:
  - Shows release name, dates, status.
  - Lists assigned ideas and defects as cards (title, type badge, status badge, lineage).
  - Clicking a card navigates to the artifact detail view.
  - "Edit" and "Delete" buttons open the respective modals.
- `web/src/router/index.ts` — add route `{ path: 'roadmap', name: 'roadmap', component: RoadmapView }` as a child of the project route.

### Acceptance criteria

- [ ] The Roadmap page is accessible at `/p/:project/roadmap`.
- [ ] Granularity toggle switches between week/month/quarter/half-year/year and re-renders the chart.
- [ ] Release bars are positioned correctly on the time axis for all granularities.
- [ ] Bar colour reflects release status.
- [ ] Summary badge shows correct idea and defect counts.
- [ ] Today marker is a vertical line at the correct position.
- [ ] Clicking a bar opens the detail modal with assigned artifacts listed.
- [ ] Clicking an artifact card in the modal navigates to the artifact editor.
- [ ] "Create Release" button opens the creation modal.
- [ ] Empty state is shown when no releases exist.
- [ ] Unscheduled releases appear in a dedicated section on the right.
- [ ] `pnpm type-check` and `pnpm build` pass.

---

## Milestone 4 — Sidebar navigation entry

### Description

Add a "Roadmap" entry to the left navigation sidebar so users can access the Roadmap page.

### Files to change

- `web/src/components/layout/AppSidebar.vue` — add a new entry to the `navItems` computed array: `{ label: 'Roadmap', to: '/p/${p}/roadmap', icon: CalendarRange }` (using `CalendarRange` from lucide-vue-next). Position it after "Graph" in the list.

### Acceptance criteria

- [ ] "Roadmap" appears in the sidebar navigation after "Graph".
- [ ] Clicking it navigates to `/p/:project/roadmap`.
- [ ] The item is highlighted when the roadmap route is active.
- [ ] The icon renders correctly in both expanded and collapsed sidebar states.

---

## Milestone 5 — Roadmap graph sub-view

### Description

Add a "Graph" toggle on the Roadmap page that shows a release-centric graph using the existing 2D/3D graph components with data from the roadmap graph endpoint.

### Files to change

- `web/src/views/project/RoadmapView.vue` — when the view toggle is set to "Graph", render `RoadmapGraphView` instead of `GanttChart`.
- `web/src/components/releases/RoadmapGraphView.vue` (new) — wrapper component that:
  - Fetches data from `getRoadmapGraph(project)`.
  - Passes nodes and edges to the existing `ForceGraph3D` (3D) or `Graph2DView` (2D) components, matching the same props interface used by `GraphView.vue`.
  - Release nodes are rendered with a distinct colour/shape (larger size, hexagonal or diamond shape) to differentiate them from artifact nodes.
  - Includes a 2D/3D toggle matching the existing graph view.
  - Listens for `release.created`, `release.updated`, `release.deleted`, and `artifact.indexed` WebSocket events to refresh.

### Acceptance criteria

- [ ] Toggling to "Graph" on the Roadmap page shows the release-centric graph.
- [ ] Release nodes are visually distinct (larger, different colour/shape) from artifact nodes.
- [ ] Releases are connected chronologically via timeline edges forming a backbone.
- [ ] Only `idea` and `defect` artifacts appear as nodes.
- [ ] 2D/3D toggle works.
- [ ] Clicking a node opens the standard artifact or release modal.
- [ ] The graph refreshes on relevant WebSocket events.
- [ ] `pnpm type-check` and `pnpm build` pass.

---

## Milestone 6 — Artifact editor release dropdown

### Description

Replace the free-text `release` input in `FrontmatterEditor.vue` with a `<select>` dropdown populated from defined releases, plus an "Unassigned" option and a "+ Create Release" inline action.

### Files to change

- `web/src/components/artifact/FrontmatterEditor.vue` — replace the `<input>` for the `release` field with a `<select>`:
  - Options: "Unassigned" (value `""`), each release name from the releases store, "+ Create Release" (special value that opens `ReleaseFormModal`).
  - On selecting "+ Create Release", open the modal; on save, set the release field to the newly created release name.
  - Import and use the releases Pinia store; ensure it fetches on mount if not already loaded.

### Acceptance criteria

- [ ] The release field renders as a `<select>` dropdown, not a text input.
- [ ] All defined releases appear as options.
- [ ] "Unassigned" option clears the release field.
- [ ] "+ Create Release" opens the creation modal inline.
- [ ] After creating a release, the dropdown selects the new release.
- [ ] Saving the artifact writes the selected release name to frontmatter.
- [ ] `pnpm type-check` passes.

---

## Milestone 7 — Kanban board release filter

### Description

Add a release filter dropdown to the kanban board toolbar so users can scope the board to a single release.

### Files to change

- `web/src/views/project/KanbanBoardView.vue` — add a `selectedRelease` ref and a `<select>` dropdown in the filter toolbar:
  - Options: "All Releases" (default, value `""`), each defined release name, "Unassigned".
  - Pass `selectedRelease` to the `applyFilters` call as the `release` parameter.
  - Persist the filter in the URL query string: `?release=<name>`. Read it from `route.query.release` on mount.
- `web/src/composables/useKanbanBoard.ts` — extend `applyFilters` to accept and forward a `release` parameter to the `GET /artifacts` API call.

### Acceptance criteria

- [ ] A release filter dropdown appears in the kanban toolbar.
- [ ] Selecting a release filters the board to show only matching artifacts.
- [ ] "All Releases" shows all artifacts (default).
- [ ] "Unassigned" shows artifacts with no release.
- [ ] The filter value is reflected in the URL as `?release=<name>`.
- [ ] Navigating to a URL with `?release=v1.0` pre-selects the filter.
- [ ] `pnpm type-check` and `pnpm build` pass.
