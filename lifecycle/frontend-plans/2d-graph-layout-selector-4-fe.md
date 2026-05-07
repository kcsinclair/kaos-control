---
title: 2D Graph Layout Selector — Frontend Plan
type: plan-frontend
status: in-development
lineage: 2d-graph-layout-selector
parent: lifecycle/requirements/2d-graph-layout-selector-2.md
release: May2026
---

# 2D Graph Layout Selector — Frontend Plan

## Summary

Add a layout algorithm selector dropdown to the 2D graph view, surface a directed-graph toggle, persist the selection in the Pinia store for session lifetime, and ensure all graph updates (filter changes, data refreshes) respect the active layout choice. Add `cytoscape-dagre` as a dynamically-imported dependency.

---

## Milestones

### Milestone 1: Layout Configuration Registry

**Description:** Create a data-driven layout configuration object that maps layout keys to their Cytoscape options. This makes adding future layouts a single-object addition with no scattered conditionals.

**Files to change:**
- `web/src/components/graph/Graph2DView.vue` — extract current hard-coded layout options into an importable config.
- `web/src/components/graph/layoutConfigs.ts` (new) — export `LAYOUT_CONFIGS` record keyed by layout ID, each entry containing `{ key, label, options, requiresPlugin? }`.

**Acceptance Criteria:**
- [ ] `LAYOUT_CONFIGS` defines entries for: `fcose`, `breadthfirst`, `concentric`, `circle`, `dagre`.
- [ ] Each entry specifies `label` (human-readable), `cyName` (Cytoscape layout name), `options` (default params including `animate: true, animationDuration: 400`), and optional `plugin` (async import function).
- [ ] `breadthfirst` entry includes `directed: true` in its options by default, overridable by the directed toggle state.
- [ ] Adding a new layout requires only appending an entry to `LAYOUT_CONFIGS`.

---

### Milestone 2: Pinia Store — Layout State

**Description:** Extend the graph store to hold the active layout key and the directed toggle state, both session-scoped (default Pinia behaviour — no localStorage).

**Files to change:**
- `web/src/stores/graph.ts` — add `activeLayout: string` (default `'fcose'`) and `directed: boolean` (default `false`) to state; add actions `setLayout(key)` and `toggleDirected()`.

**Acceptance Criteria:**
- [ ] `activeLayout` defaults to `'fcose'`.
- [ ] `directed` defaults to `false`.
- [ ] `setLayout` validates the key exists in `LAYOUT_CONFIGS` before setting.
- [ ] State survives Vue Router navigation within the SPA session (no reset on route change).

---

### Milestone 3: Layout Selector UI Component

**Description:** Build a compact dropdown control for layout selection and a toggle for directed mode, placed in the `.view-controls` area of `GraphView.vue`, visible only when the 2D view is active.

**Files to change:**
- `web/src/components/graph/LayoutSelector.vue` (new) — `<select>` or custom dropdown listing layouts from `LAYOUT_CONFIGS`, plus a directed toggle checkbox/icon button.
- `web/src/views/project/GraphView.vue` — import and render `LayoutSelector` inside `.view-controls`, conditionally shown when `view === '2d'`.

**Acceptance Criteria:**
- [ ] Selector is visible only when the 2D view is active.
- [ ] Selector lists all five layouts by their human-readable `label`.
- [ ] Selected value reflects `graphStore.activeLayout`.
- [ ] Directed toggle reflects `graphStore.directed`.
- [ ] Control is keyboard-navigable with appropriate `aria-label`.
- [ ] Styling matches existing `.view-controls` dark-theme buttons (same font, padding, border-radius, colours).
- [ ] Selector is hidden at narrow viewports via the same breakpoint logic as other toolbar controls (or remains visible per resolved question #1: "Yes").

---

### Milestone 4: Layout Application in Graph2DView

**Description:** Replace hard-coded layout logic in `Graph2DView.vue` with reactive layout application driven by the store's `activeLayout` and `directed` state.

**Files to change:**
- `web/src/components/graph/Graph2DView.vue`:
  - Remove hard-coded `breadthfirst`/`fcose` conditional in `init()` and `update()`.
  - Import `LAYOUT_CONFIGS` and read `graphStore.activeLayout` + `graphStore.directed`.
  - Add a `runLayout()` method that: resolves plugin (dynamic import if needed, idempotent registration), merges config options with directed override, calls `cy.layout(mergedOptions).run()`.
  - Watch `graphStore.activeLayout` and `graphStore.directed` — on change, call `runLayout()`.
  - Ensure `update()` (called on node/edge data change from filters) uses `runLayout()` instead of hard-coded layout.

**Acceptance Criteria:**
- [ ] Selecting a layout triggers animated relayout without destroying/recreating the Cytoscape instance.
- [ ] Animation uses `animate: true` with ~400 ms duration for layouts that support it; layouts that don't support animation fall back to instant positioning.
- [ ] fcose remains the default on first render.
- [ ] Filter changes (type/status/lineage/label/priority toggles, text search) re-layout using the active algorithm.
- [ ] Directed toggle modifies `breadthfirst` and `dagre` options (`directed: true/false`) and triggers relayout.
- [ ] fcose plugin loaded once, dagre plugin loaded on first use (code-split).

---

### Milestone 5: Add `cytoscape-dagre` Dependency

**Description:** Install `cytoscape-dagre` and wire it into the layout config as a dynamically-imported plugin.

**Files to change:**
- `web/package.json` — add `cytoscape-dagre` dependency.
- `web/src/components/graph/layoutConfigs.ts` — dagre entry uses `plugin: () => import('cytoscape-dagre')`.
- `web/src/components/graph/Graph2DView.vue` — `runLayout()` handles async plugin registration before running dagre layout.

**Acceptance Criteria:**
- [ ] `cytoscape-dagre` is listed in `package.json` dependencies.
- [ ] Dagre is NOT included in the initial JS bundle — only fetched when selected.
- [ ] Selecting "Dagre" layout works correctly after the dynamic import resolves.
- [ ] No other new npm dependencies are added.

---

### Milestone 6: Visual Polish & Edge Cases

**Description:** Handle edge cases: empty graph, single-node graph, layout switch during animation, and ensure responsive behaviour.

**Files to change:**
- `web/src/components/graph/Graph2DView.vue` — guard `runLayout()` against empty element collections; stop any running layout before starting a new one (`cy.stop()`).
- `web/src/components/graph/LayoutSelector.vue` — disable selector during layout animation (optional UX improvement).

**Acceptance Criteria:**
- [ ] Switching layout on an empty graph does not throw.
- [ ] Rapidly switching layouts does not cause visual glitches (previous animation stopped).
- [ ] Graph with 200+ nodes re-layouts in under 2 seconds.
- [ ] No console errors or warnings during normal usage.

---

## Cross-links

- [[2d-graph-layout-selector]] backend plan confirms no API changes needed.
- [[2d-graph-layout-selector]] test plan covers E2E and integration tests for layout switching.
