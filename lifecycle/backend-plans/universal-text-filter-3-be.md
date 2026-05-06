---
title: "Universal Text Filter — Backend Plan"
type: plan-backend
status: done
lineage: universal-text-filter
parent: lifecycle/requirements/universal-text-filter-2.md
---

# Universal Text Filter — Backend Plan

This plan covers the backend changes required to support the universal text filter feature ([[universal-text-filter]]). The backend work is limited to the artifact list endpoint (`GET /artifacts`), which is the only server-side filtered view. The Kanban, Graph, and Feed views all filter client-side and require no backend changes.

## Milestone 1 — Add `Q` field to `index.Filter` and update `buildWhere`

### Description

Add a free-text search field to the `Filter` struct and extend `buildWhere` to generate a case-insensitive substring match across `title`, `slug`, `lineage`, `type`, and `status` columns when the field is non-empty. Use SQLite `LIKE` with `%` wildcards; the `LIKE` operator in SQLite is already case-insensitive for ASCII characters.

### Files to change

- `internal/index/index.go`
  - Add `Q string` field to the `Filter` struct (after `Priority`).
  - In `buildWhere`, when `f.Q != ""`, append a grouped OR condition:
    ```sql
    (title LIKE ? OR slug LIKE ? OR lineage LIKE ? OR type LIKE ? OR status LIKE ?)
    ```
    with `%<q>%` as the argument for each placeholder. Escape any literal `%` or `_` characters in the user input to prevent unintended wildcards.

### Acceptance criteria

- [ ] `Filter{Q: "kanban"}` produces a WHERE clause containing the five-column `LIKE` OR group.
- [ ] The match is case-insensitive (e.g. `Q: "Kanban"` matches a row with `slug = "kanban-view"`).
- [ ] Literal `%` and `_` in the query string are escaped so they match literally.
- [ ] `Filter{Q: ""}` produces no additional WHERE condition (no change to existing behaviour).
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 2 — Wire `q` query parameter in the list handler

### Description

Update the `handleListArtifacts` handler to read a `q` query parameter from the request URL and populate `Filter.Q`. When `q` changes, the frontend will also reset `offset` to 0, but no backend enforcement is needed — the frontend handles pagination reset.

### Files to change

- `internal/http/artifacts.go`
  - In `handleListArtifacts`, add `Q: r.URL.Query().Get("q")` to the `index.Filter` literal (after `Priority`).

### Acceptance criteria

- [ ] `GET /api/p/{project}/artifacts?q=login` returns only artifacts whose title, slug, lineage, type, or status contains "login" (case-insensitive).
- [ ] `GET /api/p/{project}/artifacts?q=login&status=draft` composes the text filter with the status filter using AND logic.
- [ ] `GET /api/p/{project}/artifacts` (no `q` param) behaves identically to before this change.
- [ ] The `total` count in the response reflects the filtered count (i.e. the COUNT query also includes the `q` condition).
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 3 — Unit tests for text filter query building

### Description

Add unit tests to verify the `buildWhere` function correctly handles the `Q` field, both in isolation and composed with other filter fields.

### Files to change

- `internal/index/index_test.go` (or a new `internal/index/filter_test.go` if the existing file is large) — add test cases.

### Test cases

1. **Q only** — `Filter{Q: "hello"}` produces a WHERE clause with the five-column LIKE OR group and five `%hello%` arguments.
2. **Q + Status** — `Filter{Q: "hello", Status: "draft"}` produces both conditions joined by AND.
3. **Q with special characters** — `Filter{Q: "100%"}` escapes the `%` so it matches literally.
4. **Empty Q** — `Filter{Q: ""}` produces no Q-related condition.
5. **Case insensitivity** — Insert two rows (one with title "Kanban View", one with "kanban-board"), filter with `Q: "kanban"`, and assert both are returned.

### Acceptance criteria

- [ ] All five test cases pass.
- [ ] No existing tests are broken.
- [ ] `go test ./internal/index/ -run TestFilter` passes.

---

## Notes

- The Graph endpoint (`Graph()`) also calls `buildWhere`, so graph-level server-side filtering will automatically gain `q` support if the frontend ever sends it. For now the [[universal-text-filter]] frontend plan filters graph nodes client-side, so this is a free bonus with no additional work.
- Full-text search (FTS5) is explicitly a non-goal (NG5) for this version. The `LIKE` approach is sufficient for frontmatter-field matching on datasets up to 500 artifacts.
