---
title: Test Plan — Light Mode Colour Scheme for Graphs
type: plan-test
status: approved
lineage: light-mode-graphs
priority: medium
parent: lifecycle/requirements/light-mode-graphs-2.md
release: May2026
---

# Test Plan — Light Mode Colour Scheme for Graphs

## Overview

This plan covers manual and automated verification that the light-mode graph colour scheme meets every acceptance criterion in the requirement. Tests target the palette module (`graphConstants.ts`), the 2D graph (`Graph2DView.vue`), the 3D graph (`ForceGraph3D.vue`), and the legend (`GraphLegend.vue`).

---

## Milestone 1 — Unit tests for the palette module

### Description

Verify that `useGraphTheme()` returns correct palette objects for both themes, and that all colour pairs meet WCAG AA contrast requirements.

### Files to change

- `tests/web/graphConstants.test.ts` (new)

### Details

1. **Palette selection**: test that when `isDark` is `true`, the dark palette is returned; when `false`, the light palette is returned.
2. **Completeness**: test that both palettes define every required key (`nodeColors`, `priorityColors`, `activeStatusColors`, `edgeColors`, `approvedTestRingColor`, `canvasBg`, `labelColor`, etc.) and that no value is `undefined`.
3. **WCAG AA contrast**: for each palette, iterate every foreground/background colour pair and assert ≥ 4.5:1 contrast ratio for text-sized elements and ≥ 3:1 for graphical objects (node fill vs canvas, edge vs canvas). Use a contrast-ratio calculation utility (e.g. `wcag-contrast` or inline luminance formula).
4. **No stale exports**: assert that the old bare-named exports (`NODE_COLORS`, etc.) are no longer exported (import and expect `undefined`, or grep the module source).

### Acceptance criteria

- [ ] All unit tests pass for both light and dark palettes.
- [ ] Every foreground/background pair passes WCAG AA (4.5:1 text, 3:1 graphical).
- [ ] Test file runs in under 2 seconds.
- [ ] Related: [[light-mode-graphs]]

---

## Milestone 2 — Visual regression: 2D graph in both themes

### Description

Manual and/or screenshot-based test that the 2D Cytoscape graph renders correctly in both themes.

### Files to change

- `tests/web/graph2d-theme.test.ts` (new, if automated via Playwright or similar)
- `lifecycle/tests/light-mode-graphs-2d-visual.md` (test artifact documenting manual steps)

### Details

**Manual test procedure:**

1. Load the app in light mode (set theme to "light" via the toggle).
2. Navigate to the Graph view with a project containing ≥ 5 artifact types and ≥ 1 release node.
3. Verify:
   - Canvas background matches the app's `--color-surface` (white/near-white).
   - All node types are visually distinguishable and legible.
   - Node labels (text below nodes) are dark and readable against the light canvas.
   - Edge lines are visible against the light canvas.
   - Timeline edge labels have a readable background.
   - Label pill nodes have a light purple background with dark purple text.
   - Release nodes have legible text.
   - Priority rings are visible.
   - Active-status pulse animation is visible.
   - Search highlight (yellow ring) is visible.
4. Toggle theme to dark mode and verify the graph switches instantly — no page reload, no layout shift, no flicker.
5. Verify all dark-mode colours match the pre-change appearance (no regression).
6. Toggle back to light mode — verify instant switch back.
7. Reload the page while in light mode — verify the graph renders in light colours on first paint (no flash of dark).

**Automated (if Playwright is available):**

1. Set theme to light, navigate to graph view, take a screenshot.
2. Toggle to dark, take a screenshot.
3. Compare each against golden baselines.

### Acceptance criteria

- [ ] Manual test procedure passes for all items above.
- [ ] No visual regression in dark mode.
- [ ] No flash of wrong theme on initial load.
- [ ] Related: [[light-mode-graphs]]

---

## Milestone 3 — Visual regression: 3D graph in both themes

### Description

Manual and/or screenshot-based test that the 3D force-graph renders correctly in both themes.

### Files to change

- `lifecycle/tests/light-mode-graphs-3d-visual.md` (test artifact documenting manual steps)

### Details

**Manual test procedure:**

1. Load the app in light mode.
2. Switch to 3D graph view.
3. Verify:
   - Scene background is light (matches app surface colour).
   - Node spheres are clearly visible against the light background.
   - Label node text sprites are legible (dark text on light background context).
   - Release box/octahedron nodes have readable name sprites.
   - Edge lines are visible.
   - Tooltip hover popups have appropriate background/text for the theme.
   - Priority torus rings and approved-test blue rings are visible.
   - Active-status pulse rings are visible.
   - Search highlight yellow rings are visible.
   - Dimmed (unmatched during search) nodes use an appropriate light-mode dim colour.
4. Toggle to dark mode — verify instant switch, no re-layout.
5. Verify dark-mode appearance matches pre-change baseline.
6. Reload in light mode — verify no flash of dark scene.

### Acceptance criteria

- [ ] Manual test procedure passes for all items above.
- [ ] No visual regression in dark mode.
- [ ] No flash of wrong theme on initial load.
- [ ] Related: [[light-mode-graphs]]

---

## Milestone 4 — Performance test: theme switch on 200-node graph

### Description

Verify that switching themes on a graph with 200 nodes does not produce a frame longer than 16 ms.

### Files to change

- `lifecycle/tests/light-mode-graphs-perf.md` (test artifact documenting procedure)

### Details

1. Load a project with ≥ 200 artifacts (or mock data that produces ≥ 200 graph nodes).
2. Open DevTools Performance panel and start recording.
3. Toggle the theme.
4. Stop recording and inspect:
   - No single frame exceeds 16 ms on the main thread.
   - No layout thrashing or forced synchronous layouts during the switch.
5. Repeat for both 2D and 3D views.
6. Alternatively, instrument the theme-switch watcher with `performance.now()` before/after and log the delta.

### Acceptance criteria

- [ ] 2D graph theme switch completes in < 16 ms on a 200-node graph.
- [ ] 3D graph theme switch completes in < 16 ms on a 200-node graph.
- [ ] No visible jank or frame drops during the transition.
- [ ] Related: [[light-mode-graphs]]

---

## Milestone 5 — Code hygiene verification

### Description

Verify that no raw theme-dependent hex colour literals remain in the graph component files.

### Files to change

None (grep-based verification).

### Details

1. Grep `Graph2DView.vue`, `ForceGraph3D.vue`, and `GraphLegend.vue` for hex colour patterns (`#[0-9a-fA-F]{3,8}`).
2. Any remaining hex values must be non-theme-dependent (e.g. inside comments, or values that are identical in both themes).
3. Verify `graphConstants.ts` is the single source of truth for all theme-dependent colours.

### Acceptance criteria

- [ ] No raw theme-dependent hex colour literals in `Graph2DView.vue`.
- [ ] No raw theme-dependent hex colour literals in `ForceGraph3D.vue`.
- [ ] No raw theme-dependent hex colour literals in `GraphLegend.vue`.
- [ ] `pnpm build` succeeds with no TypeScript errors.
- [ ] Related: [[light-mode-graphs]]
