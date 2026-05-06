---
title: 'Graph: Show Tests Toggle'
type: requirement
status: draft
lineage: graph-show-tests-toggle
created: "2026-05-06"
priority: normal
parent: lifecycle/ideas/graph-show-tests-toggle.md
labels:
    - frontend
    - feature
    - vue
    - test
assignees:
    - role: product-owner
      who: agent
---

## Problem

Both the 2D and 3D graph views display every artifact type by default, including `test` artifacts. Test artifacts often outnumber feature artifacts and create visual clutter — dense edges radiating from every feature node to its associated tests make it harder to trace the ideation-to-release flow. Users who are focused on planning, development, or review need a quick way to suppress test nodes without reaching for fine-grained filter chips.

## Goals / Non-goals

### Goals

- Provide a one-click toggle to hide or show `test`-type artifacts in both graph views.
- Default the toggle to **unchecked** (tests hidden) so the graph is clean on first load.
- Reuse the established UX pattern from the existing "Show completed" toggle for visual and behavioural consistency.
- Ensure suppressed test nodes also suppress their connected edges, consistent with existing filtering behaviour.

### Non-goals

- This requirement does **not** cover filtering other artifact types via dedicated toggles (e.g. defects, prototypes). Each would be its own idea/requirement if desired.
- No changes to the graph data API or backend indexing are in scope; filtering is purely client-side.
- No persistence of the toggle state across sessions (local storage or server-side) is required in this iteration.

## Detailed Requirements

### Functional

1. **Toggle control** — Add a "Show tests" checkbox to the `GraphFilters` component, positioned alongside the existing "Show completed" checkbox.
2. **Default state** — The checkbox must be **unchecked** by default (`hideTests: true`).
3. **Node filtering** — When `hideTests` is `true` and no explicit type filter for `test` is active, nodes with `type === 'test'` must be excluded from the `filteredNodes` computed property in the graph store.
4. **Type-filter override** — If the user has explicitly selected the `test` type chip in the type filter, the "Show tests" toggle must be bypassed (same precedence logic as `hideTerminal` vs. the status filter).
5. **Edge filtering** — Edges must follow node visibility: any edge whose source or target is a suppressed test node must also be excluded. This is already handled by the existing `filteredEdges` computed (edges are kept only when both endpoints are visible), so no new edge logic is expected.
6. **Both views** — The toggle must apply identically to the 2D (Cytoscape) and 3D (3D-force-graph) views, since both consume the same store computeds.
7. **Reactivity** — Toggling the checkbox must update the graph immediately (no page reload or manual refresh).

### Non-functional

1. **Performance** — Filtering must remain O(n) over the node set; no additional network requests.
2. **Accessibility** — The checkbox must have a visible label and be keyboard-focusable, matching the existing toggle's accessibility level.
3. **Consistency** — Visual style (spacing, font, alignment) must match the "Show completed" checkbox exactly.

## Acceptance Criteria

- [ ] A "Show tests" checkbox is visible in the graph filter bar in both the 2D and 3D graph views.
- [ ] On initial page load, the checkbox is unchecked and no `test`-type nodes or their exclusive edges appear in the graph.
- [ ] Checking the box causes all `test`-type nodes and their edges to appear without a page reload.
- [ ] Unchecking the box removes `test`-type nodes and their edges again.
- [ ] If a user explicitly selects the `test` type chip in the type filter while the checkbox is unchecked, test nodes still appear (type-filter override).
- [ ] The toggle state is managed in the graph Pinia store (`hideTests` ref + `toggleHideTests` action).
- [ ] The `filteredEdges` computed correctly suppresses edges to hidden test nodes (existing behaviour — verify, do not rewrite).
- [ ] The checkbox is keyboard-accessible and has a visible `<label>`.
- [ ] Visual styling matches the existing "Show completed" checkbox.
- [ ] Related: [[graph-show-tests-toggle]]

## No Questions

_None — the idea is well-defined and follows the established pattern of the "Show completed" toggle._
