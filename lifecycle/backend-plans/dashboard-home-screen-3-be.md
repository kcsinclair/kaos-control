---
title: "Dashboard Home Screen — Backend Plan"
type: plan-backend
status: in-development
lineage: dashboard-home-screen
parent: lifecycle/requirements/dashboard-home-screen-2.md
---

# Dashboard Home Screen — Backend Plan

This plan covers the API endpoints needed to serve dashboard summary data. The requirement explicitly states "no backend API changes beyond what is needed to serve the dashboard data," so the scope is limited to aggregation queries over existing indexed data.

## Milestone 1: Dashboard Stats Endpoint

**Description:** Add a `GET /api/p/:project/dashboard/stats` endpoint returning summary counts derived from the existing `artifacts` SQLite table.

**Response shape:**
```json
{
  "total_tickets": 42,
  "in_progress": 12,
  "blocked": 3,
  "completed_this_week": 7
}
```

**Files to change:**
- `internal/index/index.go` — add `DashboardStats(sinceTime time.Time) (*DashboardStatsRow, error)` method that queries `artifacts` table for counts by status where `type = 'ticket'`.
- `internal/http/dashboard.go` (new) — handler `handleGetDashboardStats` that calls the index method and serialises JSON.
- `internal/http/server.go` — register route `GET /api/p/{project}/dashboard/stats` under the project sub-router.

**Acceptance criteria:**
- [ ] Endpoint returns correct `total_tickets` count (all tickets not `abandoned`).
- [ ] `in_progress` counts tickets with status `in-development`.
- [ ] `blocked` counts tickets with status `blocked` or `clarifying`.
- [ ] `completed_this_week` counts tickets whose last transition to `done` occurred within the current ISO week.
- [ ] Returns 200 with zero-value counts when no tickets exist.
- [ ] Response time < 50 ms on a 500-artifact project.

## Milestone 2: Status Distribution Endpoint

**Description:** Add `GET /api/p/:project/dashboard/status-distribution` returning the count of non-done, non-abandoned tickets grouped by status.

**Response shape:**
```json
{
  "distribution": [
    { "status": "draft", "count": 5 },
    { "status": "in-development", "count": 8 }
  ]
}
```

**Files to change:**
- `internal/index/index.go` — add `StatusDistribution() ([]StatusCount, error)` method: `SELECT status, COUNT(*) FROM artifacts WHERE type='ticket' AND status NOT IN ('done','abandoned') GROUP BY status`.
- `internal/http/dashboard.go` — add handler `handleGetStatusDistribution`.
- `internal/http/server.go` — register route.

**Acceptance criteria:**
- [ ] Returns all statuses that have at least one ticket, excluding `done` and `abandoned`.
- [ ] Counts are accurate after artifact re-index events.
- [ ] Returns empty array (not null) when no matching tickets exist.

## Milestone 3: Completion Velocity Endpoint

**Description:** Add `GET /api/p/:project/dashboard/velocity?granularity={daily|weekly|monthly}&days=90` returning time-bucketed counts of artifacts that transitioned to status `done`.

**Response shape:**
```json
{
  "granularity": "weekly",
  "buckets": [
    { "period": "2026-W18", "count": 3 },
    { "period": "2026-W19", "count": 5 }
  ]
}
```

**Files to change:**
- `internal/index/index.go` — add `CompletionVelocity(granularity string, days int) ([]VelocityBucket, error)` method. Queries `events` table: `SELECT timestamp FROM events WHERE event_type = 'status.transition' AND summary LIKE '%→ done%'`, then buckets in Go code.
- `internal/http/dashboard.go` — add handler `handleGetVelocity`.
- `internal/http/server.go` — register route.

**Acceptance criteria:**
- [ ] Supports `daily`, `weekly`, and `monthly` granularity query param (defaults to `weekly`).
- [ ] `days` param controls lookback window (default 90, max 365).
- [ ] Buckets are ISO-formatted (`2026-05-06` for daily, `2026-W19` for weekly, `2026-05` for monthly).
- [ ] Periods with zero completions are included in the response (no gaps).
- [ ] Data is derived from the existing `events` table — no new tables or schema migrations.

## Milestone 4: Integration with Existing Feed Endpoint

**Description:** Ensure the existing `GET /api/p/:project/feed` endpoint adequately supports the dashboard's activity panel needs (limited rows with "view all" link pattern).

**Files to change:**
- No code changes required — the existing `limit` query param (default 50, max 200) and `before` cursor already support this use case. The [[dashboard-home-screen-4-fe]] frontend plan will call `GET /feed?limit=20` for the dashboard panel.

**Acceptance criteria:**
- [ ] Confirmed: `GET /feed?limit=20` returns exactly 20 most recent events (or fewer if < 20 exist).
- [ ] Confirmed: response includes `next_cursor` for "View all" navigation.

## Cross-references

- [[dashboard-home-screen-4-fe]] — Frontend plan consumes these endpoints.
- [[dashboard-home-screen-5-test]] — Test plan covers integration testing of these endpoints.
