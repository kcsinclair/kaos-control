---
title: "Backend Plan — Releases and Roadmaps"
type: plan-backend
status: approved
lineage: releases-and-roadmaps
parent: lifecycle/requirements/releases-and-roadmaps-2.md
---

# Backend Plan — Releases and Roadmaps

This plan covers all Go backend changes required to support releases as first-class entities: SQLite schema, REST API, artifact–release assignment, rename propagation, and release-filtered queries.

Cross-references: [[releases-and-roadmaps]] (frontend plan for Gantt chart, roadmap graph, kanban filter UI), [[releases-and-roadmaps]] (test plan for integration tests).

---

## Milestone 1 — Release data model and schema migration

### Description

Add a `releases` table to the SQLite schema and bump `schemaVersion`. Add a `release` column index on the `artifacts` table to support efficient filtering. Define a Go `Release` struct with JSON serialisation tags.

### Files to change

- `internal/index/index.go` — bump `schemaVersion` to 4; add `CREATE TABLE releases (id INTEGER PRIMARY KEY AUTOINCREMENT, project_id TEXT NOT NULL, name TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'planned', start_date TEXT, end_date TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(project_id, name))` and `CREATE INDEX idx_artifacts_release ON artifacts(json_extract(frontmatter_json, '$.release'))` inside `createSchema`.
- `internal/release/release.go` (new) — define `Release` struct with fields: `ID int64`, `ProjectID string`, `Name string`, `Status string` (enum: `planned`, `active`, `shipped`), `StartDate *time.Time`, `EndDate *time.Time`, `CreatedAt time.Time`, `UpdatedAt time.Time`. Include JSON tags and a `Validate()` method enforcing: name 1–120 chars, `end_date >= start_date` when both are set, status is one of the three valid values. Unscheduled releases have nil `StartDate` and `EndDate`.

### Acceptance criteria

- [ ] `schemaVersion` is bumped and the `releases` table is created on startup.
- [ ] The `releases` table has a unique constraint on `(project_id, name)`.
- [ ] The `Release` struct validates name length, date ordering, and status vocabulary.
- [ ] Unscheduled releases (nil dates) pass validation.
- [ ] An index on the artifact `release` field exists for query performance.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 2 — Release repository (CRUD queries)

### Description

Implement a repository layer for release CRUD operations against SQLite. This keeps SQL out of HTTP handlers and mirrors the pattern used by `internal/index/` for artifacts.

### Files to change

- `internal/release/store.go` (new) — implement `Store` struct wrapping `*sql.DB` with methods:
  - `List(projectID string) ([]Release, error)` — ordered by `start_date NULLS LAST, name`.
  - `Get(projectID string, id int64) (*Release, error)` — includes summary counts of assigned ideas/defects via a subquery on `artifacts.frontmatter_json`.
  - `Create(r *Release) error` — inserts and returns populated `ID`, `CreatedAt`, `UpdatedAt`.
  - `Update(r *Release) error` — updates name, status, dates, bumps `UpdatedAt`. Returns old name for rename propagation.
  - `Delete(projectID string, id int64) error` — deletes the row. Returns the release name and count of referencing artifacts for the caller to use in warnings/reassignment.
  - `GetByName(projectID, name string) (*Release, error)` — lookup by name for assignment validation.
  - `ListArtifacts(projectID string, releaseID int64) ([]artifact.Artifact, error)` — returns artifacts whose `release` field matches this release's name.

### Acceptance criteria

- [ ] All CRUD methods execute correct SQL and return well-formed results.
- [ ] `List` returns unscheduled releases after scheduled ones.
- [ ] `Get` includes `idea_count` and `defect_count` summary fields.
- [ ] `Delete` returns the count of artifacts that reference the release.
- [ ] `Create` rejects duplicate names within the same project (unique constraint).
- [ ] `go test ./internal/release/...` passes with unit tests for validation and store methods.

---

## Milestone 3 — Release REST API endpoints

### Description

Register release API routes under `/api/p/{project}/releases` in the chi router. Implement handlers for all six endpoints defined in DR-2.

### Files to change

- `internal/http/server.go` — add `r.Route("/releases", ...)` inside the project route group, mounting the six endpoints.
- `internal/http/releases.go` (new) — implement handlers:
  - `handleListReleases` — `GET /` → calls `store.List`, returns JSON array.
  - `handleCreateRelease` — `POST /` → parses JSON body with `name`, `status`, `start_date`, `end_date` (or `duration`); calls `store.Create`; broadcasts `release.created` WS event; returns 201.
  - `handleGetRelease` — `GET /{releaseID}` → calls `store.Get`, returns release with summary counts.
  - `handleUpdateRelease` — `PUT /{releaseID}` → calls `store.Update`; if name changed, triggers rename propagation (Milestone 4); broadcasts `release.updated` WS event; returns 200.
  - `handleDeleteRelease` — `DELETE /{releaseID}` → accepts optional `reassign_to` query param (release ID to reassign artifacts to); calls `store.Delete`; if `reassign_to` is provided, updates artifact frontmatter accordingly; broadcasts `release.deleted` WS event; returns 200 with `{ "orphaned_artifact_count": N }`.
  - `handleListReleaseArtifacts` — `GET /{releaseID}/artifacts` → calls `store.ListArtifacts`, returns JSON array.
- Duration calculation: when request body contains `duration` (e.g., `"14d"` or `"2w"`), compute `end_date = start_date + duration` before passing to the store.

### Acceptance criteria

- [ ] All six endpoints are registered and respond with correct HTTP status codes.
- [ ] `POST` returns 201 with the created release including `id`.
- [ ] `POST` with duplicate name returns 409 Conflict.
- [ ] `PUT` with a new name triggers rename propagation (next milestone).
- [ ] `DELETE` returns `orphaned_artifact_count` in response body.
- [ ] `DELETE` with `reassign_to` parameter reassigns artifacts to the target release.
- [ ] Duration strings (`"14d"`, `"2w"`) are correctly parsed to compute `end_date`.
- [ ] WebSocket events `release.created`, `release.updated`, `release.deleted` are broadcast.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 4 — Release rename propagation

### Description

When a release is renamed via `PUT`, update the `release` frontmatter field on all artifacts currently assigned to the old name, commit the changes to git, and re-index.

### Files to change

- `internal/http/releases.go` — within `handleUpdateRelease`, after detecting a name change, call a propagation function.
- `internal/release/propagate.go` (new) — implement `PropagateRename(projectRoot, oldName, newName string, index *index.Index, git *git.Repo) (int, error)`:
  1. Query `index.List` with `release=oldName` filter to find all affected artifacts.
  2. For each artifact, call `artifact.PatchFrontmatterField(raw, "release", newName)` to update the file on disk.
  3. Stage all changed files and create a single git commit: `chore(releases): rename "{oldName}" → "{newName}"`.
  4. Re-index all changed paths.
  5. Return count of updated artifacts.

### Acceptance criteria

- [ ] Renaming a release updates the `release` field in all assigned artifact markdown files.
- [ ] A single git commit is created with all file changes.
- [ ] All changed artifacts are re-indexed in SQLite.
- [ ] The function returns the count of updated artifacts.
- [ ] Files not assigned to the release are untouched.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 5 — Artifact release-filter query support

### Description

Extend the existing artifact list/graph APIs to accept an optional `release` query parameter for server-side filtering. This supports the kanban release filter (DR-6) and the roadmap graph view (DR-5).

### Files to change

- `internal/index/index.go` — extend `Filter` struct with `Release string` field. Update `List()` and `Graph()` SQL queries to add `AND json_extract(frontmatter_json, '$.release') = ?` when `Release` is non-empty. Add a special value `"__unassigned__"` to match artifacts with null/empty release.
- `internal/http/artifacts.go` — read `release` query param in `handleListArtifacts` and pass it into the `Filter`.
- `internal/http/graph.go` — read `release` query param in `handleGraph` and pass it into the `Filter`.

### Acceptance criteria

- [ ] `GET /api/p/:project/artifacts?release=v1.0` returns only artifacts assigned to release "v1.0".
- [ ] `GET /api/p/:project/artifacts?release=__unassigned__` returns only artifacts with no release.
- [ ] `GET /api/p/:project/graph?release=v1.0` returns a graph filtered to that release's artifacts.
- [ ] Omitting the `release` param returns all artifacts (no change to default behaviour).
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 6 — Roadmap graph data endpoint

### Description

Add a dedicated endpoint to return graph data for the roadmap view: releases as nodes connected chronologically, with their assigned ideas/defects as child nodes.

### Files to change

- `internal/http/releases.go` — add `handleRoadmapGraph` at `GET /api/p/{project}/releases/graph`:
  1. Fetch all releases ordered by `start_date`.
  2. For each release, fetch assigned artifacts (types `idea` and `defect` only).
  3. Build nodes: release nodes (with a `"release"` node type, larger weight) + artifact nodes.
  4. Build edges: release → artifact (kind: `"assigned"`), release → next release (kind: `"timeline"`), plus any existing `depends_on`/`blocks` edges between included artifacts.
  5. Return `{ "nodes": [...], "edges": [...] }` in the same shape used by the existing `GET /graph` endpoint.
- `internal/http/server.go` — register the route.

### Acceptance criteria

- [ ] The endpoint returns releases as nodes with a distinct `type: "release"` field.
- [ ] Releases are connected in chronological order via `"timeline"` edges.
- [ ] Only `idea` and `defect` artifacts are included as nodes.
- [ ] Existing `depends_on`/`blocks` edges between included artifacts are preserved.
- [ ] The response shape matches the existing graph endpoint for frontend compatibility.
- [ ] `go build ./...` and `go vet ./...` pass.
