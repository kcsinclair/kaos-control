---
title: "Backend Plan: Dashboard Clickable Filters"
type: plan-backend
status: in-development
lineage: dashboard-clickable-filters
parent: lifecycle/requirements/dashboard-clickable-filters-2.md
created: "2026-05-09"
---

# Backend Plan: Dashboard Clickable Filters

## Overview

Per the requirement's non-goals and acceptance criteria, **no backend API changes are required**. The existing `GET /p/:project/artifacts` endpoint already accepts all filter query parameters (`status`, `stage`, `type`, `label`, `priority`, `release`, `q`, `lineage`) that the frontend click-through navigation will use. This plan covers verification of that contract and documents the backend surface area the [[dashboard-clickable-filters]] frontend plan depends on.

## Milestone 1: Verify Existing Filter API Contract

**Description:** Confirm that the artifacts list API correctly handles all filter parameters that the dashboard click-through will produce, specifically `status=blocked` and an unfiltered request (Lifecycle Total). Verify that the API returns correct results for each status value present in the status distribution chart.

**Files to review (no changes expected):**

- `internal/http/artifacts.go` — `GET /p/:project/artifacts` handler; confirm `status` query parameter is read and passed to the index query.
- `internal/index/query.go` (or equivalent) — confirm the SQLite query builder applies status filtering correctly.
- `internal/http/dashboard.go` — confirm `/dashboard/status-distribution` returns status keys that are valid values for the `status` filter parameter on the artifacts endpoint.

**Acceptance criteria:**

- [ ] `GET /p/:project/artifacts?status=blocked` returns exactly the artifacts with `status: blocked` in their frontmatter.
- [ ] `GET /p/:project/artifacts?status=<s>` works for every status value returned by `/dashboard/status-distribution` (e.g., `draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `done`, `approved`, `rejected`, `abandoned`, `blocked`).
- [ ] `GET /p/:project/artifacts` with no filter parameters returns all artifacts (matches the "Lifecycle Total" count from `/dashboard/stats`).
- [ ] No new endpoints, fields, or handler modifications are introduced.

## Milestone 2: Document API Contract for Frontend Consumption

**Description:** Ensure the status vocabulary used by the dashboard stats endpoint and the status-distribution endpoint is consistent with the vocabulary accepted by the artifacts list filter. If any discrepancy is found (e.g., the distribution returns a status string the list endpoint doesn't recognise), raise it as a defect.

**Files to review (no changes expected):**

- `internal/artifact/types.go` — status vocabulary constants.
- `internal/http/dashboard.go` — status-distribution response shape.

**Acceptance criteria:**

- [ ] Every `status` string in the distribution response is a member of the canonical status vocabulary in `artifact/types.go`.
- [ ] No status value returned by the dashboard is silently ignored by the artifacts list filter.
