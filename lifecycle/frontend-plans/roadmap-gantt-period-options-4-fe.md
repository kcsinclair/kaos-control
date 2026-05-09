---
title: "Frontend Plan: Roadmap Gantt Period Display Options"
type: plan-frontend
status: draft
lineage: roadmap-gantt-period-options
parent: lifecycle/requirements/roadmap-gantt-period-options-2.md
created: "2026-05-10T00:00:00+10:00"
labels:
    - roadmaps
    - frontend
    - enhancement
    - usability
release: KC-Release0
assignees:
    - role: frontend-developer
      who: agent
---

# Frontend Plan: Roadmap Gantt Period Display Options

This plan covers all UI and client-side logic for period display options in the
Gantt chart. The feature adds a period-mode selector to the Roadmap toolbar and
modifies the `timeRange` computation in `GanttChart.vue` to support autoscale
and fixed-period modes. The configurable default is read from the project config
API (see [[roadmap-gantt-period-options]] backend plan, Milestone 3).

---

## Milestone 1: Add period-mode state to the releases store

### Description

Add session-persisted period-mode state to the Pinia releases store (or a new
dedicated roadmap-settings store) so the selection survives tab/view switches
within the same session. State needed:

- `periodMode: 'autoscale' | 'fixed'` — defaults to `'autoscale'`.
- `fixedPeriod: 'month' | 'quarter' | 'half-year' | 'year'` — defaults to
  `'month'` (only relevant when `periodMode === 'fixed'`).
- `defaultPeriodModeLoaded: boolean` — tracks whether the config-API default
  has been applied (prevents overwriting a user selection on re-mount).

On store initialisation, fetch the project config endpoint to read
`roadmap.default_period_mode`. If the value is one of the four fixed-period
strings, set `periodMode = 'fixed'` and `fixedPeriod` accordingly; otherwise
set `periodMode = 'autoscale'`.

### Files to change

- `web/src/stores/releases.ts` — add `periodMode`, `fixedPeriod`,
  `defaultPeriodModeLoaded` refs and a `loadDefaultPeriodMode(project)` action.
  Alternatively, create `web/src/stores/roadmapSettings.ts` if separation is
  preferred.

### Acceptance criteria

- [ ] `periodMode` and `fixedPeriod` are reactive Pinia state.
- [ ] Switching between Gantt and Graph views and back preserves the period-mode
      selection.
- [ ] The config-API default is applied only once per session (not on every
      re-mount of `RoadmapView`).
- [ ] `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 2: Add period-mode selector to the Roadmap toolbar

### Description

Add a segmented control to `RoadmapView.vue` toolbar (between the granularity
control and the view-mode toggle) with two segments: **Autoscale** and **Fixed
Period**. When "Fixed Period" is active, render a secondary dropdown or
segmented control with options: Month, Quarter, Half-Year, Year.

Both controls are only visible when `viewMode === 'gantt'`.

### Files to change

- `web/src/views/project/RoadmapView.vue` — add the period-mode segmented
  control and the conditional fixed-period selector, wired to the store state
  from Milestone 1.

### Acceptance criteria

- [ ] Period-mode selector is visible in the toolbar when Gantt view is active.
- [ ] Period-mode selector is hidden when Graph view is active.
- [ ] Selecting "Fixed Period" reveals the secondary period picker.
- [ ] Selecting "Autoscale" hides the secondary period picker.
- [ ] All controls have `role="group"` and `aria-label` attributes.
- [ ] All buttons are keyboard-navigable (Tab + Enter/Space).
- [ ] Toolbar does not wrap or overflow on viewports >= 1024 px.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 3: Implement autoscale time-range computation

### Description

Modify the `timeRange` computed property in `GanttChart.vue` (lines 65-115) to
support autoscale mode. When `periodMode === 'autoscale'`:

1. Set `start` to the earliest `start_date` among scheduled releases, snapped
   to the current granularity boundary (using the existing `startOfWeek`,
   `startOfMonth`, etc. helpers). **Do not** add padding columns.
2. Set `end` to the latest `end_date` among scheduled releases, snapped to the
   end of the current granularity boundary. **Do not** add padding columns.
3. If no releases are scheduled, show a single column containing today's date.

The current logic (lines 65-115) adds padding of one column unit on each side
and includes TODAY in the range even when releases exist. Autoscale removes
both behaviours — the axis covers exactly the release span.

### Files to change

- `web/src/components/releases/GanttChart.vue` — accept `periodMode` and
  `fixedPeriod` as props; refactor `timeRange` computed to branch on
  `periodMode`.

### Acceptance criteria

- [ ] Autoscale with scheduled releases shows exactly the range they span —
      no empty leading/trailing columns beyond the granularity snap.
- [ ] Autoscale with zero scheduled releases shows a single column containing
      today.
- [ ] The existing granularity control continues to work independently of
      period mode.
- [ ] No additional API calls are made when switching modes.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 4: Implement fixed-period time-range computation

### Description

Add fixed-period logic to the `timeRange` computed property. When
`periodMode === 'fixed'`:

1. Anchor `start` to the beginning of the current calendar period containing
   today:
   - Month: `startOfMonth(TODAY)`
   - Quarter: `startOfQuarter(TODAY)`
   - Half-Year: `startOfHalfYear(TODAY)`
   - Year: `startOfYear(TODAY)`
2. Set `end` to the last day of that same period.
3. Clamp column generation to this window regardless of release dates.

### Files to change

- `web/src/components/releases/GanttChart.vue` — extend the `timeRange`
  computed with a `'fixed'` branch; add a helper function
  `endOfPeriod(period, date)` that returns the last day of the given calendar
  period.

### Acceptance criteria

- [ ] Fixed Period > Month shows only the current calendar month.
- [ ] Fixed Period > Quarter shows only the current calendar quarter.
- [ ] Fixed Period > Half-Year shows only the current half-year.
- [ ] Fixed Period > Year shows only the current calendar year.
- [ ] Switching fixed-period values re-renders within 100 ms for <= 50 releases.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 5: Bar clipping and overflow indicators

### Description

When a release bar extends outside the fixed-period window, it must render only
the visible portion (clipped at window boundaries), not be hidden entirely.
The existing `pct()` helper already clamps to 0-100%, so bars are naturally
clipped by the `overflow: hidden` on `.row-track`. Add a visual indicator
(a small arrow or chevron) at the clipped edge to signal that the bar extends
beyond the visible area.

### Files to change

- `web/src/components/releases/GanttChart.vue` — in the `scheduledBars`
  computed, detect when a bar's start or end date falls outside the time range;
  add `clippedLeft` / `clippedRight` boolean flags to `BarInfo`. Render a
  CSS-based arrow indicator on the clipped edge(s).

### Acceptance criteria

- [ ] A release bar starting before the fixed-period window renders from the
      left edge with a left-pointing clip indicator.
- [ ] A release bar ending after the fixed-period window renders to the right
      edge with a right-pointing clip indicator.
- [ ] A bar fully outside the window is not rendered (no empty row).
- [ ] Clip indicators are not shown in autoscale mode (bars always fit).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 6: Horizontal scrolling with sticky release labels

### Description

The `.gantt-wrap` container already has `overflow: auto`, so horizontal
scrolling works when content overflows. This milestone ensures that when a
fine granularity (e.g., Week) is combined with a large fixed period (e.g.,
Year), the chart becomes horizontally scrollable and the release label column
(left side) remains fixed/sticky.

Currently the release name is inside the bar itself (`.bar-name`). For sticky
labels, add a fixed-width label column on the left of each `.gantt-row`
(mirroring the existing unscheduled sticky column on the right).

Also enforce the 200-column safety cap (already at line 175): when the
combination of granularity + period would exceed 200 columns, auto-coarsen the
granularity to the next level and display a visual indicator (e.g., a small
info badge near the granularity control) explaining the override.

### Files to change

- `web/src/components/releases/GanttChart.vue` — add sticky left label column;
  add auto-coarsen logic to `columns` computed; add info badge template +
  styles.

### Acceptance criteria

- [ ] Horizontal scrolling activates when Gantt content overflows the container.
- [ ] The left label column stays fixed during horizontal scroll.
- [ ] The 200-column safety cap is respected; exceeding it coarsens granularity
      automatically.
- [ ] A visual indicator appears when granularity was auto-coarsened.
- [ ] The existing sticky unscheduled column on the right remains functional.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 7: Pass props and wire up RoadmapView

### Description

Wire everything together: pass `periodMode` and `fixedPeriod` from the store
through `RoadmapView.vue` as props to `GanttChart.vue`. Ensure the config-API
default is loaded on mount. Verify end-to-end behaviour.

### Files to change

- `web/src/views/project/RoadmapView.vue` — read period-mode state from store;
  pass as props to `GanttChart`; call `loadDefaultPeriodMode(project)` on mount.

### Acceptance criteria

- [ ] Default mode on first load matches the project config (falls back to
      Autoscale if unconfigured).
- [ ] User selection in the toolbar updates the store and re-renders the chart.
- [ ] Switching to Graph view and back preserves the period-mode selection.
- [ ] No additional API calls to the releases endpoint when switching modes.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.
