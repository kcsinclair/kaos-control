---
title: "Frontend Plan: Dashboard Clickable Filters"
type: plan-frontend
status: approved
lineage: dashboard-clickable-filters
parent: lifecycle/requirements/dashboard-clickable-filters-2.md
created: "2026-05-09"
---

# Frontend Plan: Dashboard Clickable Filters

## Overview

Make dashboard Summary Count cards and Status Distribution pie-chart segments clickable, navigating to the artifacts list view with pre-applied filters via URL query parameters. Per the resolved questions in the requirement, only **Lifecycle Total** and **Blocked** cards are clickable (In Progress and Completed This Week are excluded). All status-distribution pie segments are clickable.

A prerequisite discovery: ArtifactListView does **not** currently read filters from URL query parameters — it manages filter state solely through Pinia store refs. Milestone 1 adds query-parameter synchronisation to ArtifactListView so that navigations from the dashboard (and direct bookmarks) work correctly.

Depends on [[dashboard-clickable-filters]] backend plan confirming the filter API contract is stable. Interacts with [[dashboard-clickable-filters]] test plan for acceptance verification.

## Milestone 1: ArtifactListView — Read Filters from URL Query Parameters

**Description:** On mount (and on route query changes), ArtifactListView must initialise its local filter refs from `route.query`. This enables bookmark/deep-link support (FR-3) and is a prerequisite for click-through from the dashboard.

**Files to change:**

- `web/src/views/project/ArtifactListView.vue` — add an `onMounted` (or immediate watcher on `route.query`) block that reads supported query parameters (`status`, `stage`, `type`, `label`, `priority`, `release`, `q`, `lineage`) and sets the corresponding local refs (`selectedStatus`, `selectedStage`, etc.), then calls `applyFilters()`.

**Acceptance criteria:**

- [ ] Navigating directly to `/p/:project/artifacts?status=blocked` loads the list view with the status dropdown set to "blocked" and only blocked artifacts displayed.
- [ ] Each supported filter key (`status`, `stage`, `type`, `label`, `priority`, `release`, `q`) is read from query params on mount.
- [ ] If no query params are present, the view behaves exactly as before (all filters empty, showing all artifacts).
- [ ] Changing a filter dropdown does **not** need to write back to the URL (one-directional: URL → state on mount is sufficient for this iteration).

## Milestone 2: SummaryCountCard — Add Click Target and Navigation

**Description:** Make SummaryCountCard emit a click event and make SummaryCountsWidget wire up `router.push` navigation for the two active cards.

**Files to change:**

- `web/src/components/dashboard/widgets/SummaryCountCard.vue`:
  - Wrap the card root element in a clickable container (or make the existing root element clickable) when a new optional prop `to` (type `RouteLocationRaw | null`) is provided.
  - When `to` is set: add `@click` and `@keydown.enter` / `@keydown.space` handlers that call `router.push(to)`.
  - When `to` is null/undefined: card remains non-interactive (no cursor change, no click handler). This keeps Completed This Week and In Progress as display-only.
  - Add `role="link"` and a computed `aria-label` (e.g., `"View 3 blocked artifacts"`) when `to` is set.
  - Ensure `tabindex="0"` is present when interactive (it already has `tabindex` but may need conditional logic).

- `web/src/components/dashboard/widgets/SummaryCountsWidget.vue`:
  - Import `useRouter` and `useRoute` (or compute the project slug from props).
  - For **Lifecycle Total** card: pass `to` as `{ name: 'artifacts', params: { project }, query: {} }` (no filter — show all).
  - For **Blocked** card: pass `to` as `{ name: 'artifacts', params: { project }, query: { status: 'blocked' } }`.
  - For **In Progress** and **Completed This Week** cards: pass `to` as `null` (non-interactive, per resolved questions).

**Acceptance criteria:**

- [ ] Clicking the Lifecycle Total card navigates to `/p/:project/artifacts` with no query parameters.
- [ ] Clicking the Blocked card navigates to `/p/:project/artifacts?status=blocked`.
- [ ] In Progress and Completed This Week cards are not clickable and show no pointer cursor.
- [ ] Navigation uses `router.push`, not `window.location`.
- [ ] Browser back-button returns to the dashboard after click-through.

## Milestone 3: SummaryCountCard — Visual Affordance (FR-4)

**Description:** Add hover and focus styling to interactive cards.

**Files to change:**

- `web/src/components/dashboard/widgets/SummaryCountCard.vue`:
  - When `to` is set: add `cursor: pointer` to the card root.
  - Add a CSS hover state: subtle background shift or elevation change (e.g., `box-shadow` increase or `background-color` lightening).
  - Ensure the existing `focus-visible` ring remains visible and meets WCAG 2.1 AA contrast.

**Acceptance criteria:**

- [ ] Interactive cards show `cursor: pointer` on hover.
- [ ] A visible hover state distinguishes interactive from non-interactive cards.
- [ ] Keyboard focus ring is visible when tabbing to an interactive card.
- [ ] Non-interactive cards (In Progress, Completed This Week) show default cursor and no hover state change.

## Milestone 4: StatusDistributionWidget — Pie Segment Click-Through (FR-2)

**Description:** Add an ECharts click handler to the status distribution pie chart so that clicking a segment navigates to the artifacts list filtered by that segment's status.

**Files to change:**

- `web/src/components/dashboard/widgets/StatusDistributionWidget.vue`:
  - Import `useRouter`.
  - After the chart instance is initialised, attach a `click` event handler via `chart.on('click', handler)`.
  - The handler must extract the status key from the clicked series data (ECharts passes `params.name` or `params.data.name` for pie series).
  - Call `router.push({ name: 'artifacts', params: { project }, query: { status: statusKey } })`.
  - Set `cursor: 'pointer'` on pie series items via the ECharts series config (`emphasis.itemStyle.cursor` or global `cursor` on the series).
  - Add `aria-label="Status distribution chart — click a segment to filter artifacts by status"` to the chart container div.

**Acceptance criteria:**

- [ ] Clicking any pie segment navigates to `/p/:project/artifacts?status=<segment status>`.
- [ ] The status value in the URL matches the status key from the distribution API (e.g., `draft`, `in-development`, `blocked`).
- [ ] Cursor changes to pointer when hovering over pie segments.
- [ ] ECharts default emphasis (highlight on hover) is preserved.
- [ ] The chart container has an `aria-label` describing clickability.
- [ ] Navigation uses `router.push`.

## Milestone 5: Regression Verification

**Description:** Confirm that existing dashboard functionality is unaffected.

**Files to change:** None (testing only).

**Acceptance criteria:**

- [ ] WebSocket-driven real-time updates to SummaryCountsWidget still work (artifact.indexed event triggers refetch).
- [ ] WebSocket-driven real-time updates to ActivityFeedWidget still work (feed.new event prepends entry).
- [ ] Dashboard responsive layout (summary grid, chart column, panel column) renders correctly at common viewport widths.
- [ ] VelocityChartWidget granularity toggle still functions.
- [ ] Activity Feed "View all" button still navigates to `/p/:project/feed`.
- [ ] Activity Feed entry links still navigate to the correct artifact editor.
- [ ] Non-interactive Summary Count cards (In Progress, Completed This Week) remain display-only with no regressions to their data or styling.
