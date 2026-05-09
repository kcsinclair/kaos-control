---
title: "Test Plan — Velocity Widget Auto-Scaling Columns and Minimum Periods"
type: plan-test
status: in-development
lineage: dashboard-velocity-auto-scale
parent: lifecycle/requirements/dashboard-velocity-auto-scale-2.md
created: "2026-05-09T00:00:00+10:00"
labels:
    - test
    - integration
    - enhancement
---

# Test Plan — Velocity Widget Auto-Scaling Columns and Minimum Periods

## Context

This plan covers integration and end-to-end testing for the [[dashboard-velocity-auto-scale]] feature. It validates the backend `days` parameter behaviour (from [[dashboard-velocity-auto-scale]] backend plan) and the frontend auto-scaling, padding, and DataZoom behaviour (from [[dashboard-velocity-auto-scale]] frontend plan).

Tests are split into backend integration tests (Go, in `tests/`) and a test artifact describing frontend acceptance validation.

---

## Milestone 1 — Backend integration tests for `days` parameter

### Description

Extend the existing `tests/integration/dashboard_velocity_test.go` with test cases that exercise the `days` query parameter explicitly, since the frontend will now send it on every request.

### Test cases

1. **`TestVelocityDaysParam_Explicit`** — `GET /dashboard/velocity?granularity=daily&days=14` returns exactly 14 buckets
2. **`TestVelocityDaysParam_Zero`** — `days=0` falls back to default 90-day window
3. **`TestVelocityDaysParam_Negative`** — `days=-5` falls back to default
4. **`TestVelocityDaysParam_OverMax`** — `days=400` clamps to 365
5. **`TestVelocityDaysParam_NonNumeric`** — `days=abc` falls back to default
6. **`TestVelocityDaysParam_Omitted`** — no `days` param returns 90-day default window

### Files to change

- `tests/integration/dashboard_velocity_test.go` — add the six test cases above

### Acceptance criteria

- [ ] All six test cases pass
- [ ] Each test asserts on the number of returned buckets (not just HTTP 200)
- [ ] Tests run in CI via `make test-unit` or the integration test target
- [ ] No modifications to the backend code required (tests validate existing behaviour)

---

## Milestone 2 — Backend integration tests for zero-fill coverage

### Description

Verify that the backend returns contiguous zero-filled buckets for the full requested window, even with no completion events. This underpins the frontend's ability to pad minimally.

### Test cases

1. **`TestVelocityZeroFill_Daily7`** — empty project, `granularity=daily&days=7` → 7 buckets, all `count: 0`
2. **`TestVelocityZeroFill_Weekly28`** — empty project, `granularity=weekly&days=28` → 4 buckets, all `count: 0`
3. **`TestVelocityZeroFill_Monthly90`** — empty project, `granularity=monthly&days=90` → 3 buckets, all `count: 0`
4. **`TestVelocityZeroFill_Contiguous`** — project with one event in the middle of a 14-day daily window → 14 buckets, one with `count: 1`, rest `count: 0`, keys are contiguous dates

### Files to change

- `tests/integration/dashboard_velocity_test.go` — add the four test cases above

### Acceptance criteria

- [ ] All four tests pass
- [ ] Contiguity test validates that period keys are sequential with no gaps
- [ ] Zero-count buckets are explicitly asserted (not just array length)

---

## Milestone 3 — Frontend test artifact: minimum periods and padding

### Description

Create a test artifact documenting the manual/automated acceptance tests for the frontend padding logic. These correspond to FR-1 from [[dashboard-velocity-auto-scale]].

### Test cases

1. **Minimum daily periods** — new project with 2 days of data: Daily view renders 7 columns (5 zero-padded + 2 real)
2. **Minimum weekly periods** — new project with 1 week of data: Weekly view renders 4 columns
3. **Minimum monthly periods** — new project with 1 month of data: Monthly view renders 3 columns
4. **Padding does not inflate totals** — `aria-label` reports real completions only, not zero-padded entries
5. **Full data no padding** — project with 30 days of data: Daily view renders 30+ columns, no padding added

### Files to change

- `lifecycle/tests/dashboard-velocity-auto-scale-tests.md` — create test artifact describing these cases

### Acceptance criteria

- [ ] Each test case has clear pass/fail criteria
- [ ] Tests are traceable to FR-1 in the requirement
- [ ] Padding boundary conditions are covered (exactly at minimum, below minimum, above minimum)

---

## Milestone 4 — Frontend test artifact: auto-scaling and DataZoom

### Description

Document acceptance tests for FR-2 (auto-scaling), FR-3 (horizontal scrolling), and FR-5 (responsive resize).

### Test cases

1. **Auto-scale few bars** — Daily view with 7 periods in a 600px-wide widget: bars fill available width, no DataZoom slider visible
2. **Auto-scale cap** — Monthly view with 3 periods: bars do not exceed max width (visual check — no bar wider than ~60px)
3. **DataZoom appears** — Daily view with 30+ periods: DataZoom slider renders below the chart
4. **Scroll to recent** — DataZoom defaults to showing the rightmost (most recent) periods
5. **Shift+wheel scroll** — holding Shift and scrolling mouse wheel pans the chart horizontally
6. **Touch swipe** — on a touch device, swiping left/right pans the chart
7. **DataZoom keyboard** — Tab to DataZoom slider, arrow keys adjust the visible range
8. **Resize adds DataZoom** — start with wide window (no DataZoom), resize narrow → DataZoom appears
9. **Resize removes DataZoom** — start with narrow window (DataZoom visible), resize wide → DataZoom disappears
10. **Resize recalculates widths** — resize the widget container; bars redistribute without page reload

### Files to change

- `lifecycle/tests/dashboard-velocity-auto-scale-tests.md` — append these cases to the test artifact from Milestone 3

### Acceptance criteria

- [ ] All ten test cases have explicit pass/fail criteria
- [ ] Tests are traceable to FR-2, FR-3, and FR-5
- [ ] Resize tests cover both directions (wide→narrow and narrow→wide)

---

## Milestone 5 — Frontend test artifact: default granularity and accessibility

### Description

Document acceptance tests for FR-4 (default granularity) and NFR-2 (accessibility).

### Test cases

1. **Default daily** — on widget mount, Daily toggle is active (`aria-pressed="true"`), and chart shows daily buckets
2. **Granularity switch** — clicking Weekly then Monthly correctly fetches and renders; clicking Daily returns to daily view
3. **aria-label accuracy** — after each granularity switch, `aria-label` reports correct total and period count
4. **Keyboard navigation** — Tab through granularity buttons, Enter/Space activates each
5. **DataZoom a11y** — when DataZoom slider is visible, it is reachable via Tab and operable via keyboard
6. **Visual consistency** — bar colour `#6366f1`, emphasis `#4f46e5`, border-radius top corners rounded, widget chrome unchanged

### Files to change

- `lifecycle/tests/dashboard-velocity-auto-scale-tests.md` — append these cases to the test artifact from Milestones 3–4

### Acceptance criteria

- [ ] All six test cases have pass/fail criteria
- [ ] Tests are traceable to FR-4 and NFR-2
- [ ] `pnpm build` and `pnpm exec vue-tsc --noEmit` pass (build sanity check recorded as a test case)

---

## Milestone 6 — Performance validation

### Description

Validate NFR-1: rendering and granularity switching completes within 200ms for up to 90 periods.

### Test cases

1. **Render 90 daily periods** — measure time from `setOption` call to chart render completion; must be < 200ms
2. **Granularity switch latency** — measure time from toggle click to chart update (including API round-trip to local server); must be < 200ms for cached data
3. **No extra API calls** — switching granularity fires exactly one `GET /dashboard/velocity` request (verify via network tab or test spy)

### Files to change

- `lifecycle/tests/dashboard-velocity-auto-scale-tests.md` — append performance test cases

### Acceptance criteria

- [ ] Render time for 90 periods documented and < 200ms
- [ ] Granularity switch time documented and < 200ms
- [ ] Network request count verified: one fetch per granularity change, no duplicate requests
