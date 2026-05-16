---
title: 'GraphView.labels tests fail: ForceGraph3D never rendered because view defaults to 2d'
type: defect
status: in-development
lineage: 3d-map-node-labels-toggle
parent: lifecycle/tests/3d-map-node-labels-toggle-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Reproduction Steps

1. From the repo root, run the Milestone 5 test suite:
   ```
   cd tests/web && pnpm test GraphView.labels.test.ts
   ```
2. Observe 6 failures in the `MapView — ForceGraph3D label prop bindings (M5)` and
   `MapView — ForceGraph3D label props update reactively (M5)` describe blocks.

## Expected Behaviour

All 6 `ForceGraph3D` prop-binding and reactivity tests should find the `ForceGraph3DStub`
component in the mounted `MapView` and assert that `showNodeTitles` / `showNodeLineage`
props are correctly wired to the Pinia graph store.

## Actual Behaviour

`wrapper.findComponent(ForceGraph3DStub)` returns an empty `VueWrapper` (`.exists()` is
`false`) for every test in those two describe blocks, causing cascading failures:

- 4 tests fail with _"Cannot call props on an empty VueWrapper."_
- 2 tests fail with _"ForceGraph3D stub not found — check rawNodes/loading state"_ /
  _"ForceGraph3D stub not rendered"_

## Root Cause

`MapView.vue` initialises the view-mode ref as `'2d'`:

```ts
// web/src/views/project/MapView.vue line 29
const view = ref<'3d' | '2d'>('2d')
```

`ForceGraph3D` is only rendered when `view === '3d'` (line 119):

```html
<ForceGraph3D v-if="view === '3d'" … />
```

The `mountMapView()` helper in `GraphView.labels.test.ts` never sets the view to `'3d'`,
so `ForceGraph3D` is never mounted regardless of how many nodes the API mock returns.
The `GraphFilters` binding tests (passing) are unaffected because `GraphFilters` is
always rendered independent of the view mode.

The test plan (Milestone 5) states _"the graph template branch renders ForceGraph3D"_
but the tests do not take the necessary step of switching the view to `'3d'` before
asserting on the component.

## Fix

In `mountMapView()` (or a dedicated `mountMapView3D()` variant), access the internal
`view` ref after mounting and set it to `'3d'`, then flush promises, before attempting
`findComponent(ForceGraph3DStub)`. For example:

```ts
// After mount + flushPromises, expose view ref via wrapper.vm or use wrapper.find
// to click the '3D' toggle button, then flush again.
await wrapper.find('button[aria-pressed]').trigger('click') // assumes 3D button is first
await flushPromises()
```

Alternatively, add a prop or provide/inject mechanism so tests can set the initial view
mode without DOM interaction.

## Logs / Output

```
 FAIL  GraphView.labels.test.ts > MapView — ForceGraph3D label prop bindings (M5) > ForceGraph3D receives showNodeTitles=false (store default)
AssertionError: ForceGraph3D stub not found — check rawNodes/loading state: expected false to be true // Object.is equality
 ❯ GraphView.labels.test.ts:203:94

 FAIL  GraphView.labels.test.ts > MapView — ForceGraph3D label prop bindings (M5) > ForceGraph3D receives showNodeLineage=false (store default)
Error: Cannot call props on an empty VueWrapper.
 ❯ GraphView.labels.test.ts:210:22

 FAIL  GraphView.labels.test.ts > MapView — ForceGraph3D label prop bindings (M5) > ForceGraph3D receives showNodeTitles=true when store has it set
Error: Cannot call props on an empty VueWrapper.
 ❯ GraphView.labels.test.ts:216:22

 FAIL  GraphView.labels.test.ts > MapView — ForceGraph3D label prop bindings (M5) > ForceGraph3D receives showNodeLineage=true when store has it set
Error: Cannot call props on an empty VueWrapper.
 ❯ GraphView.labels.test.ts:222:22

 FAIL  GraphView.labels.test.ts > MapView — ForceGraph3D label props update reactively (M5) > showNodeTitles prop updates when store.toggleShowNodeTitles() is called
AssertionError: ForceGraph3D stub not rendered: expected false to be true // Object.is equality
 ❯ GraphView.labels.test.ts:236:66

 FAIL  GraphView.labels.test.ts > MapView — ForceGraph3D label props update reactively (M5) > showNodeLineage prop updates when store.toggleShowNodeLineage() is called
AssertionError: ForceGraph3D stub not rendered: expected false to be true // Object.is equality
 ❯ GraphView.labels.test.ts:248:66

 Test Files  1 failed | 5 passed (6)
      Tests  6 failed | 86 passed (92)
   Duration  996ms
```
