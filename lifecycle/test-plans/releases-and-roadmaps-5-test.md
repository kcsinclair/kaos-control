---
title: "Test Plan — Releases and Roadmaps"
type: plan-test
status: done
lineage: releases-and-roadmaps
parent: lifecycle/requirements/releases-and-roadmaps-2.md
---

# Test Plan — Releases and Roadmaps

This plan covers integration tests for the releases feature: API CRUD, rename propagation, artifact assignment, release-filtered queries, and the roadmap graph endpoint.

Cross-references: [[releases-and-roadmaps]] (backend plan for REST API and data model), [[releases-and-roadmaps]] (frontend plan for Gantt chart, roadmap graph, kanban filter UI).

---

## Milestone 1 — Release CRUD API tests

### Description

Write integration tests for all six release REST endpoints, covering happy paths, validation errors, and edge cases. These tests exercise the full HTTP stack (router → handler → store → SQLite).

### Files to change

- `tests/releases_test.go` (new) — test suite covering:
  - **Create** (`POST /api/p/:project/releases`):
    - Happy path: create a scheduled release with name, status, start_date, end_date → 201, response includes `id`.
    - Create with duration instead of end_date → end_date auto-calculated.
    - Create an unscheduled release (null dates) → 201.
    - Duplicate name → 409 Conflict.
    - Missing required field (name) → 400.
    - Name too long (121 chars) → 400.
    - end_date before start_date → 400.
    - Invalid status value → 400.
  - **List** (`GET /api/p/:project/releases`):
    - Returns all releases ordered by start_date, unscheduled last.
    - Empty project returns empty array, not null.
  - **Get** (`GET /api/p/:project/releases/:id`):
    - Returns release with `idea_count` and `defect_count` summary.
    - Non-existent ID → 404.
  - **Update** (`PUT /api/p/:project/releases/:id`):
    - Update dates → 200, dates changed.
    - Update name → 200, triggers rename propagation (tested in Milestone 3).
    - Update status → 200.
    - Rename to existing name → 409.
    - Non-existent ID → 404.
  - **Delete** (`DELETE /api/p/:project/releases/:id`):
    - Delete release with no assigned artifacts → 200, `orphaned_artifact_count: 0`.
    - Delete release with assigned artifacts → 200, returns correct orphaned count.
    - Non-existent ID → 404.
  - **List Artifacts** (`GET /api/p/:project/releases/:id/artifacts`):
    - Returns only artifacts assigned to this release.
    - Empty release returns empty array.

### Acceptance criteria

- [ ] All happy-path CRUD operations return expected status codes and response bodies.
- [ ] All validation error cases return 400 with descriptive error messages.
- [ ] Duplicate name creation returns 409.
- [ ] Non-existent IDs return 404.
- [ ] Releases persist across test queries (SQLite storage works).
- [ ] `go test ./tests/... -run TestReleases` passes.

---

## Milestone 2 — Release status lifecycle tests

### Description

Test the release `status` field behaviour: valid transitions, display filtering, and interaction with the Gantt chart data expectations.

### Files to change

- `tests/releases_test.go` — add test cases:
  - Create a release with status `planned` → update to `active` → update to `shipped` → all succeed.
  - Create with each valid status (`planned`, `active`, `shipped`) → all succeed.
  - Create with invalid status → 400.
  - List releases: verify `planned`, `active`, and `shipped` releases all appear.

### Acceptance criteria

- [ ] All three status values are accepted on create and update.
- [ ] Invalid status values are rejected with 400.
- [ ] Status changes persist correctly.
- [ ] `go test ./tests/... -run TestReleaseStatus` passes.

---

## Milestone 3 — Rename propagation tests

### Description

Test that renaming a release via `PUT` correctly updates the `release` frontmatter field in all assigned artifact markdown files and commits the changes.

### Files to change

- `tests/releases_rename_test.go` (new) — test suite:
  - **Setup**: create a project with a release named "v1.0". Create two idea artifacts and one defect artifact assigned to "v1.0" via their `release` frontmatter field.
  - **Test rename propagation**:
    - Rename release from "v1.0" to "v1.1" via `PUT`.
    - Read each artifact file from disk and verify `release: v1.1` in frontmatter.
    - Verify the artifacts are re-indexed: `GET /artifacts?release=v1.1` returns all three.
    - Verify `GET /artifacts?release=v1.0` returns empty.
  - **Test git commit**: verify a git commit was created with message containing the old and new names.
  - **Test no collateral damage**: create an artifact assigned to "v2.0" (different release). After renaming "v1.0" → "v1.1", verify the "v2.0" artifact is unchanged.
  - **Test unassigned unaffected**: create an artifact with no release field. After rename, verify it still has no release field.

### Acceptance criteria

- [ ] All artifact files assigned to the old name have their `release` field updated on disk.
- [ ] The SQLite index reflects the new release name for all affected artifacts.
- [ ] A git commit is created for the rename.
- [ ] Artifacts assigned to other releases are not modified.
- [ ] Artifacts with no release are not modified.
- [ ] `go test ./tests/... -run TestReleaseRename` passes.

---

## Milestone 4 — Delete and reassign tests

### Description

Test release deletion behaviour: orphaning artifacts when no reassignment target is given, and reassigning artifacts when a target release is specified.

### Files to change

- `tests/releases_delete_test.go` (new) — test suite:
  - **Delete without reassignment**:
    - Create release "v1.0" with two assigned artifacts.
    - `DELETE /releases/:id` → 200, `orphaned_artifact_count: 2`.
    - Verify artifact files on disk still have `release: v1.0` (not cleared).
    - Verify `GET /releases` no longer includes "v1.0".
  - **Delete with reassignment**:
    - Create releases "v1.0" and "v2.0". Assign three artifacts to "v1.0".
    - `DELETE /releases/:id?reassign_to=<v2.0 id>` → 200.
    - Read artifact files from disk: verify `release: v2.0` in frontmatter.
    - Verify `GET /releases/:v2.0_id/artifacts` returns the three reassigned artifacts.
  - **Delete empty release**: delete a release with no assigned artifacts → 200, `orphaned_artifact_count: 0`.
  - **Reassign to non-existent release** → 400 or 404.

### Acceptance criteria

- [ ] Deleting without reassignment leaves artifact files unchanged.
- [ ] Deleting with reassignment updates artifact frontmatter to the target release name.
- [ ] The `orphaned_artifact_count` in the response is accurate.
- [ ] Reassigning to a non-existent release returns an error.
- [ ] `go test ./tests/... -run TestReleaseDelete` passes.

---

## Milestone 5 — Artifact release-filter query tests

### Description

Test the `release` query parameter on the artifacts and graph endpoints for server-side filtering.

### Files to change

- `tests/releases_filter_test.go` (new) — test suite:
  - **Setup**: create two releases ("v1.0", "v2.0"). Create artifacts: two ideas assigned to "v1.0", one defect assigned to "v2.0", one idea with no release.
  - **Filter by release name**:
    - `GET /artifacts?release=v1.0` → returns exactly the two "v1.0" ideas.
    - `GET /artifacts?release=v2.0` → returns exactly the one "v2.0" defect.
  - **Filter unassigned**:
    - `GET /artifacts?release=__unassigned__` → returns only the artifact with no release.
  - **No filter**:
    - `GET /artifacts` (no release param) → returns all artifacts.
  - **Graph filter**:
    - `GET /graph?release=v1.0` → graph contains only "v1.0" artifacts as nodes.
  - **Roadmap graph endpoint**:
    - `GET /releases/graph` → returns releases as nodes, ideas/defects as child nodes, timeline edges between releases.
    - Verify only `idea` and `defect` type artifacts appear (not plans, tests, etc.).
    - Verify releases are connected chronologically.

### Acceptance criteria

- [ ] Release filter on `/artifacts` returns only matching artifacts.
- [ ] `__unassigned__` filter returns artifacts with no release field.
- [ ] Omitting the filter returns all artifacts.
- [ ] Graph endpoint respects the release filter.
- [ ] Roadmap graph includes only `idea` and `defect` artifact types.
- [ ] Roadmap graph connects releases chronologically.
- [ ] `go test ./tests/... -run TestReleaseFilter` passes.

---

## Milestone 6 — WebSocket event tests

### Description

Test that release CRUD operations broadcast the correct WebSocket events for real-time UI updates.

### Files to change

- `tests/releases_ws_test.go` (new) — test suite:
  - Connect a WebSocket client to `/api/p/:project/ws`.
  - **Create**: create a release → receive `release.created` event with release data.
  - **Update**: update a release → receive `release.updated` event with updated release data.
  - **Delete**: delete a release → receive `release.deleted` event with release ID.
  - **Rename propagation**: rename a release that has assigned artifacts → receive `release.updated` event followed by `artifact.indexed` events for each updated artifact.

### Acceptance criteria

- [ ] `release.created` event is received after creating a release.
- [ ] `release.updated` event is received after updating a release.
- [ ] `release.deleted` event is received after deleting a release.
- [ ] Rename propagation emits `artifact.indexed` events for affected artifacts.
- [ ] Event payloads contain the expected fields.
- [ ] `go test ./tests/... -run TestReleaseWebSocket` passes.

---

## Milestone 7 — Unscheduled release tests

### Description

Test the unscheduled release variant: creation with null dates, listing order, and behaviour in the roadmap graph.

### Files to change

- `tests/releases_unscheduled_test.go` (new) — test suite:
  - Create an unscheduled release (no start_date, no end_date) → 201.
  - Verify it appears after all scheduled releases in `GET /releases` list.
  - Create a second unscheduled release → both appear at the end of the list, ordered by name.
  - Assign artifacts to an unscheduled release → `GET /releases/:id/artifacts` returns them.
  - Roadmap graph endpoint: unscheduled releases appear as nodes but are not connected via timeline edges (they have no chronological position).
  - Update an unscheduled release to add dates → it moves to the correct chronological position in the list.

### Acceptance criteria

- [ ] Unscheduled releases are created successfully with null dates.
- [ ] They sort after scheduled releases in list queries.
- [ ] Artifacts can be assigned to unscheduled releases.
- [ ] The roadmap graph includes unscheduled releases as disconnected nodes (no timeline edges).
- [ ] Scheduling a previously unscheduled release repositions it correctly.
- [ ] `go test ./tests/... -run TestReleaseUnscheduled` passes.
