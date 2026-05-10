---
title: "Tests: Rename Graph to Map in UI and Routing"
type: test
status: draft
lineage: rename-graph-to-map
parent: lifecycle/test-plans/rename-graph-to-map-5-test.md
---

# Tests: Rename Graph to Map in UI and Routing

## Overview

Integration test suite covering the rename of all user-facing "Graph" references
to "Map" in the frontend, plus verification of the `/graph` → `/map` redirect.

## Scenarios Covered

### Milestone 1 — Component import paths updated to `@/components/map/`

All test files that previously imported from `@/components/graph/` already had
their imports updated to `@/components/map/` as part of the frontend rename:

- `tests/web/graph-show-tests-toggle.test.ts` — imports `MapFilters.vue`
- `tests/web/Graph2DView.layout.spec.ts` — imports `Map2DView.vue`
- `tests/web/Graph2DView.approvedRing.test.ts` — imports `Map2DView.vue`, mocks `@/components/map/graphConstants`
- `tests/web/Graph2DView.filters.spec.ts` — imports `Map2DView.vue`
- `tests/web/Graph2DView.perf.spec.ts` — imports `Map2DView.vue`
- `tests/web/graphConstants.test.ts` — imports from `@/components/map/graphConstants`
- `tests/web/LayoutSelector.spec.ts` — imports `LayoutSelector.vue` from `@/components/map/`

### Milestone 2 — Sidebar and navigation tests updated

File: `tests/web/AppSidebar.test.ts`

- `expectedLabels` array (Milestone 2 describe block): `'Graph'` → `'Map'`
- `allExpectedLabels` array (Milestone 3 describe block): `'Graph'` → `'Map'`
- Route path in Milestone 7 `views` array: `/p/testproject/graph` → `/p/testproject/map`

### Milestone 3 — Cross-view and hide-done-items tests

No changes were needed:
- `tests/web/hide-done-items/graph-toggle.test.ts` — imports only from `@/stores/graph` (unchanged per non-goals)
- `tests/web/hide-done-items/cross-view-consistency.test.ts` — no graph component imports

### Milestone 4 — Test helper factories

No changes needed in `tests/web/helpers/seed_artifacts.ts`. The factory functions
(`makeGraphNode`, `makeGraphEdge`, `makeGraphNodesForAllStatuses`) are
developer-facing identifiers and are preserved per the non-goals of the rename.

### Milestone 5 — CSS selector references

No test files referenced the old `.graph-view`, `.graph-main`, `.graph-state`,
`.graph-legend-wrap`, `.graph-hint`, or `.graph-status-panel-wrap` CSS selectors.
No changes were required.

### Milestone 6 — Redirect test (new)

File: `tests/web/graph-to-map-redirect.test.ts` (new)

Five test cases verifying the `/p/:project/graph` → `/p/:project/map` redirect:

1. Navigating to `/p/testproject/graph` resolves to path `/p/testproject/map`
2. Resolved route name is `'map'` after navigating to `/graph`
3. Route params (`:project`) are preserved through the redirect
4. Navigating directly to `/p/testproject/map` resolves correctly to name `'map'`
5. The `/graph` route does not have its own name (it is redirect-only)

### MapView mount-reset test

File: `tests/web/graph-show-tests-toggle.test.ts` — Milestone 3, test 5

Updated to import `MapView.vue` (replacing the deleted `GraphView.vue`) and navigate
to `/p/testproject/map`. Verifies that `store.hideTests` is reset to `true` on mount.

## Test Files

| File | Change |
|------|--------|
| `tests/web/graph-show-tests-toggle.test.ts` | Updated `GraphView.vue` → `MapView.vue` import and route path; renamed describe block |
| `tests/web/AppSidebar.test.ts` | Updated `'Graph'` → `'Map'` in two label arrays; updated route in views array |
| `tests/web/graph-to-map-redirect.test.ts` | **New** — 5 redirect behaviour tests |
| All other listed files | No changes needed (already updated or outside scope) |
