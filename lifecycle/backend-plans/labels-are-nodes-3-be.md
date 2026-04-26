---
title: 'Backend Plan: Labels as Graph Nodes with Priority Visualisation'
type: plan-backend
status: done
lineage: labels-are-nodes
parent: requirements/labels-are-nodes-2.md
---

# Backend Plan: Labels as Graph Nodes with Priority Visualisation

This plan covers the backend changes needed to support [[labels-are-nodes]]. The requirement is predominantly frontend, but the backend must supply a PATCH-style priority update endpoint and ensure the graph API returns all data the frontend needs. The existing graph API already returns `labels` and `priority` on `GraphNode`, and the `PUT /artifacts/:path` endpoint already supports full frontmatter updates, so backend work is minimal.

## Milestone 1: Add Priority-Only PATCH Endpoint

### Description

Add a lightweight `PATCH /api/p/:project/artifacts/:path/priority` endpoint that accepts `{ "priority": "<value>" }` and updates only the `priority` field in the artifact's YAML frontmatter without requiring the client to send the full frontmatter and body. This supports FR-3.3's inline priority edit from the modal without the overhead and conflict risk of a full PUT.

### Files to Change

- `internal/http/server.go` — register the new PATCH route
- `internal/http/write.go` — add `handlePatchPriority` handler

### Implementation Details

1. The handler reads the artifact file from disk via `sandbox.Resolve`.
2. Parses frontmatter using `artifact.Parse`.
3. Validates the incoming priority value against an allowed set: `high`, `medium`, `normal`, `low`, `""` (unset).
4. Overwrites only the `priority` field in the parsed frontmatter.
5. Re-serialises frontmatter + body and writes back to disk.
6. Triggers re-index via `p.Idx.IndexFile(path)`.
7. Broadcasts `artifact.indexed` and `file.changed` WebSocket events via the hub.
8. Returns `200` with the updated `ArtifactRow`.

### Acceptance Criteria

- [ ] `PATCH /api/p/:project/artifacts/:path/priority` with `{"priority":"high"}` updates only the priority field on disk.
- [ ] Invalid priority values (e.g. `"critical"`) return `400 bad_request`.
- [ ] The artifact body and all other frontmatter fields are unchanged after the PATCH.
- [ ] A WebSocket `artifact.indexed` event is broadcast after a successful update.
- [ ] The endpoint respects the existing auth/session middleware.

## Milestone 2: Extend Graph API Edge Kinds

### Description

The graph API currently builds edges from `parent`, `depends_on`, `blocks`, and `related_to` frontmatter fields. The frontend plan ([[labels-are-nodes]]) will synthesise label nodes client-side, but the backend must ensure the `labels` array is always present (even if empty) on every graph node so the frontend can reliably iterate without null checks.

### Files to Change

- `internal/index/graph.go` — ensure `Labels` field defaults to `[]` rather than `nil` in the graph node struct

### Implementation Details

1. In the `Graph()` function, after building the node list, replace any `nil` labels slice with an empty slice `[]string{}`.
2. This is a defensive normalisation — the existing code may already do this via the SQL scan, but it must be guaranteed.

### Acceptance Criteria

- [ ] `GET /api/p/:project/graph` returns `"labels": []` (not `null`) for artifacts with no labels.
- [ ] Artifacts with labels still return `"labels": ["foo", "bar"]` as before.
- [ ] No regression in existing graph endpoint behaviour.

## Milestone 3: Validate Priority Values on Full PUT

### Description

The existing `PUT /artifacts/:path` endpoint accepts any string for `priority`. Add validation to reject values outside the allowed set (`high`, `medium`, `normal`, `low`, `""`) so the frontend colour mapping is always deterministic.

### Files to Change

- `internal/http/write.go` — add priority validation in `handleUpdateArtifact`

### Implementation Details

1. After decoding the request body, check `req.Frontmatter.Priority` against the allowed set.
2. If invalid, return `400 bad_request` with a clear message listing allowed values.
3. Empty string or omitted priority is valid (means "unset").

### Acceptance Criteria

- [ ] `PUT` with `priority: "critical"` returns 400.
- [ ] `PUT` with `priority: "high"` succeeds.
- [ ] `PUT` with `priority: ""` or omitted priority succeeds.
- [ ] Existing tests continue to pass (`make test-unit`).
