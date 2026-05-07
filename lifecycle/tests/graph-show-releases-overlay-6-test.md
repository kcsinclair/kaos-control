---
title: "Integration Tests — Graph Releases Overlay"
type: test
status: draft
lineage: graph-show-releases-overlay
parent: lifecycle/test-plans/graph-show-releases-overlay-5-test.md
---

# Integration Tests — Graph Releases Overlay

Integration tests validating the releases overlay feature on the
`GET /api/p/:project/graph?include_releases=true` endpoint and the underlying
roadmap graph construction exposed by `GET /api/p/:project/releases/graph`.

All tests carry the `//go:build integration` tag. Run with:

```sh
go test -tags integration ./tests/... -v -run TestGraphReleases
```

---

## Test files

| File | Milestones covered |
|---|---|
| `tests/integration/graph_releases_test.go` | 1, 2, 3, 4 |
| `tests/integration/graph_releases_perf_test.go` | 7 (backend) |

---

## Scenarios covered

### Milestone 1 — `include_releases` parameter (`graph_releases_test.go`)

- **TestGraphReleases_BaselineNoParam** — `GET /graph` (no param) returns zero
  release-type nodes and no `timeline` or `assigned` edges.
- **TestGraphReleases_WithParam** — `GET /graph?include_releases=true` returns
  at least one release node and at least one timeline edge.
- **TestGraphReleases_NoDuplicateNodes** — an artifact assigned to a release
  appears exactly once; the overlay deduplication logic is exercised.
- **TestGraphReleases_FilterIndependence** — `?include_releases=true&type=idea`
  returns only `idea`-type artifact nodes plus all release nodes; non-idea
  artifact types are filtered out but release nodes are not.
- **TestGraphReleases_EmptyReleases** — with no releases, the response contains
  only the Backlog synthetic node and zero timeline edges.
- **TestGraphReleases_IncludeReleasesEdgeCountGrowth** — adding `include_releases=true`
  increases both the node count and the edge count relative to the baseline response.
- **TestGraphReleases_ReleaseNodeType** — release nodes (including Backlog) carry
  `type: "release"` in the overlay response.
- **TestGraphReleases_OverlayAssignedEdge** — artifact assigned to a release has
  an `assigned` edge from the release node in the overlay.
- **TestGraphReleases_BacklogAssignedEdgeInOverlay** — unassigned idea has an
  `assigned` edge from Backlog in the overlay.
- **TestGraphReleases_MultipleReleasesChainInOverlay** — timeline chain
  (`Backlog → v1 → v2`) is present in the overlay response.

### Milestone 2 — Backlog node semantics (`graph_releases_test.go`)

- **TestGraphReleases_BacklogPresent** — Backlog node has `id: "release:backlog"`,
  `title: "Backlog"`, `type: "release"`.
- **TestGraphReleases_BacklogEdges** — each unassigned idea/defect has an
  `assigned` edge from `release:backlog`.
- **TestGraphReleases_BacklogTimelinePosition** — Backlog connects directly to
  the earliest dated release via a timeline edge; does not skip to later releases.
- **TestGraphReleases_AllAssigned** — when all ideas/defects have a release, the
  Backlog node is still present but has zero outgoing `assigned` edges.
- **TestGraphReleases_NoArtifactsBacklog** — with no artifacts and no releases,
  only the Backlog node exists with no edges.

### Milestone 3 — Unscheduled node semantics (`graph_releases_test.go`)

- **TestGraphReleases_UnscheduledPresent** — `release:unscheduled` node exists
  when at least one release has no `start_date`; carries `title: "Unscheduled"`,
  `type: "release"`.
- **TestGraphReleases_UnscheduledAbsent** — `release:unscheduled` does not appear
  when all releases have a `start_date`.
- **TestGraphReleases_UnscheduledTerminus** — `release:unscheduled` has at least
  one incoming `timeline` edge and zero outgoing `timeline` edges.
- **TestGraphReleases_UndatedReleaseNodes** — each undated release appears as a
  distinct node (its own `release:N` ID and name) with a `timeline` edge pointing
  to `release:unscheduled`.

### Milestone 4 — Timeline ordering (`graph_releases_test.go`)

- **TestGraphReleases_ChronologicalOrder** — three releases created out of order
  form chain `Backlog → Jan → Feb → Mar` (start_date ascending).
- **TestGraphReleases_SameDateStability** — two same-date releases are sorted
  alphabetically by name as the secondary key.
- **TestGraphReleases_SingleReleaseNoUnscheduled** — a single dated release
  produces `Backlog → Release` with no Unscheduled node.

### Milestone 5 — Frontend toggle integration tests

**Not implemented.** No browser automation (Playwright/Cypress) or Vue component
testing (Vitest) infrastructure exists in this project. These tests require a
browser-level harness to exercise the "Show Releases" checkbox in the 2D and 3D
graph views. See `web/package.json` — no testing framework is listed.

### Milestone 6 — Visual distinction and rendering tests

**Not implemented.** Visual assertions (node colour `#7dd3fc`, diamond shape,
dashed edge style, legend entries) require either screenshot diffing or a DOM
inspection harness. Neither is available in the current Go integration test setup.

### Milestone 7 — Performance tests (`graph_releases_perf_test.go`)

- **TestGraphReleasesPerf_BackendResponseTime** — `GET /graph?include_releases=true`
  with 500 seeded artifacts (ideas + defects) and 20 created releases completes
  in under 500ms; response node count is validated.
- **TestGraphReleasesPerf_BaselineComparison** — both the baseline `/graph` and
  the overlay `/graph?include_releases=true` respond in under 500ms for 200
  artifacts and 10 releases, confirming no regression to the base graph path.

Frontend frame-rate and toggle-latency measurements (test plan §Milestone 7,
test cases 2–4) cannot be automated in the current setup; they require a browser
environment or manual measurement.

---

## Notes

- All backend tests use a real SQLite database via the standard `newTestEnv`
  harness — no mocking.
- Milestone 2–4 tests call `GET /releases/graph` via the existing `roadmapGraph`
  helper; Milestone 1 tests call `GET /graph?include_releases=true` via
  `graphWithReleases`.
- There is a known conflict with `TestReleaseUnscheduled_RoadmapGraphDisconnected`
  (in `releases_unscheduled_test.go`), which asserts that unscheduled releases
  have no timeline edges. The current implementation emits timeline edges from
  each undated release to `release:unscheduled`. The new Milestone 3 tests assert
  the correct implemented behaviour.
