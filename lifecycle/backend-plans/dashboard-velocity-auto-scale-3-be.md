---
title: "Backend Plan — Velocity Widget Auto-Scaling (days param support)"
type: plan-backend
status: done
lineage: dashboard-velocity-auto-scale
parent: lifecycle/requirements/dashboard-velocity-auto-scale-2.md
created: "2026-05-09T00:00:00+10:00"
labels:
    - backend
    - go
    - enhancement
---

# Backend Plan — Velocity Widget Auto-Scaling

## Context

The requirement [[dashboard-velocity-auto-scale]] calls for the frontend to enforce minimum visible periods per granularity and to pad missing periods client-side. The backend API contract (`/dashboard/velocity`) is explicitly a non-goal to change (response shape, bucket calculation). However, the frontend needs the ability to request a specific number of days so that it can guarantee enough data to meet the minimum-period targets across all granularities.

Currently the `days` query parameter already exists in the handler (`internal/http/dashboard.go:76`) with a default of 90 and a max of 365. The frontend does not send it — it always uses the default. No backend contract change is required; the parameter is already supported.

**This plan is intentionally minimal** because the requirement explicitly lists "changing the backend `/dashboard/velocity` API contract" as a non-goal. The backend work is limited to ensuring the existing `days` parameter is well-tested and documenting it for the frontend plan.

---

## Milestone 1 — Verify and harden the `days` query parameter

### Description

Audit the existing `handleGetVelocity` handler to confirm `days` works correctly for the values the frontend will send. Ensure edge cases (days=0, days=-1, days=366, days=abc, days omitted) are handled gracefully. The handler already clamps values, but we need test coverage.

### Files to change

- `internal/http/dashboard.go` — read and verify clamping logic (no changes expected unless a bug is found)
- `tests/integration/dashboard_velocity_test.go` — add test cases for explicit `days` parameter values

### Acceptance criteria

- [ ] `GET /dashboard/velocity?granularity=daily&days=14` returns exactly 14 daily buckets (or fewer if project is younger)
- [ ] `GET /dashboard/velocity?granularity=daily&days=0` falls back to default 90
- [ ] `GET /dashboard/velocity?granularity=daily&days=400` clamps to 365
- [ ] `GET /dashboard/velocity?granularity=daily&days=abc` falls back to default 90
- [ ] All existing velocity integration tests continue to pass
- [ ] `make test-unit` passes

---

## Milestone 2 — Ensure zero-filled period coverage

### Description

Verify that `CompletionVelocity` in `internal/index/index.go` always returns a contiguous, zero-filled series of period keys covering the full requested `days` window, even when no events exist for some periods. The frontend plan [[dashboard-velocity-auto-scale]] relies on the backend returning the complete period range so that its padding logic only needs to handle the case where the project is younger than the minimum period count.

### Files to change

- `internal/index/index.go` — read and verify `velocityPeriods()` zero-fill logic (no changes expected)
- `tests/integration/dashboard_velocity_test.go` — add test asserting that an empty project returns the expected number of zero-valued buckets for each granularity

### Acceptance criteria

- [ ] For an empty project, `GET /dashboard/velocity?granularity=daily&days=7` returns exactly 7 buckets, all with `count: 0`
- [ ] For an empty project, `GET /dashboard/velocity?granularity=weekly&days=28` returns exactly 4 buckets
- [ ] For an empty project, `GET /dashboard/velocity?granularity=monthly&days=90` returns exactly 3 buckets
- [ ] `make test-unit` and integration tests pass
