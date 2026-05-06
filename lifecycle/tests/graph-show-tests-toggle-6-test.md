---
title: "Tests — Graph Show Tests Toggle"
type: test
status: draft
lineage: graph-show-tests-toggle
parent: lifecycle/test-plans/graph-show-tests-toggle-5-test.md
created: "2026-05-06T00:00:00+10:00"
---

# Tests — Graph Show Tests Toggle

## Overview

This artifact documents the automated test suite that verifies the "Show tests"
toggle works correctly in both graph views.  Filtering is entirely client-side
(Pinia store computeds), so the primary test surfaces are the graph store logic
and the `GraphFilters` component rendering.

All 20 tests live in a single file:

**`tests/web/graph-show-tests-toggle.test.ts`**

---

## Milestone 1 — Store unit tests (8 tests)

Tests the `hideTests` ref, `toggleHideTests` action, and the `filteredNodes` /
`filteredEdges` computed behaviour directly against `useGraphStore`.  No DOM
rendering — Pinia store is created with `createPinia()` / `setActivePinia()` and
`rawNodes` / `rawEdges` are seeded in-memory.

| # | Scenario |
|---|----------|
| 1 | `hideTests` is `true` after store initialisation |
| 2 | `toggleHideTests()` flips `hideTests` from `true` to `false` and back |
| 3 | Nodes with `type === 'test'` are excluded from `filteredNodes` when `hideTests` is `true` and no type filter is active |
| 4 | Test nodes are included in `filteredNodes` after `toggleHideTests()` sets `hideTests` to `false` |
| 5 | Test nodes appear in `filteredNodes` when `hideTests` is `true` but the active type filter includes `'test'` (type-filter override) |
| 6 | Test nodes remain excluded when `hideTests` is `true` and the type filter does not include `'test'` (partial override — both mechanisms exclude them) |
| 7 | Edges whose source or target is a hidden test node are excluded from `filteredEdges`; edges between two visible non-test nodes remain |
| 8 | Toggling `hideTests` does not affect `hideTerminal`, and vice versa |

---

## Milestone 2 — Component tests (7 tests)

Mounts `GraphFilters.vue` with minimal required props using `@vue/test-utils`.
Asserts on the rendered DOM via CSS selectors and the component's emitted events.

| # | Scenario |
|---|----------|
| 1 | A `<label>` with text "Show tests" wrapping an `<input type="checkbox">` is present |
| 2 | The checkbox is **unchecked** when the `hideTests` prop is `true` (`:checked="!hideTests"`) |
| 3 | The checkbox is **checked** when the `hideTests` prop is `false` |
| 4 | Triggering the `change` event on the checkbox emits `toggleHideTests` |
| 5 | The checkbox is keyboard accessible: `type="checkbox"`, not disabled, and responds to the `change` event (which is fired by Space/Enter on native checkboxes) |
| 6 | The checkbox `<input>` is wrapped inside a `<label>` element (`closest('label')` is non-null) |
| (AC) | Visual consistency: "Show tests" and "Show completed" live in the same `.filter-group` container and share `toggle-label` / `toggle-input` CSS classes |

---

## Milestone 3 — Integration smoke tests (5 tests)

Tests 1–4 verify store-level filtering (the source of truth consumed by both
`Graph2DView` and `ForceGraph3D` as `:nodes` / `:edges` props).  Test 5 mounts
`GraphView.vue` with mocked API/WS and verifies the `onMounted` reset.

| # | Scenario |
|---|----------|
| 1 | **2D view — hidden on load**: `store.filteredNodes` excludes test nodes when `hideTests` defaults to `true` |
| 2 | **2D view — shown after toggle**: after `toggleHideTests()`, test nodes and their edges appear in `filteredNodes` / `filteredEdges` |
| 3 | **3D view — hidden on load**: `store.augmentedNodes` (passed to `ForceGraph3D`) excludes test nodes by default |
| 4 | **3D view — shown after toggle**: after `toggleHideTests()`, test nodes appear in `augmentedNodes` |
| 5 | **Toggle state resets on navigation**: mounting `GraphView` with a controlled Pinia where `hideTests` was pre-set to `false` results in `store.hideTests === true` after `flushPromises()`, confirming the `onMounted` hook resets the flag |

---

## Mocking strategy

- `@/api/graph` → mocked with `vi.mock` to return `{ nodes: [], edges: [] }`;
  prevents real HTTP calls.
- `@/composables/useWebSocket` → mocked to a no-op; prevents WebSocket
  connections during GraphView mount.
- `GraphFilters`, `GraphLegend`, `ArtifactModal`, `LabelModal`,
  `StatusCheckPanel`, `ForceGraph3D` → stubbed via `global.stubs` in the
  GraphView mount test so only the `onMounted` reset logic is exercised.
