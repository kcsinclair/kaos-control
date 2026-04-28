---
title: Blue Ring Indicator for Approved Test Artifacts
type: requirement
status: draft
lineage: approved-test-blue-ring
parent: lifecycle/ideas/approved-test-blue-ring.md
created: "2026-04-28"
priority: medium
labels:
  - frontend
  - feature
  - vue
---

# Blue Ring Indicator for Approved Test Artifacts

## Problem

Test artifacts that have reached the `approved` status are not visually distinguishable from other test nodes in the 2D (Cytoscape.js) or 3D (3d-force-graph) map views. Reviewers and the QA agent must open individual artifacts to determine whether a test has passed approval, which slows at-a-glance assessment of test-coverage health across a lineage.

Currently the graph views render visual rings for two concerns only:

1. **Priority rings** — a coloured border (2D) or torus (3D) derived from `PRIORITY_COLORS` (high / medium / normal / low).
2. **Active-status pulse rings** — an animated border (2D) or pulsing torus (3D) for statuses in `ACTIVE_STATUS_COLORS` (`in-development`, `in-qa`, `in-progress`, `clarifying`, `planning`).

The `approved` status is absent from both maps, so approved test nodes receive no special visual treatment.

## Goals / Non-goals

### Goals

- G1: Render a clearly visible ring around every node whose `type === 'test'` **and** `status === 'approved'` in both the 2D and 3D graph views.
- G2: The ring colour must not be confused with existing priority rings or active-status pulse rings.
- G3: The ring must not animate or pulse — it represents a settled state, not an in-progress one.
- G4: The implementation must be additive: it must not alter the appearance of non-test nodes or test nodes in other statuses.

### Non-goals

- NG1: Applying a ring to non-test artifact types that happen to be `approved` (e.g. approved requirements). This may be added later but is out of scope.
- NG2: Adding a legend or key to the graph views (desirable but separate work).
- NG3: Modifying node fill colour — the ring is an overlay on the existing type-based fill.

## Detailed Requirements

### Functional

- **FR-1**: In `graphConstants.ts`, introduce a constant (e.g. `APPROVED_TEST_RING_COLOR`) set to a blue shade that is visually distinct from:
  - The `test` node fill colour (cyan `#06b6d4`).
  - The `low` priority ring colour (blue `#3b82f6`).
  - The `clarifying` active-status colour (blue `#60a5fa`).
  A recommended value is `#2563eb` (Tailwind blue-600) or `#1d4ed8` (blue-700), but the implementer should verify contrast on the dark background (`#0f172a`).

- **FR-2 (2D — Cytoscape.js)**: Add a Cytoscape style rule in `Graph2DView.vue` that targets `node[type="test"][status="approved"]` and applies:
  - `border-color`: the approved-test ring colour.
  - `border-width`: 4 (matching the existing priority ring width).
  This rule must have higher specificity than the default node border (`1.5px rgba(255,255,255,0.25)`) and must not conflict with the priority ring selector `node[priorityColor]`. If both apply (an approved test with a priority), the approved-test ring takes precedence.

- **FR-3 (3D — 3d-force-graph)**: In `ForceGraph3D.vue`, extend `buildNodeObject` so that when `n.type === 'test' && n.status === 'approved'`, a `THREE.TorusGeometry` ring is added to the node group using the approved-test ring colour. The torus radius and tube proportions should match the existing `priorityRing` helper (radius = `sphereR * 1.45`, tube = `sphereR * 0.18`). If both a priority ring and an approved-test ring apply, render the approved-test ring at a slightly larger radius (e.g. `sphereR * 1.75`) so both rings are visible.

- **FR-4**: The ring must be static (no pulse, no scale animation). In the 2D view, the `setInterval` pulse loop must skip nodes matching the approved-test selector. In the 3D view, the ring mesh must not be added to `activeRings`.

### Non-functional

- **NFR-1**: No new runtime dependencies. Use only Cytoscape style rules and Three.js primitives already imported.
- **NFR-2**: The colour constant must be defined once in `graphConstants.ts` and imported by both view components — no duplicated hex values.
- **NFR-3**: Both `pnpm exec vue-tsc --noEmit` and `pnpm build` must pass after changes.

## Acceptance Criteria

- [ ] In the 2D graph, a test artifact with `status: approved` displays a blue ring (border) that is clearly distinguishable from cyan node fill and from priority/active-status rings.
- [ ] In the 3D graph, the same artifact displays a blue torus ring that does not animate.
- [ ] Test artifacts in statuses other than `approved` (e.g. `draft`, `in-qa`, `done`) do **not** display the blue ring.
- [ ] Non-test artifacts with `status: approved` do **not** display the blue ring.
- [ ] When an approved test also has a priority, both the priority indicator and the blue ring are visible (not overwritten).
- [ ] The active-status pulse animation does not override or flicker the approved-test ring.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass cleanly.
- [ ] No visual regression in existing node styles (spot-check nodes of each type and status in both views).

Related artifacts: [[approved-test-blue-ring]]

## Open Questions

- **OQ-1**: Should the ring also appear on `done` test nodes (which were presumably approved before transitioning to done), or strictly only on `status === 'approved'`? Current spec says `approved` only.
- **OQ-2**: If a future requirement extends this pattern to other types (e.g. approved requirements), should the colour constant be named generically (e.g. `APPROVED_RING_COLOR`) now to avoid a rename later? Current recommendation: name it `APPROVED_TEST_RING_COLOR` and rename if/when scope broadens.
