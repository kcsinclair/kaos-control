---
title: "Graph: Show Tests Toggle — Test Plan"
type: plan-test
status: approved
lineage: graph-show-tests-toggle
parent: lifecycle/requirements/graph-show-tests-toggle-2.md
---

## Overview

Verify the "Show tests" toggle works correctly in both graph views. Tests cover default state, toggle behaviour, type-filter override precedence, edge suppression, and accessibility. Since filtering is entirely client-side (Pinia store computeds), the primary test surface is the graph store logic and the GraphFilters component rendering.

---

## Milestone 1: Store unit tests — `hideTests` state and filtering

**Description:** Test the `hideTests` ref, `toggleHideTests` action, and the `filteredNodes` / `filteredEdges` computed behaviour in the graph store.

**Files to change:**

- `tests/graph-show-tests-toggle.test.ts` (new file)

**Test cases:**

1. **Default state** — `hideTests` is `true` after store initialisation.
2. **Toggle action** — calling `toggleHideTests()` flips `hideTests` from `true` to `false` and back.
3. **Nodes hidden by default** — with `hideTests: true` and no type filter, nodes with `type === 'test'` are excluded from `filteredNodes`.
4. **Nodes shown when toggled** — with `hideTests: false`, test nodes are included in `filteredNodes`.
5. **Type-filter override** — with `hideTests: true` and the `test` type chip active (`filter.types = ['test']`), test nodes appear in `filteredNodes`.
6. **Type-filter partial override** — with `hideTests: true` and a type filter that does NOT include `test` (e.g. `filter.types = ['idea']`), test nodes are excluded (both by `hideTests` bypass not triggering AND by the type filter itself).
7. **Edge suppression** — edges whose source or target is a hidden test node are excluded from `filteredEdges`. Edges between two visible non-test nodes remain.
8. **No interaction with hideTerminal** — toggling `hideTests` does not affect terminal-status hiding, and vice versa.

**Acceptance criteria:**

- [ ] All 8 test cases pass.
- [ ] Tests use the store's composable directly with mock node/edge data — no DOM rendering required.

---

## Milestone 2: Component tests — GraphFilters checkbox rendering

**Description:** Test that the "Show tests" checkbox renders correctly, reflects the `hideTests` prop, and emits the toggle event.

**Files to change:**

- `tests/graph-show-tests-toggle.test.ts` (append to file from Milestone 1)

**Test cases:**

1. **Checkbox renders** — a checkbox with label text "Show tests" is present in the rendered `GraphFilters` component.
2. **Default unchecked** — when `hideTests` prop is `true`, the checkbox is unchecked.
3. **Checked when false** — when `hideTests` prop is `false`, the checkbox is checked.
4. **Emit on change** — clicking the checkbox emits `toggleHideTests`.
5. **Keyboard accessible** — the checkbox can be focused and toggled via keyboard (Enter/Space).
6. **Label association** — the `<input>` is wrapped in or associated with a `<label>`.

**Acceptance criteria:**

- [ ] All 6 test cases pass.
- [ ] Tests mount `GraphFilters` with minimal required props.

---

## Milestone 3: Integration smoke tests — both graph views

**Description:** Verify end-to-end that both the 2D and 3D graph views correctly show/hide test nodes when the toggle is used, using browser-level or integration testing against the running app.

**Files to change:**

- `tests/graph-show-tests-toggle.test.ts` (append to file from Milestone 2)

**Test cases:**

1. **2D view — tests hidden on load** — navigate to the 2D graph view; confirm no nodes with type `test` are visible in the Cytoscape canvas (check DOM or store state).
2. **2D view — tests shown after toggle** — check the "Show tests" checkbox; confirm test nodes and their edges appear.
3. **3D view — tests hidden on load** — switch to the 3D graph view; confirm test nodes are absent.
4. **3D view — tests shown after toggle** — check the "Show tests" checkbox; confirm test nodes appear.
5. **Toggle state resets on navigation** — navigate away from the graph view and back; confirm the checkbox is unchecked and tests are hidden (verifying the `onMounted` reset).

**Acceptance criteria:**

- [ ] All 5 integration test cases pass.
- [ ] Tests confirm visual consistency: the "Show tests" checkbox is visually adjacent to "Show completed" with matching styling.

---

## Dependencies

- [[graph-show-tests-toggle]] frontend plan (`-4-fe`) must be implemented before integration tests can run.
- [[graph-show-tests-toggle]] backend plan (`-3-be`) confirms the API already provides `type` on graph nodes — no test fixtures for API changes needed.
