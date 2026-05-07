---
title: Tests — Fix stale graphConstants mock in ForceGraph3D and Graph2DView approved-ring suites
type: test
status: draft
lineage: light-mode-graphs
parent: lifecycle/defects/light-mode-graphs-7-defect.md
---

# Tests — Fix stale graphConstants mock in ForceGraph3D and Graph2DView approved-ring suites

## Overview

This artifact documents the fixes applied to two existing test suites that were
broken by the light-mode-graphs refactor.  The refactor replaced bare constant
exports (`NODE_COLORS`, `PRIORITY_COLORS`, etc.) in `graphConstants.ts` with a
single `useGraphTheme()` composable.  Both test files mocked the old API, causing
all 21 tests to fail immediately with a "No useGraphTheme export is defined" error.

No new test scenarios were added.  The scope of this work is limited to bringing
the existing 21 tests back to green by updating the mocks.

---

## Files modified

- `tests/web/ForceGraph3D.approvedRing.test.ts`
- `tests/web/Graph2DView.approvedRing.test.ts`

---

## Changes made

### `vi.mock('@/components/graph/graphConstants')` — both files

Replaced the old factory (which exported `NODE_COLORS`, `PRIORITY_COLORS`,
`ACTIVE_STATUS_COLORS`, `EDGE_COLORS`, `APPROVED_TEST_RING_COLOR` as bare
constants) with a new async factory that exports `useGraphTheme` as a function
returning `{ palette: computed(() => ({ … })), isDark: ref(true) }`.

The `palette` computed value includes all fields required by `GraphPalette`,
matching the dark-mode defaults.  In `Graph2DView.approvedRing.test.ts`,
`activeStatusColors` additionally includes `approved: '#00ff00'` so that the
pulse-loop guard test (test 4) can exercise the `type=test && status=approved`
skip path when `approved` is an active-status colour.

### `ForceGraph3D.approvedRing.test.ts` — mock graph instance

Added `linkLabel: fluent()` to the fluent mock instance for the `3d-force-graph`
library.  The `ForceGraph3D.vue` component gained a `.linkLabel()` call as part
of the light-mode-graphs implementation; the missing stub caused the fluent chain
to break before `.onEngineTick()` was registered.

### `Graph2DView.approvedRing.test.ts` — Pinia setup and cy mock fixes

1. Imported `setActivePinia` and `createPinia` from `pinia`; called
   `setActivePinia(createPinia())` in `beforeEach` so that `useGraphStore()`
   (called at the top of `Graph2DView.vue`'s `<script setup>`) finds an active
   Pinia instance.

2. Added `stop: vi.fn()` to `mockCyInstance`.  `Graph2DView.vue`'s `runLayout()`
   calls `cy.stop()` before starting a new layout; the missing stub threw inside
   the async `onMounted` handler.

3. Changed `mockCyInstance.layout` from `.mockReturnValue({ run: vi.fn() })` to
   a factory that returns `{ one: vi.fn((_e, cb) => cb()), run: vi.fn() }`.
   `runLayout()` calls `layout.one('layoutstop', cb)` to reset
   `graphStore.layoutAnimating`; the missing `one` method was the root cause of
   `init()` throwing before `pulseInterval = setInterval(…)` was reached.

---

## Scenarios covered

The test files themselves have not changed in terms of test scenarios; they are
documented in `lifecycle/tests/light-mode-graphs-6-test.md` Milestones 2 and 3.
This artifact records the mock updates needed to make those scenarios runnable
against the post-refactor module API.

### ForceGraph3D — 12 tests, all passing

1. `buildNodeObject` returns a group containing at least one torus mesh for a
   `type=test, status=approved` node.
2. The torus mesh has material colour matching `APPROVED_TEST_RING_COLOR`.
3. The approved-test torus uses `TorusGeometry` (not `SphereGeometry`).
4. The approved-test torus material is not transparent.
5. `onEngineTick` does not call `scale.setScalar` on the approved-test torus
   (it is not in `activeRings`).
6. A `type=test, status=approved` node with `priority=high` produces two torus
   meshes.
7. One torus has the priority colour and one has `APPROVED_TEST_RING_COLOR`.
8. The two torus rings have different radii.
9. `type=test, status=in-qa` produces no approved-test coloured torus.
10. `type=test, status=draft` produces no approved-test coloured torus.
11. `type=requirement, status=approved` produces no approved-test coloured torus.
12. `type=defect, status=approved` produces no approved-test coloured torus.

### Graph2DView — 9 tests, all passing

1. Cytoscape style array includes a rule for `node[type="test"][status="approved"]`.
2. That rule has `border-color` equal to `APPROVED_TEST_RING_COLOR`.
3. That rule has `border-width` of 4.
4. `APPROVED_TEST_RING_COLOR` does not appear in any other style rule.
5. No style rule targets `type=test` without also requiring `status=approved`.
6. `APPROVED_TEST_RING_COLOR` does not appear in a selector without the
   `type=test` constraint.
7. Only the exact combined selector uses `APPROVED_TEST_RING_COLOR`.
8. The pulse loop does not call `.style()` on a `type=test, status=approved`
   node (guard check).
9. The pulse loop does call `.style()` on a `type=test, status=in-qa` node
   (guard does not over-block active statuses).
