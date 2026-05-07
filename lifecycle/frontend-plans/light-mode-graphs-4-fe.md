---
title: Frontend Plan — Light Mode Colour Scheme for Graphs
type: plan-frontend
status: approved
lineage: light-mode-graphs
priority: medium
parent: lifecycle/requirements/light-mode-graphs-2.md
release: April2026
---

# Frontend Plan — Light Mode Colour Scheme for Graphs

## Overview

All graph colour values are currently hardcoded for dark backgrounds. This plan introduces a theme-aware palette system in `graphConstants.ts`, then wires both `Graph2DView.vue` and `ForceGraph3D.vue` to reactively switch palettes via `useThemeStore().isDark`. The `GraphLegend.vue` component must also adopt the active palette so the legend remains readable against either background.

The resolved question in the requirement specifies: use the app's CSS surface token (`--color-surface`) for graph canvas backgrounds, and the CSS design-token scale for light-mode node colours.

---

## Milestone 1 — Theme-aware palette in `graphConstants.ts`

### Description

Refactor `graphConstants.ts` to export dual palettes (light + dark) for every colour category and a reactive resolver that returns the active palette based on `useThemeStore().isDark`.

### Files to change

- `web/src/components/graph/graphConstants.ts`

### Details

1. Rename the current top-level exports (`NODE_COLORS`, `PRIORITY_COLORS`, `ACTIVE_STATUS_COLORS`, `EDGE_COLORS`, `APPROVED_TEST_RING_COLOR`) to `DARK_*` variants (or nest them under a `dark` object).
2. Create matching `LIGHT_*` palettes:
   - **Canvas background**: read the CSS custom property `--color-surface` at runtime (via `getComputedStyle`), or use a static light value `#ffffff` / `#f1f5f9` as the requirement says to use the app surface token. Since Cytoscape and Three.js need a hex value, resolve `--color-surface` once on theme change.
   - **Node colours (light)**: use darker, more saturated variants of the existing hues to maintain contrast against a white/light-grey canvas. The requirement says to use the CSS design-token scale — reference `--node-type-*` tokens from `tokens.css` where available, providing explicit hex values for types not yet tokenised (e.g. `plan-frontend`, `plan-test`, `defect`, `label`, `backlog`).
   - **Priority colours (light)**: use darker shades (e.g. `#dc2626` for high instead of `#ef4444`).
   - **Active status colours (light)**: darken to maintain ≥ 3:1 contrast against the light canvas.
   - **Edge colours (light)**: use mid-tone greys/colours that remain visible on a white background (e.g. `#475569` → `#64748b` or darker).
   - **Text/label colours (light)**: `#0f172a` (dark text on light background) instead of `#f1f5f9`.
   - **Label node styles (light)**: light purple background (`#f3e8ff`), dark purple text (`#6b21a8`), purple border.
   - **Release node text (light)**: dark blue (`#1e3a5f`) — same as current.
   - **Backlog node text (light)**: dark grey (`#374151`).
3. Export a composable `useGraphTheme()` that returns computed refs:
   ```ts
   export function useGraphTheme() {
     const { isDark } = useThemeStore()
     const palette = computed(() => isDark ? DARK_PALETTE : LIGHT_PALETTE)
     return { palette, isDark }
   }
   ```
   Where each palette is a typed object containing `nodeColors`, `priorityColors`, `activeStatusColors`, `edgeColors`, `approvedTestRingColor`, `canvasBg`, `labelColor`, `labelNodeBg`, `labelNodeText`, `labelNodeBorder`, `releaseText`, `backlogText`, `edgeLabelBg`, `edgeLabelText`, `borderDefault`, `searchHighlight`.
4. Keep the old named exports as thin wrappers around the dark palette for backward compatibility during migration (remove after Milestone 2–4 are complete).

### Acceptance criteria

- [ ] `useGraphTheme()` returns the correct palette for both `isDark: true` and `isDark: false`.
- [ ] Every light foreground/background pair meets WCAG 2.1 AA contrast (4.5:1 text, 3:1 graphical objects). Verify with a contrast checker during development.
- [ ] All colour values that were previously hardcoded in component files are now in the palette objects.
- [ ] No raw hex colour literals related to theming remain outside `graphConstants.ts`.
- [ ] Related: [[light-mode-graphs]]

---

## Milestone 2 — Wire `Graph2DView.vue` to the theme palette

### Description

Replace all hardcoded colour references in `Graph2DView.vue` with values from `useGraphTheme()`. Add a `watch` on `isDark` to reactively update the Cytoscape stylesheet without destroying the graph instance.

### Files to change

- `web/src/components/graph/Graph2DView.vue`

### Details

1. Import `useGraphTheme` and destructure `{ palette, isDark }`.
2. **Canvas background** (`<style scoped>` `.graph-2d { background: #0f172a }`): bind the background to `palette.value.canvasBg` via an inline style on the container div (`:style="{ background: palette.canvasBg }"`), or use a CSS variable set via JS.
3. **Cytoscape stylesheet**: refactor the `style` array in `init()` into a function `buildCyStyle(p: Palette)` that returns the style array using palette values:
   - `node` selector: `color` → `p.labelColor`, `'border-color'` → derive from palette.
   - `node:selected`: `'border-color'` → use palette-appropriate highlight (e.g. `#000000` for light, `#ffffff` for dark).
   - `node[type="label"]`: `'background-color'` → `p.labelNodeBg`, `color` → `p.labelNodeText`, `'border-color'` → `p.labelNodeBorder`.
   - `node[type="release"][synthetic="true"]`: `color` → `p.backlogText`.
   - `edge` selector: `'line-color'`, `'target-arrow-color'`, `color`, `'text-background-color'` → palette values.
   - Timeline edges: similarly palette-driven.
   - Assigned edges: palette values.
4. **Reactive switch**: add `watch(isDark, () => { ... })` that:
   - Updates the container background (if using inline style, this is automatic via reactivity).
   - Calls `cy.style().fromJson(buildCyStyle(palette.value)).update()` to batch-apply the new stylesheet.
   - Does NOT re-run layout or destroy/recreate the graph.
5. **Node data colours**: the `nodeColor()` helper and `buildElements()` already set `data(color)` — update `nodeColor()` to read from the current palette. After a theme switch, call `cy.nodes().forEach(n => n.data('color', nodeColor(n.data('type'), n.data('synthetic') === 'true')))` before the style update, so `data(color)` references resolve correctly.
6. **Pulse interval**: the pulse in `setInterval` uses `ACTIVE_STATUS_COLORS` directly — change to `palette.value.activeStatusColors`.
7. **Search highlight**: the `#facc15` highlight colour → `palette.value.searchHighlight`.
8. Remove all hardcoded hex colour literals from the component file.

### Acceptance criteria

- [ ] In light mode, `Graph2DView` renders with the CSS surface-token background and all nodes/edges/labels are legible.
- [ ] Toggling the theme updates all Cytoscape colours within one animation frame — no flicker, no layout shift, no graph rebuild.
- [ ] On first mount in light mode, the graph renders with the light palette immediately (no flash of dark).
- [ ] No raw theme-dependent hex colour literals remain in `Graph2DView.vue`.
- [ ] Existing dark-mode appearance is pixel-identical (no regression).
- [ ] Related: [[light-mode-graphs]]

---

## Milestone 3 — Wire `ForceGraph3D.vue` to the theme palette

### Description

Replace all hardcoded colour references in `ForceGraph3D.vue` with values from `useGraphTheme()`. Add a `watch` on `isDark` to reactively update Three.js materials and the scene background.

### Files to change

- `web/src/components/graph/ForceGraph3D.vue`

### Details

1. Import `useGraphTheme` and destructure `{ palette, isDark }`.
2. **Scene background**: replace `.backgroundColor('#0f172a')` with `.backgroundColor(palette.value.canvasBg)`.
3. **Node colours**: update `nodeColor()` to read from `palette.value.nodeColors`. The dim-for-unmatched blend target must also switch (`#1e2535` for dark → a light-grey equivalent for light).
4. **Text sprites**: the `textSprite()` colour parameter and all call sites must use palette values. For label nodes, use `palette.value.labelNodeText`; for release nodes, use `palette.value.releaseText` / `palette.value.backlogText`.
5. **Tooltip HTML**: the inline `background:#1e293b` and `color:#f1f5f9` in `nodeLabel` callbacks must use palette-derived values.
6. **Edge colours**: `edgeColor()` reads from `palette.value.edgeColors`.
7. **Reactive switch**: add `watch(isDark, () => { ... })` that:
   - Calls `graph.backgroundColor(palette.value.canvasBg)` to update the Three.js renderer clear colour.
   - Calls `graph.nodeColor(...)` with the updated `nodeColor` function to refresh sphere materials.
   - Rebuilds `nodeThreeObject` via `graph.nodeThreeObject(...)` to re-create sprites and rings with new colours. (Three.js materials don't bind to reactive data, so objects must be rebuilt — but this does NOT re-run the force layout.)
   - Updates `linkColor` via `graph.linkColor(...)`.
   - The graph data and layout positions are preserved — only visual appearance changes.
8. **Performance**: ensure theme switch on a 200-node graph stays under 16 ms. The `nodeThreeObject` rebuild is O(n) with lightweight geometry — profile and verify.
9. Remove all hardcoded hex colour literals from the component file.

### Acceptance criteria

- [ ] In light mode, `ForceGraph3D` renders with a light scene background and all visual elements are legible.
- [ ] Toggling the theme updates the 3D scene without a full graph rebuild or layout re-computation.
- [ ] Text sprites (label nodes, release names) are legible against both backgrounds.
- [ ] Tooltip popups are legible in both themes.
- [ ] On first mount in light mode, the 3D graph renders with the light palette immediately.
- [ ] No raw theme-dependent hex colour literals remain in `ForceGraph3D.vue`.
- [ ] Existing dark-mode appearance is unchanged.
- [ ] Related: [[light-mode-graphs]]

---

## Milestone 4 — Wire `GraphLegend.vue` to the theme palette

### Description

The legend overlay uses hardcoded dark background and light text. It must adapt to the active theme so it remains readable.

### Files to change

- `web/src/components/graph/GraphLegend.vue`

### Details

1. Import `useGraphTheme` and use `palette` to source legend dot/ring/line colours.
2. Replace the hardcoded CSS `background: rgba(15, 23, 42, 0.85)` and `color: #f1f5f9` with theme-aware values — e.g. `rgba(255,255,255,0.9)` in light mode, the current dark glass in dark mode. Bind via `:class` or `:style`.
3. Legend colour swatches must reflect the active palette's node/priority/edge colours (they may differ between light and dark).
4. The `PRIORITY_COLORS` and `EDGE_COLORS` references at module top level are not reactive — move them into `computed` properties that read from `palette.value`.

### Acceptance criteria

- [ ] Legend is readable in both light and dark mode.
- [ ] Legend colour swatches match the colours actually rendered in the graph.
- [ ] Toggling the theme updates the legend reactively.
- [ ] Related: [[light-mode-graphs]]

---

## Milestone 5 — Remove backward-compatible re-exports

### Description

Once all consumers are migrated to `useGraphTheme()`, remove the old top-level named exports from `graphConstants.ts` and verify no file imports them.

### Files to change

- `web/src/components/graph/graphConstants.ts`

### Acceptance criteria

- [ ] No file imports `NODE_COLORS`, `PRIORITY_COLORS`, `ACTIVE_STATUS_COLORS`, `EDGE_COLORS`, or `APPROVED_TEST_RING_COLOR` as bare top-level exports.
- [ ] `pnpm build` succeeds with no errors.
- [ ] Related: [[light-mode-graphs]]
