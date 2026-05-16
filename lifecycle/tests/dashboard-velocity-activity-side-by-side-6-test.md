---
title: "Tests: Dashboard Velocity and Activity Side-by-Side Layout"
type: test
status: approved
lineage: dashboard-velocity-activity-side-by-side
parent: lifecycle/test-plans/dashboard-velocity-activity-side-by-side-5-test.md
---

# Tests: Dashboard Velocity and Activity Side-by-Side Layout

Companion artifact for the test plan at
`lifecycle/test-plans/dashboard-velocity-activity-side-by-side-5-test.md`.

## Test file

**`tests/web/dashboard-velocity-activity-layout.test.ts`** — Vitest component
tests covering all milestones that are verifiable in happy-dom.

## Scenarios covered

### Milestone 1 — Mock fixture verification

| Scenario | Outcome |
|---|---|
| Velocity mock resolves a bucket with count > 0 | passes |
| Feed mock resolves at least one event | passes |

Fixtures are provided entirely through Vitest module mocks (`@/api/feed`,
`@/api/client`), so each test run is isolated with no shared state.

### Milestone 2 — DashboardGrid: side-by-side section DOM structure

| Scenario | Outcome |
|---|---|
| `section[aria-label="Velocity and activity"]` is rendered | passes |
| Velocity stub is inside the side-by-side section | passes |
| Activity feed stub is inside the side-by-side section | passes |
| Velocity stub precedes feed stub in DOM tree (TC DOM order) | passes |
| With 3 other chart widgets registered, velocity/feed absent from Charts and Panels sections | passes |
| Container carries class `dashboard-side-by-side` | passes |
| `project` prop is forwarded to both side-by-side widgets | passes |

**Deferred to Playwright:** M2-TC1 (bounding rects equal at 1280 px),
M2-TC2 (900 px side-by-side, neither narrower than 360 px), M2-TC3 (600 px
stacked, full-width), M2-TC4 (breakpoint boundary 768 px/767 px).

### Milestone 3 — Responsive transitions

All test cases in Milestone 3 (wide→narrow and narrow→wide CSS transitions,
no JS errors during transitions) require real CSS `@media` evaluation and
viewport resize via `page.setViewportSize` — **deferred to Playwright**.

### Milestone 4 — ECharts resize behaviour

| Scenario | Outcome |
|---|---|
| `chart.resize()` called after data loads on mount | passes |
| `chart.setOption` called with bar series containing correct count | passes |
| `chart.dispose()` called on component unmount | passes |
| `.velocity-chart` container rendered when data has non-zero counts | passes |
| `.widget-empty` shown when all buckets are zero-count | passes |

**Deferred to Playwright:** M4-TC1 (canvas width ≈ half dashboard at 1280 px),
M4-TC2 (canvas width increases after ResizeObserver fires at 600 px),
M4-TC3 (container width >= 360 px).

### Milestone 5 — Widget functionality regression

| Scenario | Outcome |
|---|---|
| TC1: `feed.new` WebSocket event prepends new event to feed list | passes |
| TC1: Feed list caps at 7 entries on WebSocket overflow | passes |
| TC3: Clicking Weekly triggers API call with `granularity=weekly` | passes |
| TC3: Active granularity button has `aria-pressed="true"` | passes |
| TC3: Clicking Monthly sets Monthly button `aria-pressed="true"` | passes |
| TC3: Granularity change causes `setOption` to be called with updated data | passes |
| TC4: "View all" calls `router.push` with project feed path | passes |
| TC4: Feed path uses the `project` prop, not a hardcoded value | passes |
| TC6: Velocity element precedes activity element in DOM | passes |

**Deferred to Playwright:** M5-TC2 (tooltip appears on bar hover — real
canvas), M5-TC5 (keyboard tab order — real focus management).

### Milestone 6 — CLS prevention

| Scenario | Outcome |
|---|---|
| `VelocityChartWidget` root has `.velocity-widget` class (carries `min-height: 240px`) | passes |
| `ActivityFeedWidget` body has `.activity-feed-body` class (carries `min-height: 240px`) | passes |
| `DashboardGrid` side-by-side section has `.dashboard-side-by-side` class (carries `min-height: 240px`) | passes |

**Deferred to Playwright:** M6-TC1 (PerformanceObserver CLS score < 0.01).

### Regression

| Scenario | Outcome |
|---|---|
| All 6 named dashboard widgets render alongside the side-by-side row | passes |
| velocity-chart and activity-feed not rendered in `.dashboard-charts-top` | passes |

## Implementation notes

- **ECharts mocked** via `vi.mock('echarts/core')` following the same pattern
  as `tests/web/dashboard-clickable-filters.test.ts`. The mock captures the
  chart instance so `resize` / `setOption` / `dispose` calls can be asserted.
- **FeedEntry mocked** as a minimal stub (renders `event.summary`) so
  `ActivityFeedWidget` can mount without the full component tree.
- **`vi.useFakeTimers()`** is activated in `beforeEach` and restored in
  `afterEach` so the 150 ms resize debounce does not cause test-timeout races.
- Granularity-toggle test queues `mockResolvedValueOnce` *after* the initial
  mount flush so the weekly API response is consumed only by the watch-triggered
  refetch, not the mount-time fetch.
- TC5 (widget exclusion from Charts section) uses a production-like registration
  of 3 chart widgets before `velocity-chart` (orders 0, 1, 2, then 3), which
  matches `registerWidgets.ts` and ensures velocity-chart falls into
  `bottomChartWidgets` where `SIDE_BY_SIDE_IDS` are filtered.
