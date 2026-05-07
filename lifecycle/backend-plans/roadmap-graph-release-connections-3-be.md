---
title: 'Backend Plan: Directed Release Chain Graph Endpoint'
type: plan-backend
status: in-development
lineage: roadmap-graph-release-connections
created: "2026-05-07"
priority: high
parent: lifecycle/requirements/roadmap-graph-release-connections-2.md
release: May2026
---

# Backend Plan: Directed Release Chain Graph Endpoint

Refactor the `/api/p/:project/releases/graph` endpoint to produce a directed chain of release nodes with a synthetic Backlog root, chronologically ordered scheduled releases, and alphabetically ordered unscheduled terminal leaves.

## Milestone 1: Synthetic Backlog Node

**Description:** Add a synthetic "Backlog" node as the root of the roadmap graph. This replaces any current "Undefined" label logic.

**Files to change:**
- `internal/http/releases.go` — modify `handleRoadmapGraph` to inject a Backlog node with id `"release:backlog"`, title `"Backlog"`, type `"release"`.

**Acceptance criteria:**
- The graph response always includes a node with `id: "release:backlog"` and `title: "Backlog"`.
- The Backlog node is present even when no releases exist (empty state).
- The Backlog node has a distinguishing field (e.g., `synthetic: true`) so the frontend can style it differently.

## Milestone 2: Chronological Directed Chain for Scheduled Releases

**Description:** Sort all releases with a `start_date` in ascending order (ties broken alphabetically by name). Emit directed edges forming a chain: Backlog → first scheduled → second scheduled → … → last scheduled.

**Files to change:**
- `internal/http/releases.go` — replace existing timeline edge logic in `handleRoadmapGraph` with chain construction.
- `internal/release/store.go` — ensure `List()` query supports ordering by `start_date ASC, name ASC` (verify existing behaviour or add sort parameter).

**Acceptance criteria:**
- Directed edges with `kind: "timeline"` connect each release to its successor in chronological order.
- A directed edge connects the Backlog node to the earliest scheduled release.
- When two releases share the same `start_date`, the one alphabetically first precedes the other.
- Each edge includes a `label` field containing the human-readable duration between the two releases' `start_date` values (e.g., "2 weeks", "3 months"). The Backlog→first edge has no duration label.

## Milestone 3: Unscheduled Release Terminal Leaves

**Description:** Releases without a `start_date` appear after all scheduled releases, sorted alphabetically, connected by directed edges.

**Files to change:**
- `internal/http/releases.go` — extend `handleRoadmapGraph` to partition releases into scheduled/unscheduled, sort unscheduled alphabetically, and append chain edges.

**Acceptance criteria:**
- Unscheduled releases are sorted alphabetically by name.
- A directed edge connects the last scheduled release to the first unscheduled release.
- Directed edges connect consecutive unscheduled releases in alphabetical order.
- If no scheduled releases exist, the Backlog node connects directly to the first unscheduled release.
- If only one unscheduled release exists, it is a single terminal leaf connected from the last scheduled release (or Backlog if none scheduled).

## Milestone 4: Edge Metadata — Duration Labels

**Description:** Each timeline edge between scheduled releases carries a `label` field showing the time gap between their `start_date` values.

**Files to change:**
- `internal/http/releases.go` — compute duration between consecutive `start_date` values and attach as edge metadata.

**Acceptance criteria:**
- Edges between scheduled releases include `label` with a human-readable duration string.
- Duration uses the largest appropriate unit: days (< 8), weeks (< 5), months (< 13), years.
- Edges involving the Backlog node or unscheduled releases have no duration label (empty string or omitted).

## Milestone 5: Artifact Assignment Edges and Click Support

**Description:** Maintain existing assignment edges from release nodes to their artifacts (ideas, defects). Ensure the Backlog node also has assignment edges to artifacts with no release.

**Files to change:**
- `internal/http/releases.go` — query unassigned artifacts via `Filter.Release = "__unassigned__"` and emit `assigned` edges from the Backlog node to each.
- `internal/index/index.go` — verify `"__unassigned__"` filter works correctly (already exists per codebase).

**Acceptance criteria:**
- Artifacts assigned to a release have `kind: "assigned"` edges from that release node.
- Artifacts with no release assignment have `kind: "assigned"` edges from the Backlog node.
- Each artifact node includes sufficient data (`id`, `title`, `type`, `status`, `lineage`) for the frontend to display its modal on click.

## Cross-references

- [[roadmap-graph-release-connections]] frontend plan handles node styling (light blue rounded cubes for releases, Backlog differentiation) and click-to-modal behaviour.
- [[roadmap-graph-release-connections]] test plan validates the endpoint response structure and edge ordering.
