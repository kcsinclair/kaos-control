---
title: "Integration Tests — Releases and Roadmaps"
type: test
status: in-qa
lineage: releases-and-roadmaps
parent: lifecycle/test-plans/releases-and-roadmaps-5-test.md
---

# Integration Tests — Releases and Roadmaps

Integration tests for the releases feature: REST API CRUD, status lifecycle, rename propagation, delete/reassign, release-filter queries, WebSocket events, and unscheduled release behaviour.

## Test files

All tests live in `tests/integration/` and carry the `//go:build integration` tag. Run the full suite with:

```sh
go test -tags integration ./tests/... -v -run TestRelease
```

## Scenarios covered

### Milestone 1 — Release CRUD (`tests/integration/releases_test.go`)

Run with: `go test -tags integration ./tests/... -run TestReleases`

- **Create happy path** — POST with name, status, start_date, end_date → 201, id present.
- **Create with duration** — `duration: "14d"` auto-calculates end_date.
- **Create unscheduled** — no dates → 201, start_date/end_date are null.
- **Duplicate name** → 409 Conflict.
- **Missing name** → 400.
- **Name too long** (121 chars) → 400.
- **end_date before start_date** → 400.
- **Invalid status value** → 400.
- **List ordered by start_date** — scheduled releases first, unscheduled last.
- **List empty project** — returns empty array (not null).
- **Get with counts** — idea_count and defect_count summary populated from indexed artifacts.
- **Get non-existent ID** → 404.
- **Update dates** → 200, dates changed.
- **Update status** → 200.
- **Update to duplicate name** → 409.
- **Update non-existent ID** → 404 (spec behaviour; current impl returns 500 — documents a known gap).
- **Delete no artifacts** → 200, orphaned_artifact_count=0.
- **Delete with artifacts** → 200, correct orphaned count.
- **Delete non-existent ID** → 404 (spec behaviour; current impl returns 500 — documents a known gap).
- **List release artifacts** — returns only assigned artifacts.
- **List release artifacts empty** — empty array (not null).

### Milestone 2 — Release status lifecycle (`tests/integration/releases_test.go`)

Run with: `go test -tags integration ./tests/... -run TestReleaseStatus`

- **Valid transition** — planned → active → shipped all succeed.
- **All statuses valid on create** — planned, active, shipped each return 201.
- **Invalid status rejected** — on create and update, unknown status → 400.
- **All statuses in list** — planned, active, and shipped releases all appear in GET /releases.

### Milestone 3 — Rename propagation (`tests/integration/releases_rename_test.go`)

Run with: `go test -tags integration ./tests/... -run TestReleaseRename`

- **Propagates release field** — renaming via PUT rewrites `release:` frontmatter in all assigned artifact files on disk and re-indexes them.
- **Git commit created** — a commit is produced whose message contains both old and new release names.
- **No collateral damage** — artifacts assigned to a different release are not modified.
- **Unassigned unaffected** — artifacts with no release field remain unchanged.

### Milestone 4 — Delete and reassign (`tests/integration/releases_delete_test.go`)

Run with: `go test -tags integration ./tests/... -run TestReleaseDelete`

- **Delete without reassignment** — artifact files retain original `release:` field; release removed from list; orphaned_artifact_count accurate.
- **Delete with reassignment** — `?reassign_to=<id>` rewrites artifact files to target release; target artifact list contains reassigned artifacts.
- **Delete empty release** → 200, orphaned_artifact_count=0.
- **Reassign to non-existent release** → 400 or 404.

### Milestone 5 — Artifact release-filter queries (`tests/integration/releases_filter_test.go`)

Run with: `go test -tags integration ./tests/... -run TestReleaseFilter`

- **Filter by release name** — `?release=<name>` returns only artifacts with that release.
- **Filter unassigned** — `?release=__unassigned__` returns only artifacts with no release field.
- **No filter** — all artifacts returned.
- **Graph endpoint filter** — `GET /graph?release=<name>` returns only matching artifact nodes.
- **Roadmap graph** — `GET /releases/graph` includes release nodes, idea and defect child nodes, timeline edges between scheduled releases; plan-backend excluded.

### Milestone 6 — WebSocket events (`tests/integration/releases_ws_test.go`)

Run with: `go test -tags integration ./tests/... -run TestReleaseWebSocket`

Uses the hub channel pattern (`env.proj.Hub.Register`) for in-process event capture.

- **release.created** — broadcast on POST /releases with release payload.
- **release.updated** — broadcast on PUT /releases/:id with updated release data.
- **release.deleted** — broadcast on DELETE /releases/:id with release ID.
- **Rename propagation** — rename with assigned artifacts emits both `release.updated` and at least one `artifact.indexed` event.

### Milestone 7 — Unscheduled releases (`tests/integration/releases_unscheduled_test.go`)

Run with: `go test -tags integration ./tests/... -run TestReleaseUnscheduled`

- **Create succeeds** — null dates → 201.
- **Sorts after scheduled** — appears after all scheduled releases in list.
- **Two unscheduled ordered by name** — alphabetical ordering within unscheduled group.
- **Artifact assignment** — artifacts can be assigned; appear in /artifacts endpoint.
- **Roadmap graph disconnected** — unscheduled release appears as node but has no timeline edges.
- **Update to add dates** — scheduling a previously unscheduled release moves it to correct chronological position.

## Known gaps identified during implementation

- `PUT /releases/:id` on a non-existent ID returns 500 instead of 404 — the handler does not distinguish a not-found error from `Store.Update` vs other DB errors.
- `DELETE /releases/:id` on a non-existent ID returns 500 instead of 404 — same root cause.

Both gaps are covered by `TestReleases_UpdateNotFound` and `TestReleases_DeleteNotFound` which assert the correct spec behaviour (404) and will pass once the handlers are fixed.
