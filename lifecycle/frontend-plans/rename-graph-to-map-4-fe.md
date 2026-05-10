---
title: "Frontend Plan: Rename Graph to Map in UI and Routing"
type: plan-frontend
status: done
lineage: rename-graph-to-map
parent: lifecycle/requirements/rename-graph-to-map-2.md
---

# Frontend Plan: Rename Graph to Map in UI and Routing

## Overview

Rename all user-facing references from "Graph" to "Map" in the Vue 3 SPA: navigation labels, route path and name, view and component filenames, aria labels, loading text, and CSS class selectors. Internal developer-facing files (stores, API client, composables, types) are explicitly out of scope per the resolved questions in the requirement.

## Milestones

### Milestone 1 — Route and Navigation Update

**Description:** Change the Vue Router path from `graph` to `map`, rename the route, add a redirect from the old path, and update the sidebar label.

**Files to change:**

- `web/src/router/index.ts` — Change the route definition: `path: 'graph'` → `path: 'map'`, `name: 'graph'` → `name: 'map'`. Update the lazy import path from `@/views/project/GraphView.vue` to `@/views/project/MapView.vue`. Add a redirect entry: `{ path: 'graph', redirect: { name: 'map' } }`.
- `web/src/components/layout/AppSidebar.vue` — Change the navigation item label from `'Graph'` to `'Map'` and update the `to` path from `/p/${p}/graph` to `/p/${p}/map` (line ~91).

**Acceptance criteria:**

- [ ] Sidebar displays "Map" label
- [ ] Clicking "Map" navigates to `/p/:project/map`
- [ ] Route name is `map`
- [ ] Navigating to `/p/:project/graph` redirects to `/p/:project/map`
- [ ] No `router-link` or `router.push` references the old path without the redirect

### Milestone 2 — Rename View Component

**Description:** Rename the main view file from `GraphView.vue` to `MapView.vue`.

**Files to change:**

- `web/src/views/project/GraphView.vue` → rename to `web/src/views/project/MapView.vue`

**Acceptance criteria:**

- [ ] `GraphView.vue` no longer exists
- [ ] `MapView.vue` exists and is correctly imported by the router (updated in Milestone 1)
- [ ] `make build-web` succeeds

### Milestone 3 — Rename Component Directory and Files

**Description:** Rename the `components/graph/` directory to `components/map/` and rename user-facing component files within it.

**Files to change:**

- `web/src/components/graph/` → rename directory to `web/src/components/map/`
- Within the renamed directory:
  - `Graph2DView.vue` → `Map2DView.vue`
  - `GraphFilters.vue` → `MapFilters.vue`
  - `GraphLegend.vue` → `MapLegend.vue`
- Files that stay named as-is (internal, not user-facing):
  - `ForceGraph3D.vue` — internal 3D rendering wrapper
  - `LabelModal.vue` — no "graph" in name
  - `LayoutSelector.vue` — no "graph" in name
  - `graphConstants.ts` — developer-facing constant file
  - `layoutConfigs.ts` — no "graph" in name

**Import updates required in:**

- `web/src/views/project/MapView.vue` (formerly GraphView.vue) — update all component imports from `@/components/graph/...` to `@/components/map/...` and from `Graph2DView` / `GraphFilters` / `GraphLegend` to `Map2DView` / `MapFilters` / `MapLegend`.
- Any other files importing from `@/components/graph/` — check `RoadmapView.vue`, `ArtifactEditorView.vue`, and any other consumers. Note: `RoadmapGraphView.vue` in `components/releases/` is NOT renamed per the resolved questions.

**Acceptance criteria:**

- [ ] `components/graph/` directory no longer exists
- [ ] `components/map/` directory exists with all files
- [ ] `Graph2DView.vue`, `GraphFilters.vue`, `GraphLegend.vue` are renamed with `Map` prefix
- [ ] `ForceGraph3D.vue`, `LabelModal.vue`, `LayoutSelector.vue`, `graphConstants.ts`, `layoutConfigs.ts` retain original names
- [ ] All imports resolve correctly
- [ ] `make build-web` succeeds with zero TypeScript errors

### Milestone 4 — Update User-Facing Strings

**Description:** Replace all user-visible text that says "Graph" or "graph" with "Map" or "map" in renamed and related component files.

**Files to change:**

- `web/src/views/project/MapView.vue` (formerly GraphView.vue):
  - `aria-label="Graph view mode"` → `aria-label="Map view mode"` (line ~87)
  - `"Loading graph…"` → `"Loading map…"` (line ~107)
- `web/src/components/map/Map2DView.vue` (formerly Graph2DView.vue):
  - `aria-label="2D artifact graph"` → `aria-label="2D artifact map"` (line ~435)
- `web/src/components/map/LayoutSelector.vue`:
  - `aria-label="2D graph layout controls"` → `aria-label="2D map layout controls"` (line ~19)
  - `aria-label="Select graph layout algorithm"` → `aria-label="Select map layout algorithm"` (line ~26)
  - `aria-label="Toggle directed graph mode"` → `aria-label="Toggle directed map mode"` (line ~37)

**Acceptance criteria:**

- [ ] No user-visible string in view or component files references "graph" (verified by grep of `web/src/views/` and `web/src/components/map/` for case-insensitive "graph" in string literals)
- [ ] Aria labels use "map" terminology
- [ ] Loading text says "Loading map…"

### Milestone 5 — Rename CSS Classes

**Description:** Rename CSS class selectors used as test hooks and styling anchors from `graph-*` to `map-*`.

**Files to change:**

- `web/src/views/project/MapView.vue` — rename CSS classes in both template and `<style>` block:
  - `.graph-view` → `.map-view`
  - `.graph-main` → `.map-main`
  - `.graph-state` → `.map-state`
  - `.graph-legend-wrap` → `.map-legend-wrap`
  - `.graph-hint` → `.map-hint`
  - `.graph-status-panel-wrap` → `.map-status-panel-wrap`

**Acceptance criteria:**

- [ ] No CSS class starting with `.graph-` exists in the renamed view file
- [ ] Styling and layout are visually identical to the pre-rename state
- [ ] `make build-web` succeeds

### Milestone 6 — Final Verification

**Description:** End-to-end verification that all changes work together.

**Verification steps:**

- [ ] `make build-web` succeeds with zero errors
- [ ] Dev server starts and the Map view loads at `/p/:project/map`
- [ ] Redirect from `/p/:project/graph` to `/p/:project/map` works
- [ ] 3D visualisation renders and responds to interaction (zoom, drag, click)
- [ ] 2D visualisation renders and responds to interaction
- [ ] Filter panel, legend, and layout selector function correctly
- [ ] `RoadmapGraphView.vue` in releases context is unaffected

## Dependencies

- [[rename-graph-to-map]] backend plan confirms no backend changes needed
- [[rename-graph-to-map]] test plan must update test references after this plan is complete
