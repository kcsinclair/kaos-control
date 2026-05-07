---
title: 'Backend Plan: Artefacts List Release & Priority Columns'
type: plan-backend
status: done
lineage: artefacts-list-release-priority-columns
priority: high
parent: requirements/artefacts-list-release-priority-columns-2.md
release: May2026
---

# Backend Plan: Artefacts List Release & Priority Columns

## Summary

The backend already supports `release` and `priority` fields in the artifact model, SQLite index, HTTP query filters, and JSON responses. **No new API endpoints or schema changes are required.** This plan covers minor gaps and ensures the backend contract is solid for the frontend work in [[artefacts-list-release-priority-columns]].

---

## Milestone 1 — Verify `release` parameter in API response payload

### Description

Confirm that the `ArtifactRow` JSON serialisation includes `release` and `priority` at the top level (or within `frontmatter`) so the frontend can access them without parsing raw frontmatter. Currently, `priority` is a dedicated column on the `artifacts` table and appears in the response, while `release` lives inside `frontmatter_json`. Both are available in the `frontmatter` object of the JSON response. No code change is expected — this milestone is verification only.

### Files to check

- `internal/http/artifacts.go` — `handleListArtifacts`, response serialisation
- `internal/http/write.go` — `writeArtifactRow` / JSON marshalling helpers
- `internal/index/index.go` — `scanRow` / row-to-struct mapping

### Acceptance criteria

- [ ] `GET /api/p/:project/artifacts` returns each artifact with `frontmatter.release` and `frontmatter.priority` present (or omitted when empty), confirming the frontend can read these fields without changes.
- [ ] No code change needed if the above holds; document the finding in a commit message if verified.

---

## Milestone 2 — Add `release` to `filterParams` on the API client (frontend-adjacent)

### Description

The `filterParams()` function in `web/src/api/artifacts.ts` builds query-string parameters for the list endpoint. It currently omits `release` even though `ArtifactFilter` declares the field and the backend handler reads `r.URL.Query().Get("release")`. Add the missing line so server-side release filtering works from the frontend.

> **Note:** This is a frontend file but is included here because it directly exercises the backend API contract and the [[artefacts-list-release-priority-columns]] frontend plan depends on it.

### Files to change

- `web/src/api/artifacts.ts` — add `if (f.release) p.set('release', f.release)` inside `filterParams()`

### Acceptance criteria

- [ ] `filterParams({ release: 'v1.0' })` produces a query string containing `release=v1.0`.
- [ ] `filterParams({ release: '__unassigned__' })` produces `release=__unassigned__` so the backend's special-case handling is exercised.
- [ ] Existing filter parameters are unaffected.

---

## Milestone 3 — Ensure priority sort-order values are well-defined

### Description

The requirement specifies that priority sorts by logical severity: `critical > high > normal > low`. The backend stores priority as a plain string with no ordinal mapping. Sorting will be handled client-side (see [[artefacts-list-release-priority-columns]] frontend plan), but the backend must guarantee that the priority values returned match the expected vocabulary. Verify that the existing `listPriorities` endpoint (`GET /api/p/:project/priorities`) returns the distinct priority values and that no normalisation issues exist (e.g. mixed case).

### Files to check

- `internal/http/artifacts.go` or `internal/http/priorities.go` — priorities endpoint handler
- `internal/index/index.go` — `ListPriorities` query

### Acceptance criteria

- [ ] The priorities endpoint returns distinct values matching the artifact data.
- [ ] Values are consistently cased (lowercase) as stored in frontmatter.
- [ ] No backend code change needed if the above holds.

---

## Milestone 4 — Add integration test coverage for release filter

### Description

Add or extend an integration test that exercises the `release` query parameter on the artifact list endpoint, including the `__unassigned__` special value. This ensures the backend contract is regression-tested for the new frontend feature.

### Files to change

- `tests/` — new or existing integration test file covering `GET /api/p/:project/artifacts?release=...`

### Acceptance criteria

- [ ] Test verifies that `?release=<name>` returns only artifacts with matching release.
- [ ] Test verifies that `?release=__unassigned__` returns only artifacts with no release value.
- [ ] Test verifies that release filter composes with other filters (e.g. `?release=v1&status=draft`).
- [ ] All existing tests continue to pass.
