---
title: Integration Tests — Directed Release Chain Graph
type: test
status: done
lineage: roadmap-graph-release-connections
parent: lifecycle/test-plans/roadmap-graph-release-connections-5-test.md
---

# Integration Tests — Directed Release Chain Graph

Integration tests for the `GET /api/p/:project/releases/graph` endpoint: directed chain construction, edge duration labels, artifact assignment, regression, and performance.

## Test files

All tests carry the `//go:build integration` tag. Run the full suite with:

```sh
go test -tags integration ./tests/... -v -run TestRoadmapGraph
```

## Scenarios covered

### Milestone 1 — Chain construction logic (`tests/integration/releases_graph_test.go`)

Run with: `go test -tags integration ./tests/... -run TestRoadmapGraph_EmptyState\|TestRoadmapGraph_Single\|TestRoadmapGraph_Multiple\|TestRoadmapGraph_TieBreaking\|TestRoadmapGraph_Mixed\|TestRoadmapGraph_Backlog\|TestRoadmapGraph_NoScheduled`

- **Empty state** — no releases → exactly 1 node (`release:backlog`, `synthetic: true`), 0 edges.
- **Backlog always present** — Backlog node with `title: "Backlog"`, `type: "release"`, `synthetic: true` always exists.
- **Single scheduled** — one scheduled release → chain `[Backlog, R1]`, exactly one timeline edge.
- **Multiple scheduled chronological** — three releases created out of order; chain is `Backlog → R1(Jan) → R2(Apr) → R3(Jul)`.
- **Tie-breaking alphabetical** — two releases with same `start_date`; alphabetically-first name precedes the other.
- **Single unscheduled** — one unscheduled release → `Backlog → unsched` timeline edge.
- **Multiple unscheduled alphabetical** — two unscheduled releases created in reverse alphabetical order; chain is alphabetical.
- **No scheduled, direct to unscheduled** — three unscheduled releases → `Backlog → aaa → bbb → ccc`.
- **Mixed scheduled + unscheduled** — two scheduled, two unscheduled → `Backlog → S1 → S2 → UA → UZ`.
- **Timeline edge count** — 3 releases → exactly 3 timeline edges.

### Milestone 2 — Edge duration labels (`tests/integration/releases_graph_test.go`)

Run with: `go test -tags integration ./tests/... -run TestRoadmapGraph_BacklogEdge\|TestRoadmapGraph_Unscheduled\|TestRoadmapGraph_EdgeLabel`

- **Backlog edge no label** — `Backlog → first scheduled` edge has empty `label`.
- **Unscheduled edges no label** — `scheduled → unscheduled` and `unscheduled → unscheduled` edges have empty `label`.
- **1-day gap** → `"1 day"`.
- **7-day gap** → `"1 week"`.
- **14-day gap** → `"2 weeks"`.
- **30-day gap** → `"4 weeks"` (30 days / 7 = 4 weeks, below the 5-week→months threshold).
- **35-day gap** → `"1 month"` (35 days → 5 weeks, crosses into months).
- **390-day gap** → `"1 year"`.

### Milestone 3 — Artifact assignment (`tests/integration/releases_graph_test.go`)

Run with: `go test -tags integration ./tests/... -run TestRoadmapGraph_Artifact\|TestRoadmapGraph_Plans\|TestRoadmapGraph_DependsOn`

- **Artifact assigned to release** — idea with `release: v1.0` has `assigned` edge from the `release:N` node.
- **Artifact unassigned from Backlog** — idea with no `release` field has `assigned` edge from `release:backlog`.
- **Artifact node fields** — artifact nodes carry `id`, `title`, `type`, and `status`.
- **Plans excluded** — `plan-backend`, `plan-frontend`, `plan-test`, `plan-dev` artifacts are not included as nodes.
- **depends_on edges preserved** — a `depends_on` relationship between two included artifacts appears as an edge with `kind: "depends_on"`.

### Milestone 4 & 5 — Frontend rendering and click interactions

Not implemented as part of this test suite. No Vitest/browser automation infrastructure exists in the project. These milestones remain open for future UI testing work.

### Milestone 6 — Performance (`tests/integration/releases_graph_perf_test.go`)

Run with: `go test -tags integration ./tests/... -run TestRoadmapGraph_Perf`

- **50 releases, no artifacts** — `GET /releases/graph` responds in < 100ms; at least 51 nodes and 50 timeline edges present.
- **50 releases + 100 artifacts** — response in < 200ms; at least 151 nodes (1 Backlog + 50 releases + 100 artifacts).
- **20 releases, response shape** — every node has a non-empty `id`; every timeline edge has non-empty `source`, `target`, and `kind`.

### Milestone 7 — Edge cases and regression (`tests/integration/releases_graph_test.go`)

Run with: `go test -tags integration ./tests/... -run TestRoadmapGraph_Delete\|TestRoadmapGraph_Insert\|TestRoadmapGraph_Rename\|TestRoadmapGraph_Main`

- **Delete only scheduled updates chain** — after deleting the sole scheduled release, chain becomes `Backlog → unscheduled`.
- **Insert release in middle chain** — creating R2 (Apr) after R1 (Jan) and R3 (Jul) already exist places R2 between them on next fetch.
- **Rename updates node label** — after renaming a release, the node's `title` reflects the new name on next fetch.
- **Main graph unaffected** — `GET /graph` (main artifact graph) still returns artifact nodes correctly when releases exist.

## WebSocket re-render

WebSocket events (`release.created`, `release.updated`, `release.deleted`) that would trigger a frontend graph re-render are already covered by `TestReleaseWebSocket_*` in `tests/integration/releases_ws_test.go`. The backend correctly broadcasts these events; verifying that the frontend re-fetches the chain on receipt is a frontend integration concern (Milestone 4/5 gap above).

## Known conflict with existing test

`TestReleaseUnscheduled_RoadmapGraphDisconnected` (in `releases_unscheduled_test.go`) asserts that unscheduled releases have **no** timeline edges. This contradicts the backend plan (Milestone 3) and the implementation in `internal/http/releases.go`, which **does** emit timeline edges for unscheduled releases. The new tests in this suite assert the correct behaviour per the spec. The existing test should be updated to reflect the chain-connected design.
