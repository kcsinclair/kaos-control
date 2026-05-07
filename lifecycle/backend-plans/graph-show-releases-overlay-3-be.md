---
title: 'Backend Plan: Graph Releases Overlay'
type: plan-backend
status: done
lineage: graph-show-releases-overlay
parent: lifecycle/requirements/graph-show-releases-overlay-2.md
release: May2026
---

# Backend Plan: Graph Releases Overlay

## Overview

The existing `/api/p/:project/releases/graph` endpoint already produces release nodes, timeline spine edges, and release-to-artifact assignment edges. The main graph endpoint (`/api/p/:project/graph`) does not include release data. This plan adds an opt-in mechanism to merge release overlay data into the main graph response so the frontend can toggle releases without a separate fetch, and ensures the roadmap graph logic fully satisfies the requirement's Backlog/Unscheduled semantics.

Cross-links: [[graph-show-releases-overlay-4-fe]] (frontend consumes the merged data), [[graph-show-releases-overlay-5-test]] (integration tests validate the endpoint).

---

## Milestone 1 — Add `include_releases` query parameter to the main graph endpoint

### Description

Extend `GET /api/p/:project/graph` to accept an optional `include_releases=true` query parameter. When set, the handler merges the roadmap graph data (release nodes, timeline edges, assignment edges) into the standard graph response. This allows the frontend to fetch a single dataset that contains both the artifact graph and the release overlay.

### Files to change

- `internal/http/graph.go` — `handleGraph()`: read `include_releases` query param; when truthy, call the existing roadmap-graph builder and merge its nodes/edges into the response.
- `internal/http/releases.go` — extract the roadmap-graph building logic from `handleRoadmapGraph()` into a reusable function (e.g., `buildRoadmapGraph(project *project.Project) (*index.GraphData, error)`) so both endpoints can call it without duplication.

### Acceptance criteria

- `GET /graph` without `include_releases` returns the same response as before (no release nodes or timeline edges).
- `GET /graph?include_releases=true` returns the artifact graph plus release nodes (`type: "release"`), timeline edges (`kind: "timeline"`), and assignment edges (`kind: "assigned"`).
- No duplicate nodes when an artifact appears in both the artifact graph and the release overlay.
- Existing query-parameter filters (`stage`, `status`, `label`, `lineage`, `type`) still apply to artifact nodes; release nodes are always included when `include_releases=true`.

---

## Milestone 2 — Backlog synthetic node

### Description

When `include_releases=true`, the response must include a synthetic "Backlog" node representing all ideas and defects that have no `release` field in their frontmatter. This node anchors the start of the timeline spine. Edges of kind `assigned` connect it to each unassigned idea/defect.

### Files to change

- `internal/http/releases.go` (the shared `buildRoadmapGraph` function) — add logic to:
  1. Query ideas/defects where `release` is empty/null (use the existing `Filter.Release = "__unassigned__"` sentinel).
  2. Create a synthetic `GraphNode` with `id: "release:backlog"`, `title: "Backlog"`, `type: "release"`, `status: "planned"`.
  3. Add `assigned` edges from `release:backlog` to each matched artifact.
  4. Insert a `timeline` edge from `release:backlog` to the first chronologically-sorted release node.

### Acceptance criteria

- A "Backlog" node is present when at least one idea or defect has no `release` field.
- The "Backlog" node is the first node in the timeline chain (a `timeline` edge connects it to the earliest-dated release).
- Each unassigned idea/defect has an `assigned` edge from `release:backlog`.
- If no unassigned artifacts exist, the "Backlog" node is still present as the chain anchor (per requirement §3: "anchors the start").

---

## Milestone 3 — Unscheduled synthetic node

### Description

Add an "Unscheduled" terminus node for releases that have no `start_date`. Per the resolved question in the requirement, this node should only appear if there are releases without a start date. These releases are not necessarily called "Unscheduled" — they are real release entities that lack scheduling.

### Files to change

- `internal/http/releases.go` (the shared `buildRoadmapGraph` function) — add logic to:
  1. Identify releases with no `start_date` (already computed as "unscheduled" in the releases store).
  2. Create a synthetic `GraphNode` with `id: "release:unscheduled"`, `title: "Unscheduled"`, `type: "release"`, `status: "planned"`.
  3. Connect each undated release to the "Unscheduled" group node via `timeline` edges.
  4. Add a `timeline` edge from the last dated release to `release:unscheduled`.
  5. Aggregate assignment edges: artifacts assigned to any undated release connect to the undated release node (not the synthetic Unscheduled node).
  6. Only emit this node when at least one undated release exists.

### Acceptance criteria

- The "Unscheduled" synthetic node appears only when at least one release lacks a `start_date`.
- Undated release nodes still appear individually with their real names; the "Unscheduled" node groups them at the end of the spine.
- The timeline chain terminates at the "Unscheduled" node when it is present; otherwise it terminates at the last dated release.
- Artifact assignments to undated releases resolve to the individual undated release nodes (not collapsed into "Unscheduled").

---

## Milestone 4 — Ensure timeline ordering uses `start_date`

### Description

Per the resolved requirement question, release chronological order must be determined by the `start_date` field. Verify and, if necessary, correct the `handleRoadmapGraph` / `buildRoadmapGraph` sorting logic.

### Files to change

- `internal/http/releases.go` — confirm the sort in timeline-edge generation uses `start_date` (not `created_at`). If the existing code already does this, no change is needed; document the verification.

### Acceptance criteria

- Releases in the timeline spine are ordered by `start_date` ascending.
- Releases with the same `start_date` have a stable secondary sort (e.g., by name).
- Releases without `start_date` are excluded from the dated spine and attached to the "Unscheduled" terminus instead.
