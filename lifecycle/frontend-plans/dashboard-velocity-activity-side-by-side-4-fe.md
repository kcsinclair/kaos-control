---
title: "Frontend Plan: Dashboard Velocity and Activity Side-by-Side Layout"
type: plan-frontend
status: in-development
lineage: dashboard-velocity-activity-side-by-side
parent: lifecycle/requirements/dashboard-velocity-activity-side-by-side-2.md
created: "2026-05-10T00:00:00+10:00"
---

# Frontend Plan: Dashboard Velocity and Activity Side-by-Side Layout

## Overview

Restructure the dashboard layout so that the Completion Velocity chart and the Recent Activity panel render side by side in a two-column row on desktop viewports (>= 768 px), collapsing to a single stacked column below that breakpoint. The change is purely CSS/template — no new dependencies, no API changes, no new components.

Currently, VelocityChartWidget sits in the `chart` slot (bottom sub-row, full-width) and ActivityFeedWidget sits in the `panel` slot (separate section below). The plan moves both widgets into a shared two-column container within DashboardGrid.

Depends on [[dashboard-velocity-activity-side-by-side]] backend plan confirming API contracts are stable. Interacts with [[dashboard-velocity-activity-side-by-side]] test plan for acceptance verification.

## Milestone 1: Create Side-by-Side Container in DashboardGrid

**Description:** Modify `DashboardGrid.vue` to render VelocityChartWidget and ActivityFeedWidget together in a new two-column CSS Grid row, replacing the current separate chart-bottom and panel sections for these two widgets.

**Files to change:**

- `web/src/components/dashboard/DashboardGrid.vue`:
  - Add a new `<section>` between the charts section and the panels section (or replace the bottom chart sub-row and panel section) that contains exactly the velocity-chart and activity-feed widgets.
  - Use CSS Grid with `grid-template-columns: 1fr 1fr` for the two-column layout.
  - Define a CSS custom property `--dashboard-side-by-side-bp: 768px` for the responsive breakpoint.
  - At `@media (max-width: 767px)`, switch to `grid-template-columns: 1fr` so widgets stack vertically (velocity on top).
  - Apply `gap: var(--space-4)` to match existing inter-widget spacing.
  - Ensure `align-items: start` so widgets top-align (FR-5).
  - The velocity widget must be the first child in DOM order (NFR-3: reading order).

- `web/src/components/dashboard/registerWidgets.ts`:
  - Optionally adjust the widget registry to distinguish the velocity and activity widgets from other chart/panel widgets, or handle them specially in DashboardGrid. The simplest approach: add a new slot value (e.g., `'side-by-side'`) to the `WidgetSlot` type and update the two registrations, or handle them by ID in DashboardGrid without changing the registry.

- `web/src/components/dashboard/widgetRegistry.ts`:
  - If adding a new slot type, extend the `WidgetSlot` union to include `'side-by-side'`.

**Acceptance criteria:**

- [ ] On viewports >= 1280 px, VelocityChartWidget and ActivityFeedWidget render side by side in a single row, both fully visible without vertical scrolling.
- [ ] On viewports >= 768 px and < 1280 px, the side-by-side layout is maintained with columns flexing to fill available width.
- [ ] On viewports < 768 px, the two widgets stack vertically with VelocityChartWidget on top.
- [ ] The container uses CSS Grid or Flexbox — no JavaScript-driven layout.
- [ ] Both columns have equal width (`1fr 1fr`).
- [ ] Gap between columns matches `var(--space-4)`.
- [ ] Widgets align at the top of the row (no vertical centring).
- [ ] DOM order is velocity-then-activity in both layout modes.

## Milestone 2: Minimum Widget Width and Layout Collapse

**Description:** Ensure each widget respects its minimum usable width. If the column width falls below the velocity chart's 360 px minimum, the layout must collapse to single-column mode.

**Files to change:**

- `web/src/components/dashboard/DashboardGrid.vue`:
  - Set `grid-template-columns: repeat(2, minmax(360px, 1fr))` (or equivalent) so that when the container cannot fit two 360 px columns plus gap, CSS Grid auto-wraps to a single column.
  - Alternatively, use a media query calculated from `2 × 360px + gap` as an additional breakpoint.

- `web/src/components/dashboard/widgets/VelocityChartWidget.vue`:
  - Add `min-width: 360px` to the `.velocity-widget` style to ensure the chart never renders narrower than 360 px.

- `web/src/components/dashboard/widgets/ActivityFeedWidget.vue`:
  - Confirm existing `min-width: 0` does not conflict; the panel's natural content width is already sufficient. No changes expected unless testing reveals truncation.

**Acceptance criteria:**

- [ ] The velocity chart never renders narrower than 360 px.
- [ ] When the container width is too narrow for two 360 px columns, the layout collapses to single-column.
- [ ] Activity feed entries are not truncated in two-column mode at any supported viewport width.

## Milestone 3: ECharts Resize Handling

**Description:** Verify and ensure the ECharts velocity chart resizes correctly when column width changes due to the new layout.

**Files to change:**

- `web/src/components/dashboard/widgets/VelocityChartWidget.vue`:
  - The existing `ResizeObserver` on `chartEl` with 150 ms debounce already calls `chart.resize()` on width changes. Verify this works correctly when the chart is in a narrower column (approximately half viewport width instead of full width).
  - Confirm `containerWidth` is updated correctly and that the DataZoom scroll threshold adapts to the narrower column width.
  - No code changes expected — this milestone is verification that the existing resize logic handles the new layout.

**Acceptance criteria:**

- [ ] On initial mount in two-column mode, the chart renders proportionally within its column.
- [ ] Resizing the browser window within two-column mode causes the chart to resize smoothly (bars and labels remain proportional and legible).
- [ ] Crossing the 768 px breakpoint triggers a chart resize as the layout transitions between single and two-column modes.
- [ ] DataZoom (scroll/slider) activates correctly based on the narrower column width.

## Milestone 4: Prevent Layout Shift (CLS)

**Description:** Ensure the two-column layout does not cause visible content layout shift during page load.

**Files to change:**

- `web/src/components/dashboard/DashboardGrid.vue`:
  - Set a `min-height` on the side-by-side container to reserve vertical space while async widget components load.

- `web/src/components/dashboard/widgets/VelocityChartWidget.vue`:
  - The chart container already has an explicit `height` style (`chartHeight + 'px'`, defaulting to 240 px). Confirm this prevents CLS. Consider setting `min-height: 240px` on the widget wrapper as a fallback during async load.

- `web/src/components/dashboard/widgets/ActivityFeedWidget.vue`:
  - The `.activity-feed-body` already has `max-height: 560px`. Consider adding a `min-height` matching the velocity chart height to prevent the row from shifting as feed items load.

**Acceptance criteria:**

- [ ] No measurable Cumulative Layout Shift (CLS > 0) on initial dashboard load attributable to the side-by-side container.
- [ ] Widget containers reserve their height before content loads.

## Milestone 5: Regression Verification

**Description:** Verify that all existing widget functionality is preserved after the layout change.

**Files to change:** None — this is a verification-only milestone.

**Acceptance criteria:**

- [ ] WebSocket-driven real-time updates in the activity feed still work (new events appear without page refresh).
- [ ] Chart tooltips on the velocity chart work correctly in the narrower column.
- [ ] Activity feed "View all" link navigates correctly.
- [ ] Granularity toggle (Daily/Weekly/Monthly) on the velocity chart works.
- [ ] [[dashboard-clickable-filters]] click-through functionality is unaffected.
- [ ] Keyboard navigation order remains logical: velocity widget is reachable before activity widget.
- [ ] Other dashboard sections (summary cards, top chart row) are unaffected.
