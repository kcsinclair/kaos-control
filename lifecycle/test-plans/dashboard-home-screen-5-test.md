---
title: Dashboard Home Screen — Test Plan
type: plan-test
status: in-development
lineage: dashboard-home-screen
parent: lifecycle/requirements/dashboard-home-screen-2.md
assignees:
    - role: product-owner
      who: agent
---

# Dashboard Home Screen — Test Plan

This plan covers integration tests for the dashboard backend endpoints and end-to-end tests for the frontend dashboard view.

## Milestone 1: Backend Integration Tests — Stats Endpoint

**Description:** Test `GET /api/p/:project/dashboard/stats` with various artifact configurations.

**Files to change:**
- `tests/integration/dashboard_stats_test.go` (new) — Integration tests against a real SQLite index.

**Acceptance criteria:**
- [ ] Test: empty project returns all counts as 0.
- [ ] Test: project with mixed ticket statuses returns correct `total_tickets`, `in_progress`, `blocked` counts.
- [ ] Test: `completed_this_week` only counts tickets with done-transition events in the current ISO week.
- [ ] Test: non-ticket artifacts (ideas, plans) are excluded from counts.
- [ ] Test: `abandoned` tickets are excluded from `total_tickets`.
- [ ] Test: response time assertion < 100 ms with 500 seeded artifacts.

## Milestone 2: Backend Integration Tests — Status Distribution Endpoint

**Description:** Test `GET /api/p/:project/dashboard/status-distribution` returns correct grouping.

**Files to change:**
- `tests/integration/dashboard_distribution_test.go` (new) — Integration tests.

**Acceptance criteria:**
- [ ] Test: returns empty array when no tickets exist.
- [ ] Test: returns correct counts grouped by status.
- [ ] Test: excludes tickets with status `done` and `abandoned`.
- [ ] Test: adding a new ticket and re-indexing updates the distribution.

## Milestone 3: Backend Integration Tests — Velocity Endpoint

**Description:** Test `GET /api/p/:project/dashboard/velocity` with seeded transition events.

**Files to change:**
- `tests/integration/dashboard_velocity_test.go` (new) — Integration tests.

**Acceptance criteria:**
- [ ] Test: daily granularity returns one bucket per day within the lookback window.
- [ ] Test: weekly granularity aggregates correctly by ISO week.
- [ ] Test: monthly granularity aggregates correctly by calendar month.
- [ ] Test: days with zero completions appear as `{ "count": 0 }` (no gaps).
- [ ] Test: invalid granularity param defaults to weekly.
- [ ] Test: `days` param limits the lookback window (e.g., `days=7` returns only last 7 days of buckets).

## Milestone 4: Frontend Component Tests — Widget Registry

**Description:** Unit tests for the widget registry ensuring extensibility contract.

**Files to change:**
- `web/src/components/dashboard/__tests__/widgetRegistry.spec.ts` (new) — Vitest unit tests.

**Acceptance criteria:**
- [ ] Test: `registerWidget()` adds a widget to the reactive list.
- [ ] Test: widgets are sorted by `order` within their slot.
- [ ] Test: duplicate IDs throw or overwrite (document chosen behaviour).
- [ ] Test: all three slots ('summary', 'chart', 'panel') are supported.

## Milestone 5: Frontend Component Tests — Dashboard View & Grid

**Description:** Component-level tests for the dashboard layout at different viewport widths.

**Files to change:**
- `web/src/views/project/__tests__/DashboardView.spec.ts` (new) — Vitest + Vue Test Utils.

**Acceptance criteria:**
- [ ] Test: DashboardGrid renders registered widgets in correct slot positions.
- [ ] Test: at viewport ≥ 1024 px, grid has two columns (charts and panel side-by-side).
- [ ] Test: at viewport < 1024 px, grid has single column (stacked).
- [ ] Test: summary counts display after API response resolves.

## Milestone 6: End-to-End Tests — Navigation & Default Route

**Description:** E2E tests verifying routing behaviour and sidebar placement.

**Files to change:**
- `tests/integration/dashboard_e2e_test.go` (new) — HTTP-level tests against the running server verifying route redirects and HTML content.

**Acceptance criteria:**
- [ ] Test: `GET /p/:project` responds with 302 redirect to `/p/:project/dashboard`.
- [ ] Test: dashboard page HTML includes all expected widget containers.
- [ ] Test: sidebar HTML has "Dashboard" as the first navigation item.

## Milestone 7: Performance & Bundle Size Validation

**Description:** Validate non-functional requirements: render time and bundle size.

**Files to change:**
- `web/src/components/dashboard/__tests__/performance.spec.ts` (new) — Measures mount time of DashboardView with mocked API.
- Script or CI step to measure `echarts` contribution to bundle size.

**Acceptance criteria:**
- [ ] Test: DashboardView mounts and renders summary counts within 500 ms (mocked API latency ≤ 50 ms).
- [ ] Validation: `echarts` tree-shaken bundle ≤ 80 KB gzipped (measured via `npx vite-bundle-visualizer` or `source-map-explorer`).
- [ ] Document the measurement method so it can be repeated in CI.

## Cross-references

- [[dashboard-home-screen-3-be]] — Backend endpoints under test.
- [[dashboard-home-screen-4-fe]] — Frontend components under test.

## Resolved Questions

Milestones 1-3 (Go backend integration tests) have been implemented in
`tests/integration/dashboard_stats_test.go`, `dashboard_distribution_test.go`,
and `dashboard_velocity_test.go`. The questions below block Milestones 4-7.

### Q1 - Milestone 6: Server-side redirect assumption is incorrect

The plan states: "Test: `GET /p/:project` responds with 302 redirect to
`/p/:project/dashboard`."

The server catch-all (`r.Get("/*", s.handleFrontend)`) serves the Vue SPA's
`index.html` with HTTP 200 for all frontend routes. The dashboard redirect
is client-side via Vue Router; no 302 is ever emitted by the Go server.

Pick one:
a) Remove the assertion; test instead that `GET /p/:project` returns 200.
b) Add an explicit server-side redirect in `handleFrontend` and update the test.
c) Move the test to a browser-based E2E suite (Playwright/Cypress).

> a

### Q2 - Milestone 6: HTML content assertions against a SPA shell

"Dashboard page HTML includes all expected widget containers" and "sidebar HTML
has Dashboard as the first navigation item" assume server-rendered HTML. The
server only returns an empty `<div id="app"></div>`; widgets and nav are
injected by JavaScript at runtime.

Pick one:
a) Drop these assertions; cover widget registration at the unit level (Milestone 4).
b) Use Playwright to test the rendered DOM in a real browser.

> b

### Q3 - Milestones 4, 5, 7: Vitest is not installed

The frontend spec files (`widgetRegistry.spec.ts`, `DashboardView.spec.ts`,
`performance.spec.ts`) require Vitest and `@vue/test-utils`. Neither appears in
`web/package.json` and there is no `vitest.config.ts`. The write scope for this
agent is `tests/**` and `lifecycle/tests/` - modifying `web/package.json`,
`web/vite.config.ts`, or writing files under `web/src/` is out of scope.

Decision needed: Should a separate task install Vitest and configure the
frontend test harness before a subsequent agent implements Milestones 4, 5, 7?

> Vitest is installed.  I have given access to agent for web/src

### Q4 - Milestone 5: Viewport layout testing in jsdom

The criterion "at viewport >= 1024 px, grid has two columns" tests a CSS
`@media` rule. jsdom does not evaluate CSS media queries.

Pick one:
a) Replace the CSS breakpoint with a JS reactive breakpoint (`useWindowSize`)
   so tests can set `window.innerWidth` in jsdom.
b) Use Playwright where real CSS is applied.
c) Drop the column-count assertion; test only DOM slot structure.

> b
