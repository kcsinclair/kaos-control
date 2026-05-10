---
title: "Test Plan: Rename Graph to Map in UI and Routing"
type: plan-test
status: done
lineage: rename-graph-to-map
parent: lifecycle/requirements/rename-graph-to-map-2.md
---

# Test Plan: Rename Graph to Map in UI and Routing

## Overview

Update all existing test files that reference "Graph" route paths, component names, CSS selectors, or labels to use the new "Map" names. Verify that all tests pass after the [[rename-graph-to-map]] frontend rename is complete. No new test logic is required — this is a mechanical rename of references within existing tests, plus one new test for the redirect behaviour.

## Milestones

### Milestone 1 — Update Graph Component Test Imports and References

**Description:** Update test files that directly test renamed graph components to import from the new paths and use the new component names.

**Files to change:**

- `tests/web/graph-show-tests-toggle.test.ts`
  - Update imports from `@/components/graph/GraphFilters.vue` → `@/components/map/MapFilters.vue`
  - Update imports from `@/components/graph/Graph2DView.vue` → `@/components/map/Map2DView.vue`
  - Update imports from `@/components/graph/ForceGraph3D.vue` → `@/components/map/ForceGraph3D.vue`
  - Update component name references in test descriptions if user-facing (e.g. `"GraphFilters"` → `"MapFilters"`)

- `tests/web/Graph2DView.layout.spec.ts`
  - Update import from `@/components/graph/Graph2DView.vue` → `@/components/map/Map2DView.vue`
  - Update component references in `describe` blocks and assertions

- `tests/web/Graph2DView.approvedRing.test.ts`
  - Update import from `@/components/graph/Graph2DView.vue` → `@/components/map/Map2DView.vue`
  - Update component references

- `tests/web/Graph2DView.filters.spec.ts`
  - Update import from `@/components/graph/Graph2DView.vue` → `@/components/map/Map2DView.vue`

- `tests/web/Graph2DView.perf.spec.ts`
  - Update import from `@/components/graph/Graph2DView.vue` → `@/components/map/Map2DView.vue`

- `tests/web/graphConstants.test.ts`
  - Update import from `@/components/graph/graphConstants` → `@/components/map/graphConstants`

- `tests/web/graph.store.layout.spec.ts`
  - Update import if referencing component path (store import `@/stores/graph` stays as-is per non-goals)

- `tests/web/LayoutSelector.spec.ts`
  - Update import from `@/components/graph/LayoutSelector.vue` → `@/components/map/LayoutSelector.vue`

**Acceptance criteria:**

- [ ] All test files import from `@/components/map/` instead of `@/components/graph/`
- [ ] Component names in test descriptions reflect the rename where user-facing
- [ ] All listed test files compile without TypeScript errors

### Milestone 2 — Update Sidebar and Navigation Tests

**Description:** Update the AppSidebar test to expect "Map" instead of "Graph" in navigation labels and route paths.

**Files to change:**

- `tests/web/AppSidebar.test.ts`
  - Line ~204: Change `expectedLabels` entry from `'Graph'` to `'Map'`
  - Line ~326: Change `allExpectedLabels` entry from `'Graph'` to `'Map'`
  - Line ~574: Change route reference from `'/p/testproject/graph'` to `'/p/testproject/map'`

**Acceptance criteria:**

- [ ] AppSidebar test expects "Map" label
- [ ] AppSidebar test expects `/p/testproject/map` route path
- [ ] `npm run test` (or vitest) passes for `AppSidebar.test.ts`

### Milestone 3 — Update Cross-View and Hide-Done-Items Tests

**Description:** Update tests that reference GraphView or the graph store in cross-view contexts.

**Files to change:**

- `tests/web/hide-done-items/graph-toggle.test.ts`
  - Update any imports from `@/components/graph/` to `@/components/map/`
  - Store import (`@/stores/graph`) stays as-is per non-goals

- `tests/web/hide-done-items/cross-view-consistency.test.ts`
  - Update any references to `GraphView` → `MapView`
  - Update import path if it references `@/views/project/GraphView.vue` → `@/views/project/MapView.vue`
  - Store import stays as-is

**Acceptance criteria:**

- [ ] All hide-done-items tests compile and pass
- [ ] Cross-view consistency tests correctly reference MapView

### Milestone 4 — Update Test Helper Factories

**Description:** Update the test seed helper if it has user-facing graph references. Note: function names like `makeGraphNode()` are developer-facing and stay per non-goals.

**Files to review (changes only if user-facing strings exist):**

- `tests/web/helpers/seed_artifacts.ts`
  - Review `makeGraphNode()`, `makeGraphEdge()`, `makeGraphNodesForAllStatuses()` — these are developer-facing factory functions. No rename needed per non-goals.

**Acceptance criteria:**

- [ ] Helper file reviewed; no user-facing "graph" strings exist in seed data
- [ ] Helper functions continue to work with all consuming tests

### Milestone 5 — Update CSS Selector References in Tests

**Description:** Update any test files that use the old CSS class selectors as query targets.

**Files to change:**

- Search all files in `tests/` for `.graph-view`, `.graph-main`, `.graph-state`, `.graph-legend-wrap`, `.graph-hint`, `.graph-status-panel-wrap`
- Replace with `.map-view`, `.map-main`, `.map-state`, `.map-legend-wrap`, `.map-hint`, `.map-status-panel-wrap`

**Acceptance criteria:**

- [ ] No test file references `.graph-view`, `.graph-main`, or other renamed CSS classes
- [ ] All CSS selector-based test queries use the new `.map-*` class names

### Milestone 6 — Add Redirect Test

**Description:** Add a test verifying that navigating to the old `/p/:project/graph` route redirects to `/p/:project/map`.

**Files to change:**

- Create test case within an existing router or navigation test file (e.g. `tests/web/AppSidebar.test.ts` or a new focused test)
- Test should: push to `/p/testproject/graph`, assert `router.currentRoute.value.path` resolves to `/p/testproject/map` and `router.currentRoute.value.name` is `'map'`

**Acceptance criteria:**

- [ ] A test exists that verifies the `/graph` → `/map` redirect
- [ ] The redirect test passes

### Milestone 7 — Full Test Suite Pass

**Description:** Run the complete test suite and verify no regressions.

**Verification steps:**

- [ ] `make test-unit` passes (Go tests unaffected)
- [ ] `npx vitest run` (or equivalent) passes for all web tests
- [ ] `make build-web` succeeds with zero TypeScript errors
- [ ] No test file imports from `@/components/graph/` (verified by grep)
- [ ] No test file references the old route path `/graph` except in redirect test assertions

## Dependencies

- [[rename-graph-to-map]] frontend plan must be completed first — test updates depend on the renamed files existing at their new paths
- [[rename-graph-to-map]] backend plan confirms no backend test changes needed

## Test File Inventory

| Test file | Milestone |
|-----------|-----------|
| `tests/web/graph-show-tests-toggle.test.ts` | 1 |
| `tests/web/Graph2DView.layout.spec.ts` | 1 |
| `tests/web/Graph2DView.approvedRing.test.ts` | 1 |
| `tests/web/Graph2DView.filters.spec.ts` | 1 |
| `tests/web/Graph2DView.perf.spec.ts` | 1 |
| `tests/web/graphConstants.test.ts` | 1 |
| `tests/web/graph.store.layout.spec.ts` | 1 |
| `tests/web/LayoutSelector.spec.ts` | 1 |
| `tests/web/AppSidebar.test.ts` | 2 |
| `tests/web/hide-done-items/graph-toggle.test.ts` | 3 |
| `tests/web/hide-done-items/cross-view-consistency.test.ts` | 3 |
| `tests/web/helpers/seed_artifacts.ts` | 4 |
