---
title: "Test Plan: Dashboard Velocity and Activity Side-by-Side Layout"
type: plan-test
status: approved
lineage: dashboard-velocity-activity-side-by-side
parent: lifecycle/requirements/dashboard-velocity-activity-side-by-side-2.md
created: "2026-05-10T00:00:00+10:00"
---

# Test Plan: Dashboard Velocity and Activity Side-by-Side Layout

## Overview

Integration and visual tests for the [[dashboard-velocity-activity-side-by-side]] layout change. Tests verify the two-column layout behaviour across viewport sizes, responsive breakpoint transitions, chart resize handling, and absence of regressions in widget functionality. Tests run against a live dev server with seeded lifecycle artifacts.

Depends on [[dashboard-velocity-activity-side-by-side]] frontend plan for implementation. Interacts with [[dashboard-velocity-activity-side-by-side]] backend plan for API contract stability.

## Milestone 1: Test Fixture Verification

**Description:** Confirm existing test fixtures produce dashboard data sufficient for both the velocity chart and activity feed to render with content. The layout tests need both widgets to have visible content to verify side-by-side rendering.

**Files to change:**

- `tests/fixtures/` — review existing fixtures; ensure at least one artifact has `status: done` (to produce a velocity chart bar) and that the feed endpoint returns at least one event. Add fixtures if needed.
- `tests/helpers/` — if a seed/reset helper exists, verify it loads fixtures that produce both velocity data and feed events.

**Acceptance criteria:**

- [ ] Test fixtures produce a dashboard where both the velocity chart and activity feed render with visible content.
- [ ] Fixtures are isolated per test run.

## Milestone 2: Two-Column Layout Tests

**Description:** Test that the velocity chart and activity feed render side by side on desktop viewports and stack vertically on narrow viewports.

**Files to change:**

- `tests/web/dashboard-velocity-activity-layout.test.ts` (new file).

**Test cases:**

1. **Desktop side-by-side (1280 px)** — Set viewport to 1280 × 800. Navigate to the dashboard. Assert: the velocity chart and activity feed are in the same horizontal row (their `getBoundingClientRect().top` values are equal or within a small tolerance). Assert: both widgets are fully visible without vertical scrolling past the dashboard header.

2. **Narrow desktop side-by-side (900 px)** — Set viewport to 900 × 800. Assert: widgets are still side by side; neither is narrower than 360 px.

3. **Mobile stacked (600 px)** — Set viewport to 600 × 800. Assert: the velocity chart is above the activity feed (velocity `top` < activity `top`). Assert: both are full container width.

4. **At breakpoint boundary (768 px)** — Set viewport to 768 × 800. Assert: widgets are side by side. Set viewport to 767 × 800. Assert: widgets are stacked.

**Acceptance criteria:**

- [ ] All four test cases pass.
- [ ] Tests use viewport resizing, not mocked CSS.

## Milestone 3: Responsive Transition Tests

**Description:** Test that resizing the browser window across the breakpoint transitions the layout smoothly.

**Files to change:**

- `tests/web/dashboard-velocity-activity-layout.test.ts` (continued).

**Test cases:**

1. **Wide to narrow transition** — Start at 1280 px. Resize to 600 px. Assert: layout transitions to stacked without page reload. Assert: velocity chart is above activity feed.

2. **Narrow to wide transition** — Start at 600 px. Resize to 1280 px. Assert: layout transitions to side by side. Assert: the velocity chart resizes (its `clientWidth` changes to approximately half the container).

3. **No visible glitch** — During transitions in test cases 1 and 2, assert no JavaScript errors are thrown and no layout elements disappear temporarily.

**Acceptance criteria:**

- [ ] Both transition directions work correctly.
- [ ] No console errors during transitions.

## Milestone 4: ECharts Resize Tests

**Description:** Test that the velocity chart's ECharts instance resizes correctly in the new layout.

**Files to change:**

- `tests/web/dashboard-velocity-activity-layout.test.ts` (continued).

**Test cases:**

1. **Chart proportional in column** — At 1280 px viewport, assert: the velocity chart canvas width is approximately half the dashboard content width (accounting for gap and padding), not full width.

2. **Chart resize on breakpoint cross** — Start at 1280 px. Record chart canvas width. Resize to 600 px. Wait 300 ms (debounce). Assert: chart canvas width has increased to approximately full container width.

3. **Axis labels legible** — At 1280 px viewport, assert: the chart container width is >= 360 px (the minimum from FR-3).

**Acceptance criteria:**

- [ ] Chart canvas width adapts to column width, not viewport width.
- [ ] Chart resize completes within the 150 ms debounce window + a small tolerance.

## Milestone 5: Widget Functionality Regression Tests

**Description:** Verify existing widget functionality is not broken by the layout change.

**Files to change:**

- `tests/web/dashboard-velocity-activity-layout.test.ts` (continued) or extend existing test files:
  - `tests/web/dashboard-clickable-filters.test.ts` — run existing tests to confirm no regressions.

**Test cases:**

1. **Activity feed real-time update** — On the dashboard at 1280 px viewport, trigger a `feed.new` WebSocket event. Assert: the new event appears in the activity feed without page reload.

2. **Velocity chart tooltips** — At 1280 px viewport, hover over a bar in the velocity chart. Assert: a tooltip appears with the period and count.

3. **Granularity toggle** — At 1280 px viewport, click the "Weekly" granularity button. Assert: the chart re-renders with weekly data.

4. **Activity feed "View all" link** — Click the "View all" button. Assert: navigates to the project feed page.

5. **Keyboard navigation order** — Tab through the dashboard. Assert: focus reaches the velocity widget before the activity widget.

6. **DOM reading order** — Assert: in the DOM tree, the velocity widget element precedes the activity widget element (regardless of CSS layout mode).

**Acceptance criteria:**

- [ ] All six test cases pass.
- [ ] No regressions in existing dashboard test suites (`tests/web/dashboard-clickable-filters.test.ts`, `tests/integration/dashboard_e2e_test.go`).

## Milestone 6: Layout Shift (CLS) Test

**Description:** Verify the layout change does not introduce cumulative layout shift on page load.

**Files to change:**

- `tests/web/dashboard-velocity-activity-layout.test.ts` (continued).

**Test cases:**

1. **No CLS on cold load** — Navigate to the dashboard with a cold browser context. Use the `PerformanceObserver` API (or Playwright's `page.evaluate` to query `performance.getEntriesByType('layout-shift')`) to measure CLS attributable to the side-by-side container. Assert: CLS contribution is 0 or negligible (< 0.01).

**Acceptance criteria:**

- [ ] CLS attributable to the side-by-side container is < 0.01 on initial load.
- [ ] Widget containers have reserved height before content loads.
