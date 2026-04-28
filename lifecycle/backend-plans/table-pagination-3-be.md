---
title: "Table Pagination — Backend Plan"
type: plan-backend
status: approved
lineage: table-pagination
parent: lifecycle/requirements/table-pagination-2.md
---

# Table Pagination — Backend Plan

This plan covers the backend changes needed to support client-side table pagination ([[table-pagination]]). The requirement explicitly states pagination is a rendering concern with no new API endpoints. However, the `ArtifactListView` currently uses server-side offset/limit pagination via the `/p/:project/artifacts` endpoint. To allow the frontend to paginate client-side (and support shareable deep-link URLs), the backend must support returning all artifacts in a single response when requested.

## Milestone 1 — Allow unbounded artifact list fetch

### Description

The artifacts list endpoint currently applies a default `limit` (50). When the frontend requests all artifacts for client-side pagination, it needs a way to fetch the full dataset in one call. Add support for `limit=0` (or `limit=-1`) to mean "return all rows" so the frontend can opt into client-side slicing.

### Files to change

- `internal/http/artifacts.go` — Update the list handler to treat `limit=0` as "no limit" instead of defaulting to 50. When limit is omitted, keep the existing default of 50 for backward compatibility.

### Acceptance criteria

- [ ] `GET /p/:project/artifacts?limit=0` returns all artifacts without truncation
- [ ] `GET /p/:project/artifacts` (no limit param) still defaults to 50 for backward compatibility
- [ ] Response still includes `total` count field
- [ ] No performance regression for the default case

## Milestone 2 — Verify agents runs and parse-errors endpoints return full datasets

### Description

Confirm that the agents runs endpoint (`GET /p/:project/agents/runs`) and the parse-errors endpoint (`GET /p/:project/parse-errors`) already return all rows without server-side pagination. If either applies a limit, update it to return the full dataset since client-side pagination will handle display.

### Files to change

- `internal/http/agents.go` — Verify/update runs list handler (likely no change needed)
- `internal/http/parse_errors.go` — Verify/update parse errors handler (likely no change needed)

### Acceptance criteria

- [ ] `GET /p/:project/agents/runs` returns all runs without truncation
- [ ] `GET /p/:project/parse-errors` returns all parse errors without truncation
- [ ] Existing API consumers (frontend views) are not broken
