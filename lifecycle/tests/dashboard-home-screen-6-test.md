---
title: Dashboard Home Screen — Integration Test Suite
type: test
status: approved
lineage: dashboard-home-screen
parent: lifecycle/test-plans/dashboard-home-screen-5-test.md
---

# Dashboard Home Screen — Integration Test Suite

Full test coverage for the dashboard backend endpoints, widget registry,
dashboard grid, summary counts widget, E2E route handling, and performance.
All milestones from the test plan are now implemented.

## Scenarios covered

### Milestone 1 — Stats endpoint (`GET /api/p/:project/dashboard/stats`)

File: `tests/integration/dashboard_stats_test.go`

| Test | Description |
|---|---|
| `TestDashboardStats_Empty` | Empty project returns all counts as zero |
| `TestDashboardStats_MixedStatuses` | Mixed ticket statuses yield correct `total_tickets`, `in_progress` (in-development), and `blocked` (blocked + clarifying) counts |
| `TestDashboardStats_CompletedThisWeek` | `completed_this_week` counts only done-transition events whose timestamp falls within the current ISO week; events from the prior week are excluded |
| `TestDashboardStats_NonTicketsExcluded` | Ideas and plan artifacts contribute zero to all ticket counts |
| `TestDashboardStats_AbandonedExcluded` | Abandoned tickets are not counted in `total_tickets` |
| `TestDashboardStats_Performance` | Response time < 100 ms with 500 seeded artifacts |

Done-transition events are injected directly via `env.proj.Idx.InsertEvent` so
that timestamps can be controlled precisely (e.g. placed in a prior ISO week
to verify exclusion).

### Milestone 2 — Status-distribution endpoint (`GET /api/p/:project/dashboard/status-distribution`)

File: `tests/integration/dashboard_distribution_test.go`

| Test | Description |
|---|---|
| `TestStatusDistribution_Empty` | Returns an empty (non-null) array when no tickets exist |
| `TestStatusDistribution_CorrectCounts` | Tickets grouped and counted correctly by status |
| `TestStatusDistribution_ExcludesDoneAndAbandoned` | Tickets with status `done` or `abandoned` do not appear |
| `TestStatusDistribution_UpdatesAfterReindex` | Writing a new artifact to disk and waiting for the 150 ms watcher debounce causes the distribution to update |

### Milestone 3 — Velocity endpoint (`GET /api/p/:project/dashboard/velocity`)

File: `tests/integration/dashboard_velocity_test.go`

| Test | Description |
|---|---|
| `TestVelocity_DailyGranularity` | One bucket per day; counts match seeded events; correct `period` key format (YYYY-MM-DD) |
| `TestVelocity_WeeklyGranularity` | Events aggregated by ISO week; correct `period` key format (YYYY-Www) |
| `TestVelocity_MonthlyGranularity` | Events aggregated by calendar month; correct `period` key format (YYYY-MM) |
| `TestVelocity_ZeroGapsIncluded` | Days with no completions still appear with `count: 0` |
| `TestVelocity_InvalidGranularityDefaultsWeekly` | Unrecognised `granularity` param yields `granularity: "weekly"` in response |
| `TestVelocity_DaysParamLimitsWindow` | Events older than the `days` window are excluded; total count reflects only events inside the window |

Events are seeded via `env.proj.Idx.InsertEvent` with explicit Unix timestamps
so that the distribution across daily/weekly/monthly buckets can be verified
deterministically.

### Milestone 4 — Widget registry unit tests

File: `tests/web/widgetRegistry.test.ts`

| Test group | Scenarios |
|---|---|
| `registerWidget adds widgets` | Adds widget to reactive list; stores component, slot, and order; supports multiple widgets |
| `sorting by order within slot` | Out-of-order registration is sorted ascending by `order`; slot sorts alphabetically (chart < panel < summary); independent slot ordering |
| `duplicate ID handling` | Duplicate ID is silently skipped (first registration wins); idempotent re-registration (HMR safety) |
| `all three slot types` | `summary`, `chart`, and `panel` slots each work individually and coexist |

The `widgetList` reactive singleton is reset via `widgetList.splice(0)` in
`beforeEach` to ensure test isolation.

### Milestone 5 — DashboardGrid and SummaryCountsWidget component tests

File: `tests/web/DashboardView.test.ts`

| Test group | Scenarios |
|---|---|
| `DashboardGrid — slot rendering` | Renders widgets in summary, chart, and panel slots; omits sections when no widgets registered; renders widgets in ascending `order` within a slot; passes `project` prop to each widget |
| `SummaryCountsWidget — summary counts` | Renders four stat cards; shows zeroes before API resolves; displays API counts after response; keeps zeroes on API failure; calls API with correct project-scoped URL; displays correct card labels |

**Viewport layout tests** (two-column at ≥1024 px, single-column at <1024 px)
are deferred to a Playwright suite: happy-dom does not evaluate CSS `@media`
rules (Q4 resolution, option b).

### Milestone 6 — E2E route tests (HTTP-level)

File: `tests/integration/dashboard_e2e_test.go`

| Test | Description |
|---|---|
| `TestDashboardE2E_ProjectRouteReturns200` | `GET /p/:project` returns HTTP 200 with the SPA shell (no server-side redirect, per Q1 resolution option a) |
| `TestDashboardE2E_DashboardSubRouteReturns200` | `GET /p/:project/dashboard` also returns HTTP 200 |
| `TestDashboardE2E_ResponseBodyContainsSPAShell` | Response body is non-empty (index.html content served) |
| `TestDashboardE2E_FrontendUnavailableReturns500` | Server returns 500 when no frontend FS is configured |
| `TestDashboardE2E_ArbitrarySubRouteReturns200` | Any unrecognised sub-path falls back to index.html (HTML5 pushState) |

Tests use `newTestEnvWithFrontend` with a `fstest.MapFS` stub containing
`dist/index.html`.

**HTML content assertions** (widget containers, "Dashboard" as first nav item)
require JavaScript execution in a real browser and are deferred to a Playwright
suite (Q2 resolution, option b).

### Milestone 7 — Performance tests

File: `tests/web/performance.test.ts`

| Test | Description |
|---|---|
| Mount and render within 500 ms | `SummaryCountsWidget` mounts and renders all four stat cards within 500 ms with 30 ms mocked API latency |
| Synchronous mount under 100 ms | Component mounts synchronously (before API resolves) in under 100 ms |
| No degradation across five runs | Five consecutive mounts all stay within the 500 ms budget |

**Bundle size measurement** (echarts ≤ 80 KB gzipped) cannot be automated in
Vitest. The recommended method:

```sh
cd web && pnpm build
npx vite-bundle-visualizer
# or: npx source-map-explorer dist/assets/*.js --gzip
```

Look for the `echarts` chunk in the visualiser output. Target: ≤ 80 KB gzipped.
This can be run as a CI step after `make build-web`.

## Resolved questions (from test plan)

| Question | Resolution |
|---|---|
| Q1: `GET /p/:project` redirect | Option a: test for HTTP 200 (redirect is client-side Vue Router) |
| Q2: HTML content assertions against SPA | Option b: deferred to Playwright; widget registry covered at unit level (Milestone 4) |
| Q3: Vitest not installed | Vitest is installed; agent granted access to `web/src` |
| Q4: Viewport layout in jsdom | Option b: deferred to Playwright; DOM slot structure tested via Vitest |
