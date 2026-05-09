---
title: "Backend Plan: Stages Distribution Pie Chart"
type: plan-backend
status: approved
lineage: stages-distribution-pie-chart
parent: lifecycle/requirements/stages-distribution-pie-chart-2.md
created: "2026-05-09"
---

# Backend Plan: Stages Distribution Pie Chart

## Overview

Add a `GET /api/p/:project/dashboard/stage-distribution` endpoint that returns artifact counts grouped by lifecycle stage (directory). The endpoint mirrors the existing `StatusDistribution` pattern in `internal/index/index.go` and `internal/http/dashboard.go`, but groups by the `stage` column instead of `status`, and excludes `done`/`abandoned` artifacts per the resolved requirement question.

The [[stages-distribution-pie-chart]] frontend plan depends on this endpoint's response shape.

## Milestone 1: Index method — `StageDistribution`

**Description:** Add a `StageDistribution(trackedTypes []string) ([]StageCount, error)` method to `internal/index/index.go` alongside the existing `StatusDistribution` method.

**Files to change:**

- `internal/index/index.go` — add `StageCount` struct and `StageDistribution` method.

**Implementation details:**

- Define `StageCount` struct: `Stage string` (`json:"stage"`), `Count int` (`json:"count"`).
- Query: `SELECT stage, COUNT(*) FROM artifacts WHERE type IN (<trackedTypes>) AND status NOT IN ('done','abandoned') GROUP BY stage ORDER BY stage`.
- Reuse `trackedTypesClause()` for the type filter, same as `StatusDistribution`.
- Return an empty non-nil slice (`[]StageCount{}`) when no rows match.
- Exclude `done` and `abandoned` statuses per the resolved requirement (Resolved Question 1).

**Acceptance criteria:**

- [ ] `StageDistribution` returns `[]StageCount` with one entry per stage that has at least one non-done/non-abandoned tracked artifact.
- [ ] Stages with zero qualifying artifacts are not included in the result.
- [ ] The result is an empty slice (not nil) when no artifacts exist.
- [ ] The method respects `trackedTypes` — only artifacts whose `type` is in the list are counted.
- [ ] Results are ordered alphabetically by stage name.

## Milestone 2: HTTP handler — `handleGetStageDistribution`

**Description:** Add the REST handler and wire it into the dashboard route group.

**Files to change:**

- `internal/http/dashboard.go` — add `handleGetStageDistribution` method.
- `internal/http/server.go` — register `r.Get("/stage-distribution", s.handleGetStageDistribution)` in the `/dashboard` route group.

**Implementation details:**

- Follow the exact pattern of `handleGetStatusDistribution`: extract project from context, call `p.Idx.StageDistribution(p.Cfg.Dashboard.TrackedTypes)`, wrap result as `{"distribution": [...]}`.
- Return HTTP 200 with `{"distribution": []}` when no artifacts exist.
- Return HTTP 500 with `apiError` on database errors.

**Acceptance criteria:**

- [ ] `GET /api/p/:project/dashboard/stage-distribution` returns `{"distribution": [{"stage": "ideas", "count": 5}, ...]}`.
- [ ] The endpoint returns `{"distribution": []}` (empty array, not null) when no artifacts exist.
- [ ] The endpoint respects `Dashboard.TrackedTypes` from project config.
- [ ] The endpoint excludes artifacts with `done` or `abandoned` status.
- [ ] The response Content-Type is `application/json`.
- [ ] Invalid/missing project returns an appropriate error response.

## Milestone 3: Verify stage column index

**Description:** Confirm that the `stage` column on the `artifacts` table is indexed (the requirement notes it already is, NFR-1). If not, add an index.

**Files to review:**

- `internal/index/index.go` — schema creation / migration statements.

**Acceptance criteria:**

- [ ] The `stage` column has a database index, ensuring the `GROUP BY stage` query performs within 50 ms for up to 1,000 artifacts.
- [ ] No schema migration is needed if the index already exists; document the finding.
