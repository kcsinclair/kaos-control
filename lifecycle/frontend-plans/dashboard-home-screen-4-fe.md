---
title: "Dashboard Home Screen — Frontend Plan"
type: plan-frontend
status: draft
lineage: dashboard-home-screen
parent: lifecycle/requirements/dashboard-home-screen-2.md
---

# Dashboard Home Screen — Frontend Plan

This plan implements the dashboard view as the default landing page per project, using Apache ECharts for charting and a widget-slot architecture for extensibility.

## Milestone 1: Route & Navigation Setup

**Description:** Add the `/p/:project/dashboard` route and make it the default landing page. Add a "Dashboard" entry as the first item in the sidebar navigation.

**Files to change:**
- `web/src/router/index.ts` — Add `dashboard` route as the first child of `/p/:project`. Change the default redirect from `graph` to `dashboard`.
- `web/src/components/layout/AppSidebar.vue` — Insert a "Dashboard" nav item at index 0 in the `navItems` array with a `LayoutDashboard` icon from lucide-vue-next.
- `web/src/views/project/DashboardView.vue` (new) — Skeleton view component with widget-slot layout.

**Acceptance criteria:**
- [ ] `/p/:project` redirects to `/p/:project/dashboard`.
- [ ] "Dashboard" appears as the first item in the left sidebar.
- [ ] Direct navigation to `/p/:project/dashboard` renders the view.
- [ ] Existing routes (`/graph`, `/artifacts`, etc.) continue to work unchanged.

## Milestone 2: Widget Registry & Layout Architecture

**Description:** Implement a widget-slot system so the dashboard page is composed of registered widget components without hard-coding them in the template.

**Files to change:**
- `web/src/components/dashboard/widgetRegistry.ts` (new) — Export a `registerWidget(id, component, options)` function and a reactive `widgetList` array. Each widget entry has: `id`, `component` (async import), `slot` ('summary' | 'chart' | 'panel'), `order`.
- `web/src/components/dashboard/DashboardGrid.vue` (new) — Renders widgets by slot: summary row at top, chart column left, panel column right. Uses CSS Grid with responsive breakpoint at 1024 px.
- `web/src/views/project/DashboardView.vue` — Imports `DashboardGrid`, passes project context.

**Acceptance criteria:**
- [ ] Adding a new widget requires only calling `registerWidget()` — no edits to `DashboardView.vue` or `DashboardGrid.vue`.
- [ ] Layout is two-column (charts left, panels right) at ≥ 1024 px viewport.
- [ ] Layout stacks vertically at < 1024 px viewport.
- [ ] Widget order within a slot is respected.

## Milestone 3: Summary Count Widgets

**Description:** Four numeric stat cards showing total tickets, in-progress, blocked, and completed this week.

**Files to change:**
- `web/src/components/dashboard/widgets/SummaryCountCard.vue` (new) — Reusable card taking `label`, `value`, and optional `icon` props.
- `web/src/components/dashboard/widgets/SummaryCountsWidget.vue` (new) — Fetches `GET /api/p/:project/dashboard/stats` from [[dashboard-home-screen-3-be]], renders four `SummaryCountCard` instances.
- `web/src/components/dashboard/widgetRegistry.ts` — Register `SummaryCountsWidget` in slot `summary` with order 0.

**Acceptance criteria:**
- [ ] Four cards rendered: Total Tickets, In Progress, Blocked, Completed This Week.
- [ ] Values update when WebSocket `artifact.indexed` event fires.
- [ ] Cards render within 500 ms of route entry (meets performance NFR).
- [ ] Zero-state: shows "0" gracefully, not a loading spinner forever.

## Milestone 4: Status Distribution Chart (ECharts)

**Description:** A donut chart showing status distribution for non-done tickets using Apache ECharts.

**Files to change:**
- `web/package.json` — Add `echarts` and `vue-echarts` dependencies.
- `web/src/components/dashboard/widgets/StatusDistributionWidget.vue` (new) — Fetches `GET /api/p/:project/dashboard/status-distribution`, renders an ECharts donut chart. Includes interactive tooltips on hover.
- `web/src/components/dashboard/widgetRegistry.ts` — Register in slot `chart` with order 0.

**Acceptance criteria:**
- [ ] Donut chart renders showing each status as a segment with correct proportions.
- [ ] Interactive: hovering a segment shows tooltip with status name and count.
- [ ] Accessible: `aria-label` on chart container summarises the distribution.
- [ ] Colour palette meets WCAG 2.1 AA contrast against the background.
- [ ] `echarts` gzipped bundle contribution ≤ 80 KB (use tree-shaking: import only pie chart + tooltip modules).
- [ ] Empty state handled: if no tickets exist, show a message instead of an empty chart.

## Milestone 5: Completion Velocity Chart (ECharts)

**Description:** A time-series bar chart showing artifact completion counts over time with a granularity toggle (daily/weekly/monthly).

**Files to change:**
- `web/src/components/dashboard/widgets/VelocityChartWidget.vue` (new) — Fetches `GET /api/p/:project/dashboard/velocity?granularity=<g>`. Renders ECharts bar chart. Includes a toggle button group for daily/weekly/monthly. Interactive tooltips on hover.
- `web/src/components/dashboard/widgetRegistry.ts` — Register in slot `chart` with order 1.

**Acceptance criteria:**
- [ ] Bar chart renders with correct data from the velocity endpoint.
- [ ] Toggle between daily, weekly, and monthly granularity re-fetches and re-renders.
- [ ] Interactive: hover shows tooltip with period and count.
- [ ] Accessible: `aria-label` summarises the trend.
- [ ] X-axis labels are readable (rotated or abbreviated if needed).
- [ ] Handles zero-data gracefully (shows "No completions in this period" message).

## Milestone 6: Activity Feed Panel

**Description:** Right-hand panel showing the N most recent activity entries with a "View all" link to the existing feed view.

**Files to change:**
- `web/src/components/dashboard/widgets/ActivityFeedWidget.vue` (new) — Fetches `GET /api/p/:project/feed?limit=15`. Reuses existing `FeedEntry` component from `@/components/feed/FeedEntry.vue`. Includes "View all" link to `/p/:project/feed`. Subscribes to WebSocket for live updates.
- `web/src/components/dashboard/widgetRegistry.ts` — Register in slot `panel` with order 0.

**Acceptance criteria:**
- [ ] Shows 15 most recent feed entries using the existing `FeedEntry` component.
- [ ] New events appear at the top in real-time via WebSocket push.
- [ ] "View all" link navigates to `/p/:project/feed`.
- [ ] Scrollable if entries exceed panel height.

## Milestone 7: Responsive Polish & Accessibility Audit

**Description:** Final pass ensuring responsive layout, WCAG compliance, and performance budget.

**Files to change:**
- `web/src/components/dashboard/DashboardGrid.vue` — Refine CSS Grid media queries and spacing.
- All widget components — verify `aria-label` attributes, colour contrast, keyboard navigation of toggles.

**Acceptance criteria:**
- [ ] Two-column layout at ≥ 1024 px, single-column stacked at < 1024 px (verified with dev tools responsive mode).
- [ ] All charts have `aria-label` describing the data.
- [ ] Colour contrast of chart segments passes WCAG 2.1 AA (4.5:1 ratio for text, 3:1 for UI components).
- [ ] Tab order is logical: summary cards → charts → feed panel.
- [ ] No horizontal scrollbar on any tested viewport (320 px – 2560 px).

## Cross-references

- [[dashboard-home-screen-3-be]] — Backend endpoints consumed by this plan.
- [[dashboard-home-screen-5-test]] — Test plan covers E2E and component testing.
