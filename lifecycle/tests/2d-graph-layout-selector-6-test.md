---
title: "Tests — 2D Graph Layout Selector"
type: test
status: draft
lineage: 2d-graph-layout-selector
parent: lifecycle/test-plans/2d-graph-layout-selector-5-test.md
created: "2026-05-07T00:00:00+10:00"
---

# Tests — 2D Graph Layout Selector

## Overview

This artifact documents the automated test suite covering the 2D graph layout
selector feature.  Tests are written in TypeScript using Vitest + `@vue/test-utils`
and run from the `tests/web/` directory.

All component and integration tests run in a happy-dom environment with Cytoscape
mocked (no real DOM rendering of graph nodes).  Milestone 7 (bundle verification)
reads `web/dist/assets/` and requires `make build-web` to be run first.

Milestone 5 (E2E with Playwright/Cypress) is **not implemented**: neither
Playwright nor Cypress is configured in this repository.  A separate task is
required to introduce E2E infrastructure before those scenarios can be automated.

---

## Milestone 1 — LayoutSelector component unit tests

**File:** `tests/web/LayoutSelector.spec.ts`

| # | Scenario |
|---|----------|
| 1 | `<select>` renders with exactly five `<option>` elements |
| 2 | Options include all five keys: `fcose`, `breadthfirst`, `concentric`, `circle`, `dagre` |
| 3 | Changing the select calls `store.setLayout()` with the selected key |
| 4 | Select value reflects `store.activeLayout` on mount (default `fcose`) |
| 5 | Select value reflects `store.activeLayout` when set to `circle` or `dagre` before mount |
| 6 | Clicking the Directed button calls `store.toggleDirected()` |
| 7 | Directed button has no `active` class when `store.directed` is `false` |
| 8 | Directed button has `active` class and `aria-pressed="true"` when `store.directed` is `true` |
| 9 | `<select>` has an `aria-label` attribute |
| 10 | Directed button has `aria-label` and `aria-pressed` attributes |
| 11 | `<select>` is not disabled when `store.layoutAnimating` is `false` |
| 12 | `<select>` and Directed button are disabled when `store.layoutAnimating` is `true` |
| 13 | `<select>` is a native `SELECT` element (keyboard-accessible by default) |

---

## Milestone 2 — Graph store layout state unit tests

**File:** `tests/web/graph.store.layout.spec.ts`

| # | Scenario |
|---|----------|
| 1 | `activeLayout` defaults to `'fcose'` |
| 2 | `directed` defaults to `false` |
| 3 | `layoutAnimating` defaults to `false` |
| 4–8 | `setLayout()` with each valid key updates `activeLayout` |
| 9–12 | `setLayout()` with invalid keys (`'invalid-key'`, `''`, `'FCOSE'`, `'force-directed'`) does not change state |
| 13 | `toggleDirected()` changes `directed` from `false` to `true` |
| 14 | `toggleDirected()` changes `directed` from `true` back to `false` |
| 15 | Three calls to `toggleDirected()` end with `directed === true` |
| 16 | `activeLayout` and `directed` persist when the same Pinia instance is accessed again (store singleton) |

---

## Milestone 3 — Graph2DView layout switching integration tests

**File:** `tests/web/Graph2DView.layout.spec.ts`

| # | Scenario |
|---|----------|
| 1 | `cy.layout()` is called at least once on mount |
| 2 | First `cy.layout()` call uses `name: 'fcose'` |
| 3 | `fcose` plugin is registered via `Cy.use()` on mount |
| 4 | After `store.setLayout('concentric')`, `cy.layout()` is called with `name: 'concentric'` |
| 5 | `cy.layout().run()` is called on layout change |
| 6 | Concentric layout call includes `animate: true` and `animationDuration` |
| 7 | `store.setLayout('dagre')` → `cy.layout()` called with `name: 'dagre'` |
| 8 | `Cy.use()` is called with the dagre plugin when dagre is first selected |
| 9 | Dagre plugin is NOT registered a second time on repeated dagre selections |
| 10 | `store.toggleDirected()` causes `cy.layout()` to be called again |
| 11 | Layout call after `toggleDirected()` includes `directed: true` |
| 12 | Layout call after second `toggleDirected()` includes `directed: false` |
| 13 | Animated layout change uses `animate: true` and numeric `animationDuration` |
| 14 | Initial mount layout uses `animate: false` |
| 15 | Cytoscape constructor is called only once across multiple layout changes |
| 16 | `cy.destroy()` is NOT called on layout change — only on component unmount |

---

## Milestone 4 — Filter + layout interaction tests

**File:** `tests/web/Graph2DView.filters.spec.ts`

| # | Scenario |
|---|----------|
| 1 | With `activeLayout = 'circle'`, updating the `nodes` prop triggers relayout with `name: 'circle'` |
| 2 | Circle filter relayout does NOT use `name: 'fcose'` |
| 3 | Circle filter relayout uses `animate: false` (the `update()` path) |
| 4 | With `activeLayout = 'breadthfirst'`, updating `nodes` triggers relayout with `name: 'breadthfirst'` |
| 5 | Breadthfirst filter relayout includes `spacingFactor` and `avoidOverlap` options |
| 6 | Changing `matchedNodeIds` does NOT call `cy.layout()` |
| 7 | Changing `matchedNodeIds` does not increment `layoutCallOptions` count |
| 8 | Clearing `matchedNodeIds` (empty set) does not call `cy.layout()` |
| 9 | Simultaneous `nodes` + `matchedNodeIds` change uses current layout (concentric) |

---

## Milestone 5 — E2E tests (not implemented)

End-to-end Playwright/Cypress tests are **not implemented** because neither
testing framework is configured in this repository.  The following scenarios
from the test plan remain open:

- Navigate to graph view → 2D toggle → layout selector is visible.
- Switch to 3D view → layout selector is hidden.
- Select "Hierarchical" → nodes visibly reposition.
- Select "Circle" → nodes arrange in circular pattern.
- Navigate away → back → previously selected layout is restored.
- Apply filter while "Concentric" selected → relayouts in concentric.
- Toggle directed → graph updates layout.
- Keyboard-only interaction.

To implement these, add Playwright (or Cypress) to the project and create
`tests/e2e/graph-layout-selector.spec.ts`.

---

## Milestone 6 — Performance tests

**File:** `tests/web/Graph2DView.perf.spec.ts`

| # | Scenario |
|---|----------|
| 1–5 | Each of the five layout keys completes `cy.layout().run()` in < 2000 ms with 200 nodes / 400 edges |
| 6–10 | Each of the five layout keys completes in < 2000 ms with 500 nodes / 1000 edges |
| 11 | Switching through all 5 layouts rapidly does not throw or leave unhandled rejections |
| 12 | Cytoscape instance is not re-created after 5 rapid layout switches |
| 13 | `cy.stop()` is called before each layout to cancel in-progress animations |

**Note:** Cytoscape is mocked in these tests. The mock completes synchronously, so
the elapsed-time assertions verify component orchestration overhead (store watch →
`runLayout()` → `cy.layout()`), not the actual layout algorithm performance.
Real-scale performance testing at 200–500 nodes requires a headful browser
environment (browser-mode Vitest or Playwright).

---

## Milestone 7 — Bundle size / dagre code-split check

**File:** `tests/web/bundle/dagre-codesplit.spec.ts`

| # | Scenario |
|---|----------|
| 1 | Main chunk does not contain `"cytoscape-dagre"` string |
| 2 | Main chunk does not contain `"graphlib"` (dagre dependency) |
| 3 | At least one non-main chunk contains a dagre fingerprint string |
| 4 | The dagre chunk is a different file from the main chunk |
| 5 | Total JS chunk count is between 2 and 30 |

**Prerequisite:** Tests skip gracefully when `web/dist/assets/` does not exist.
Run `make build-web` before executing Milestone 7 tests.

---

## Mocking strategy

- `@/api/graph` → `vi.mock` returning `{ nodes: [], edges: [] }` — prevents HTTP calls.
- `@/composables/useWebSocket` → no-op mock — prevents WebSocket connections.
- `cytoscape` → mock constructor that captures `cy.layout()` calls, tracks options passed.
- `cytoscape-fcose` → `{ default: { name: 'fcoseMock' } }` — tracks `Cy.use()` registration.
- `cytoscape-dagre` → `{ default: { name: 'dagreMock' } }` — tracks lazy registration on first dagre selection.
- Mock layout objects fire `'layoutstop'` synchronously so `store.layoutAnimating` resets immediately.
