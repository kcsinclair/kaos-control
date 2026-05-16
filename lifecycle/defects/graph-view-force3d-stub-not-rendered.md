---
title: MapView — ForceGraph3D Stub Not Rendered in Unit Tests; showNodeTitles/showNodeLineage Props Untestable
type: defect
status: done
lineage: graph-view-force3d-stub-not-rendered
created: "2026-05-16T14:00:00+10:00"
priority: normal
labels:
    - defect
    - test
    - frontend
    - map
    - 3d-graph
release: KC-Release2
assignees:
    - role: test-developer
      who: agent
---

# MapView — ForceGraph3D Stub Not Rendered in Unit Tests; showNodeTitles/showNodeLineage Props Untestable

## Reproduction Steps

1. Run `cd tests/web && pnpm test`.
2. Observe failures in `GraphView.labels.test.ts`.

## Expected Behaviour

The `MapView` component renders a `ForceGraph3D` child when nodes are available. The test stubs `ForceGraph3D` and asserts that `showNodeTitles` and `showNodeLineage` props are passed correctly from the store.

## Actual Behaviour

Six tests in `GraphView.labels.test.ts > MapView — ForceGraph3D label prop bindings (M5)` fail:

- `ForceGraph3D stub not found — check rawNodes/loading state: expected false to be true`
- `Cannot call props on an empty VueWrapper.` (for all prop-binding and reactivity tests)

`wrapper.findComponent(ForceGraph3DStub)` returns an empty wrapper, meaning the `ForceGraph3D` stub is not present in the mounted `MapView`. The component is either conditionally rendered (guarded behind a loading state or empty-nodes check that the test setup does not satisfy) or the component name/import used in the test stub no longer matches what `MapView` imports.

Failing tests (all in `tests/web/GraphView.labels.test.ts`):
1. `ForceGraph3D receives showNodeTitles=false (store default)`
2. `ForceGraph3D receives showNodeLineage=false (store default)`
3. `ForceGraph3D receives showNodeTitles=true when store has it set`
4. `ForceGraph3D receives showNodeLineage=true when store has it set`
5. `showNodeTitles prop updates when store.toggleShowNodeTitles() is called`
6. `showNodeLineage prop updates when store.toggleShowNodeLineage() is called`

## Notes

Likely root causes:
1. `MapView` wraps `ForceGraph3D` in a `v-if` conditioned on `rawNodes.length > 0` or `!loading`, and the test setup does not seed any nodes or await the loading state to resolve.
2. The component was renamed or its import path changed, breaking the stub registration.

Verify by logging `wrapper.html()` in the test and checking whether `MapView` renders anything at all once `mountMapView()` resolves.
