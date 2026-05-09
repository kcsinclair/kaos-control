---
title: "Frontend Plan: Recent Ideas and Defects Dashboard Widget"
type: plan-frontend
status: draft
lineage: dashboard-recent-ideas-defects-widget
parent: lifecycle/requirements/dashboard-recent-ideas-defects-widget-2.md
created: "2026-05-09"
---

# Frontend Plan: Recent Ideas and Defects Dashboard Widget

This plan covers the Vue 3 frontend changes: a new `RecentIdeasDefectsWidget` component, its registration in the widget registry, and the dashboard layout restructure. Depends on [[dashboard-recent-ideas-defects-widget]] backend plan (-3-be) for multi-value `type` filter and `sort` query parameter on the artifacts API.

---

## Milestone 1: Create `RecentIdeasDefectsWidget` component

### Description

Build a new widget component that fetches the 6 most recent ideas and defects and displays them as a clickable list with type badges and relative timestamps.

### Files to change

- `web/src/components/dashboard/widgets/RecentIdeasDefectsWidget.vue` (new file)
  - **Props**: `project: string` (same pattern as all other widgets).
  - **Data fetching**: `onMounted`, call `GET /api/p/{project}/artifacts?type=idea,defect&sort=created:desc&limit=6`. Store result in a `ref<ArtifactRow[]>`.
  - **WebSocket**: Use `useWebSocket(project, 'artifact.indexed', handler)` to refetch when an idea or defect is created/updated/deleted. The handler should call the same fetch function.
  - **Template**:
    - Card wrapper matching existing widget styling (see `ActivityFeedWidget` for reference).
    - Title: "Recent Ideas & Defects".
    - List of up to 6 items. Each item is a `<router-link :to="'/p/' + project + '/artifacts/' + item.path">` containing:
      - Type badge: `<span>` with `idea` or `defect` text, distinct background colours per type. Colours must meet WCAG 2.1 AA contrast against the badge text.
      - Artifact title (truncated with CSS `text-overflow: ellipsis` if needed).
      - Relative timestamp (e.g. "2 hours ago") computed from `item.created`. Use a small utility or inline `Intl.RelativeTimeFormat`.
    - Empty state: "No recent ideas or defects" message when the list is empty.
  - **Accessibility**:
    - Each list item is a `<router-link>` (renders `<a>`, natively keyboard-focusable).
    - Visible focus indicators (`:focus-visible` outline).
    - Type badges have `aria-label="Type: idea"` or `aria-label="Type: defect"`.
  - **Styling**: Scoped `<style>`. Card max-height should accommodate 6 items without scroll. Consistent spacing and font sizes with `ActivityFeedWidget`.

### Acceptance criteria

- Widget renders a list of up to 6 items with title, type badge, and relative timestamp.
- Each item links to `/p/{project}/artifacts/{path}`.
- Widget auto-refreshes on `artifact.indexed` WebSocket event.
- Empty state message displays when no ideas or defects exist.
- Type badges have distinct colours for `idea` (e.g. blue-tinted) and `defect` (e.g. red/amber-tinted) meeting WCAG 2.1 AA contrast.
- All items are keyboard-navigable with visible focus indicators.

---

## Milestone 2: Register widget in the registry

### Description

Register the new widget so it appears in the dashboard grid in the correct slot and position.

### Files to change

- `web/src/components/dashboard/registerWidgets.ts`
  - Add a `defineAsyncComponent` import for `RecentIdeasDefectsWidget.vue`.
  - Call `registerWidget('recent-ideas-defects', component, { slot: 'chart', order: 1.5 })` — order between `stages-distribution` (order 1) and `velocity-chart` (order 2) so it renders third in the chart slot, after the two pie charts and before velocity.

  Note: The exact order value may need adjustment in milestone 3 when the layout restructure changes how chart-slot widgets are rendered. The intent is for this widget to appear in the top row alongside the two pie charts.

### Acceptance criteria

- Widget appears in the dashboard without manual configuration.
- Widget is lazy-loaded (async component) — does not increase initial bundle size.
- Widget renders in the chart slot, positioned after the two pie charts.

---

## Milestone 3: Dashboard layout restructure

### Description

Restructure `DashboardGrid.vue` to implement the new column layout:
- Main grid changes from `1fr 340px` to `2fr 1fr` (charts column gets two-thirds, panels column one-third, both flexible).
- Within the charts section, the first three chart widgets (StatusDistribution, StagesDistribution, RecentIdeasDefects) render in a three-column top row.
- The VelocityChart widget spans two-thirds width beneath the top row, aligned under the two pie charts.

### Files to change

- `web/src/components/dashboard/DashboardGrid.vue`
  - **Template changes**:
    - Split `chartWidgets` into two groups: `topRowWidgets` (first 3 by order: status-distribution, stages-distribution, recent-ideas-defects) and `remainingChartWidgets` (velocity-chart and any future widgets).
    - Wrap `topRowWidgets` in a `<div class="dashboard-charts-top">` with a 3-column grid.
    - Wrap `remainingChartWidgets` in a `<div class="dashboard-charts-bottom">` — velocity widget gets a class for two-thirds width.
  - **CSS changes**:
    - `.dashboard-main` at `min-width: 1024px`: change `grid-template-columns` from `1fr 340px` to `2fr 1fr`.
    - `.dashboard-charts-top`: `display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--space-4)`.
    - `.dashboard-charts-bottom`: velocity widget spans `width: 66.66%` or uses a 3-column sub-grid with `grid-column: span 2`.
    - Responsive: at `max-width: 1023px`, `.dashboard-charts-top` collapses to `grid-template-columns: 1fr` (single column, widgets stack).
  - **Computed changes**:
    - Add `topRowWidgets` computed: `chartWidgets.slice(0, 3)`.
    - Add `bottomChartWidgets` computed: `chartWidgets.slice(3)`.

### Acceptance criteria

- The charts column is wider than the panels column (approximately 2:1 ratio, both flexible).
- The top row of the charts column shows three widgets side-by-side in equal-width cells.
- The Completion Velocity widget appears below the top row, spanning two-thirds of the charts column width.
- On viewports narrower than 1024px, the top-row widgets stack vertically and the two-column main layout collapses to a single column.
- Existing widgets (summary counts, activity feed) continue to render correctly in their respective slots.
- No horizontal overflow at any supported viewport width.

---

## Milestone 4: Visual polish and responsive testing

### Description

Final pass on visual consistency, responsive behaviour, and accessibility.

### Files to change

- `web/src/components/dashboard/widgets/RecentIdeasDefectsWidget.vue` — minor CSS tweaks if needed after layout integration.
- `web/src/components/dashboard/DashboardGrid.vue` — responsive breakpoint adjustments if testing reveals issues.

### Acceptance criteria

- Widget card style, spacing, font sizes, and colours match existing dashboard widgets.
- At 1024px+ viewport: three-column top row renders cleanly with no overflow or wrapping.
- At <1024px viewport: all widgets stack vertically, layout remains usable.
- Relative timestamps update correctly (or are static — consistent with the rest of the app).
- Type badge colours are visually distinct and pass WCAG 2.1 AA contrast check.
- Widget data renders within 200 ms of API response arrival (no unnecessary re-renders or layout shifts).
