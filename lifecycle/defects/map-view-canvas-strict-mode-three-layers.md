---
title: Map View — locator('canvas') Fails in Strict Mode; Cytoscape Renders Three Canvas Layers
type: defect
status: in-development
lineage: map-view-canvas-strict-mode-three-layers
created: "2026-05-16T14:00:00+10:00"
priority: high
labels:
    - defect
    - test
    - map
    - cytoscape
release: KC-Release2
assignees:
    - role: test-developer
      who: agent
---

# Map View — locator('canvas') Fails in Strict Mode; Cytoscape Renders Three Canvas Layers

## Reproduction Steps

1. Run the E2E suite.
2. Observe failures in `Flow 05 — Graph node click` and all three tests in `Flow 09 — Graph rendering for doc nodes`.

## Expected Behaviour

The tests target the Cytoscape 2D map canvas and proceed with node inspection and click interactions.

## Actual Behaviour

All four failing tests fail at the same locator:

```
await expect(page.locator('canvas')).toBeVisible({ timeout: 15_000 })
```

Error:

```
Error: expect(locator).toBeVisible() failed
Locator: locator('canvas')
strict mode violation: locator('canvas') resolved to 3 elements:
  1) <canvas data-id="layer0-selectbox">
  2) <canvas data-id="layer1-drag">
  3) <canvas data-id="layer2-node">
```

Cytoscape.js renders three internal canvas layers (`layer0-selectbox`, `layer1-drag`, `layer2-node`). The test locator `page.locator('canvas')` is ambiguous in strict mode and immediately throws.

Failing tests:
- `flows/05-graph-click.spec.ts:12` — `Flow 05 — Graph node click`
- `flows/09-doc-graph.spec.ts:16` — `TC1: doc node exists in the 2D map view`
- `flows/09-doc-graph.spec.ts:51` — `TC2: an edge connects the doc node to its parent artifact`
- `flows/09-doc-graph.spec.ts:97` — `TC3: doc node uses a distinct colour from idea and requirement nodes`

## Fix

Update all four tests to target the primary drawing layer specifically:

```ts
// Before
await expect(page.locator('canvas')).toBeVisible({ timeout: 15_000 })

// After
await expect(page.locator('canvas[data-id="layer2-node"]')).toBeVisible({ timeout: 15_000 })
```
