---
title: "Backend Plan: Dashboard Velocity and Activity Side-by-Side Layout"
type: plan-backend
status: done
lineage: dashboard-velocity-activity-side-by-side
parent: lifecycle/requirements/dashboard-velocity-activity-side-by-side-2.md
created: "2026-05-10T00:00:00+10:00"
---

# Backend Plan: Dashboard Velocity and Activity Side-by-Side Layout

## Overview

Per the requirement's non-goals, **no backend or API changes are required**. This is a purely frontend layout change. This plan covers verification that the existing backend endpoints consumed by the two affected widgets continue to function correctly and documents the API surface area the [[dashboard-velocity-activity-side-by-side]] frontend plan depends on.

## Milestone 1: Verify Velocity API Contract

**Description:** Confirm that the `GET /p/:project/dashboard/velocity` endpoint returns data in the shape expected by VelocityChartWidget, and that no backend changes are needed for the widget to function in a narrower column layout.

**Files to review (no changes expected):**

- `internal/http/dashboard.go` — velocity endpoint handler; confirm response shape (`{ buckets: VelocityItem[], granularity: string }`) is stable.
- `internal/index/query.go` — confirm velocity query logic is independent of any frontend layout concerns.

**Acceptance criteria:**

- [ ] `GET /p/:project/dashboard/velocity?granularity=daily&days=90` returns a well-formed `VelocityResponse` with `buckets` and `granularity` fields.
- [ ] The endpoint behaviour is identical regardless of how the frontend renders the chart (no layout-coupled logic exists in the backend).
- [ ] No new endpoints, fields, or handler modifications are introduced.

## Milestone 2: Verify Feed API Contract

**Description:** Confirm that the `GET /p/:project/feed` endpoint (consumed by ActivityFeedWidget) returns data in the expected shape and that the WebSocket `feed.new` event continues to be broadcast correctly.

**Files to review (no changes expected):**

- `internal/http/feed.go` — feed list endpoint; confirm the `limit` query parameter and response shape are stable.
- `internal/hub/hub.go` — confirm `feed.new` WebSocket event broadcast is independent of frontend layout.

**Acceptance criteria:**

- [ ] `GET /p/:project/feed?limit=7` returns the expected feed event list.
- [ ] WebSocket `feed.new` events continue to be broadcast with a `FeedEvent` payload.
- [ ] No new endpoints, fields, or handler modifications are introduced.
