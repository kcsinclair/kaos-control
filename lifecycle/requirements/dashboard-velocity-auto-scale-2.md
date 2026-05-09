---
title: Velocity Widget Auto-Scaling Columns and Minimum Periods
type: requirement
status: blocked
lineage: dashboard-velocity-auto-scale
created: "2026-05-09"
priority: high
parent: lifecycle/ideas/dashboard-velocity-auto-scale.md
labels:
    - enhancement
    - frontend
    - vue
    - usability
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

# Velocity Widget Auto-Scaling Columns and Minimum Periods

## Problem

The Completion Velocity widget currently renders all returned buckets at a fixed chart width with no minimum period guarantees. When the daily view has only a few data points the bars are excessively wide; when there are many points the labels overlap and become unreadable. There is no horizontal scrolling, so data beyond the visible area is either compressed to illegibility or silently clipped by ECharts. Users cannot comfortably compare velocity across planning horizons (sprint vs. quarter) because the widget does not adapt its layout to the chosen granularity.

## Goals / Non-goals

### Goals

1. Enforce **minimum visible periods** per granularity: daily = 7, weekly = 4, monthly = 3.
2. **Auto-scale column widths** so bars distribute evenly across available widget width when the number of periods is at or below the visible area.
3. When periods **exceed the visible area**, introduce horizontal scrolling so all data remains accessible without truncation or compression.
4. Default granularity remains **daily** (changed from current `weekly` default).
5. Maintain existing accessibility attributes (`role`, `aria-label`, keyboard-navigable granularity toggle).

### Non-goals

- Changing the backend `/dashboard/velocity` API contract (bucket calculation, period format, response shape).
- Adding user-configurable minimum periods or custom date ranges.
- Replacing ECharts with a different charting library.
- Adding animations or transitions between granularity switches beyond what ECharts provides natively.

## Detailed Requirements

### Functional

**FR-1 Minimum periods.** The widget must request and display at least the minimum number of periods for the active granularity:

| Granularity | Minimum periods |
|-------------|-----------------|
| Daily       | 7               |
| Weekly      | 4               |
| Monthly     | 3               |

If the API returns fewer buckets than the minimum (e.g. a new project with only 2 days of data), the widget must still render all returned buckets — the minimum is a request/display target, not a hard filter.

**FR-2 Auto-scaling column widths.** When the number of returned periods fits within the widget's visible width, bar widths must scale proportionally to fill the available horizontal space. No fixed pixel width per bar should be hard-coded.

**FR-3 Horizontal scrolling.** When the number of returned periods exceeds what can be displayed legibly at the current widget width, the chart area must become horizontally scrollable. Scrolling must be available via:
- Mouse wheel (horizontal scroll or shift+wheel).
- Touch swipe on touch-capable devices.
- ECharts DataZoom slider (already imported but currently unused).

The scroll position must default to the most recent period (right-most).

**FR-4 Default granularity.** The default granularity on widget mount must be `daily` (currently `weekly`).

**FR-5 Responsive resize.** The existing `ResizeObserver`-based resize behaviour must continue to function correctly after these changes. Column widths must recalculate on container resize.

### Non-functional

**NFR-1 Performance.** Rendering and granularity switching must complete within 200 ms for up to 90 periods (daily = ~3 months). No additional API calls beyond the existing single fetch per granularity change.

**NFR-2 Accessibility.** The `aria-label` must continue to report total completions and period count. The DataZoom slider, if rendered, must be keyboard-accessible.

**NFR-3 Visual consistency.** Bar styling (colour `#6366f1`, border-radius, emphasis colour) and widget chrome (border, padding, header layout) must remain unchanged.

## Acceptance Criteria

- [ ] Selecting **Daily** shows at least 7 day columns; bars auto-scale to fill width when <= 14 days are returned.
- [ ] Selecting **Weekly** shows at least 4 week columns; bars auto-scale to fill width when <= 8 weeks are returned.
- [ ] Selecting **Monthly** shows at least 3 month columns; bars auto-scale to fill width when <= 6 months are returned.
- [ ] When periods exceed the visible area, horizontal scroll is available and the view defaults to the most recent period.
- [ ] Widget opens in **Daily** granularity by default.
- [ ] Resizing the browser window or dashboard layout causes columns to recalculate widths without a page reload.
- [ ] `aria-label` accurately reflects the displayed data after every granularity switch.
- [ ] No regressions in the granularity toggle (keyboard navigation, active state styling).
- [ ] `pnpm build` and `pnpm exec vue-tsc --noEmit` pass with zero errors.
- [ ] Related: [[dashboard-velocity-auto-scale]]

## Open Questions

1. Should the backend enforce the minimum-periods guarantee (always return >= N buckets, padding empty periods with `count: 0`), or should the frontend pad missing periods client-side? — Current assumption: backend already returns the full range; frontend simply renders what it receives.
2. Is ECharts' built-in `dataZoom` slider the preferred scroll UX, or should a native CSS `overflow-x: auto` container wrap the chart canvas? — Current assumption: use ECharts `dataZoom` since it is already imported.
