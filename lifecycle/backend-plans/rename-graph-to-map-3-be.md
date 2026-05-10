---
title: "Backend Plan: Rename Graph to Map in UI and Routing"
type: plan-backend
status: approved
lineage: rename-graph-to-map
parent: lifecycle/requirements/rename-graph-to-map-2.md
---

# Backend Plan: Rename Graph to Map in UI and Routing

## Overview

The requirement explicitly states: **do not rename the backend API endpoint** (`/api/p/:project/graph`). The API is not user-facing and renaming it adds risk for no user benefit. Therefore, the backend scope for this change is **zero**.

No Go source files, no API routes, no handler functions, and no backend tests require modification for this rename.

## Milestones

### Milestone 1 — Confirm No Backend Changes Required

**Description:** Verify that the backend has no user-facing references to "Graph" that need renaming, and that the existing `/api/p/:project/graph` endpoint will continue to serve data correctly after the frontend route change.

**Files to review (no changes expected):**

- `internal/http/graph.go` — serves `GET /api/p/:project/graph` and `GET /api/p/:project/releases/graph`. These endpoints are consumed by the frontend API client (`web/src/api/graph.ts`) which is explicitly out of scope per the requirement's non-goals.
- `internal/http/server.go` — route registration at lines 182 and 205. No changes.

**Acceptance criteria:**

- [ ] `make build` succeeds with no changes to Go source files
- [ ] `make test-unit` passes with no changes to Go test files
- [ ] `GET /api/p/:project/graph` continues to return graph data after the frontend rename
- [ ] No user-facing strings containing "Graph" exist in Go source files (confirmed by grep)

## Dependencies

- None. The backend is not blocked by and does not block [[rename-graph-to-map]] frontend or test plans, but should be verified after the frontend plan is complete.

## Notes

If a future requirement arises to rename the API endpoint from `/graph` to `/map`, that should be treated as a separate lineage or a new requirement — it would involve API versioning considerations, potential client breakage, and migration of any external consumers.
