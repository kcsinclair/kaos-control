---
title: "Test Plan — Blue Ring Indicator for Approved Test Artifacts"
type: plan-test
status: approved
lineage: approved-test-blue-ring
parent: lifecycle/requirements/approved-test-blue-ring-2.md
created: "2026-04-28"
---

# Test Plan — Blue Ring Indicator for Approved Test Artifacts

## Overview

Verify that the approved-test blue ring renders correctly in both 2D and 3D graph views, applies only to the correct combination of `type === 'test'` and `status === 'approved'`, does not animate, and does not regress existing ring visuals.

## Milestone 1: Unit tests for `graphConstants.ts`

**Description:** Add a small test to verify the `APPROVED_TEST_RING_COLOR` constant exists and has the expected value, guarding against accidental removal or modification.

**Files to change:**
- `tests/web/graphConstants.test.ts` (new file)

**Acceptance criteria:**
- [ ] Test imports `APPROVED_TEST_RING_COLOR` from `@/components/graph/graphConstants`.
- [ ] Test asserts the value is a valid hex colour string (e.g. `#2563eb`).
- [ ] Test asserts it differs from `NODE_COLORS.test`, `PRIORITY_COLORS.low`, and `ACTIVE_STATUS_COLORS.clarifying` to guarantee visual distinction.
- [ ] `pnpm test` passes.

## Milestone 2: Component tests for `Graph2DView.vue` — 2D ring logic

**Description:** Write component-level tests (vitest + happy-dom) that mount `Graph2DView` with controlled node data and verify the Cytoscape style rules apply correctly. Since Cytoscape requires a real DOM container, these tests use a shallow approach: mock `cytoscape` and assert the style array passed to it contains the approved-test selector with the correct properties.

**Files to change:**
- `tests/web/Graph2DView.approvedRing.test.ts` (new file)

**Test cases:**
1. **Approved test node gets blue ring style:** Provide a node with `type: 'test', status: 'approved'`. Assert the Cytoscape style array includes a selector matching `node[type="test"][status="approved"]` with `border-color` equal to `APPROVED_TEST_RING_COLOR` and `border-width: 4`.
2. **Non-approved test node does not match:** Provide a node with `type: 'test', status: 'draft'`. Assert the approved-test selector does not override its border.
3. **Non-test approved node does not match:** Provide a node with `type: 'requirement', status: 'approved'`. Assert the approved-test selector does not apply.
4. **Pulse loop skips approved test nodes:** Verify that the `setInterval` callback does not mutate border style on a `test`/`approved` node (inspect the guard condition).

**Acceptance criteria:**
- [ ] All four test cases pass.
- [ ] Tests do not depend on a visible browser DOM (happy-dom is sufficient).
- [ ] `pnpm test` passes.

## Milestone 3: Component tests for `ForceGraph3D.vue` — 3D ring logic

**Description:** Write tests that verify `buildNodeObject` produces the correct Three.js group structure for approved test nodes. Since `ForceGraph3D` depends on WebGL, isolate `buildNodeObject` for testing (or extract it if not already testable).

**Files to change:**
- `tests/web/ForceGraph3D.approvedRing.test.ts` (new file)

**Test cases:**
1. **Approved test node gets a static torus:** Call `buildNodeObject` with `{ type: 'test', status: 'approved' }`. Assert the returned group contains a `THREE.Mesh` with `TorusGeometry` and material colour matching `APPROVED_TEST_RING_COLOR`.
2. **Torus is not in `activeRings`:** Assert the mesh is not added to the `activeRings` map (no animation).
3. **Approved test with priority gets two rings:** Provide `{ type: 'test', status: 'approved', priority: 'high' }`. Assert the group contains two torus meshes at different radii.
4. **Non-approved test node — no blue torus:** Provide `{ type: 'test', status: 'in-qa' }`. Assert no torus with the approved-test colour exists in the group.
5. **Non-test approved node — no blue torus:** Provide `{ type: 'requirement', status: 'approved' }`. Assert no torus with the approved-test colour exists.

**Acceptance criteria:**
- [ ] All five test cases pass.
- [ ] Tests mock Three.js minimally (geometry + material assertions only).
- [ ] `pnpm test` passes.

## Milestone 4: Manual visual verification checklist

**Description:** Manual spot-checks to be performed by the QA agent or developer after deploying to a dev environment. These complement automated tests by verifying rendering fidelity on an actual canvas.

**Checklist:**
- [ ] In 2D graph: an `approved` test artifact shows a blue border ring, clearly distinct from the cyan fill.
- [ ] In 3D graph: the same artifact shows a blue torus ring that does not pulse or scale.
- [ ] A `draft` test artifact shows no blue ring in either view.
- [ ] An `in-qa` test artifact shows its active-status pulse but no blue ring.
- [ ] An `approved` requirement shows no blue ring.
- [ ] An `approved` test with `priority: high` shows both the red priority ring and the blue approved-test ring (both visible, not overlapping destructively).
- [ ] Switching between 2D and 3D views preserves correct ring display.
- [ ] No visual regression on other node types or statuses.

**Files to change:** None (manual verification).

**Acceptance criteria:**
- [ ] All checklist items confirmed visually in both 2D and 3D views.

## Milestone 5: Lifecycle test artifact

**Description:** Create the lifecycle test artifact documenting what the test code covers, per project convention.

**Files to change:**
- `lifecycle/tests/approved-test-blue-ring-6-test.md` (new file, next lineage index)

**Acceptance criteria:**
- [ ] Artifact has correct frontmatter (`type: test`, `status: draft`, `lineage: approved-test-blue-ring`, `parent` pointing to this test plan).
- [ ] Body summarises the test files and what they cover.

## Cross-references

- [[approved-test-blue-ring]] — frontend plan ([[approved-test-blue-ring-4-fe]]) defines the implementation this plan verifies.
- [[approved-test-blue-ring]] — backend plan ([[approved-test-blue-ring-3-be]]) confirms no backend testing needed.
