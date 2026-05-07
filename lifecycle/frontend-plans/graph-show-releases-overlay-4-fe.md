---
title: 'Frontend Plan: Graph Releases Overlay'
type: plan-frontend
status: done
lineage: graph-show-releases-overlay
parent: lifecycle/requirements/graph-show-releases-overlay-2.md
release: May2026
---

# Frontend Plan: Graph Releases Overlay

## Overview

Add a "Show Releases" toggle to the graph toolbar that overlays release nodes, a chronological timeline spine, and release-to-artifact assignment edges onto both the 2D (Cytoscape.js) and 3D (3d-force-graph) graph views. The overlay is opt-in (off by default), reactive (no full re-render on toggle), and visually distinct (light blue nodes per the resolved colour question).

Cross-links: [[graph-show-releases-overlay-3-be]] (backend supplies merged data via `?include_releases=true`), [[graph-show-releases-overlay-5-test]] (integration tests validate overlay behaviour).

---

## Milestone 1 — Add release colour and edge kind to graph constants

### Description

Register the `release` node type colour (light blue) and new edge kinds (`timeline`, `assigned`) in the shared constants file so both graph views render them consistently.

### Files to change

- `web/src/components/graph/graphConstants.ts`:
  - Add `release: '#7dd3fc'` (sky-300, light blue) to `NODE_COLORS`.
  - Add `timeline: '#7dd3fc'` and `assigned: '#7dd3fc'` to `EDGE_COLORS` (matching release nodes for visual grouping).

### Acceptance criteria

- `NODE_COLORS.release` resolves to a light-blue hex value.
- `EDGE_COLORS.timeline` and `EDGE_COLORS.assigned` resolve to matching hex values.
- Existing node/edge colours are unchanged.

---

## Milestone 2 — Extend the graph Pinia store with release overlay state

### Description

Add state and logic to the graph store to manage the release overlay toggle, fetch release-augmented graph data, and compute overlay nodes/edges separately so they can be added/removed without re-fetching.

### Files to change

- `web/src/stores/graph.ts`:
  - Add state field: `showReleases: boolean` (default `false`).
  - Add action: `toggleShowReleases()` — flips `showReleases`; if transitioning to `true` and release data hasn't been fetched yet, calls `fetchGraph` with `include_releases=true`.
  - Add computed: `releaseNodes` — filters `rawNodes` where `type === 'release'`.
  - Add computed: `releaseEdges` — filters `rawEdges` where `kind === 'timeline' || kind === 'assigned'`.
  - Modify `augmentedNodes` computed to include `releaseNodes` when `showReleases` is true.
  - Modify `augmentedEdges` computed to include `releaseEdges` when `showReleases` is true.
  - Ensure `filteredNodes` does NOT filter out release nodes by type/status/lineage filters (release nodes are always shown when overlay is on).

- `web/src/api/graph.ts`:
  - Modify `getGraph()` to accept an optional `includeReleases?: boolean` parameter and append `?include_releases=true` to the request URL when set.

### Acceptance criteria

- `showReleases` defaults to `false`.
- Toggling `showReleases` to `true` causes release nodes and edges to appear in `augmentedNodes`/`augmentedEdges`.
- Toggling back to `false` removes them.
- Release nodes are not filtered out by type/status chip filters.
- `showReleases` state persists across in-session navigation (Pinia store lifetime).

---

## Milestone 3 — Add "Show Releases" toggle to GraphFilters

### Description

Add a checkbox toggle to the graph toolbar (GraphFilters component) labelled "Show Releases", consistent with the existing "Show label nodes", "Show completed", "Show tests" toggles. Must be keyboard-accessible with an accessible label.

### Files to change

- `web/src/components/graph/GraphFilters.vue`:
  - Add a new checkbox/toggle: label "Show Releases", bound to a new `showReleases` prop and emitting `toggleShowReleases` event.
  - Position it in the toggle group alongside existing toggles.
  - Ensure the checkbox has `id` and `<label for>` attributes, plus `aria-label` for screen readers.

- `web/src/views/project/GraphView.vue`:
  - Wire the new `toggleShowReleases` emit from GraphFilters to `graphStore.toggleShowReleases()`.
  - Pass `graphStore.showReleases` as a prop to GraphFilters.

### Acceptance criteria

- A "Show Releases" checkbox is visible in the graph toolbar area.
- Default state is unchecked.
- Toggling the checkbox calls `toggleShowReleases()` on the store.
- The checkbox is keyboard-focusable and operable (Enter/Space).
- The checkbox has an accessible label readable by screen readers.

---

## Milestone 4 — Render release overlay in 2D graph (Cytoscape.js)

### Description

Update the 2D graph component to render release nodes with distinct visual styling and handle incremental add/remove when the overlay is toggled, avoiding a full graph re-render.

### Files to change

- `web/src/components/graph/Graph2DView.vue`:
  - In `buildElements()`: release nodes get a distinct shape (e.g., `diamond` or `round-diamond`) to differentiate from artifact circles, using `NODE_COLORS.release` as background.
  - Timeline spine edges get a dashed line style (`line-style: 'dashed'`) and `EDGE_COLORS.timeline` colour.
  - Assignment edges use `EDGE_COLORS.assigned` colour with a lighter weight (1px).
  - Watch for changes in the nodes/edges props to incrementally add/remove release elements using Cytoscape's `cy.add()` and `cy.remove()` rather than full `init()` re-call.

### Acceptance criteria

- Release nodes render as diamonds (or another non-circle shape) in light blue.
- Timeline edges render as dashed lines connecting releases chronologically.
- Assignment edges render as solid light-blue lines from release nodes to artifacts.
- "Backlog" and "Unscheduled" synthetic nodes render with the same style as regular release nodes.
- Toggling the overlay on adds elements incrementally (no full layout re-run).
- Toggling off removes release elements incrementally.
- Node labels (release title) are visible on release nodes.

---

## Milestone 5 — Render release overlay in 3D graph (3d-force-graph)

### Description

Update the 3D graph component to render release nodes with a distinct Three.js geometry (e.g., octahedron or box instead of sphere) and light-blue colour, and handle incremental data updates.

### Files to change

- `web/src/components/graph/ForceGraph3D.vue`:
  - In `buildNodeObject()`: detect `type === 'release'` and use `THREE.OctahedronGeometry` (or `BoxGeometry`) instead of `SphereGeometry`, with `NODE_COLORS.release` material colour.
  - Timeline spine links: use dashed line material or a distinct colour matching `EDGE_COLORS.timeline`.
  - Assignment links: use `EDGE_COLORS.assigned`.
  - Ensure `graphData()` is called with the updated nodes/edges when the overlay is toggled, using the force-graph's `graphData()` method for incremental update (it re-runs the force simulation only for new nodes).

### Acceptance criteria

- Release nodes render as octahedra (or another non-sphere geometry) in light blue.
- Timeline edges are visually distinct from lineage/dependency edges.
- Assignment edges are visually distinct and use light-blue colouring.
- "Backlog" and "Unscheduled" nodes render identically to regular release nodes.
- Toggling the overlay updates the graph data without destroying and re-creating the entire scene.
- Release node labels (title text) are visible.
- No perceptible frame-rate drop with ≤20 release nodes on a graph of ≤500 artifacts.

---

## Milestone 6 — Update GraphLegend

### Description

Add a "Release" entry to the graph legend so users understand the new node shape and colour.

### Files to change

- `web/src/components/graph/GraphLegend.vue`:
  - Add a "Release" entry with the light-blue colour swatch and diamond/octahedron shape indicator.
  - Add "Timeline" and "Assigned" edge entries with their respective styles.

### Acceptance criteria

- The legend shows a "Release" node entry with light-blue colour.
- The legend shows "Timeline" and "Assigned" edge kinds.
- Legend entries only appear when the releases overlay is enabled (or always — follow existing legend behaviour for label nodes).
