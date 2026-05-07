---
title: 2D Graph Layout Selector — Test Plan
type: plan-test
status: approved
lineage: 2d-graph-layout-selector
parent: lifecycle/requirements/2d-graph-layout-selector-2.md
release: May2026
---

# 2D Graph Layout Selector — Test Plan

## Summary

Verify that the layout selector control works correctly across all acceptance criteria: visibility, layout switching, animation, session persistence, filter interaction, directed toggle, accessibility, and performance.

---

## Milestones

### Milestone 1: Component Unit Tests — LayoutSelector

**Description:** Unit test the `LayoutSelector.vue` component in isolation using Vue Test Utils, verifying rendering, event emission, and accessibility attributes.

**Files to change:**
- `tests/web/components/graph/LayoutSelector.spec.ts` (new)

**Acceptance Criteria:**
- [ ] Test: renders a select/dropdown with all five layout options (fcose, breadthfirst, concentric, circle, dagre).
- [ ] Test: emits layout change event when user selects a different option.
- [ ] Test: reflects the current `activeLayout` from the store as the selected value.
- [ ] Test: directed toggle emits the correct action.
- [ ] Test: control has `aria-label` attribute.
- [ ] Test: control is keyboard-navigable (can be focused, options selectable via keyboard).

---

### Milestone 2: Store Unit Tests — Layout State

**Description:** Unit test the Pinia graph store's layout-related state and actions.

**Files to change:**
- `tests/web/stores/graph.spec.ts` (new or extend existing)

**Acceptance Criteria:**
- [ ] Test: `activeLayout` defaults to `'fcose'`.
- [ ] Test: `directed` defaults to `false`.
- [ ] Test: `setLayout('breadthfirst')` updates `activeLayout`.
- [ ] Test: `setLayout('invalid-key')` does not change state (validation).
- [ ] Test: `toggleDirected()` flips `directed` boolean.
- [ ] Test: state persists across simulated route changes (store not destroyed).

---

### Milestone 3: Integration Tests — Layout Switching in Graph2DView

**Description:** Integration tests verifying that `Graph2DView` correctly applies layouts when the store state changes. Uses a real (jsdom-compatible) or mocked Cytoscape instance.

**Files to change:**
- `tests/web/components/graph/Graph2DView.layout.spec.ts` (new)

**Acceptance Criteria:**
- [ ] Test: on mount, default layout `fcose` is applied via `cy.layout()`.
- [ ] Test: changing `activeLayout` to `'concentric'` calls `cy.layout({ name: 'concentric', ... }).run()`.
- [ ] Test: changing `activeLayout` to `'dagre'` triggers dynamic import of `cytoscape-dagre` before running layout.
- [ ] Test: toggling `directed` to `true` re-runs the active layout with `directed: true` in options.
- [ ] Test: animation options (`animate: true`, `animationDuration`) are passed to layout.
- [ ] Test: Cytoscape instance is NOT destroyed and recreated on layout change.

---

### Milestone 4: Integration Tests — Filter + Layout Interaction

**Description:** Verify that applying filters re-layouts using the currently selected algorithm, not the hard-coded default.

**Files to change:**
- `tests/web/components/graph/Graph2DView.filters.spec.ts` (new or extend existing)

**Acceptance Criteria:**
- [ ] Test: with `activeLayout` set to `'circle'`, applying a type filter triggers relayout with `{ name: 'circle', ... }`.
- [ ] Test: with `activeLayout` set to `'breadthfirst'`, toggling a status filter triggers relayout with breadthfirst options.
- [ ] Test: text search highlight does NOT trigger a relayout (only visual dimming).

---

### Milestone 5: E2E Tests — Full User Flow

**Description:** End-to-end tests (Playwright or Cypress) exercising the complete user journey through layout selection in a running application.

**Files to change:**
- `tests/e2e/graph-layout-selector.spec.ts` (new)

**Acceptance Criteria:**
- [ ] Test: navigate to graph view → 2D toggle → layout selector is visible.
- [ ] Test: switch to 3D view → layout selector is hidden.
- [ ] Test: select "Hierarchical" → nodes visibly reposition (screenshot or position assertion).
- [ ] Test: select "Circle" → nodes arrange in circular pattern.
- [ ] Test: navigate away from graph view → navigate back → previously selected layout is restored.
- [ ] Test: apply a filter while "Concentric" is selected → graph re-layouts in concentric pattern (not fcose).
- [ ] Test: toggle directed → graph updates layout.
- [ ] Test: keyboard-only interaction — tab to selector, change value with arrow keys, layout updates.

---

### Milestone 6: Performance Tests

**Description:** Verify that layout computation meets the <2 second requirement for graphs up to 500 nodes.

**Files to change:**
- `tests/web/components/graph/Graph2DView.perf.spec.ts` (new)

**Acceptance Criteria:**
- [ ] Test: generate a synthetic graph of 200 nodes and 400 edges; each layout algorithm completes `cy.layout().run()` in under 2 seconds.
- [ ] Test: generate a synthetic graph of 500 nodes and 1000 edges; each layout algorithm completes in under 2 seconds (may need `quality: 'default'` for fcose at this scale).
- [ ] Test: rapidly switching layouts 5 times does not cause memory leaks or unhandled promise rejections.

---

### Milestone 7: Bundle Size Verification

**Description:** Confirm that dagre is code-split and not included in the initial bundle.

**Files to change:**
- `tests/web/bundle/dagre-codesplit.spec.ts` (new) — or a build-time check script.

**Acceptance Criteria:**
- [ ] Test: after `pnpm build`, the main chunk does NOT contain `cytoscape-dagre` code.
- [ ] Test: a separate async chunk exists containing dagre.
- [ ] No other new dependencies added to the bundle beyond `cytoscape-dagre`.

---

## Cross-links

- [[2d-graph-layout-selector]] frontend plan defines the implementation these tests verify.
- [[2d-graph-layout-selector]] backend plan confirms no API-level tests needed for this feature.
