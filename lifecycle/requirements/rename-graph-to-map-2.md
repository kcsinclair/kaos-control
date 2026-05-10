---
title: Rename Graph to Map in UI and Routing
type: requirement
status: blocked
lineage: rename-graph-to-map
created: "2026-05-10"
priority: medium
parent: lifecycle/ideas/rename-graph-to-map.md
labels:
    - frontend
    - usability
    - enhancement
    - vue
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

# Rename Graph to Map in UI and Routing

## Problem

The navigation and routing currently use the term "Graph" for the artifact visualisation view. While technically accurate (the view renders a force-directed graph), the term is unintuitive for non-technical users. Users naturally refer to these visualisations as "maps" — a term that better conveys the navigational, exploratory purpose of the view within the Innovation Maker product.

The mismatch between user mental model ("map") and UI label ("Graph") creates unnecessary cognitive friction and reduces discoverability.

## Goals / Non-goals

### Goals

- Rename the sidebar navigation label from "Graph" to "Map".
- Change the route path from `/p/:project/graph` to `/p/:project/map`.
- Change the route name from `graph` to `map`.
- Rename user-facing component filenames from `Graph*` to `Map*` (e.g. `GraphView.vue` → `MapView.vue`).
- Update all user-facing strings: aria labels, loading text, page titles, and help text that reference "graph".
- Update all Playwright and other end-to-end tests that reference the graph route, menu label, or selectors.
- Maintain a redirect from the old `/p/:project/graph` path to `/p/:project/map` so that bookmarks and shared links continue to work.

### Non-goals

- Renaming internal graph-rendering library references (three.js, Cytoscape, `3d-force-graph`) — these are implementation details, not user-facing.
- Renaming the backend API endpoint (`/api/graph` or similar) — the API is not user-facing and renaming it adds risk for no user benefit. If the API is renamed, that is a separate change.
- Changing the graph icon in the sidebar (currently `Network` from lucide) — icon choice is orthogonal to the label rename. May be revisited separately.
- Renaming internal store files (`stores/graph.ts`), API client files (`api/graph.ts`), or composables (`useGraphData.ts`) — these are developer-facing, not user-facing, and renaming them adds churn without user benefit.

## Detailed Requirements

### Functional

1. **Sidebar label** — The `AppSidebar.vue` navigation item must display "Map" instead of "Graph".

2. **Route path** — The Vue Router path must change from `graph` to `map` under the `/p/:project/` prefix.

3. **Route name** — The named route must change from `graph` to `map`.

4. **Redirect** — A route redirect must be configured from `/p/:project/graph` to `/p/:project/map` so existing bookmarks resolve correctly. The redirect must be a `301` (permanent) or Vue Router `redirect` property.

5. **View component filename** — `GraphView.vue` must be renamed to `MapView.vue`. The router import must be updated accordingly.

6. **User-facing component filenames** — The following components in `web/src/components/graph/` must be renamed:
   - `Graph2DView.vue` → `Map2DView.vue`
   - `GraphFilters.vue` → `MapFilters.vue`
   - `GraphLegend.vue` → `MapLegend.vue`
   - The containing directory `components/graph/` → `components/map/`

7. **User-facing strings** — All occurrences of "Graph" or "graph" in user-visible text must be replaced with "Map" or "map":
   - Aria labels (e.g. `aria-label="Graph view mode"` → `aria-label="Map view mode"`)
   - Loading text (e.g. `"Loading graph…"` → `"Loading map…"`)
   - CSS class names used as test selectors (e.g. `.graph-view`, `.graph-main`) should be renamed to `.map-view`, `.map-main`.

8. **Tests** — All Playwright end-to-end tests and any other test files that reference the old route (`/graph`), the old menu label (`Graph`), or old CSS selectors (`.graph-view`) must be updated to use the new names.

### Non-functional

1. **No broken links** — After the rename, no internal `router-link`, `router.push`, or `<a>` element should reference the old path without going through the redirect.

2. **Build passes** — `make build-web` must succeed with zero TypeScript errors after the rename.

3. **No runtime regressions** — The 2D and 3D visualisation views must render identically to their pre-rename behaviour. Node interaction (click, drag, zoom) must be unaffected.

## Acceptance Criteria

- [ ] Sidebar navigation displays "Map" label instead of "Graph"
- [ ] Clicking "Map" in the sidebar navigates to `/p/:project/map`
- [ ] The route name is `map` (verified via Vue DevTools or `router.currentRoute.value.name`)
- [ ] Navigating to `/p/:project/graph` redirects to `/p/:project/map`
- [ ] `GraphView.vue` has been renamed to `MapView.vue`
- [ ] The `components/graph/` directory has been renamed to `components/map/`
- [ ] `Graph2DView.vue`, `GraphFilters.vue`, and `GraphLegend.vue` are renamed with `Map` prefix
- [ ] All user-facing strings (aria labels, loading text) say "map" not "graph"
- [ ] CSS classes `.graph-view` and `.graph-main` are renamed to `.map-view` and `.map-main`
- [ ] All Playwright tests pass with updated selectors and route references
- [ ] `make build-web` succeeds with no TypeScript errors
- [ ] 3D and 2D visualisations render and behave correctly after the rename
- [ ] Related artifacts: [[rename-graph-to-map]]

## Open Questions

- Should the `RoadmapGraphView.vue` component (used in the releases/roadmap context) also be renamed, or does "graph" remain appropriate there since it is not the primary map view?
- Should internal developer-facing files (`stores/graph.ts`, `api/graph.ts`, `useGraphData.ts`) be renamed for consistency, or left as-is to minimise churn? The current recommendation is to leave them, but the product owner may prefer full consistency.
