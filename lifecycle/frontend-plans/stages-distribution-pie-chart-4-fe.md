---
title: "Frontend Plan: Stages Distribution Pie Chart"
type: plan-frontend
status: draft
lineage: stages-distribution-pie-chart
parent: lifecycle/requirements/stages-distribution-pie-chart-2.md
created: "2026-05-09"
---

# Frontend Plan: Stages Distribution Pie Chart

## Overview

Create a `StagesDistributionWidget.vue` component that renders an ECharts donut chart showing artifact counts per lifecycle stage. The widget follows the same architecture as `StatusDistributionWidget.vue` — async component registration, ECharts tree-shaking, ResizeObserver, and click-through navigation. It consumes the endpoint defined in the [[stages-distribution-pie-chart]] backend plan.

## Milestone 1: Create StagesDistributionWidget component

**Description:** Build the core widget component that fetches stage distribution data and renders an ECharts donut chart.

**Files to change:**

- `web/src/components/dashboard/widgets/StagesDistributionWidget.vue` — new file.

**Implementation details:**

- Import ECharts modules using the same tree-shaking pattern as `StatusDistributionWidget`: `PieChart`, `TooltipComponent`, `LegendComponent`, `CanvasRenderer`.
- Define `StageDistributionItem` interface: `{ stage: string; count: number }`.
- Define a WCAG 2.1 AA compliant colour palette for stages, visually distinct from the status palette in `StatusDistributionWidget`:
  - `ideas`: `#0ea5e9` (sky blue)
  - `requirements`: `#f97316` (orange)
  - `backend-plans`: `#14b8a6` (teal)
  - `frontend-plans`: `#a855f7` (purple)
  - `test-plans`: `#eab308` (yellow)
  - `tests`: `#22c55e` (green)
  - `prototypes`: `#64748b` (slate)
  - `releases`: `#e11d48` (rose)
  - `sprints`: `#06b6d4` (cyan)
  - `defects`: `#dc2626` (red)
  - Fallback: `#94a3b8` (grey) for any unlisted stage.
- Fetch from `GET /p/${project}/dashboard/stage-distribution` on mount and when `project` prop changes.
- Chart config: donut with inner radius 40%, outer radius 70%, horizontal legend at bottom, tooltip `{stage}: {count} ({percent}%)`, `cursor: pointer` on segments, labels hidden.
- Set `isEmpty` flag when distribution is empty or all counts are zero; show "No artifacts yet" placeholder.

**Acceptance criteria:**

- [ ] The widget renders a donut chart with one slice per stage returned by the API.
- [ ] Each stage has a distinct, accessible colour from the defined palette.
- [ ] Tooltip shows stage name, count, and percentage.
- [ ] Legend is horizontal, positioned at the bottom of the chart.
- [ ] "No artifacts yet" is shown when there are no artifacts.
- [ ] The chart height is 280 px.
- [ ] Card styling uses `var(--color-surface)`, `var(--color-border)`, `var(--radius-lg)`, `var(--space-4)`.
- [ ] Widget title is "Stages Distribution".

## Milestone 2: Click-through navigation

**Description:** Make pie slices clickable to navigate to the artifacts list filtered by the clicked stage.

**Files to change:**

- `web/src/components/dashboard/widgets/StagesDistributionWidget.vue` — add click handler.

**Implementation details:**

- Attach an ECharts `click` event listener on the chart instance (same as `StatusDistributionWidget`).
- On click, extract `params.name` (the stage value) and call `router.push({ name: 'artifacts', params: { project }, query: { stage } })`.
- The `stage` query parameter already exists on the artifacts list route, so no router changes are needed.

**Acceptance criteria:**

- [ ] Clicking a pie slice navigates to `/p/:project/artifacts?stage=<stage>`.
- [ ] The URL is bookmarkable and produces the correct filtered list when loaded directly.
- [ ] Browser back-button returns to the dashboard after navigation.
- [ ] `cursor: pointer` appears on hover over chart segments.

## Milestone 3: Accessibility

**Description:** Add ARIA attributes to the chart container for screen reader support.

**Files to change:**

- `web/src/components/dashboard/widgets/StagesDistributionWidget.vue` — add `role` and `aria-label`.

**Implementation details:**

- Set `role="img"` on the chart container div.
- Compute a dynamic `aria-label` from the fetched data, e.g., `"Stages distribution: ideas 5, requirements 12, ... — click a segment to filter artifacts by stage"`.
- When empty, use `aria-label="Stages distribution: no artifacts"`.

**Acceptance criteria:**

- [ ] The chart container has `role="img"`.
- [ ] The `aria-label` lists each stage and its count.
- [ ] The `aria-label` mentions that segments are clickable.

## Milestone 4: Widget registration and layout

**Description:** Register the widget in the widget registry so it appears on the dashboard in the correct position.

**Files to change:**

- `web/src/components/dashboard/registerWidgets.ts` — add registration entry.

**Implementation details:**

- Register with `id: 'stages-distribution'`, `slot: 'chart'`.
- Per Resolved Question 2 in the requirement, renumber existing chart widgets: set `status-distribution` to `order: 0`, `stages-distribution` to `order: 1`, `velocity-chart` to `order: 2`.
- Use `defineAsyncComponent` for lazy loading (NFR-1).

**Acceptance criteria:**

- [ ] The "Stages Distribution" widget appears on the dashboard between Status Distribution and Velocity Chart.
- [ ] The widget is lazy-loaded via `defineAsyncComponent` (not in the initial JS bundle).
- [ ] The widget renders correctly in single-column (< 1024 px) and two-column (>= 1024 px) layouts.
- [ ] Existing widgets (Status Distribution, Velocity Chart, Summary Counts, Activity Feed) are unaffected.

## Milestone 5: Responsive resize

**Description:** Ensure the chart resizes correctly when the viewport changes.

**Files to change:**

- `web/src/components/dashboard/widgets/StagesDistributionWidget.vue` — add ResizeObserver.

**Implementation details:**

- Use `ResizeObserver` to call `chart.resize()` on container size changes, same pattern as `StatusDistributionWidget`.
- Disconnect observer and dispose chart in `onUnmounted`.

**Acceptance criteria:**

- [ ] The chart resizes correctly on window/container resize.
- [ ] ResizeObserver is disconnected on component unmount.
- [ ] ECharts instance is disposed on component unmount (no memory leaks).
