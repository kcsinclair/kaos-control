---
title: "Graph: Show Tests Toggle — Frontend Plan"
type: plan-frontend
status: draft
lineage: graph-show-tests-toggle
parent: lifecycle/requirements/graph-show-tests-toggle-2.md
---

## Overview

Add a "Show tests" checkbox to the graph filter bar, following the exact pattern established by the existing "Show completed" (`hideTerminal`) toggle. The new `hideTests` state lives in the graph Pinia store and is consumed by the `filteredNodes` computed, which already drives both the 2D and 3D graph views. Edge filtering requires no changes — `filteredEdges` already excludes edges whose endpoints are not in `filteredNodes`.

---

## Milestone 1: Add `hideTests` state and toggle action to the graph store

**Description:** Introduce a `hideTests` ref (default `true`) and a `toggleHideTests` action in the graph store, mirroring `hideTerminal` / `toggleHideTerminal`.

**Files to change:**

- `web/src/stores/graph.ts`

**Changes:**

1. Add `const hideTests = ref(true)` alongside `hideTerminal` (near line 19).
2. Add a `toggleHideTests()` function alongside `toggleHideTerminal` (near line 114):
   ```ts
   function toggleHideTests(): void {
     hideTests.value = !hideTests.value
   }
   ```
3. Export `hideTests` and `toggleHideTests` from the composable return object (near line 136).

**Acceptance criteria:**

- [ ] `hideTests` defaults to `true`.
- [ ] Calling `toggleHideTests()` flips the value.
- [ ] Both are exported from the store composable.

---

## Milestone 2: Filter `test`-type nodes in `filteredNodes`

**Description:** Extend the `filteredNodes` computed to exclude nodes with `type === 'test'` when `hideTests` is true and no explicit type filter is active, using the same precedence pattern as `hideTerminal` / status filters.

**Files to change:**

- `web/src/stores/graph.ts`

**Changes:**

1. In the `filteredNodes` computed (near line 31), add a `noTypeFilter` guard:
   ```ts
   const noTypeFilter = !(f.types?.length)
   ```
2. Insert a filter clause after the existing `hideTerminal` check (after line 35):
   ```ts
   if (hideTests.value && noTypeFilter && n.type === 'test') return false
   ```

**Precedence logic:** When the user explicitly selects the `test` type chip, `f.types` is non-empty, so `noTypeFilter` is `false` and the `hideTests` clause is bypassed — the type filter alone governs visibility. This mirrors how `hideTerminal` is bypassed when a status filter is active.

**Acceptance criteria:**

- [ ] With `hideTests: true` and no type filter, nodes where `type === 'test'` are excluded from `filteredNodes`.
- [ ] With `hideTests: true` and the `test` type chip selected, test nodes appear (type-filter override).
- [ ] With `hideTests: false`, test nodes always appear (subject to other filters).
- [ ] `filteredEdges` automatically excludes edges to hidden test nodes (existing behaviour — no code change).

---

## Milestone 3: Add "Show tests" checkbox to `GraphFilters`

**Description:** Add a new checkbox to the `GraphFilters` component, positioned alongside the existing "Show completed" checkbox, following the same prop/emit pattern.

**Files to change:**

- `web/src/components/graph/GraphFilters.vue`

**Changes:**

1. Add `hideTests: boolean` to the component's `defineProps` (near line 14).
2. Add `toggleHideTests: []` to the component's `defineEmits` (near line 21).
3. Add the checkbox markup immediately after the "Show completed" `<label>` (after line 63):
   ```vue
   <label class="toggle-label">
     <input
       type="checkbox"
       class="toggle-input"
       :checked="!hideTests"
       @change="emit('toggleHideTests')"
     />
     <span class="toggle-text">Show tests</span>
   </label>
   ```

**Acceptance criteria:**

- [ ] The "Show tests" checkbox renders next to "Show completed" in the filter bar.
- [ ] Checkbox is unchecked on initial load (reflecting `hideTests: true`).
- [ ] Toggling emits `toggleHideTests`.
- [ ] Checkbox has a visible `<label>` and is keyboard-focusable.
- [ ] Visual style (spacing, font, alignment) matches the "Show completed" checkbox.

---

## Milestone 4: Wire the toggle through `GraphView`

**Description:** Pass `hideTests` and the toggle handler from the graph store to the `GraphFilters` component in `GraphView.vue`.

**Files to change:**

- `web/src/views/project/GraphView.vue`

**Changes:**

1. In the `onMounted` hook (near line 63), add `store.hideTests = true` alongside the existing `store.hideTerminal = true` reset.
2. Add prop binding on the `<GraphFilters>` component (near line 67):
   ```vue
   :hide-tests="store.hideTests"
   ```
3. Add event handler:
   ```vue
   @toggle-hide-tests="store.toggleHideTests"
   ```

**Acceptance criteria:**

- [ ] On mount, `hideTests` resets to `true` (tests hidden by default).
- [ ] The "Show tests" checkbox reflects the store state and toggles it on click.
- [ ] Both 2D and 3D graph views update reactively when the toggle changes — no page reload required.
- [ ] No changes to `Graph2DView.vue` or `ForceGraph3D.vue` are needed (they consume pre-filtered data from the store).

---

## Dependencies

- [[graph-show-tests-toggle]] backend plan (`-3-be`) confirms no API changes needed.
- [[graph-show-tests-toggle]] test plan (`-5-test`) covers integration and visual verification.
