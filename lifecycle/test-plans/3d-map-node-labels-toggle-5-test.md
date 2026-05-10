---
title: 'Test Plan: 3D Map Node Labels Toggle'
type: plan-test
status: in-development
lineage: 3d-map-node-labels-toggle
parent: lifecycle/requirements/3d-map-node-labels-toggle-2.md
---

## Summary

Test the two new label-toggle checkboxes ("Show node titles", "Show node lineage") across both 3D views (map and roadmap). Coverage spans checkbox rendering, default state, label content correctness, truncation logic, combined display, toggle reactivity, node exclusions, and accessibility. Tests use the existing Vitest + Vue Test Utils setup for component tests and Playwright for integration/visual tests.

## Milestone 1: Unit Tests — Truncation Helper

### Description

If the truncation logic is extracted to a utility function (recommended), write unit tests for the 15-character truncation rule. If inline, test via component tests in Milestone 2.

### Files to change

- `tests/web/truncateTitle.test.ts` (new) — unit tests for title truncation.

### Acceptance criteria

- [ ] Strings of exactly 15 characters return unchanged (no ellipsis).
- [ ] Strings of 16+ characters are truncated to 15 characters followed by `…` (U+2026).
- [ ] Empty string and single-character inputs handled correctly.
- [ ] Unicode characters (multi-byte) are truncated by character count, not byte count.

## Milestone 2: Component Tests — GraphFilters Checkboxes

### Description

Test that `GraphFilters.vue` renders the two new checkboxes, that they reflect prop state, and that toggling emits the correct events.

### Files to change

- `tests/web/GraphFilters.labels.test.ts` (new) — mount `GraphFilters` with the new props and verify rendering and events.

### Acceptance criteria

- [ ] A checkbox labelled "Show node titles" is rendered.
- [ ] A checkbox labelled "Show node lineage" is rendered.
- [ ] Both are unchecked when `showNodeTitles: false` and `showNodeLineage: false`.
- [ ] Clicking "Show node titles" emits `toggleShowNodeTitles`.
- [ ] Clicking "Show node lineage" emits `toggleShowNodeLineage`.
- [ ] Both checkboxes have associated `<label>` elements (accessibility).

## Milestone 3: Component Tests — ForceGraph3D Label Rendering

### Description

Test that `ForceGraph3D.vue` correctly calls the label-building logic based on prop values. Since Three.js rendering cannot be fully tested in JSDOM, these tests verify that `buildNodeObject` produces the expected Three.js Group structure (sprite count and content) by spying on or extracting the build function.

### Files to change

- `tests/web/ForceGraph3D.labels.test.ts` (new) — test `buildNodeObject` output under different prop combinations.

### Acceptance criteria

- [ ] With both props `false`, non-label/non-release nodes produce a Group with no title or lineage sprites (only rings/existing overlays).
- [ ] With `showNodeTitles: true`, nodes get a title sprite; title text matches `node.title` truncated to 15 chars.
- [ ] With `showNodeLineage: true`, nodes get a lineage sprite; text matches `node.lineage` (full, untruncated).
- [ ] With both `true`, nodes get two sprites at different y-offsets (no overlap).
- [ ] Release nodes (`type === 'release'`) are unaffected by either prop.
- [ ] Label nodes (`type === 'label'`) are unaffected by either prop.

## Milestone 4: Store Tests — Toggle State

### Description

Test the Pinia graph store's new refs and toggle methods.

### Files to change

- `tests/web/graphStore.labels.test.ts` (new) — test `showNodeTitles`, `showNodeLineage`, and their toggle methods.

### Acceptance criteria

- [ ] `showNodeTitles` defaults to `false`.
- [ ] `showNodeLineage` defaults to `false`.
- [ ] `toggleShowNodeTitles()` flips the value.
- [ ] `toggleShowNodeLineage()` flips the value.

## Milestone 5: Integration Tests — GraphView Toggle Wiring

### Description

Test that `GraphView.vue` correctly wires the store state to `GraphFilters` props and `ForceGraph3D` props. Use shallow-mount to avoid Three.js rendering.

### Files to change

- `tests/web/GraphView.labels.test.ts` (new) — shallow-mount `GraphView` with a mocked store, verify prop bindings and event handling.

### Acceptance criteria

- [ ] `GraphFilters` receives `showNodeTitles` and `showNodeLineage` props matching store state.
- [ ] `ForceGraph3D` receives `showNodeTitles` and `showNodeLineage` props matching store state.
- [ ] Emitting `toggleShowNodeTitles` from `GraphFilters` calls `store.toggleShowNodeTitles()`.
- [ ] Emitting `toggleShowNodeLineage` from `GraphFilters` calls `store.toggleShowNodeLineage()`.

## Milestone 6: Integration Tests — RoadmapGraphView Toggle Wiring

### Description

Test that `RoadmapGraphView.vue` renders the two checkboxes and passes local ref state to `ForceGraph3D`.

### Files to change

- `tests/web/RoadmapGraphView.labels.test.ts` (new) — shallow-mount `RoadmapGraphView`, verify checkbox rendering and prop passing.

### Acceptance criteria

- [ ] Two checkboxes appear in the roadmap 3D view.
- [ ] Both default to unchecked.
- [ ] Clicking a checkbox updates the prop passed to `ForceGraph3D`.

## Milestone 7: E2E / Manual Test — Visual Verification

### Description

Manual or Playwright-based visual test covering the golden path in a running dev server. This milestone verifies the full integrated behaviour that component tests cannot cover (actual Three.js rendering in a browser).

### Files to change

- `tests/web/3dMapLabels.e2e.test.ts` (new, if Playwright is set up) — or document a manual test script.

### Acceptance criteria

- [ ] Navigate to the 3D map view; both checkboxes are visible and unchecked; no labels on nodes.
- [ ] Check "Show node titles": truncated title labels appear on non-release, non-label nodes.
- [ ] Check "Show node lineage": lineage slug labels appear in a distinct style.
- [ ] Enable both: labels stack vertically without overlap.
- [ ] Uncheck each: corresponding labels disappear immediately, no page reload.
- [ ] Navigate to 3D roadmap view: same checkboxes exist and function identically.
- [ ] Release nodes and label-type nodes remain unaffected throughout.
- [ ] Toggle state persists when switching between 2D and 3D mode within the same session.
- [ ] Toggle state resets on full page reload.
- [ ] No visible frame-rate regression with both labels enabled on a graph with ~100+ nodes.

## Cross-references

- Backend: [[3d-map-node-labels-toggle]] (no backend changes; no API tests needed)
- Frontend implementation: [[3d-map-node-labels-toggle]] (frontend plan details all code changes)
