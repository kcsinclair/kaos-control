---
title: 'Fix: GraphView.labels ForceGraph3D tests use 3D view mode'
type: test
status: draft
lineage: 3d-map-node-labels-toggle
parent: lifecycle/defects/3d-map-node-labels-toggle-7-defect.md
---

## Summary

Fixes the 6 failing tests in `tests/web/GraphView.labels.test.ts` that were
caused by the `mountMapView()` helper never switching the view to `'3d'`.
`ForceGraph3D` is gated on `v-if="view === '3d'"` in `MapView.vue` (line 119),
so any test that asserts on that component must first switch to 3D view.

## What Changed

A new `mountMapView3D()` helper was added to
`tests/web/GraphView.labels.test.ts` (after line 148). It calls `mountMapView()`
then clicks the first button in `.view-toggle` (the "3D" toggle) and flushes
promises so the `ForceGraph3D` branch renders before assertions run.

The two describe blocks that test `ForceGraph3D` were updated to call
`mountMapView3D()` instead of `mountMapView()`:

- `MapView — ForceGraph3D label prop bindings (M5)` — 4 tests
- `MapView — ForceGraph3D label props update reactively (M5)` — 2 tests

The 4 `GraphFilters` and event-wiring describe blocks continue to use
`mountMapView()` unchanged, since `GraphFilters` is always rendered regardless
of view mode.

## Test File

`tests/web/GraphView.labels.test.ts`

## Scenarios Covered

All 6 previously failing scenarios now pass:

- `ForceGraph3D` stub receives `showNodeTitles=false` (store default) after
  switching to 3D view.
- `ForceGraph3D` stub receives `showNodeLineage=false` (store default) after
  switching to 3D view.
- `ForceGraph3D` stub receives `showNodeTitles=true` when pre-set on the store.
- `ForceGraph3D` stub receives `showNodeLineage=true` when pre-set on the store.
- `showNodeTitles` prop on `ForceGraph3D` updates reactively when
  `store.toggleShowNodeTitles()` is called.
- `showNodeLineage` prop on `ForceGraph3D` updates reactively when
  `store.toggleShowNodeLineage()` is called.

Total suite: 28 tests, all passing (was 6 failed / 22 passed before this fix).
