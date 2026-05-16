---
title: "Test: Map View Canvas Locator — Strict Mode Fix"
type: test
status: draft
lineage: map-view-canvas-strict-mode-three-layers
parent: lifecycle/defects/map-view-canvas-strict-mode-three-layers.md
---

# Test: Map View Canvas Locator — Strict Mode Fix

Fixes four E2E tests that were failing due to Playwright strict-mode rejection
of the ambiguous `locator('canvas')` selector. Cytoscape.js renders three
internal canvas layers (`layer0-selectbox`, `layer1-drag`, `layer2-node`), so
the generic selector resolves to three elements and throws immediately. All four
tests are updated to target the primary drawing layer with
`locator('canvas[data-id="layer2-node"]')`.

## Scenarios Covered

### `tests/e2e/flows/05-graph-click.spec.ts`

| Test | Change |
|---|---|
| `Flow 05 — Graph node click` | Canvas readiness check updated from `locator('canvas')` to `locator('canvas[data-id="layer2-node"]')`. Remainder of the test (node count assertion, tap-to-navigate flow) is unchanged. |

### `tests/e2e/flows/09-doc-graph.spec.ts`

| Test | Change |
|---|---|
| `TC1: doc node exists in the 2D map view` | Canvas readiness check updated. |
| `TC2: an edge connects the doc node to its parent artifact` | Canvas readiness check updated. |
| `TC3: doc node uses a distinct colour from idea and requirement nodes` | Canvas readiness check updated. |

All three TC tests in Flow 09 still exercise the full scenario — graph
rendering, node existence, edge existence, and per-type colour distinctness —
only the initial canvas-visibility guard is narrowed to the specific layer.

## Test Files

- `tests/e2e/flows/05-graph-click.spec.ts` — Flow 05 graph node click
- `tests/e2e/flows/09-doc-graph.spec.ts` — Flow 09 doc node graph rendering
