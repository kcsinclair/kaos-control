---
title: '3D Map: Toggleable Node Title and Lineage Labels'
type: requirement
status: done
lineage: 3d-map-node-labels-toggle
created: "2026-05-10T00:00:00+10:00"
priority: normal
parent: lifecycle/ideas/3d-map-node-labels-toggle.md
labels:
    - frontend
    - enhancement
    - vue
    - usability
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

## Problem

The 3D map and 3D roadmap views currently render nodes as unlabelled spheres (except for release and label-type nodes which have special treatment). Users must hover over a node to see its tooltip in order to identify it. This makes spatial orientation slow — users cannot scan the graph to locate a specific artifact or understand lineage groupings at a glance. The 2D map view already displays node labels by default, creating an inconsistent experience when switching between views.

## Goals / Non-goals

### Goals

- Provide two independent checkboxes in both 3D views (map and roadmap): **"Show node titles"** and **"Show node lineage"**.
- Default both checkboxes to **off**, preserving the current clean, unlabelled 3D experience.
- When "Show node titles" is enabled, render a text label above each node showing the artifact title, truncated to 15 characters with an ellipsis (`…`) when exceeded.
- When "Show node lineage" is enabled, render a text label showing the full lineage slug (untruncated) so users can identify which lineage a node belongs to.
- When both are enabled, display both labels (title above lineage) without overlapping.
- Persist checkbox state for the duration of the browser session (not across sessions).
- Leverage the existing `3d-force-graph` label/overlay mechanism (`nodeThreeObject` / `nodeThreeObjectExtend`) and the existing `textSprite` helper in `ForceGraph3D.vue`.

### Non-goals

- No changes to the 2D map view. Its labels are controlled separately via Cytoscape styling.
- No server-side or localStorage persistence of toggle state.
- No user-configurable truncation length; 15 characters is fixed for this iteration.
- No per-node toggle — this is a global on/off for all non-synthetic nodes.
- No changes to release-node or label-node rendering; those already have their own label logic.

## Detailed Requirements

### Functional

1. **Checkbox controls** — Add two checkboxes to the 3D view controls area in both `GraphView.vue` (3D map) and `RoadmapGraphView.vue` (3D roadmap): "Show node titles" and "Show node lineage". Position them alongside existing toggle controls (e.g. "Show completed", "Show tests").

2. **Default state** — Both checkboxes must default to **unchecked** (`showNodeTitles: false`, `showNodeLineage: false`).

3. **Title label rendering** — When `showNodeTitles` is `true`, each non-release, non-label node must display a text sprite above the node sphere showing `node.title || node.slug`, truncated to 15 characters. If the original string exceeds 15 characters, append `…` (Unicode ellipsis U+2026). The label must be rendered via the existing `textSprite()` helper in `ForceGraph3D.vue`.

4. **Lineage label rendering** — When `showNodeLineage` is `true`, each non-release, non-label node must display a text sprite showing `node.lineage` in full (no truncation). Use a distinct, smaller font size or muted colour so it is visually distinguishable from the title label.

5. **Combined display** — When both toggles are active, stack the title label above the lineage label with sufficient vertical offset to prevent overlap.

6. **Toggle reactivity** — Changing either checkbox must immediately update the 3D scene without a page reload. Use the existing `nodeThreeObject` refresh path (already demonstrated in `ForceGraph3D.vue` for theme changes) to rebuild node objects when toggle state changes.

7. **Node exclusions** — Release nodes and label-type nodes already have bespoke rendering. Do not add title/lineage labels to these node types; their existing labels are sufficient.

8. **Session persistence** — Store toggle state in a reactive ref (Pinia store or component-level ref). State resets on page navigation or reload; no persistence to localStorage or the server.

9. **Both 3D views** — The toggles must function identically in the standalone 3D map (`GraphView.vue`) and the roadmap 3D graph (`RoadmapGraphView.vue`), since both consume `ForceGraph3D.vue`.

### Non-functional

1. **Performance** — Label sprite creation must remain O(n) over the visible node set. Text sprites are lightweight canvas textures; no additional network requests or GPU-heavy geometry.

2. **Readability** — Title labels must use a legible font size that scales appropriately with camera zoom. Follow the sizing precedent set by existing `textSprite` calls for label nodes.

3. **Accessibility** — Checkboxes must have visible `<label>` elements and be keyboard-focusable, matching the style and behaviour of existing graph filter checkboxes.

4. **Visual consistency** — Checkbox styling (spacing, font, alignment) must match existing toggles in the graph filter bar.

## Acceptance Criteria

- [ ] A "Show node titles" checkbox appears in the 3D map view controls.
- [ ] A "Show node lineage" checkbox appears in the 3D map view controls.
- [ ] Both checkboxes also appear in the 3D roadmap view controls.
- [ ] On initial load, both checkboxes are unchecked and no title/lineage labels are rendered on nodes.
- [ ] Checking "Show node titles" causes a truncated title label (max 15 chars + `…`) to appear above each non-release, non-label node.
- [ ] Titles of 15 characters or fewer are displayed without an ellipsis.
- [ ] Checking "Show node lineage" causes the full lineage slug to appear on each non-release, non-label node.
- [ ] Enabling both checkboxes displays both labels stacked vertically without overlap.
- [ ] Unchecking either box immediately removes the corresponding labels without a page reload.
- [ ] Release nodes and label-type nodes are unaffected by either toggle.
- [ ] Toggle state persists while navigating within the same browser session (e.g. switching between 2D/3D mode and back).
- [ ] Toggle state resets on full page reload.
- [ ] Both checkboxes are keyboard-accessible and have visible `<label>` elements.
- [ ] Visual styling of the checkboxes matches existing graph filter toggles.
- [ ] No measurable frame-rate regression on graphs with up to 500 nodes when both labels are enabled.
- [ ] Related: [[3d-map-node-labels-toggle]]

## Resolved Questions

_None — the idea is well-scoped with clear defaults, truncation rules, and implementation guidance._
