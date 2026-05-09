---
title: "Backend Plan: Recent Ideas and Defects Widget API Support"
type: plan-backend
status: approved
lineage: dashboard-recent-ideas-defects-widget
parent: lifecycle/requirements/dashboard-recent-ideas-defects-widget-2.md
created: "2026-05-09"
---

# Backend Plan: Recent Ideas and Defects Widget API Support

This plan covers the backend changes required to support the Recent Ideas and Defects dashboard widget. The widget needs to fetch the 6 most recent artifacts where type is `idea` or `defect`, sorted by `created` descending. The existing `GET /api/p/{project}/artifacts` endpoint supports single-value `type` filtering and has no `sort` parameter. Both must be extended.

Related plans: [[dashboard-recent-ideas-defects-widget]] (frontend: -4-fe, test: -5-test)

---

## Milestone 1: Multi-value `type` filter

### Description

Extend the `type` query parameter on `GET /api/p/{project}/artifacts` to accept comma-separated values (e.g. `?type=idea,defect`). When multiple types are provided, the SQL filter uses `IN (?, ?)` instead of `= ?`.

### Files to change

- `internal/index/index.go` — `Filter` struct and `buildWhere` function
  - `Filter.Type` remains a `string`. When the value contains a comma, `buildWhere` splits on `,`, generates `type IN (?, ?, ...)` with one placeholder per value, and appends each value to `args`.
  - No changes to `Filter.withDefaults()` or `List()`.

- `internal/http/artifacts.go` — No handler changes needed; `r.URL.Query().Get("type")` already passes the raw value through.

### Acceptance criteria

- `?type=idea,defect` returns only artifacts whose type is `idea` or `defect`.
- `?type=idea` (single value, no comma) continues to work identically via `type = ?`.
- Empty `?type=` or omitted `type` applies no type filter (existing behaviour).
- SQL injection is impossible — all values are parameterised.

---

## Milestone 2: `sort` query parameter

### Description

Add a `sort` query parameter to `GET /api/p/{project}/artifacts`. Format: `<column>:<direction>` where column is one of an allowlist and direction is `asc` or `desc`. Default (when omitted): `lineage, idx, path` (existing behaviour).

### Files to change

- `internal/index/index.go`
  - Add `Sort string` field to `Filter` struct.
  - Add a `buildOrderBy(f Filter) string` helper that:
    - Defines an allowlist map: `{"created": "created", "mtime": "mtime", "title": "title", "status": "status", "type": "type", "lineage": "lineage"}`.
    - Parses `f.Sort` as `column:direction`. If the column is in the allowlist and direction is `asc` or `desc`, returns `ORDER BY <mapped_column> <DIR>`. Otherwise falls back to `ORDER BY lineage, idx, path`.
    - Column names are mapped from the allowlist, never interpolated from user input.
  - Update `List()` to call `buildOrderBy(f)` instead of the hardcoded `ORDER BY lineage, idx, path`.

- `internal/http/artifacts.go`
  - Parse `sort` from `r.URL.Query().Get("sort")` and assign to `f.Sort`.

### Acceptance criteria

- `?sort=created:desc` returns artifacts ordered by `created` descending.
- `?sort=created:asc` returns artifacts ordered by `created` ascending.
- Invalid sort values (unknown column, missing direction, malformed format) silently fall back to the default order.
- No SQL injection risk — column names come from an allowlist, direction is validated.
- Existing callers with no `sort` parameter see no behaviour change.

---

## Milestone 3: Integration verification

### Description

Verify the combined query `?type=idea,defect&sort=created:desc&limit=6` works end-to-end and returns the expected result set for the widget.

### Files to change

- No additional code changes. This milestone is a verification step.

### Acceptance criteria

- `GET /api/p/{project}/artifacts?type=idea,defect&sort=created:desc&limit=6` returns at most 6 artifacts, of type `idea` or `defect` only, ordered by creation date most-recent-first.
- The `total` field in the response reflects the full count of matching artifacts (not capped by `limit`).
- The response completes within 300 ms on a dataset of 500 artifacts (per the non-functional requirement).
- The `created` field is present on all returned `ArtifactRow` objects.
