---
title: 'Frontend Plan: Labels as Graph Nodes with Priority Visualisation'
type: plan-frontend
status: done
lineage: labels-are-nodes
parent: requirements/labels-are-nodes-2.md
---

# Frontend Plan: Labels as Graph Nodes with Priority Visualisation

This plan covers all frontend changes for [[labels-are-nodes]]. It depends on the backend PATCH priority endpoint from [[labels-are-nodes]] backend plan and is validated by the [[labels-are-nodes]] test plan.

## Milestone 1: Create Shared Graph Constants

### Description

Extract `NODE_COLORS` and introduce `PRIORITY_COLORS` and `EDGE_COLORS` into a single shared constants file. Both 2D and 3D graph components, plus the legend, will import from this file — satisfying NFR-3's single-source-of-truth requirement. Extend `NODE_COLORS` to cover every spec-defined type (FR-5).

### Files to Change

- `web/src/components/graph/graphConstants.ts` — **new file**: shared colour maps
- `web/src/components/graph/ForceGraph3D.vue` — remove inline `NODE_COLORS` / `EDGE_COLORS`, import from constants
- `web/src/components/graph/Graph2DView.vue` — remove inline `NODE_COLORS`, import from constants
- `web/src/components/graph/GraphLegend.vue` — import node types and edge kinds from constants

### Implementation Details

Create `graphConstants.ts` with:

```ts
export const NODE_COLORS: Record<string, string> = {
  idea:           '#f59e0b',  // amber
  ticket:         '#3b82f6',  // blue
  epic:           '#1d4ed8',  // darker blue
  'plan-backend': '#8b5cf6',  // violet
  'plan-frontend':'#a78bfa',  // lighter violet
  'plan-dev':     '#7c3aed',  // deep violet
  'plan-test':    '#c084fc',  // lavender
  test:           '#06b6d4',  // cyan
  prototype:      '#14b8a6',  // teal
  release:        '#ef4444',  // red
  sprint:         '#ec4899',  // pink
  defect:         '#f43f5e',  // rose
  label:          '#a855f7',  // purple — synthetic label nodes
}

export const PRIORITY_COLORS: Record<string, string> = {
  high:   '#ef4444',
  medium: '#f97316',
  normal: '#22c55e',
  low:    '#3b82f6',
}

export const EDGE_COLORS: Record<string, string> = {
  parent:     '#94a3b8',
  depends_on: '#f97316',
  blocks:     '#ef4444',
  related_to: '#64748b',
  label:      '#a855f7',
}
```

The `GraphLegend` component's inline arrays must be replaced with derived data from these maps.

### Acceptance Criteria

- [ ] `NODE_COLORS` covers all 12 spec types plus `label`.
- [ ] `PRIORITY_COLORS` has entries for `high`, `medium`, `normal`, `low`.
- [ ] `ForceGraph3D.vue`, `Graph2DView.vue`, and `GraphLegend.vue` all import from `graphConstants.ts` — no inline colour definitions remain.
- [ ] All spec-defined types render with a distinct, non-grey colour (FR-5.1, FR-5.2).
- [ ] Fallback grey (`#6b7280`) still exists for genuinely unknown types (FR-5.3).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

## Milestone 2: Label Node Synthesis and Toggle

### Description

Add logic to synthesise label nodes and label edges from the filtered artifact set (FR-1), and add a show/hide toggle to `GraphFilters.vue` (FR-2). The toggle defaults to off. When enabled, synthetic label nodes and edges are injected into the graph data reactively without a full rebuild.

### Files to Change

- `web/src/stores/graph.ts` — add `showLabelNodes` ref and computed properties that inject synthetic label nodes/edges
- `web/src/components/graph/GraphFilters.vue` — add "Show label nodes" toggle control
- `web/src/views/project/GraphView.vue` — pass the augmented node/edge arrays (with or without label nodes) to graph components

### Implementation Details

In `graph.ts`:
1. Add `showLabelNodes = ref(false)`.
2. Add a computed `labelNodes` that iterates `filteredNodes`, collects distinct labels, and creates a synthetic `GraphNode` for each with `type: 'label'`, `id: 'label::' + labelName`, `title: labelName`, and sensible defaults for other fields.
3. Add a computed `labelEdges` that creates an edge `{ source: artifactId, target: 'label::' + label, kind: 'label' }` for each artifact-label pair.
4. Add a computed `augmentedNodes` that returns `showLabelNodes ? [...filteredNodes, ...labelNodes] : filteredNodes`.
5. Add a computed `augmentedEdges` that returns `showLabelNodes ? [...filteredEdges, ...labelEdges] : filteredEdges`.

In `GraphFilters.vue`:
1. Accept a new prop `showLabelNodes: boolean`.
2. Emit a new event `toggleLabelNodes`.
3. Render a toggle switch between the filter-count and the first filter group.

In `GraphView.vue`:
1. Pass `store.augmentedNodes` and `store.augmentedEdges` to both graph components instead of `filteredNodes`/`filteredEdges`.
2. Pass `store.showLabelNodes` and `store.toggleShowLabelNodes` to `GraphFilters`.

### Acceptance Criteria

- [ ] With toggle off (default), no label nodes or label edges appear in the graph.
- [ ] With toggle on, each distinct label in the filtered set appears as a purple node.
- [ ] Each artifact with a label has an edge of kind `label` to the corresponding label node.
- [ ] Toggling does not cause a full page reload or graph rebuild (FR-2.3) — Cytoscape/force-graph receive updated arrays reactively.
- [ ] Label nodes have `type: 'label'` and render with the label colour from `graphConstants.ts`.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

## Milestone 3: Priority Ring on Artifact Nodes

### Description

Add a coloured ring/border to artifact nodes in both 2D and 3D views that encodes priority (FR-4). The ring colour comes from `PRIORITY_COLORS`. Nodes with no priority get no ring.

### Files to Change

- `web/src/components/graph/Graph2DView.vue` — add border styling based on priority
- `web/src/components/graph/ForceGraph3D.vue` — add priority ring via canvas node painting

### Implementation Details

**2D (Cytoscape)**:
1. In `buildElements()`, add `priorityColor` to the node data, derived from `PRIORITY_COLORS[n.priority]` or `null`.
2. Update the Cytoscape style selector for `node` to use `data(priorityColor)` for `border-color` when set, with `border-width: 3` for nodes with a priority.
3. Use a conditional style: nodes without priority keep the current thin neutral border.

**3D (force-graph)**:
1. Replace the default sphere rendering with `nodeCanvasObject` (for 2D canvas mode) or use `nodeThreeObject` to add a torus/ring around the sphere.
2. Simpler approach: use the `nodeCanvasObjectMode` with `'before'` to draw a larger circle behind each node in the priority colour, creating a ring effect.
3. The ring should scale with the node — thicker when the node is larger (FR-4 open question answer: visible when small, thicker when large).

### Acceptance Criteria

- [ ] Artifact nodes with `priority: "high"` display a red ring in both 2D and 3D.
- [ ] `medium` → orange, `normal` → green, `low` → blue.
- [ ] Nodes without priority display no ring (or a thin neutral border only).
- [ ] The ring is visually distinct from the node fill colour (FR-4.2).
- [ ] In 3D, the ring remains visible at low zoom levels and scales with node size.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

## Milestone 4: Modal Enhancements — Labels, Priority Display, and Inline Edit

### Description

Enhance `ArtifactModal.vue` to display labels as badges, display priority with colour, and allow inline priority editing via a dropdown that PATCHes the backend (FR-3).

### Files to Change

- `web/src/components/artifact/ArtifactModal.vue` — add labels badges, priority badge, priority dropdown
- `web/src/api/artifacts.ts` — add `patchPriority(project, path, priority)` function calling `PATCH /api/p/:project/artifacts/:path/priority`
- `web/src/stores/graph.ts` — add an action to update a node's priority locally after a successful PATCH (for instant ring update)

### Implementation Details

1. **Labels display** (FR-3.1): Below the status badge in `modal-meta`, render `node.labels` as a row of small pill badges with a subtle border.

2. **Priority badge** (FR-3.2): Next to labels, show a priority badge coloured with `PRIORITY_COLORS[node.priority]`. If unset, show "No priority" in muted text.

3. **Priority dropdown** (FR-3.3): Clicking the priority badge opens an inline `<select>` (or a small dropdown menu) with options: High, Medium, Normal, Low, None. On selection:
   - Call `patchPriority(project, node.id, newValue)`.
   - On success, update `detail.value.frontmatter.priority` and emit an event or update the graph store so the node ring updates without a manual refresh (FR-3.4).
   - On error, show a brief error message and revert.

4. **API function**: `patchPriority` sends `PATCH` with `{ priority: value }` to the new backend endpoint from [[labels-are-nodes]] backend plan milestone 1.

### Acceptance Criteria

- [ ] Labels appear as styled badges/chips below the status badge in the modal.
- [ ] Priority is displayed with the correct colour mapping.
- [ ] Clicking priority opens a dropdown; selecting a value triggers a PATCH request.
- [ ] After a successful PATCH, the modal reflects the new priority without page refresh.
- [ ] The graph node's priority ring updates reactively after the priority change.
- [ ] The round-trip (click → select → save → UI update) completes in under 1 second on localhost (NFR-2).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

## Milestone 5: Label Node Click — Mini-Modal

### Description

When a user clicks a label node in either graph view, show a mini-modal summarising which artifacts carry that label (FR-1.5, per open question 2).

### Files to Change

- `web/src/components/graph/LabelModal.vue` — **new file**: a small modal listing artifacts for a label
- `web/src/views/project/GraphView.vue` — detect label-node clicks and show `LabelModal` instead of `ArtifactModal`

### Implementation Details

1. In `GraphView.vue`, the `onNodeClick` handler checks if `node.type === 'label'`. If so, it shows `LabelModal` instead of `ArtifactModal`.
2. `LabelModal` receives the label name and the current filtered artifact list, filters to those carrying the label, and displays them in a compact list with title, type, and status.
3. Clicking an artifact in the list navigates to the artifact editor or opens the full `ArtifactModal`.

### Acceptance Criteria

- [ ] Clicking a label node opens a mini-modal, not the artifact detail modal.
- [ ] The mini-modal lists all artifacts carrying that label with title, type, and status.
- [ ] Clicking an artifact in the list navigates to the artifact editor.
- [ ] The mini-modal can be dismissed with Escape or clicking outside.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

## Milestone 6: Update GraphLegend

### Description

Update the `GraphLegend` component to reflect the expanded colour palette — all spec types, the label node type, priority ring colours, and the label edge kind.

### Files to Change

- `web/src/components/graph/GraphLegend.vue` — derive legend entries from `graphConstants.ts`

### Implementation Details

1. Import `NODE_COLORS`, `PRIORITY_COLORS`, `EDGE_COLORS` from `graphConstants.ts`.
2. Build the node legend from `NODE_COLORS` entries, excluding `label` unless label nodes are currently shown (accept `showLabelNodes` as a prop).
3. Add a "Priority" section showing the four priority ring colours.
4. Build the edge legend from `EDGE_COLORS`, including the `label` edge kind when label nodes are shown.

### Acceptance Criteria

- [ ] Legend shows all spec-defined types with their correct colours.
- [ ] Label node entry appears in the legend only when label nodes are toggled on.
- [ ] Priority ring colours appear in a dedicated "Priority" section.
- [ ] Label edge kind appears only when label nodes are toggled on.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.
