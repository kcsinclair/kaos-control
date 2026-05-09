---
title: "Velocity Widget Auto-Scaling Columns and Minimum Periods — Test Suite"
type: test
status: approved
lineage: dashboard-velocity-auto-scale
parent: lifecycle/test-plans/dashboard-velocity-auto-scale-5-test.md
created: "2026-05-09T00:00:00+10:00"
labels:
    - test
    - integration
    - enhancement
---

# Velocity Widget Auto-Scaling Columns and Minimum Periods — Test Suite

## Overview

This artifact documents the test coverage for the [[dashboard-velocity-auto-scale]] feature.
Backend integration tests live in `tests/integration/dashboard_velocity_test.go`.
Frontend acceptance tests (Milestones 3–6) are documented below as manual/automated
acceptance criteria.

---

## Milestone 1 — Backend: `days` parameter behaviour

**File:** `tests/integration/dashboard_velocity_test.go`

| Test | Scenario | Pass criteria |
|---|---|---|
| `TestVelocityDaysParam_Explicit` | `days=14` daily | ≥ 14 buckets returned; ≤ 20 (not the default 90-day window) |
| `TestVelocityDaysParam_Zero` | `days=0` weekly | Falls back to 90-day default; ≥ 12 weekly buckets |
| `TestVelocityDaysParam_Negative` | `days=-5` weekly | Falls back to 90-day default; ≥ 12 weekly buckets |
| `TestVelocityDaysParam_OverMax` | `days=400` weekly | Clamped to 365; same bucket count as `days=365` |
| `TestVelocityDaysParam_NonNumeric` | `days=abc` weekly | Falls back to 90-day default; ≥ 12 weekly buckets |
| `TestVelocityDaysParam_Omitted` | no `days` param weekly | Same bucket count as explicit `days=90` |

All six tests assert on bucket count, not just HTTP 200.

---

## Milestone 2 — Backend: zero-fill coverage

**File:** `tests/integration/dashboard_velocity_test.go`

| Test | Scenario | Pass criteria |
|---|---|---|
| `TestVelocityZeroFill_Daily7` | Empty project, `daily&days=7` | ≥ 7 buckets, all `count: 0` |
| `TestVelocityZeroFill_Weekly28` | Empty project, `weekly&days=28` | ≥ 4 buckets, all `count: 0` |
| `TestVelocityZeroFill_Monthly90` | Empty project, `monthly&days=90` | ≥ 2 buckets, all `count: 0`; YYYY-MM format verified |
| `TestVelocityZeroFill_Contiguous` | 1 event at day -7 of a 14-day daily window | ≥ 14 buckets; exactly 1 non-zero bucket (count=1 at day -7); all adjacent periods are exactly 1 day apart (no gaps) |

---

## Milestone 3 — Frontend: minimum periods and padding (FR-1)

These are manual acceptance tests to be performed against a running dev server.

### Test cases

1. **Minimum daily periods**
   - Setup: new project with artifact completions on 2 days only.
   - Action: open Dashboard, select Daily granularity.
   - Pass: chart renders 7 columns (5 zero-padded + 2 real).
   - Fail: fewer than 7 columns appear.

2. **Minimum weekly periods**
   - Setup: new project with completions in 1 week only.
   - Action: select Weekly granularity.
   - Pass: chart renders 4 columns.
   - Fail: fewer than 4 columns appear.

3. **Minimum monthly periods**
   - Setup: new project with completions in 1 month only.
   - Action: select Monthly granularity.
   - Pass: chart renders 3 columns.
   - Fail: fewer than 3 columns appear.

4. **Padding does not inflate totals**
   - Action: inspect the `aria-label` attribute on the velocity widget after zero-padding is applied.
   - Pass: `aria-label` reports the count of real completions only (zero-padded entries excluded).
   - Fail: `aria-label` total includes zero-padded periods.

5. **Full data — no padding**
   - Setup: project with 30 days of completion events.
   - Action: select Daily granularity.
   - Pass: 30+ columns render; no artificial zero-padding is prepended.
   - Fail: fewer than 30 columns, or the total column count exceeds 30 + a small boundary.

---

## Milestone 4 — Frontend: auto-scaling and DataZoom (FR-2, FR-3, FR-5)

### Test cases

1. **Auto-scale few bars** — Daily view, 7 periods, 600 px-wide widget. Bars fill available width; no DataZoom slider visible.
2. **Auto-scale cap** — Monthly view, 3 periods. No bar exceeds ~60 px wide (visual check via DevTools computed style).
3. **DataZoom appears** — Daily view, 30+ periods. DataZoom slider renders below the chart.
4. **Scroll to recent** — With DataZoom active, the visible range defaults to the rightmost (most recent) periods.
5. **Shift+wheel scroll** — Holding Shift and scrolling the mouse wheel pans the chart horizontally. Verify by watching the visible range shift.
6. **Touch swipe** — On a touch device (or Chrome DevTools touch emulation), swiping left/right pans the chart.
7. **DataZoom keyboard** — Tab to the DataZoom slider, then use arrow keys. Verify the visible window adjusts.
8. **Resize adds DataZoom** — Start with a wide browser window (no DataZoom). Resize narrow. DataZoom slider appears automatically.
9. **Resize removes DataZoom** — Start narrow (DataZoom visible). Resize wide. DataZoom slider disappears.
10. **Resize recalculates widths** — Resize the widget container. Bars redistribute to fill the new width without a page reload.

---

## Milestone 5 — Frontend: default granularity and accessibility (FR-4, NFR-2)

### Test cases

1. **Default daily** — On widget mount `aria-pressed="true"` is on the Daily button; chart shows daily buckets.
2. **Granularity switch** — Click Weekly → Monthly → Daily. Each switch fetches and renders correctly; returning to Daily restores daily buckets.
3. **`aria-label` accuracy** — After each granularity switch, `aria-label` reports the correct total and period count for the active view.
4. **Keyboard navigation** — Tab through the granularity buttons. Enter/Space activates each.
5. **DataZoom a11y** — When the DataZoom slider is visible, it is reachable by Tab and operable via keyboard arrow keys.
6. **Visual consistency** — Bar colour is `#6366f1`, hover/emphasis colour is `#4f46e5`, top corners of bars are rounded, widget chrome matches the existing dashboard design.

**Build sanity check (recorded as a test case):**

```
pnpm build            # must exit 0
pnpm exec vue-tsc --noEmit  # must exit 0
```

---

## Milestone 6 — Performance validation (NFR-1)

### Test cases

1. **Render 90 daily periods** — Measure time from `setOption` call to chart render completion. Must be < 200 ms (verify via browser Performance panel or ECharts `finished` event timestamp).
2. **Granularity switch latency** — Measure time from toggle click to chart update (including API round-trip to local server). Must be < 200 ms for locally-served data.
3. **No extra API calls** — Switching granularity fires exactly one `GET /dashboard/velocity` request per click. Verify via browser Network panel or a test spy on `fetch`.

---

## Traceability

| Milestone | Requirement reference |
|---|---|
| 1, 2 | Backend `days` parameter (backend plan) |
| 3 | FR-1 — Minimum periods and padding |
| 4 | FR-2 (auto-scale), FR-3 (horizontal scroll), FR-5 (responsive resize) |
| 5 | FR-4 (default daily granularity), NFR-2 (accessibility) |
| 6 | NFR-1 (< 200 ms render) |
