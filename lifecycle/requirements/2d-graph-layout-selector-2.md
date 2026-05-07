---
title: 2D Graph Layout Selector
type: requirement
status: blocked
lineage: 2d-graph-layout-selector
created: "2026-05-07"
priority: normal
parent: lifecycle/ideas/2d-graph-layout-selector.md
labels:
    - frontend
    - feature
    - usability
    - vue
assignees:
    - role: product-owner
      who: agent
---

# 2D Graph Layout Selector

## Problem

The 2D graph view currently uses a single layout algorithm (fcose for undirected, breadthfirst for directed) with no user control. Different layout algorithms reveal different structural patterns — hierarchical layouts clarify lineage chains, force-directed layouts show clustering, and circular layouts expose orphaned nodes. Users exploring complex artifact graphs cannot switch perspectives without modifying code, limiting the graph's value as an analysis tool.

## Goals / Non-goals

### Goals

- Let users switch between layout algorithms from an in-graph control without page reload.
- Persist the selected layout within the browser session so it survives navigation away from and back to the graph view.
- Animate transitions between layouts so users can track node movement.
- Support at least fcose (current default), breadthfirst, concentric, and circle layouts out of the box.

### Non-goals

- Custom per-layout parameter tuning (e.g. adjusting edge length or spacing) — not in scope for this iteration.
- Persisting layout selection across browser sessions (localStorage) — session-only for now.
- Adding new Cytoscape plugins that require additional npm dependencies beyond what is already installed or available as Cytoscape built-ins, except dagre if included.
- Changing the 3D graph view — this requirement applies only to the 2D Cytoscape graph.

## Detailed Requirements

### Functional

1. **Layout selector control**: Add a dropdown (or equivalent compact control) to the `.view-controls` area in `GraphView.vue`, visible only when the 2D view is active. The control must list available layout algorithms by human-readable name.

2. **Supported layouts**: The selector must offer at minimum:
   | Key           | Display name    | Cytoscape `name` | Notes                          |
   |---------------|-----------------|-------------------|--------------------------------|
   | `fcose`       | Force-directed  | `fcose`           | Current default; requires plugin already imported |
   | `breadthfirst`| Hierarchical    | `breadthfirst`    | Built-in; use `directed: true` |
   | `concentric`  | Concentric      | `concentric`      | Built-in                       |
   | `circle`      | Circle          | `circle`          | Built-in                       |

   Additional layouts (e.g. dagre, grid, cose-bilkent) may be added later; the implementation should make adding a new entry straightforward (data-driven config, not scattered conditionals).

3. **Layout application**: When the user selects a layout, the graph must re-run `cy.layout(options).run()` with the chosen algorithm's options. The graph must not be destroyed and recreated.

4. **Animation**: Layout transitions should animate node positions (Cytoscape's `animate: true` with a reasonable duration, e.g. 300–500 ms) so users can follow where nodes move. If a layout algorithm does not support animation natively, snap without animation rather than breaking.

5. **Default layout**: fcose remains the default when the 2D view is first rendered.

6. **Session persistence**: Store the selected layout key in the Pinia graph store (or a dedicated composable). When the user navigates away from the graph view and returns within the same browser session, the previously selected layout must be restored automatically.

7. **Interaction with filters**: Changing filters (type, status, lineage, text search) triggers a graph update. The active layout algorithm must be used for re-layout after filter changes — not hard-coded to fcose.

8. **Interaction with directed prop**: If the `directed` prop is ever activated on `Graph2DView`, the selector should still be functional. The `directed` prop may set an initial layout but should not lock the selector.

### Non-functional

1. **Performance**: Layout computation for graphs up to 500 nodes must complete in under 2 seconds on a mid-range laptop.
2. **Bundle size**: Do not add new npm dependencies for built-in Cytoscape layouts. If dagre is added, it must be dynamically imported (code-split) like fcose is today.
3. **Accessibility**: The selector control must be keyboard-navigable and have appropriate `aria-label`.
4. **Visual consistency**: The selector must match the existing dark-theme styling used by the 3D/2D toggle and other `.view-controls` elements.

## Acceptance Criteria

- [ ] A layout selector control is visible in the 2D graph view toolbar area.
- [ ] The selector is hidden when the 3D view is active.
- [ ] Selecting a layout re-renders the graph with the chosen algorithm without page reload.
- [ ] At least four layout options are available: force-directed, hierarchical, concentric, circle.
- [ ] The default layout on first render is force-directed (fcose).
- [ ] Switching layouts animates node positions (where supported by the algorithm).
- [ ] The selected layout persists when navigating away from and back to the graph view within the same session.
- [ ] Applying a filter re-layouts using the currently selected algorithm, not the hard-coded default.
- [ ] The selector is keyboard-accessible.
- [ ] The selector visually matches the existing graph toolbar controls (dark theme, same font/spacing).
- [ ] No new npm dependencies are required for built-in Cytoscape layouts.
- [ ] Graph with 200+ nodes re-layouts in under 2 seconds.

## Open Questions

1. Should the selector also appear in a future mobile/narrow viewport, or is it acceptable to hide it below a breakpoint?
2. Is dagre (hierarchical with edge routing) desired as a fifth option? It would require adding the `cytoscape-dagre` npm package.
3. Should the "directed" toggle (currently a prop, not exposed in UI) be surfaced alongside or merged with the layout selector?
