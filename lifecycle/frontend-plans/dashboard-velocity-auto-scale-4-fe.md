---
title: "Frontend Plan — Velocity Widget Auto-Scaling Columns and Minimum Periods"
type: plan-frontend
status: draft
lineage: dashboard-velocity-auto-scale
parent: lifecycle/requirements/dashboard-velocity-auto-scale-2.md
created: "2026-05-09"
labels:
    - frontend
    - vue
    - enhancement
---

# Frontend Plan — Velocity Widget Auto-Scaling Columns and Minimum Periods

## Context

The CompletionVelocity widget (`web/src/components/dashboard/widgets/VelocityChartWidget.vue`) currently renders bars at whatever width ECharts auto-calculates, with no minimum period guarantees, no horizontal scrolling, and a `weekly` default granularity. `DataZoomComponent` is imported but unused.

This plan implements all five functional requirements from [[dashboard-velocity-auto-scale]]:
- FR-1: Minimum periods per granularity
- FR-2: Auto-scaling column widths
- FR-3: Horizontal scrolling via ECharts DataZoom
- FR-4: Default granularity → daily
- FR-5: Responsive resize

---

## Milestone 1 — Change default granularity to daily and send `days` parameter

### Description

Change the default `granularity` ref from `'weekly'` to `'daily'`. Compute the appropriate `days` query parameter based on the active granularity to ensure the backend returns at least the minimum number of periods:

| Granularity | Min periods | Days to request |
|-------------|-------------|-----------------|
| Daily       | 7           | max(7, 90) = 90 |
| Weekly      | 4           | max(28, 90) = 90 |
| Monthly     | 3           | max(90, 90) = 90 |

Since the default lookback (90 days) already covers all minimums, we send `days=90` for all granularities. If future requirements increase minimums, this computed value will scale.

### Files to change

- `web/src/components/dashboard/widgets/VelocityChartWidget.vue` — line 26: change `'weekly'` to `'daily'`; update fetch URL to include `&days=` parameter

### Acceptance criteria

- [ ] Widget mounts with Daily granularity selected and the daily toggle button shows active state
- [ ] The API request includes `days=90` in the query string
- [ ] Switching granularity still works and triggers a new fetch
- [ ] `pnpm exec vue-tsc --noEmit` passes

---

## Milestone 2 — Client-side period padding for minimum periods

### Description

After fetching buckets from the API, ensure at least the minimum number of periods are present. If the API returns fewer buckets than the minimum (e.g. a brand-new project), pad the beginning of the series with zero-count entries using the appropriate date format. This implements FR-1.

Add a `MIN_PERIODS` constant map and a `padBuckets` function that:
1. Checks if `buckets.length < MIN_PERIODS[granularity]`
2. If so, generates missing period keys by back-dating from the earliest returned period (or from today if no buckets)
3. Prepends zero-count entries
4. Returns the padded array

### Files to change

- `web/src/components/dashboard/widgets/VelocityChartWidget.vue` — add `MIN_PERIODS` constant and `padBuckets()` function; call it in `fetchAndRender()` before building chart options

### Acceptance criteria

- [ ] With a project that has only 2 days of data, Daily view shows 7 columns (5 padded + 2 real)
- [ ] With a project that has 1 week of data, Weekly view shows 4 columns
- [ ] With a project that has 1 month of data, Monthly view shows 3 columns
- [ ] Padded periods show `count: 0` and appear at the left of the chart
- [ ] `aria-label` correctly reflects the actual total (not counting padding as completions)

---

## Milestone 3 — Auto-scaling column widths

### Description

Implement FR-2: when the number of periods fits within the visible chart width, bars should distribute evenly. Define a `MIN_BAR_WIDTH` constant (e.g. 20px) to determine the threshold between "fits in view" and "needs scrolling".

Calculate: `maxVisibleBars = floor(chartWidth / MIN_BAR_WIDTH)`. If `periods.length <= maxVisibleBars`, let ECharts auto-size (which fills the container). Set `barMaxWidth` to prevent excessively wide bars when there are very few periods (cap at e.g. 60px).

Store the chart container width by reading it from the `ResizeObserver` callback (already wired up) and save it to a reactive ref.

### Files to change

- `web/src/components/dashboard/widgets/VelocityChartWidget.vue` — add `containerWidth` ref updated by ResizeObserver; add `MIN_BAR_WIDTH` and `MAX_BAR_WIDTH` constants; apply `barMaxWidth` in series config; compute `needsScroll` boolean

### Acceptance criteria

- [ ] With <= 14 daily periods, bars fill the available width proportionally
- [ ] Bars never exceed 60px width even with very few periods (e.g. 3 monthly)
- [ ] No fixed pixel bar width — widths adapt to container size
- [ ] Resizing the browser window recalculates bar widths without reload

---

## Milestone 4 — Horizontal scrolling via ECharts DataZoom

### Description

Implement FR-3: when `periods.length > maxVisibleBars`, enable ECharts `dataZoom` for horizontal scrolling. Use two dataZoom components:

1. **`type: 'inside'`** — enables mouse wheel horizontal scroll and touch swipe
2. **`type: 'slider'`** — visible slider bar below the chart for drag-based scrolling

Configure both to show the rightmost data by default (`start`/`end` percentages calculated so the last `maxVisibleBars` periods are visible). The slider must be keyboard-accessible (ECharts provides this natively).

Increase the grid `bottom` margin when the slider is shown to avoid label/slider overlap. Increase the `.velocity-chart` height by ~30px when the slider is visible to accommodate it.

### Files to change

- `web/src/components/dashboard/widgets/VelocityChartWidget.vue` — add `dataZoom` array to `setOption` conditionally based on `needsScroll`; adjust grid bottom padding and chart container height dynamically

### Acceptance criteria

- [ ] With 30+ daily periods, a DataZoom slider appears below the chart
- [ ] View defaults to showing the most recent (rightmost) periods
- [ ] Shift+mouse wheel scrolls horizontally
- [ ] Touch swipe scrolls the chart on touch devices
- [ ] DataZoom slider is keyboard-navigable (Tab to focus, arrow keys to scroll)
- [ ] With few periods (e.g. 7 daily), no DataZoom slider appears
- [ ] Grid bottom margin adjusts so x-axis labels don't overlap the slider

---

## Milestone 5 — Responsive resize integration

### Description

Ensure FR-5: the existing `ResizeObserver` continues to work and now also triggers a recalculation of `needsScroll` and `maxVisibleBars`. When the container resizes:

1. Update `containerWidth` ref
2. Recalculate whether scrolling is needed
3. Re-apply `setOption` with updated `dataZoom` config (add or remove slider)
4. Call `chart.resize()`

Add debouncing (150ms) to the ResizeObserver callback to avoid excessive recalculations during continuous resize.

### Files to change

- `web/src/components/dashboard/widgets/VelocityChartWidget.vue` — update ResizeObserver callback to set `containerWidth` and call a debounced re-render; add debounce utility (inline, no new file)

### Acceptance criteria

- [ ] Resizing browser from wide to narrow triggers DataZoom to appear when bars would be too compressed
- [ ] Resizing browser from narrow to wide removes DataZoom when all bars fit
- [ ] No janky re-renders during continuous resize (debounce working)
- [ ] `chart.resize()` is still called so ECharts canvas matches container
- [ ] `pnpm build` passes with zero errors
- [ ] `pnpm exec vue-tsc --noEmit` passes with zero errors

---

## Milestone 6 — Accessibility and visual consistency check

### Description

Verify NFR-2 and NFR-3 from [[dashboard-velocity-auto-scale]]:

- `aria-label` accurately reports total completions and period count after every granularity switch (already exists, verify it still works with padding)
- DataZoom slider, when rendered, is keyboard-accessible
- Bar styling unchanged: colour `#6366f1`, border-radius `[3, 3, 0, 0]`, emphasis `#4f46e5`
- Widget chrome (border, padding, header layout) unchanged

No code changes expected — this milestone is a verification pass. Fix any regressions found.

### Files to change

- `web/src/components/dashboard/widgets/VelocityChartWidget.vue` — only if regressions are found

### Acceptance criteria

- [ ] `aria-label` reads correctly after switching between all three granularities
- [ ] `aria-label` includes padded period count but only real completion total
- [ ] DataZoom slider can be reached via Tab key and operated via arrow keys
- [ ] Bar colour is `#6366f1`, emphasis is `#4f46e5`, border-radius is `[3, 3, 0, 0]`
- [ ] No visual changes to widget border, padding, or header layout
- [ ] Granularity toggle remains keyboard-navigable with visible focus indicator
