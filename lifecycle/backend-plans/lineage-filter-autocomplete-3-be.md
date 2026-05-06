---
title: "Backend Plan: Lineage Filter with Autocomplete"
type: plan-backend
status: abandoned
lineage: lineage-filter-autocomplete
parent: lifecycle/requirements/lineage-filter-autocomplete-2.md
---

# Backend Plan: Lineage Filter with Autocomplete

## Summary

The requirement explicitly states that no backend API changes are needed — the frontend already receives lineage data via the existing `GET /p/:project/lineages` endpoint, which returns `LineageSummary[]` containing both the lineage slug and per-status artifact counts. The existing `GET /p/:project/artifacts` endpoint already accepts a `lineage` query parameter for server-side filtering.

This plan documents the existing backend surface that the [[lineage-filter-autocomplete]] frontend plan relies on, and identifies one minor enhancement to improve UX.

---

## Milestone 1: Verify existing endpoint contract

### Description

Confirm that the existing `/lineages` endpoint returns all data the frontend autocomplete needs: distinct lineage slugs and artifact counts.

### Files to review (no changes expected)

- `internal/http/artifacts.go` — handler for `GET /p/:project/lineages`
- `internal/index/queries.go` — SQL query backing the lineages list

### Acceptance Criteria

- [ ] `GET /p/:project/lineages` returns `{ lineages: [{ lineage: string, members: string[], statuses: Record<string, number> }] }`.
- [ ] The `statuses` map values can be summed client-side to derive a total artifact count per lineage.
- [ ] Response time is < 50 ms for projects with ≤ 500 distinct lineage slugs (existing SQLite index on `lineage` column).

---

## Milestone 2: Ensure `lineage` filter parameter on artifact list endpoint

### Description

Confirm that the existing `GET /p/:project/artifacts?lineage=<value>` performs substring matching consistent with the frontend's free-text filter behaviour. If it currently does exact match only, this is acceptable — the frontend will handle substring filtering client-side from the already-loaded artifact list.

### Files to review (no changes expected)

- `internal/http/artifacts.go` — query param parsing
- `internal/index/queries.go` — WHERE clause construction for `lineage` filter

### Acceptance Criteria

- [ ] The `lineage` query parameter is documented and functional.
- [ ] Filtering composes correctly with `stage`, `status`, `type`, `label`, and `priority` parameters (AND logic).
- [ ] No regressions to existing filter behaviour.

---

## Milestone 3: Add `total` field to LineageSummary response (enhancement)

### Description

To avoid the frontend needing to sum status counts, add a pre-computed `total` integer field to each `LineageSummary` object in the API response. This is a non-breaking additive change.

### Files to change

- `internal/index/queries.go` — add `COUNT(*)` to the lineages query and map it to the response struct.
- `internal/http/artifacts.go` — include `total` in the JSON serialization of `LineageSummary`.

### Acceptance Criteria

- [ ] Each object in the `lineages` array includes `"total": <int>` representing the count of all artifacts in that lineage.
- [ ] Existing fields (`lineage`, `members`, `statuses`) remain unchanged.
- [ ] Unit tests for the lineages query pass with the new field.
- [ ] The endpoint remains < 50 ms for ≤ 500 lineages.
