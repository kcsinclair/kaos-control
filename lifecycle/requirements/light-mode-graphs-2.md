---
title: Light Mode Colour Scheme for 2D and 3D Graphs
type: requirement
status: approved
lineage: light-mode-graphs
created: "2026-05-07"
priority: medium
parent: lifecycle/ideas/light-mode-graphs.md
labels:
    - enhancement
    - frontend
    - usability
release: May2026
assignees:
    - role: product-owner
      who: agent
---

# Light Mode Colour Scheme for 2D and 3D Graphs

## Problem

The 2D (Cytoscape.js) and 3D (3d-force-graph / three.js) graph visualisations use hardcoded dark-mode colours for backgrounds, nodes, edges, and labels. These values are defined in `graphConstants.ts` and inline within `Graph2DView.vue` and `ForceGraph3D.vue`. When a user switches the application to light mode via the theme store, the graphs remain dark — creating a jarring visual mismatch and reducing legibility (e.g. dark text on a dark background, or dark edges against the application's light chrome).

## Goals / Non-goals

### Goals

- **G1** Both graph renderers adopt a light colour palette when the application theme is set to light mode.
- **G2** Theme changes are applied reactively — toggling the theme updates graph colours without a page reload or route change.
- **G3** Colour choices maintain WCAG 2.1 AA contrast ratios (minimum 4.5:1 for normal text, 3:1 for large text and graphical objects) in both themes.
- **G4** The colour-mapping strategy is centralised so that adding a new type or status colour requires a single-place change, not edits in multiple files.

### Non-goals

- Providing a user-configurable colour picker or custom theme editor.
- Adding additional themes beyond the existing light / dark pair.
- Redesigning the graph layout algorithms or interaction model.
- Changing the application-level theme toggle mechanism itself.

## Detailed Requirements

### Functional

1. **Theme-aware colour constants** — `graphConstants.ts` must export both a light and a dark variant for every colour category: `NODE_COLORS`, `PRIORITY_COLORS`, `ACTIVE_STATUS_COLORS`, `EDGE_COLORS`, `APPROVED_TEST_RING_COLOR`, and any label/text colours. A helper (function or reactive computed) must resolve the active palette based on the current value of `useThemeStore().isDark`.

2. **2D graph (Cytoscape.js) — `Graph2DView.vue`**
   - Canvas background must switch between the current dark value (`#0f172a`) and an appropriate light value.
   - The Cytoscape stylesheet must reference the active palette so that node fills, node label colours, edge colours, and selection/hover highlights update when the theme changes.
   - The component must watch for theme changes and call `cy.style().update()` (or equivalent batch style refresh) to apply the new palette without destroying and recreating the graph instance.

3. **3D graph (three.js) — `ForceGraph3D.vue`**
   - Scene background colour must switch to match the theme.
   - Node sphere materials, edge line materials, and text sprite colours must update from the active palette.
   - The component must watch for theme changes and update Three.js materials / sprites in place (no full graph rebuild).

4. **Reactive switching** — Both renderers must respond to `useThemeStore().isDark` via a Vue `watch` (or equivalent reactivity). The transition must complete within one animation frame after the store value changes, with no visible flicker or layout shift.

5. **Initial render** — On first mount, each graph component must read the current theme and render with the correct palette immediately; there must be no flash of the wrong theme.

### Non-functional

1. **Performance** — Switching themes must not cause a measurable frame drop (> 16 ms main-thread block) on a graph of up to 200 nodes.
2. **Maintainability** — All theme-dependent colour values must live in `graphConstants.ts` (or a dedicated `graphTheme.ts` alongside it). No raw hex colour literals related to theming should remain in the Vue component files.
3. **Accessibility** — Every foreground/background colour pair in both palettes must meet WCAG 2.1 AA contrast. The light palette must be verified against a white or near-white canvas background.

## Acceptance Criteria

- [ ] With the application in light mode, `Graph2DView` renders with a light canvas background and all node/edge/label colours are legible against it.
- [ ] With the application in light mode, `ForceGraph3D` renders with a light scene background and all node/edge/label colours are legible against it.
- [ ] Toggling the theme via the existing toggle updates both graph renderers instantly without a page reload, route change, or graph re-initialisation.
- [ ] No raw theme-dependent hex colour literals remain in `Graph2DView.vue` or `ForceGraph3D.vue`; all such values are sourced from `graphConstants.ts` (or a co-located theme module).
- [ ] All foreground-on-background colour pairs in both light and dark palettes satisfy WCAG 2.1 AA contrast (4.5:1 for text, 3:1 for graphical objects).
- [ ] Switching themes on a 200-node graph does not produce a frame longer than 16 ms (measured via DevTools Performance panel or `performance.now()` in a test).
- [ ] On first page load in light mode, graphs render in light colours with no flash of dark colours.
- [ ] Existing dark-mode appearance is unchanged (no visual regression).
- [ ] Related: [[light-mode-graphs]]

## Resolved Questions

1. **Light palette specifics** — Should the light-mode canvas background be pure white (`#ffffff`), the app's surface token (`--color-surface`), or a slightly tinted neutral? Using the CSS token would keep graphs consistent with the rest of the UI but may not suit the graph aesthetic.

> app's surface token.  I will need to see it.

2. **Node colour adjustment strategy** — The current node colours (amber, violet, cyan, etc.) were chosen for a dark background. Should the light palette use the same hues at a darker shade, or should it use the existing CSS design-token scale (e.g. `--color-primary`, `--color-accent`)?

> CSS design-token scale
