---
title: 'Frontend Plan: Directed Release Chain Graph Rendering'
type: plan-frontend
status: approved
lineage: roadmap-graph-release-connections
created: "2026-05-07"
priority: high
parent: lifecycle/requirements/roadmap-graph-release-connections-2.md
release: May2026
---

# Frontend Plan: Directed Release Chain Graph Rendering

Implement the visual rendering of the directed release chain in both 2D (Cytoscape.js) and 3D (3d-force-graph) graph views, including node styling, directional arrows, edge labels, and click-to-modal interactions.

## Milestone 1: Graph Constants and Node Styling

**Description:** Add colour and styling constants for release nodes and the Backlog synthetic node. Releases should be light blue; the Backlog node should be visually distinct.

**Files to change:**
- `web/src/components/graph/graphConstants.ts` — add `release: '#93c5fd'` (light blue) to `NODE_COLORS`, add `backlog: '#6b7280'` (gray) or a distinct colour for the Backlog node.

**Acceptance criteria:**
- Release nodes render in light blue (`#93c5fd` or similar).
- The Backlog node renders in a distinct colour (e.g., gray or slate) differentiating it from regular releases.
- Constants are exported and available to both 2D and 3D graph components.

## Milestone 2: 3D Graph — Release Node Geometry

**Description:** Render release nodes as rounded cube shapes (BoxGeometry with rounded edges) in the 3D view. The Backlog node uses a different shape (e.g., sphere or octahedron) for visual distinction.

**Files to change:**
- `web/src/components/graph/ForceGraph3D.vue` — extend `nodeThreeObject` to handle `type: "release"` nodes with `THREE.BoxGeometry` (or `RoundedBoxGeometry` from three examples) in light blue, and `synthetic: true` nodes with distinct geometry.

**Acceptance criteria:**
- Release nodes appear as rounded cubes (or box shapes if rounded is not feasible) in 3D view.
- The Backlog node appears with distinct geometry (sphere or similar).
- Node labels display the release name.
- Nodes are sized consistently with other graph nodes.

## Milestone 3: 2D Graph — Release Node Shape

**Description:** Render release nodes with a distinct shape in Cytoscape.js (e.g., `round-rectangle` for releases, `diamond` or `ellipse` for Backlog).

**Files to change:**
- `web/src/components/graph/Graph2DView.vue` — add Cytoscape style selectors for `[type="release"]` using `round-rectangle` shape with light blue background, and for `[synthetic="true"]` using distinct shape/colour.

**Acceptance criteria:**
- Release nodes display as rounded rectangles in the 2D graph.
- The Backlog node displays with a visually distinct shape.
- Labels are readable and do not overlap at default zoom for up to 20 nodes.

## Milestone 4: Directional Arrows on Edges

**Description:** Ensure all timeline edges display arrowheads indicating direction (earlier → later).

**Files to change:**
- `web/src/components/graph/Graph2DView.vue` — set `target-arrow-shape: 'triangle'` on edges with `kind: "timeline"`.
- `web/src/components/graph/ForceGraph3D.vue` — verify `linkDirectionalArrowLength` is set for timeline edges (may already be configured for all edges).

**Acceptance criteria:**
- All timeline/chain edges display a visible arrowhead at the target end.
- Arrow direction is from earlier (source) to later (target) in the chain.
- Arrows are visible in both 2D and 3D views.

## Milestone 5: Edge Duration Labels (Tooltips)

**Description:** Display the time-gap label on timeline edges between scheduled releases. In 2D, show as edge label text. In 3D, show as tooltip on hover.

**Files to change:**
- `web/src/components/graph/Graph2DView.vue` — add edge label styling for `kind: "timeline"` edges that have a `label` field.
- `web/src/components/graph/ForceGraph3D.vue` — add hover tooltip or mid-link label for timeline edges with duration metadata.

**Acceptance criteria:**
- Duration labels (e.g., "2 weeks") are visible on edges between scheduled releases.
- Labels do not clutter the graph — font size is smaller than node labels.
- Edges from/to Backlog or between unscheduled releases show no duration label.

## Milestone 6: Click-to-Modal Interactions

**Description:** Clicking a release node opens the ReleaseDetailModal. Clicking an idea or defect node opens the existing artifact modal used in the main graph view.

**Files to change:**
- `web/src/components/releases/RoadmapGraphView.vue` — handle `nodeClick` event: if node type is `"release"`, emit event to open ReleaseDetailModal; if node is an artifact, open the artifact detail modal.
- `web/src/views/project/RoadmapView.vue` — wire `nodeClick` from RoadmapGraphView to open the appropriate modal (ReleaseDetailModal for releases, existing artifact modal for ideas/defects).

**Acceptance criteria:**
- Clicking a release node opens ReleaseDetailModal with that release's data.
- Clicking an idea or defect artifact node opens the same modal as used in the main graph view.
- Clicking the Backlog synthetic node does nothing (or shows a simple info tooltip).
- Modals can be dismissed and graph interaction resumes.

## Milestone 7: Layout and Performance

**Description:** Ensure the directed chain layout is readable and performant. For 2D, configure fcose layout to respect the chain ordering (left-to-right or top-to-bottom flow). For 3D, ensure force-directed simulation settles with the chain visually linear.

**Files to change:**
- `web/src/components/graph/Graph2DView.vue` — adjust fcose layout options or switch to `dagre`/`klay` layout for the roadmap graph to enforce hierarchical left-to-right flow.
- `web/src/components/graph/ForceGraph3D.vue` — optionally set `dagMode('lr')` or similar to linearise the chain in 3D.

**Acceptance criteria:**
- The release chain reads left-to-right (or top-to-bottom) with Backlog on one end and unscheduled leaves on the other.
- No label overlap with up to 20 release nodes at default zoom.
- Graph renders in under 200ms for 50 release nodes (no perceptible delay).
- The layout is stable — nodes do not jump or rearrange on re-render.

## Cross-references

- [[roadmap-graph-release-connections]] backend plan provides the graph data structure (nodes, edges, edge labels, synthetic flags).
- [[roadmap-graph-release-connections]] test plan covers visual regression and interaction testing.
