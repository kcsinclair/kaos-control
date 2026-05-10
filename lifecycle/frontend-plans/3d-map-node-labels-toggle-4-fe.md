---
title: 'Frontend Plan: 3D Map Node Labels Toggle'
type: plan-frontend
status: in-development
lineage: 3d-map-node-labels-toggle
parent: lifecycle/requirements/3d-map-node-labels-toggle-2.md
---

## Summary

Add two independent checkboxes — "Show node titles" and "Show node lineage" — to both 3D views (map and roadmap). When enabled, render text sprite labels on non-release, non-label nodes using the existing `textSprite()` helper in `ForceGraph3D.vue`. State lives in the Pinia graph store as session-scoped reactive refs.

## Milestone 1: Add Toggle State to the Graph Store

### Description

Add two new reactive boolean refs (`showNodeTitles`, `showNodeLineage`) to the Pinia `useGraphStore` in `web/src/stores/graph.ts`, both defaulting to `false`. Expose toggle methods following the existing pattern used by `showLabelNodes`, `hideTerminal`, etc.

### Files to change

- `web/src/stores/graph.ts` — add `showNodeTitles` ref, `showNodeLineage` ref, and toggle methods `toggleShowNodeTitles()` / `toggleShowNodeLineage()`.

### Acceptance criteria

- [ ] `showNodeTitles` and `showNodeLineage` exist as `ref(false)` in the graph store.
- [ ] `toggleShowNodeTitles()` and `toggleShowNodeLineage()` flip the respective ref.
- [ ] Both refs and methods are returned from the store setup function.
- [ ] State resets on page reload (no localStorage persistence).

## Milestone 2: Add Checkbox Controls to GraphFilters

### Description

Extend `GraphFilters.vue` to accept and emit the two new toggle props/events. Add two checkboxes ("Show node titles", "Show node lineage") in the existing filter-group `<div>`, positioned after the current "Show Releases" checkbox. Style and accessibility must match the existing checkboxes (visible `<label>`, keyboard-focusable).

### Files to change

- `web/src/components/graph/GraphFilters.vue` — add props `showNodeTitles: boolean`, `showNodeLineage: boolean`; add emits `toggleShowNodeTitles`, `toggleShowNodeLineage`; add two `<label><input type="checkbox" ...> ...</label>` elements.

### Acceptance criteria

- [ ] Two new checkboxes appear in the graph filter bar: "Show node titles" and "Show node lineage".
- [ ] Both are unchecked on initial load.
- [ ] Checking/unchecking emits the corresponding toggle event.
- [ ] Both checkboxes have visible `<label>` elements and are keyboard-focusable.
- [ ] Visual style (spacing, font, alignment) matches the existing checkboxes.

## Milestone 3: Wire GraphView to the New Toggles

### Description

In `GraphView.vue`, pass the two new store refs as props to `GraphFilters` and handle the emitted toggle events by calling the store methods. Also pass `showNodeTitles` and `showNodeLineage` as new props to `ForceGraph3D`.

### Files to change

- `web/src/views/project/GraphView.vue` — bind new props to `<GraphFilters>`, handle toggle events, pass new props to `<ForceGraph3D>`.

### Acceptance criteria

- [ ] `GraphView.vue` reads `store.showNodeTitles` and `store.showNodeLineage` and passes them to both `GraphFilters` and `ForceGraph3D`.
- [ ] Toggle events from `GraphFilters` call `store.toggleShowNodeTitles()` / `store.toggleShowNodeLineage()`.

## Milestone 4: Add Toggle Controls to RoadmapGraphView

### Description

`RoadmapGraphView.vue` currently has no filter panel. Add a minimal controls section containing the two new checkboxes (inline, not the full `GraphFilters` component). Use local component-level refs since the roadmap view does not use `useGraphData`/`useGraphStore`. Pass the refs as props to `ForceGraph3D`.

### Files to change

- `web/src/components/releases/RoadmapGraphView.vue` — add two local `ref(false)` toggles, render two checkboxes in the template above the graph container, pass as props to `<ForceGraph3D>`.

### Acceptance criteria

- [ ] Two checkboxes ("Show node titles", "Show node lineage") appear in the 3D roadmap view controls.
- [ ] Both default to unchecked.
- [ ] Toggling them passes updated props to `ForceGraph3D`.
- [ ] Style matches existing view-mode toggle buttons.

## Milestone 5: Accept Label Props in ForceGraph3D

### Description

Extend `ForceGraph3D.vue` props to accept `showNodeTitles` and `showNodeLineage` booleans (both optional, defaulting to `false`).

### Files to change

- `web/src/components/graph/ForceGraph3D.vue` — add `showNodeTitles?: boolean` and `showNodeLineage?: boolean` to `defineProps`.

### Acceptance criteria

- [ ] `ForceGraph3D.vue` accepts the two new optional boolean props.
- [ ] Defaults to `false` when not provided, preserving backward compatibility.

## Milestone 6: Render Title and Lineage Labels in buildNodeObject

### Description

Modify `buildNodeObject()` in `ForceGraph3D.vue` to conditionally add text sprites for title and lineage labels on non-release, non-label nodes.

**Title label**: When `props.showNodeTitles` is `true`, call `textSprite()` with `node.title || node.slug` truncated to 15 characters (append `…` U+2026 if exceeded). Position above the node sphere at `y = 9` (matching existing label-node sprite offset).

**Lineage label**: When `props.showNodeLineage` is `true`, call `textSprite()` with `node.lineage` (no truncation), using a smaller font size or muted colour to distinguish from the title. Position at `y = 5` (below the title position).

**Combined**: When both are active, title at `y = 12`, lineage at `y = 5` — stacked with no overlap.

### Files to change

- `web/src/components/graph/ForceGraph3D.vue` — modify `buildNodeObject()` to read props and conditionally add title/lineage sprites. Optionally add a second `textSprite` variant or parameterise font size/colour.

### Acceptance criteria

- [ ] When `showNodeTitles` is true, a truncated title label (max 15 chars + `…`) appears above each non-release, non-label node.
- [ ] Titles of 15 characters or fewer display without an ellipsis.
- [ ] When `showNodeLineage` is true, the full lineage slug appears on each non-release, non-label node in a visually distinct style.
- [ ] When both are true, title and lineage are stacked vertically without overlap.
- [ ] Release nodes and label-type nodes are unaffected.

## Milestone 7: Reactive Toggle — Rebuild Node Objects on Prop Change

### Description

Add a `watch` on the two label props that triggers `graph.nodeThreeObject(...)` to rebuild all node objects, following the existing pattern used for theme changes in the `watch(isDark, ...)` block. This ensures toggling a checkbox immediately updates the 3D scene without a page reload or force-layout restart.

### Files to change

- `web/src/components/graph/ForceGraph3D.vue` — add a `watch` on `[() => props.showNodeTitles, () => props.showNodeLineage]` that calls `graph.nodeThreeObject(...)` to rebuild.

### Acceptance criteria

- [ ] Checking or unchecking either checkbox immediately adds or removes the corresponding labels in the 3D scene.
- [ ] No page reload or force-simulation restart occurs.
- [ ] No frame-rate regression on graphs with up to 500 nodes.

## Cross-references

- Backend: [[3d-map-node-labels-toggle]] (backend plan confirms no server changes needed)
- Test coverage: [[3d-map-node-labels-toggle]] (test plan covers integration and visual verification)
