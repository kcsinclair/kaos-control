---
title: "Sortable Table Columns — Backend Plan"
type: plan-backend
status: in-development
lineage: sortable-table-columns
parent: lifecycle/requirements/sortable-table-columns-2.md
---

# Sortable Table Columns — Backend Plan

Sorting is a client-side-only concern per [[sortable-table-columns]]. No new API query parameters or server-side sort logic is required. The backend work is limited to ensuring every table's API response includes the data the frontend needs to sort correctly, and that the artifact list endpoint can return the full dataset for client-side sorting to operate across all rows (not just the current page).

## Milestone 1 — Ensure artifact list endpoint supports full-dataset fetch

### Description

The `ArtifactListView` currently uses server-side offset/limit pagination (`GET /p/:project/artifacts`). Client-side sorting must operate on the full dataset, not a single page. The `table-pagination` feature already added support for `limit=0` to return all rows. Verify this works correctly and that the frontend can rely on it.

If `limit=0` support is not yet present, add it: treat `limit=0` in the list handler as "no limit" so all artifacts are returned in a single response.

### Files to change

- `internal/http/artifacts.go` — Verify that `limit=0` returns all artifacts. If not already implemented, update the handler to treat `limit=0` as unbounded.

### Acceptance criteria

- [ ] `GET /p/:project/artifacts?limit=0` returns all artifacts without truncation
- [ ] Response includes accurate `total` count matching the number of returned items
- [ ] Default behaviour (no `limit` param) is unchanged — still returns 50

## Milestone 2 — Verify date fields are returned in a sortable format

### Description

The frontend needs `created` and `mtime` fields in ISO 8601 format (or any format that is both chronologically sortable as a string and parseable by `new Date()`). Verify the artifact list endpoint returns these fields consistently. The agent runs endpoint (`started_at`, `finished_at`) should also be checked.

### Files to change

- `internal/http/artifacts.go` — Verify `created` and `mtime` are serialized as RFC 3339 / ISO 8601 strings.
- `internal/http/agents.go` — Verify `started_at` and `finished_at` are serialized as RFC 3339 / ISO 8601 strings.

### Acceptance criteria

- [ ] `created` and `mtime` on every `ArtifactRow` in the JSON response are valid ISO 8601 strings
- [ ] `started_at` and `finished_at` on every agent run in the JSON response are valid ISO 8601 strings
- [ ] No change to existing response shapes — this is a verification milestone

## Milestone 3 — Verify agent runs and parse-errors endpoints return full datasets

### Description

Confirm that `GET /p/:project/agents/runs` and `GET /p/:project/parse-errors` return all rows without server-side truncation. Client-side sorting (and pagination, per [[table-pagination]]) requires the full dataset. These endpoints likely already return everything, but this must be verified.

### Files to change

- `internal/http/agents.go` — Verify the runs list handler has no implicit limit.
- `internal/http/parse_errors.go` — Verify the parse-errors handler has no implicit limit.

### Acceptance criteria

- [ ] `GET /p/:project/agents/runs` returns all runs without truncation
- [ ] `GET /p/:project/parse-errors` returns all parse errors without truncation
- [ ] No breaking changes to response shapes
