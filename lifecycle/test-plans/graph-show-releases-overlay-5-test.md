---
title: 'Test Plan: Graph Releases Overlay'
type: plan-test
status: approved
lineage: graph-show-releases-overlay
parent: lifecycle/requirements/graph-show-releases-overlay-2.md
release: May2026-2
---

# Test Plan: Graph Releases Overlay

## Overview

Integration and end-to-end tests validating the releases overlay feature across the backend API and both graph views. Tests ensure correct data merging, timeline ordering, Backlog/Unscheduled semantics, visual rendering, toggle reactivity, accessibility, and performance.

Cross-links: [[graph-show-releases-overlay-3-be]] (backend milestones under test), [[graph-show-releases-overlay-4-fe]] (frontend milestones under test).

---

## Milestone 1 — Backend API tests for `include_releases` parameter

### Description

Test the `GET /api/p/:project/graph?include_releases=true` endpoint to verify that release nodes, timeline edges, and assignment edges are correctly merged into the standard graph response.

### Files to change

- `tests/graph_releases_test.go` (new file) — integration tests against the HTTP handler.

### Test cases

1. **Baseline: no parameter** — `GET /graph` returns no nodes with `type: "release"` and no edges with `kind: "timeline"` or `kind: "assigned"`.
2. **With parameter** — `GET /graph?include_releases=true` returns release nodes alongside artifact nodes.
3. **No duplicate nodes** — artifacts that appear in both the artifact graph and release assignments are not duplicated.
4. **Filter independence** — `GET /graph?include_releases=true&type=idea` returns only idea-type artifact nodes plus all release nodes (release nodes are not filtered by `type`).
5. **Empty releases** — when no releases exist, `include_releases=true` returns only the Backlog synthetic node (no timeline edges, Backlog has no outgoing timeline edges).

### Acceptance criteria

- All test cases pass.
- Tests run against a real SQLite database (not mocked) with seeded artifacts and releases.
- Tests complete in under 5 seconds.

---

## Milestone 2 — Backend tests for Backlog node semantics

### Description

Verify the "Backlog" synthetic node is correctly generated and connected.

### Files to change

- `tests/graph_releases_test.go` — additional test cases in the same file.

### Test cases

1. **Backlog present** — when at least one idea/defect has no `release` field, a node with `id: "release:backlog"` and `title: "Backlog"` exists.
2. **Backlog edges** — each unassigned idea/defect has an `assigned` edge from `release:backlog`.
3. **Backlog timeline position** — a `timeline` edge connects `release:backlog` to the earliest dated release.
4. **All assigned** — when every idea/defect has a release, the Backlog node is still present (anchors the chain) but has no `assigned` edges.
5. **No artifacts at all** — when no ideas/defects exist, the Backlog node is present with no edges.

### Acceptance criteria

- All test cases pass with deterministic seed data.

---

## Milestone 3 — Backend tests for Unscheduled node semantics

### Description

Verify the "Unscheduled" synthetic node only appears when undated releases exist and terminates the timeline chain.

### Files to change

- `tests/graph_releases_test.go` — additional test cases.

### Test cases

1. **Unscheduled present** — when at least one release has no `start_date`, a node with `id: "release:unscheduled"` exists.
2. **Unscheduled absent** — when all releases have `start_date`, no `release:unscheduled` node exists.
3. **Unscheduled terminus** — the `release:unscheduled` node has an incoming `timeline` edge from the last dated release but no outgoing `timeline` edge.
4. **Undated release nodes** — individual undated releases still appear as separate nodes with their real names, connected to `release:unscheduled` via `timeline` edges.

### Acceptance criteria

- All test cases pass with deterministic seed data.

---

## Milestone 4 — Backend tests for timeline ordering

### Description

Verify releases are chained in `start_date` ascending order.

### Files to change

- `tests/graph_releases_test.go` — additional test cases.

### Test cases

1. **Chronological order** — given releases with start dates 2026-01-01, 2026-03-01, 2026-02-01, timeline edges form: Backlog → Jan → Feb → Mar.
2. **Same-date stability** — releases with identical `start_date` are sorted by name (deterministic secondary sort).
3. **Single release** — one dated release produces: Backlog → Release (no Unscheduled node if no undated releases).

### Acceptance criteria

- All test cases pass; edge source/target pairs assert exact chronological ordering.

---

## Milestone 5 — Frontend toggle integration tests

### Description

Test the "Show Releases" toggle in both 2D and 3D graph views using end-to-end or component tests. Verify that toggling adds/removes release elements and that the default state is off.

### Files to change

- `tests/graph_releases_overlay_e2e_test.go` (new file) — if using Playwright/Cypress-style browser tests; otherwise add to the existing E2E test harness.
- Alternatively, `web/src/components/graph/__tests__/GraphFilters.spec.ts` (new file) — Vue component tests for the toggle.

### Test cases

1. **Default off** — on initial graph load, no release nodes are visible; the "Show Releases" checkbox is unchecked.
2. **Toggle on** — checking "Show Releases" causes release nodes to appear on the graph.
3. **Toggle off** — unchecking removes release nodes and edges without a full page reload.
4. **Persists in session** — navigating away from the graph view and back retains the toggle state.
5. **Keyboard accessible** — the toggle is focusable via Tab and operable via Space/Enter.

### Acceptance criteria

- All test cases pass in both 2D and 3D view modes.
- Toggle state does not persist across browser sessions (fresh load = off).

---

## Milestone 6 — Visual distinction and rendering tests

### Description

Validate that release nodes and overlay edges are visually distinct from artifact nodes and lineage edges.

### Files to change

- `tests/graph_releases_overlay_e2e_test.go` — additional visual assertions.

### Test cases

1. **Node colour** — release nodes use the light-blue colour (`#7dd3fc` or configured value) in both 2D and 3D.
2. **Node shape** — release nodes use a non-circle shape (diamond in 2D, octahedron in 3D), differentiating them from artifact circles/spheres.
3. **Timeline edge style** — timeline spine edges are dashed (2D) or visually distinct from lineage edges.
4. **Backlog and Unscheduled styling** — synthetic nodes use the same release styling (light blue, distinct shape).
5. **Legend updated** — the graph legend includes Release node and Timeline/Assigned edge entries.
6. **Dark theme** — release node colour is legible on dark backgrounds.

### Acceptance criteria

- Visual assertions pass (colour hex comparison or screenshot diffing depending on test framework).
- Release nodes are distinguishable from all existing node types without relying solely on colour.

---

## Milestone 7 — Performance tests

### Description

Verify that the release overlay does not degrade graph performance beyond acceptable thresholds.

### Files to change

- `tests/graph_releases_perf_test.go` (new file) — benchmarks for the backend endpoint.
- Manual or automated frontend performance measurement (document procedure if not automatable).

### Test cases

1. **Backend response time** — `GET /graph?include_releases=true` with 500 artifacts and 20 releases responds in under 500ms.
2. **Frontend frame rate (2D)** — Cytoscape graph with overlay enabled maintains ≥30 fps (manual test on mid-range hardware or automated via `requestAnimationFrame` timing).
3. **Frontend frame rate (3D)** — 3d-force-graph with overlay enabled has no visible stall during force simulation settling.
4. **Toggle latency** — toggling the overlay on/off completes within 200ms (no perceptible UI freeze).

### Acceptance criteria

- Backend benchmark passes with the defined thresholds.
- Frontend performance is verified and documented (screenshot or metric log).
- No regressions to baseline graph performance (without overlay) are introduced.
