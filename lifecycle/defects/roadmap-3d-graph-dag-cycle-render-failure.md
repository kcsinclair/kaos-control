---
title: Roadmap 3D graph nodes don't lay out — DAG mode breaks on cyclic graph (no onDagError)
type: defect
status: done
lineage: roadmap-3d-graph-dag-cycle
created: "2026-06-26T00:00:00+10:00"
priority: high
labels:
    - defect
    - frontend
    - roadmap
    - graph
release: KC-Release4
assignees:
    - role: frontend-developer
      who: agent
---

# Roadmap 3D graph nodes don't lay out — DAG mode breaks on cyclic graph (no onDagError)

## Reproduction Steps

1. Open a project with releases and assigned artifacts.
2. Go to **Roadmap** and switch to the **Graph** view in **3D**.
3. Observe the 3D canvas: the engine renders (a release node and edge arrowheads
   are visible), but nodes are **not laid out** — they pile up near the origin
   with overlapping/garbled label sprites and scattered, disconnected
   arrowheads, with no left-to-right ordering. (Switching to **2D** works; the
   regular **Map** 3D view also works.)

## Expected Behaviour

The roadmap 3D graph should lay out left-to-right (DAG mode `lr`) like a normal
force/DAG graph — nodes spread and separated, edges connecting them, labels
legible.

## Actual Behaviour

The WebGL render engine **does** start (release sphere, edge arrow cones, and
label sprites are drawn), but the **layout fails**: nodes are not positioned by
the DAG layout — they collapse to the origin with overlapping labels and the
arrowheads scatter, so the graph is unreadable. The regular 3D map
([web/src/components/map/ForceGraph3D.vue](../../web/src/components/map/ForceGraph3D.vue))
lays out correctly with the *same* underlying component.

## Root Cause

The roadmap reuses the shared `ForceGraph3D.vue` component but is the only caller
that enables DAG layout:
[RoadmapGraphView.vue:162](../../web/src/components/releases/RoadmapGraphView.vue#L162)
passes `dag-mode="lr"`, and `ForceGraph3D` applies it
([ForceGraph3D.vue:251-252](../../web/src/components/map/ForceGraph3D.vue#L251)):

```ts
if (props.dagMode) {
  graph.dagMode(props.dagMode as any)
}
```

No `onDagError` / `dagError` handler is registered anywhere in `web/src`. When
DAG mode is enabled on graph data that contains a **cycle** and no `onDagError`
is set, 3d-force-graph cannot assign valid DAG depths, so the layout breaks — the
engine still renders but nodes are left unconstrained/at the origin (and the
position solver can produce NaN/collapsed coordinates), which is exactly the
piled-up, non-laid-out result observed.

The roadmap graph combines `timeline`, `parent`, `depends_on`, `related_to`,
`blocks`, and `assigned` edges across releases and artifacts, which readily forms
cycles (e.g. symmetric `related_to`, or a `depends_on`/timeline/assignment loop).
So DAG mode can't lay out the data. The regular map never sets `dagMode`, so it
is unaffected — explaining "the regular 3D map is working well".

## Suggested Fix

In `ForceGraph3D.vue`, when `dagMode` is set, also register an `onDagError`
handler so cyclic data degrades gracefully instead of throwing. 3d-force-graph's
`onDagError(null)` skips the DAG constraint for nodes involved in cycles and
still renders; a no-op callback `onDagError(() => {})` ignores the error:

```ts
if (props.dagMode) {
  graph.dagMode(props.dagMode as any)
  // Tolerate cycles: without this, a cyclic graph throws and the engine never
  // starts (the roadmap graph is frequently cyclic).
  graph.onDagError(() => {})
}
```

(Order also matters: ensure `dagMode`/`onDagError` are set before/around
`graphData(...)` so the constraint is applied to the loaded data.)

## Verification

- Roadmap 3D view renders for a project whose graph contains a cycle.
- Add a Vitest case mounting `ForceGraph3D` with `dag-mode` and cyclic edges,
  asserting the graph instance is created and no exception is thrown (extend the
  existing `tests/web/3d-graph-edge-contrast.linkWidth.test.ts` / ForceGraph3D
  specs).

## Resolution — 2026-06-26 (verified)

Registered `graph.onDagError(() => {})` before `graph.dagMode(...)` in
`web/src/components/map/ForceGraph3D.vue`, so cyclic data no longer breaks the
DAG layout. Verified live: the Roadmap 3D graph now lays out instead of piling
nodes at the origin.
