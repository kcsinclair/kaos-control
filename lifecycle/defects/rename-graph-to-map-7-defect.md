---
title: "MapFilters test fixture missing required props showReleases, showNodeTitles, showNodeLineage"
type: defect
status: draft
lineage: rename-graph-to-map
parent: lifecycle/tests/rename-graph-to-map-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# MapFilters test fixture missing required props showReleases, showNodeTitles, showNodeLineage

## Reproduction Steps

1. Run the integration test suite:
   ```
   cd tests/web
   pnpm exec vitest run graph-show-tests-toggle.test.ts --reporter=verbose
   ```
2. Observe stderr output during the **Milestone 2** describe block (`GraphFilters — Show tests checkbox`). Each of the 7 tests in that block emits three Vue prop warnings before passing.

## Expected Behaviour

All 7 Milestone 2 tests mount `MapFilters.vue` with all required props provided. No Vue prop warnings appear in the test output. The component is exercised in a complete, production-equivalent state.

## Actual Behaviour

The `defaultProps` object defined in the Milestone 2 describe block of `tests/web/graph-show-tests-toggle.test.ts` (lines 161–180) is missing three required props that `MapFilters.vue` declares in its `defineProps<{...}>()`:

- `showReleases: boolean`
- `showNodeTitles: boolean`
- `showNodeLineage: boolean`

Vue 3 emits a warning for each missing required prop on every mount. Because `defineProps` in Vue 3 does not enforce types at runtime, the component still renders and all assertions pass — masking the incomplete fixture. Any test that relies on the rendering behaviour controlled by these props (e.g. the "Show releases" toggle or the node-label visibility options) would pass vacuously.

Additionally, when `MapView.vue` is mounted in Milestone 3 test 5, the same two props (`showNodeTitles`, `showNodeLineage`) appear in warnings because the `stubs: { GraphFilters: true }` configuration does not fully suppress Vue's prop validation before the stub is applied.

## Logs / Output

```
stderr | graph-show-tests-toggle.test.ts > GraphFilters — Show tests checkbox (Milestone 2) > 1. a checkbox with label text "Show tests" is present
[Vue warn]: Missing required prop: "showReleases"
  at <MapFilters filter=... >
  at <VTUROOT>
[Vue warn]: Missing required prop: "showNodeTitles"
  at <MapFilters filter=... >
  at <VTUROOT>
[Vue warn]: Missing required prop: "showNodeLineage"
  at <MapFilters filter=... >
  at <VTUROOT>

(same three warnings repeated for each of the 7 Milestone 2 tests)

stderr | graph-show-tests-toggle.test.ts > MapView integration — Show tests toggle (Milestone 3) > 5. toggle state resets to hidden when MapView is (re)mounted
[Vue warn]: Missing required prop: "showNodeTitles"
  at <GraphFilters ref="graphFiltersRef" ...>
  at <MapView ref="VTU_COMPONENT">
[Vue warn]: Missing required prop: "showNodeLineage"
  at <GraphFilters ref="graphFiltersRef" ...>
  at <MapView ref="VTU_COMPONENT">

(pair repeated three times)
```

## Fix Required

In `tests/web/graph-show-tests-toggle.test.ts`, update the `defaultProps` constant in the Milestone 2 describe block to include the three missing required props:

```ts
const defaultProps = {
  // …existing props…
  showLabelNodes: false,
  showReleases: false,       // add
  hideTerminal: true,
  hideTests: true,
  showNodeTitles: true,      // add
  showNodeLineage: false,    // add
  searchText: '',
}
```

For Milestone 3 test 5, investigate whether `stubs: { GraphFilters: true }` is correctly matched to the `GraphFilters` local alias of `MapFilters.vue`. If not, switch to `stubs: { MapFilters: true }` or use a component reference stub to ensure prop validation is fully suppressed.
