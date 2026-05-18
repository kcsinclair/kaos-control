---
title: 'Tests: 3D Map Node Labels Toggle'
type: test
status: done
lineage: 3d-map-node-labels-toggle
parent: lifecycle/test-plans/3d-map-node-labels-toggle-5-test.md
---

## Summary

Integration test suite covering the "Show node titles" and "Show node lineage"
label-toggle checkboxes added to the 3D map and roadmap views.  All 6 planned
milestones are implemented as Vitest + Vue Test Utils component and store tests.
Milestone 7 (E2E / visual) requires a live browser and is documented as a manual
test checklist rather than automated code.

**Total: 92 automated tests across 6 files, all passing.**

## Test Files

### `tests/web/truncateTitle.test.ts` — Milestone 1

Unit tests for the 15-character title truncation rule (inlined in
`ForceGraph3D.vue`'s `buildNodeObject`).  The logic is extracted to a pure
function inside the test file so the behaviour can be exercised independently.

Scenarios covered:
- Strings of exactly 15 characters return unchanged (no ellipsis).
- Strings of 16+ characters are truncated to 15 chars followed by `…` (U+2026).
- Empty string and single-character inputs handled correctly.
- Unicode / multi-byte characters truncated by character count (JS code-units).

### `tests/web/GraphFilters.labels.test.ts` — Milestone 2

Component tests for `web/src/components/map/MapFilters.vue` (imported as
`GraphFilters` in `MapView.vue`).

Scenarios covered:
- A checkbox labelled "Show node titles" is rendered.
- A checkbox labelled "Show node lineage" is rendered.
- Both are unchecked when `showNodeTitles=false` and `showNodeLineage=false`.
- Both are checked when their respective props are `true`.
- Triggering each checkbox emits `toggleShowNodeTitles` / `toggleShowNodeLineage`.
- Each toggle emits independently (no cross-contamination).
- Both inputs are wrapped in `<label>` elements (a11y).
- Both inputs carry an `id` attribute for explicit label association.
- Inputs are not disabled.
- Both checkboxes are in the same `.filter-group` as other toggles.

### `tests/web/ForceGraph3D.labels.test.ts` — Milestone 3

Component tests for `web/src/components/map/ForceGraph3D.vue` — verifies that
`buildNodeObject` produces the correct `THREE.Group` structure depending on prop
values.  Three.js is mocked with lightweight stand-ins; canvas `getContext('2d')`
is stubbed globally via `vi.spyOn(document, 'createElement')` to capture text
rendered via `fillText()`.

Scenarios covered:
- Both props `false` → regular nodes produce no sprites.
- `showNodeTitles=true` → exactly one sprite; 15-char title unchanged; 16+-char
  title truncated to 15 + `…`; empty title falls back to slug.
- `showNodeLineage=true` → exactly one sprite; full lineage text rendered; empty
  lineage produces no sprite.
- Both props `true` → two sprites at different y-offsets; title sprite is higher
  than lineage sprite (y=12 vs y=5).
- Release nodes (`type='release'`) route to `buildReleaseObject` which always
  emits exactly one sprite regardless of props.
- Label nodes (`type='label'`) always get exactly one sprite (their own label
  text) regardless of props.

### `tests/web/graphStore.labels.test.ts` — Milestone 4

Pure Pinia store unit tests for `web/src/stores/graph.ts`.

Scenarios covered:
- `showNodeTitles` defaults to `false`.
- `showNodeLineage` defaults to `false`.
- `toggleShowNodeTitles()` flips the value (false→true→false, odd calls→true).
- `toggleShowNodeLineage()` flips the value.
- The two refs are independent (toggling one leaves the other unchanged).
- Toggling `hideTerminal`/`hideTests` does not affect label refs.
- State resets to `false` on a fresh pinia (no localStorage persistence).
- Both toggle methods are exported as callable functions on the store.

### `tests/web/GraphView.labels.test.ts` — Milestone 5

Integration tests for `web/src/views/project/MapView.vue`.  The API mock returns
a visible idea node so the graph template branch renders `ForceGraph3D`.  All
child components are stubbed; store action spies are installed before mounting.

Scenarios covered:
- `GraphFilters` stub receives `showNodeTitles` / `showNodeLineage` props
  matching store state (both `false` and `true` variants).
- `ForceGraph3D` stub receives the same props matching store state.
- `ForceGraph3D` props update reactively when `store.toggleShowNodeTitles()` /
  `store.toggleShowNodeLineage()` are called.
- Emitting `toggleShowNodeTitles` from `GraphFilters` stub calls
  `store.toggleShowNodeTitles()` (verified by `vi.spyOn` installed pre-mount).
- Emitting `toggleShowNodeLineage` from `GraphFilters` stub calls
  `store.toggleShowNodeLineage()`.
- Both toggle events result in the correct store state mutation.

### `tests/web/RoadmapGraphView.labels.test.ts` — Milestone 6

Integration tests for `web/src/components/releases/RoadmapGraphView.vue`.  The
releases API mock returns a small graph (one synthetic backlog release + one idea
node) so the 3D graph template branch renders.  `ForceGraph3D` is stubbed.

Scenarios covered:
- A checkbox labelled "Show node titles" is rendered.
- A checkbox labelled "Show node lineage" is rendered.
- Both checkboxes are of `type="checkbox"`.
- Both default to unchecked on initial render.
- `ForceGraph3D` stub receives `showNodeTitles=false` / `showNodeLineage=false`
  initially.
- Clicking "Show node titles" sets `ForceGraph3D` `showNodeTitles` prop to `true`.
- Clicking "Show node lineage" sets `ForceGraph3D` `showNodeLineage` prop to `true`.
- The two checkboxes toggle independently.
- Re-clicking a checked checkbox returns the prop to `false`.
- Both inputs are wrapped in `<label>` elements (a11y).

## Milestone 7 — Manual / E2E Checklist

Playwright is not configured in this repo.  The following steps should be
performed manually against a running dev server (`make run`):

1. Navigate to the 3D map view; both checkboxes are visible and unchecked; no
   labels on nodes.
2. Check "Show node titles": truncated title labels appear on non-release,
   non-label nodes.
3. Check "Show node lineage": lineage slug labels appear in a distinct smaller
   grey style.
4. Enable both: labels stack vertically without overlap (title above lineage).
5. Uncheck each: corresponding labels disappear immediately, no page reload.
6. Navigate to the 3D roadmap view: same two checkboxes exist and function
   identically.
7. Release nodes and label-type nodes remain unlabelled throughout.
8. Toggle state persists when switching between 2D and 3D within the same
   session.
9. Toggle state resets on full page reload.
10. No visible frame-rate regression with both labels on a graph of 100+ nodes.
