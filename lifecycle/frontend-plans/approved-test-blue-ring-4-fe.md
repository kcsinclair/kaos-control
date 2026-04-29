---
title: "Frontend Plan — Blue Ring Indicator for Approved Test Artifacts"
type: plan-frontend
status: done
lineage: approved-test-blue-ring
parent: lifecycle/requirements/approved-test-blue-ring-2.md
created: "2026-04-28"
---

# Frontend Plan — Blue Ring Indicator for Approved Test Artifacts

## Overview

Add a static blue ring around graph nodes whose `type === 'test'` and `status === 'approved'` in both the 2D (Cytoscape.js) and 3D (3d-force-graph) views. The ring must not animate, must not conflict with priority or active-status rings, and must be driven by a single colour constant.

## Milestone 1: Add colour constant to `graphConstants.ts`

**Description:** Define `APPROVED_TEST_RING_COLOR` in `graphConstants.ts` so both view components can import it. The colour must be visually distinct from:
- Test node fill: cyan `#06b6d4`
- Low priority ring: blue `#3b82f6`
- Clarifying active-status: blue `#60a5fa`

Recommended value: `#2563eb` (Tailwind blue-600). Verify contrast against the dark background `#0f172a`.

**Files to change:**
- `web/src/components/graph/graphConstants.ts` — add `APPROVED_TEST_RING_COLOR` export.

**Acceptance criteria:**
- [ ] `APPROVED_TEST_RING_COLOR` is exported from `graphConstants.ts` with value `#2563eb` (or adjusted if contrast check warrants it).
- [ ] No existing constants are modified.
- [ ] `pnpm exec vue-tsc --noEmit` passes.

## Milestone 2: 2D Cytoscape.js ring in `Graph2DView.vue`

**Description:** Add a Cytoscape style rule that targets `node[type="test"][status="approved"]` and applies a static border ring using the constant from Milestone 1. This rule must:
1. Appear after the `node[priorityColor]` rule so it takes precedence for approved tests with a priority.
2. Use `border-width: 4` and `border-color: APPROVED_TEST_RING_COLOR`.
3. Not be overridden by the pulse animation — the `setInterval` pulse loop must skip nodes matching `type === 'test'` and `status === 'approved'`.

**Files to change:**
- `web/src/components/graph/Graph2DView.vue`
  - Import `APPROVED_TEST_RING_COLOR` from `./graphConstants`.
  - Add a new Cytoscape style block after the `node[priorityColor]` selector (around line 87–92).
  - In the `setInterval` callback (around line 147–155), add a guard: if `n.data('type') === 'test' && n.data('status') === 'approved'`, skip the pulse style override.

**Acceptance criteria:**
- [ ] In the 2D graph, a `test` artifact with `status: approved` displays a blue border ring.
- [ ] The ring is static — no pulse animation.
- [ ] Test artifacts in other statuses (`draft`, `in-qa`, `done`) do **not** get the blue ring.
- [ ] Non-test artifacts with `status: approved` do **not** get the blue ring.
- [ ] When an approved test also has a priority, the blue ring is visible (the approved-test rule wins over the priority rule).
- [ ] Active-status pulse animation does not flicker or override the approved-test ring.

## Milestone 3: 3D three.js ring in `ForceGraph3D.vue`

**Description:** Extend `buildNodeObject` so that when `n.type === 'test' && n.status === 'approved'`, a static `THREE.TorusGeometry` ring is added to the node group. Details:
1. Import `APPROVED_TEST_RING_COLOR` from `./graphConstants`.
2. Compute torus dimensions from `sphereR = Math.cbrt(nodeVal(n)) * 4`:
   - If a priority ring is also present (both apply), use `sphereR * 1.75` for the approved-test torus radius so both rings are visible concentrically.
   - If no priority ring, use `sphereR * 1.45` (matching the existing priority ring proportions).
   - Tube radius: `sphereR * 0.18` in both cases.
3. Use `THREE.MeshLambertMaterial` with the approved-test colour, `transparent: false`.
4. **Do not** add the mesh to the `activeRings` map — it must remain static (no scale animation from `onEngineTick`).

**Files to change:**
- `web/src/components/graph/ForceGraph3D.vue`
  - Import `APPROVED_TEST_RING_COLOR`.
  - Add approved-test ring logic inside `buildNodeObject` (after priority ring, before active ring, around line 68–71).

**Acceptance criteria:**
- [ ] In the 3D graph, a `test` artifact with `status: approved` displays a blue torus ring.
- [ ] The torus does not animate (no scale pulsing from `onEngineTick`).
- [ ] When both a priority ring and approved-test ring apply, both are visible at different radii.
- [ ] Test artifacts in other statuses do not get the blue torus.
- [ ] Non-test artifacts with `status: approved` do not get the blue torus.

## Milestone 4: Build verification

**Description:** Ensure the full frontend toolchain passes after all changes.

**Files to change:** None (verification only).

**Acceptance criteria:**
- [ ] `pnpm exec vue-tsc --noEmit` exits 0.
- [ ] `pnpm build` exits 0.
- [ ] No visual regression in existing node styles — spot-check nodes of each type and status in both 2D and 3D views.

## Cross-references

- [[approved-test-blue-ring]] — backend plan ([[approved-test-blue-ring-3-be]]) confirms no backend changes needed.
- [[approved-test-blue-ring]] — test plan ([[approved-test-blue-ring-5-test]]) covers automated and manual verification.
