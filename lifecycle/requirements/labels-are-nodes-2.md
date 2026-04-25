---
title: "Labels as Graph Nodes with Priority Visualisation"
type: ticket
status: draft
lineage: labels-are-nodes
parent: ideas/labels-are-nodes.md
labels:
    - enhancement
    - frontend
---

# Labels as Graph Nodes with Priority Visualisation

## Problem

The graph views (2D Cytoscape and 3D force-graph) only display artifact nodes. Labels — free-form tags set in frontmatter — are invisible in the graph even though they represent a meaningful clustering dimension. Users cannot visually discover which artifacts share a label without manually inspecting each one.

Additionally, priority is stored in frontmatter and usable as a filter, but it has no visual representation on artifact nodes. The node detail modal omits both labels and priority, making it impossible to see or change them without opening the editor.

Finally, the current `NODE_COLORS` map only covers a subset of artifact types (`idea`, `requirement`, `plan`, `implementation`, `test`, `release`), so many node types defined in the spec — `ticket`, `epic`, `plan-backend`, `plan-frontend`, `plan-dev`, `plan-test`, `prototype`, `sprint`, `defect` — render as the grey fallback colour.

## Goals / Non-goals

### Goals

1. Render each distinct label as a dedicated node in both graph views, with edges connecting it to every artifact that carries that label.
2. Provide a UI toggle to show/hide label nodes so the graph can be decluttered on demand.
3. Display labels and priority in the artifact detail modal.
4. Allow inline editing of priority directly from the modal (without navigating to the full editor).
5. Visually encode priority on artifact nodes as a coloured ring/border.
6. Update `NODE_COLORS` to assign a distinct colour to every artifact `type` in the spec vocabulary, eliminating grey fallback nodes under normal use.

### Non-goals

- Creating a managed label taxonomy or admin UI for labels (labels remain free-form per §3.6 of the spec).
- Changing the backend indexer or API schema for labels (the backend already indexes `labels` from frontmatter and returns them on graph nodes).
- Persisting the label-node toggle state server-side (local component or Pinia state is sufficient).

## Detailed Requirements

### FR-1 Label Nodes

- **FR-1.1** When building the graph data, the frontend must create a synthetic node for each distinct label value present in the current (possibly filtered) artifact set.
- **FR-1.2** Label nodes must use a visually distinct shape or colour that differentiates them from artifact nodes. A suggested default colour is `#a855f7` (purple) but the exact value is an implementation detail.
- **FR-1.3** For every artifact that has a given label in its `labels` array, an edge of kind `label` must be created from the artifact node to the label node.
- **FR-1.4** Label nodes must display the label text as their title in both 2D and 3D views.
- **FR-1.5** Clicking a label node in either graph view should open a filtered list or highlight all connected artifacts — behaviour should be consistent with clicking any other node (i.e., show a modal or info panel for the label).

### FR-2 Label Node Toggle

- **FR-2.1** A toggle control (checkbox or switch) must be added to the graph filter panel (`GraphFilters.vue` or equivalent) labelled "Show label nodes" (or similar).
- **FR-2.2** When the toggle is off (default), label nodes and their edges are excluded from the rendered graph.
- **FR-2.3** Toggling must not cause a full graph rebuild; label nodes and edges should be added/removed reactively.

### FR-3 Modal Enhancements

- **FR-3.1** The artifact detail modal (`ArtifactModal.vue`) must display the artifact's `labels` as a list of badges/chips below the status badge.
- **FR-3.2** The modal must display the artifact's `priority` value (if set) as a badge, using the same colour scheme as the priority ring (FR-5).
- **FR-3.3** Priority must be editable inline via a dropdown or select control in the modal. Selecting a new value issues a `PUT` to the artifact API endpoint to update the frontmatter `priority` field.
- **FR-3.4** After a successful priority update the modal must reflect the new value without requiring a manual refresh.

### FR-4 Priority Ring on Artifact Nodes

- **FR-4.1** Each artifact node in both graph views must render a coloured ring (border/stroke) whose colour maps to the artifact's priority:
  - `high` → red (`#ef4444`)
  - `medium` → orange (`#f97316`)
  - `normal` → green (`#22c55e`)
  - `low` → blue (`#3b82f6`)
  - unset / unknown → no ring (or a thin neutral border)
- **FR-4.2** The ring must be visually distinct from the node fill colour so both type-colour and priority-colour are simultaneously readable.
- **FR-4.3** In 3D mode, the ring may be implemented as a sprite outline, a second concentric sphere, or a torus — as long as the colour is clearly visible.

### FR-5 Updated Node Type Colours

- **FR-5.1** `NODE_COLORS` must be extended to cover every `type` in the spec vocabulary: `idea`, `ticket`, `epic`, `plan-backend`, `plan-frontend`, `plan-dev`, `plan-test`, `test`, `prototype`, `release`, `sprint`, `defect`.
- **FR-5.2** Each type must have a visually distinguishable colour. The palette should remain accessible (WCAG AA contrast against the graph background).
- **FR-5.3** The fallback colour for unrecognised types must still exist but should only apply to genuinely unknown types, not to spec-defined types.

### Non-functional Requirements

- **NFR-1** Adding label nodes must not degrade graph render performance for projects with up to 500 artifacts and 50 distinct labels.
- **NFR-2** The priority inline-edit round-trip (click → select → save → UI update) should complete in under 1 second on localhost.
- **NFR-3** Colour choices for type nodes and priority rings should be defined in a single shared constants file or composable so both 2D and 3D views stay in sync.

## Acceptance Criteria

- [ ] With label nodes enabled, each distinct label appears as a node in the 2D graph with edges to its artifacts.
- [ ] With label nodes enabled, each distinct label appears as a node in the 3D graph with edges to its artifacts.
- [ ] A toggle in the graph filter panel hides/shows label nodes; default is hidden.
- [ ] Toggling label nodes does not trigger a full page reload or graph rebuild.
- [ ] The artifact modal displays labels as badges/chips.
- [ ] The artifact modal displays priority with the correct colour.
- [ ] Priority can be changed from the modal via a dropdown; the change persists to disk.
- [ ] After changing priority, the node's ring colour updates without a manual refresh.
- [ ] Artifact nodes display a coloured ring matching their priority value.
- [ ] All spec-defined artifact types render with a distinct, non-grey colour.
- [ ] No visible performance regression on a project with 200+ artifacts and 30+ labels.
- [ ] Colour constants are shared between 2D and 3D graph components (single source of truth).
- [ ] [[labels-are-nodes]] idea requirements are fully addressed.

## Open Questions

1. Should label nodes be filterable independently (e.g., show only the "auth" label node), or is the global show/hide toggle sufficient for v1?
2. What should happen when a label node is clicked — open a mini-modal summarising which artifacts carry that label, or apply a graph filter to isolate those artifacts?
3. Should the priority ring be visible when the node is very small (zoomed out in 3D)? If so, should ring thickness scale with zoom?
