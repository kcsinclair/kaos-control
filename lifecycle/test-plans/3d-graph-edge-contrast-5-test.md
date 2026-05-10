---
title: "Test Plan: Improve Edge Line Contrast in 3D Graph"
type: plan-test
status: in-development
lineage: 3d-graph-edge-contrast
parent: lifecycle/requirements/3d-graph-edge-contrast-2.md
---

# Test Plan: Improve Edge Line Contrast in 3D Graph

## Summary

Verify that the updated edge colours meet WCAG 3:1 contrast, link widths establish the correct hierarchy, opacity is applied correctly, theme toggling works, and no regressions are introduced.

---

## Milestone 1: Contrast Ratio Verification (Unit)

### Description

Write a unit test that validates the contrast ratios of all edge colours against their respective canvas backgrounds in both palettes. This is a pure computation test — no DOM or canvas needed.

### Files to Change

- `web/src/components/graph/__tests__/graphConstants.spec.ts` (new file)

### Implementation

Import `DARK_PALETTE` and `LIGHT_PALETTE` (may need to export them or test via `useGraphTheme`). For each palette, compute the relative luminance contrast ratio (WCAG formula) of every `edgeColors` entry against `canvasBg`. Assert >= 3.0.

### Acceptance Criteria

- [ ] Test asserts >= 3:1 contrast for every edge colour in dark palette against `#0f172a`.
- [ ] Test asserts >= 3:1 contrast for every edge colour in light palette against `#ffffff`.
- [ ] Test fails if a future colour change violates the minimum contrast.

---

## Milestone 2: Palette Consistency Verification (Unit)

### Description

Assert that named edge colour properties (`timelineEdgeColor`, `assignedEdgeColor`) are identical to their corresponding `edgeColors` map entries within each palette.

### Files to Change

- `web/src/components/graph/__tests__/graphConstants.spec.ts`

### Acceptance Criteria

- [ ] `palette.edgeColors.timeline === palette.timelineEdgeColor` for both palettes.
- [ ] `palette.edgeColors.assigned === palette.assignedEdgeColor` for both palettes.

---

## Milestone 3: Link Width Hierarchy Verification (Unit)

### Description

Test the `linkWidth` callback logic to confirm the correct hierarchy: `timeline` > semantic edges > `assigned`.

### Files to Change

- `web/src/components/graph/__tests__/ForceGraph3D.spec.ts` (new or existing)

### Implementation

Extract the `linkWidth` logic into a testable function (or test it inline by mocking edge objects). Assert:

- `linkWidth({ kind: 'timeline' }) >= 2.0`
- `linkWidth({ kind: 'parent' }) >= 1.0`
- `linkWidth({ kind: 'assigned' }) >= 0.6`
- `linkWidth({ kind: 'timeline' }) > linkWidth({ kind: 'parent' }) > linkWidth({ kind: 'assigned' })`

### Acceptance Criteria

- [ ] Width hierarchy assertion passes.
- [ ] All edge kinds return a width >= their specified minimum.

---

## Milestone 4: Visual Regression — Manual Test Script

### Description

Document a manual verification procedure for QA to confirm visual correctness on a graph with mixed edge kinds.

### Files to Change

- `tests/manual/3d-graph-edge-contrast.md` (new file)

### Test Steps

1. Load a project with >= 20 artifacts and mixed edge kinds (parent, depends_on, related_to, timeline, assigned).
2. In dark theme: confirm all edges are clearly visible against the dark background.
3. In light theme: confirm all edges are clearly visible against the white background.
4. Verify timeline edges are visually most prominent (thickest, fully opaque).
5. Verify assigned edges are visually least prominent (thinnest, slightly transparent).
6. Toggle theme: confirm edges update immediately without page reload or layout jitter.
7. Verify no change to node colours, sizes, or label rendering.
8. Verify force simulation does not restart or alter node positions on theme toggle.

### Acceptance Criteria

- [ ] All 8 manual steps pass.
- [ ] QA confirms no visual regression on nodes or layout.

---

## Milestone 5: Performance Spot-Check

### Description

Verify no FPS regression from adding `linkOpacity`.

### Files to Change

- None (manual browser DevTools verification).

### Procedure

1. Open 3D graph with ~500 edges (or as many as available).
2. Record baseline FPS in Chrome DevTools performance panel (before change — use main branch).
3. Record FPS on feature branch with `linkOpacity` applied.
4. Assert no measurable regression (< 5% drop).

### Acceptance Criteria

- [ ] FPS on 500-edge graph remains within 5% of baseline.
- [ ] No visible stuttering or dropped frames during camera rotation.

---

## Cross-references

- [[3d-graph-edge-contrast]] — frontend plan defines the implementation being tested.
