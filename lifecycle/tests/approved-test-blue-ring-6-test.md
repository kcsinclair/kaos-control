---
title: "Tests — Blue Ring Indicator for Approved Test Artifacts"
type: test
status: draft
lineage: approved-test-blue-ring
parent: lifecycle/test-plans/approved-test-blue-ring-5-test.md
created: "2026-04-29"
---

# Tests — Blue Ring Indicator for Approved Test Artifacts

## Overview

This artifact documents the automated test suite that verifies the blue ring visual
indicator applied to graph nodes whose `type === 'test'` and `status === 'approved'`
in both the 2D (Cytoscape.js) and 3D (three.js) graph views.

Three test files were created under `tests/web/`:

---

## Test files

### `tests/web/graphConstants.test.ts`

**Covers:** Milestone 1 — Unit tests for `graphConstants.ts`.

Verifies that `APPROVED_TEST_RING_COLOR` is exported from
`web/src/components/graph/graphConstants.ts` and meets the visual-distinction
requirements laid out in the frontend plan:

- The exported value is a string matching the 6-digit hex pattern `#rrggbb`.
- It differs from `NODE_COLORS.test` (cyan `#06b6d4`).
- It differs from `PRIORITY_COLORS.low` (blue `#3b82f6`).
- It differs from `ACTIVE_STATUS_COLORS.clarifying` (light blue `#60a5fa`).

These tests act as a regression guard against accidental removal or colour collision.

---

### `tests/web/Graph2DView.approvedRing.test.ts`

**Covers:** Milestone 2 — 2D Cytoscape.js ring logic in `Graph2DView.vue`.

Testing approach: `cytoscape` and `cytoscape-fcose` are mocked (intercepting the
dynamic imports in `init()`). The Cytoscape constructor mock records the full
options object, allowing direct assertion on the `style` array.
`graphConstants` is also mocked so `APPROVED_TEST_RING_COLOR` has a known sentinel
value and `ACTIVE_STATUS_COLORS` includes `'approved'` (making the pulse-loop guard
exercisable).

**Scenarios covered:**

1. **Approved-test selector present** — the style array includes an entry with
   selector `node[type="test"][status="approved"]`, `border-color` equal to
   `APPROVED_TEST_RING_COLOR`, and `border-width` of `4`.
2. **Non-approved test is unaffected** — `APPROVED_TEST_RING_COLOR` does not appear
   in any other Cytoscape style rule; no overly-broad selector (e.g.
   `node[type="test"]` without a status constraint) carries the blue colour.
3. **Non-test approved node is unaffected** — no selector lacking the `type="test"`
   constraint uses `APPROVED_TEST_RING_COLOR` as its border colour.
4. **Pulse loop guard** — after advancing fake timers past the 700 ms interval, a
   mock cy-node with `type=test` / `status=approved` has its `.style()` method
   never called; in contrast, an `in-qa` test node's `.style()` is called normally.

---

### `tests/web/ForceGraph3D.approvedRing.test.ts`

**Covers:** Milestone 3 — 3D three.js ring logic in `ForceGraph3D.vue`.

Testing approach: `3d-force-graph` is mocked to capture the `nodeThreeObject` and
`onEngineTick` callbacks registered during component mount. `three` is replaced
with lightweight class stand-ins (`MockGroup`, `MockTorusGeometry`,
`MockMeshLambertMaterial`, `MockMesh`, etc.) that record constructor arguments.
`buildNodeObject` is exercised indirectly by calling the captured
`nodeThreeObject` callback with controlled `GraphNode` fixtures.

**Scenarios covered:**

1. **Approved test gets a blue torus** — the returned `THREE.Group` contains a
   `Mesh` with `TorusGeometry` whose `MeshLambertMaterial` colour is
   `APPROVED_TEST_RING_COLOR`. The material is non-transparent.
2. **Torus is static (not in activeRings)** — firing the `onEngineTick` callback
   does not call `scale.setScalar` on the approved-test torus mesh, proving it was
   not registered in the `activeRings` animation map.
3. **Approved test with priority gets two rings** — a node with `priority: 'high'`
   produces two torus meshes at different radii: one red (priority) and one blue
   (approved-test).
4. **Non-approved test gets no blue torus** — nodes with `type=test` /
   `status=in-qa` and `type=test` / `status=draft` produce no torus with
   `APPROVED_TEST_RING_COLOR`.
5. **Non-test approved node gets no blue torus** — nodes with
   `type=requirement` / `status=approved` and `type=defect` / `status=approved`
   produce no torus with `APPROVED_TEST_RING_COLOR`.

---

## Dependency on implementation

These tests are written ahead of the frontend implementation (TDD).  They will
**fail** until `approved-test-blue-ring-4-fe` is implemented:

- `APPROVED_TEST_RING_COLOR` exported from `graphConstants.ts`
- `node[type="test"][status="approved"]` style rule added to `Graph2DView.vue`
- Pulse-loop guard added to `Graph2DView.vue`
- Approved-test torus added to `buildNodeObject` in `ForceGraph3D.vue`

Once the implementation is merged, `pnpm test` (run from `tests/web/`) should
show all suites passing.

## Manual verification

Milestone 4 of the test plan covers visual spot-checks that complement these
automated tests.  Those must be performed by the QA agent or developer after
deploying to a dev environment.
