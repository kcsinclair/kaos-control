---
title: MapView ForceGraph3D stub rendering and label-prop binding tests
type: test
status: draft
lineage: graph-view-force3d-stub-not-rendered
parent: lifecycle/defects/graph-view-force3d-stub-not-rendered.md
---

## Summary

Integration tests verifying that `MapView.vue` correctly renders the `ForceGraph3D`
component in 3D view mode and wires `showNodeTitles` / `showNodeLineage` props and
toggle events through the Pinia graph store.

The fix that unblocked these tests was adding a `mountMapView3D()` helper that
switches the view to `'3d'` (by clicking the first toggle button in `.view-toggle`)
after mount and `flushPromises()`. `ForceGraph3D` is gated on `v-if="view === '3d'"`
in `MapView.vue` (line 119), so tests asserting on that component must activate 3D
view first.

## Test File

`tests/web/GraphView.labels.test.ts`

## Scenarios Covered

### MapView — GraphFilters label prop bindings (M5) — 4 tests

- `GraphFilters` stub receives `showNodeTitles=false` by default.
- `GraphFilters` stub receives `showNodeLineage=false` by default.
- `GraphFilters` stub receives `showNodeTitles=true` when pre-set on the store.
- `GraphFilters` stub receives `showNodeLineage=true` when pre-set on the store.

### MapView — ForceGraph3D label prop bindings (M5) — 4 tests

- `ForceGraph3D` stub receives `showNodeTitles=false` after switching to 3D view.
- `ForceGraph3D` stub receives `showNodeLineage=false` after switching to 3D view.
- `ForceGraph3D` stub receives `showNodeTitles=true` when pre-set on the store.
- `ForceGraph3D` stub receives `showNodeLineage=true` when pre-set on the store.

### MapView — ForceGraph3D label props update reactively (M5) — 2 tests

- `showNodeTitles` prop on `ForceGraph3D` updates to `true` when
  `store.toggleShowNodeTitles()` is called after mount.
- `showNodeLineage` prop on `ForceGraph3D` updates to `true` when
  `store.toggleShowNodeLineage()` is called after mount.

### MapView — GraphFilters toggle events wire to store actions (M5) — 4 tests

- Emitting `toggleShowNodeTitles` from `GraphFilters` calls `store.toggleShowNodeTitles()`.
- Emitting `toggleShowNodeLineage` from `GraphFilters` calls `store.toggleShowNodeLineage()`.
- Emitting `toggleShowNodeTitles` from `GraphFilters` mutates `store.showNodeTitles` to `true`.
- Emitting `toggleShowNodeLineage` from `GraphFilters` mutates `store.showNodeLineage` to `true`.

## Approach

- `MapView` is mounted with a real Pinia store and a memory-history router.
- `getGraph` API is mocked to return one `idea` node with `status: 'in-development'`
  so the `v-else` branch (nodes present) renders.
- `useWebSocket` is mocked to prevent real WebSocket connections.
- All heavy child components are stubbed; `ForceGraph3DStub` and `GraphFiltersStub`
  are defined as named objects so `findComponent(StubDefinition)` locates them
  reliably after mount.
- `mountMapView3D()` calls `mountMapView()` then triggers a click on the first
  `.view-toggle button` to activate 3D view mode before assertions run.
- Store action spies are installed before mounting so Vue captures the spy references.

## Result

28 tests, all passing.
