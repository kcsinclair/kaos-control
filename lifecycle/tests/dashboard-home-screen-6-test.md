---
title: Dashboard Home Screen — Integration Test Suite
type: test
status: draft
lineage: dashboard-home-screen
parent: lifecycle/test-plans/dashboard-home-screen-5-test.md
---

# Dashboard Home Screen — Integration Test Suite

Backend integration tests for the three dashboard API endpoints. Milestones
4-7 (frontend unit tests and E2E) are blocked pending resolution of the open
questions recorded in the test plan.

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

## Blocked milestones

Milestones 4-7 require resolution of four open questions documented in
`lifecycle/test-plans/dashboard-home-screen-5-test.md`:

- **Q1/Q2** (Milestone 6): The HTTP-level E2E test plan assumes a server-side
  302 redirect and server-rendered HTML, neither of which applies to this SPA
  architecture.
- **Q3** (Milestones 4, 5, 7): Vitest and `@vue/test-utils` are not installed;
  the write scope for this agent excludes `web/`.
- **Q4** (Milestone 5): jsdom does not evaluate CSS media queries, so the
  two-column layout assertion cannot be tested without a design change or a
  browser-based tool.
