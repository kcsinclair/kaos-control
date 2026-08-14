---
title: "Frontend Plan — Architecture Relationship Map"
type: plan-frontend
status: done
lineage: architecture-relationship-map
parent: lifecycle/requirements/architecture-relationship-map-2.md
labels:
    - frontend
    - architecture
    - visualization
    - graph
release: KC-Release5
---

# Frontend Plan — Architecture Relationship Map

Implements the read-only browse surface from [[architecture-relationship-map-2]]: a dedicated
**Architecture** navigation section whose relationship-map view renders the catalog architectures as a
typed-relationship graph, in both 2D (Cytoscape) and 3D (3d-force-graph), reusing the existing map
engines and consuming the endpoint from [[architecture-relationship-map-3-be]].

**Scope boundary.** No editing of any kind (FR-10) — pan/zoom/drag stay ephemeral. This plan reuses
the existing graph *engines* (`Map2DView.vue` / `ForceGraph3D.vue`) rather than adding a new
rendering dependency (NFR-1); where those components need per-node/per-edge styling not currently
supported, it adds **optional style props with today's behaviour as the default** so the main map
(`MapView.vue`) is unaffected. The guided questionnaire/scoring is **not** here — it is
[[onboarding-architecture-selection]]; the chosen-architecture overview is [[architecture-overview-view]].
The scale→colour mapping is fixed by the requirement's resolved question: labels starting `low` →
green, `medium` → yellow, `high` → orange, otherwise blue.

Cross-references:
- [[architecture-relationship-map-3-be]] — the `architecture-map` endpoint this consumes.
- [[architecture-relationship-map-5-test]] — Test plan.
- [[architecture-overview-view]] — co-tenant of the Architecture section; owns the section's
  chosen-architecture default destination (this plan wires the map route and a map-first default).
- [[onboarding-architecture-selection]] — links into this map as its browse surface (FR-11).

---

## Milestone 1 — API client, types & data composable

### Description

Add the typed client call and a composable that loads the derived map payload and exposes reactive
`nodes`/`edges`, mirroring `useGraphData`/`api/graph.ts` so downstream components stay engine-agnostic.

### Files to change

- **Edit** `web/src/api/graph.ts`: add `getArchitectureMap(project, stackFor?)` → `GET
  /p/:project/architecture-map` (append `?stack_for=` only when `stackFor` is set).
- **Edit** `web/src/types/api.ts`: reuse `GraphNode`/`GraphEdge`/`GraphData`; add the typed edge
  `kind` union (`'related' | 'evolves_into' | 'alternative_to' | 'composed_with' | 'related_to'`) as a
  loose string-compatible type so unknown kinds degrade to generic (NFR-5).
- **Add** `web/src/composables/useArchitectureMap.ts`: loads the base map on mount, exposes
  `nodes`, `edges`, `loading`, `error`, a `selectedArchId` ref, a `showStacks` boolean (default
  **false**, FR-8), and a `reload()` that re-fetches with `stack_for = selectedArchId` when
  `showStacks` is on. Reacts to the `artifact.indexed` / `file.changed` WS events already used by the
  map so the view refreshes on disk changes (FR-12).

### Acceptance criteria

- `getArchitectureMap('proj')` requests the base endpoint; `getArchitectureMap('proj', 'id')` appends
  `stack_for=id`.
- The composable exposes architecture-only nodes by default (`showStacks === false`), and a WS
  index/file event triggers a reload without a manual refresh (FR-12).

---

## Milestone 2 — Architecture nav section & route

### Description

Give architecture its own left-menu entry that opens the relationship map on its own URL (FR-1).

### Files to change

- **Edit** `web/src/router/index.ts`: add child routes under `/p/:project`:
  `architecture` (section landing → redirects to the map for now) and
  `architecture/map` (name `architecture-map`) → `ArchitectureMapView.vue` (below). Keep the map
  directly reachable by its own name/URL in all cases. Leave a documented seam so
  [[architecture-overview-view]] can later make the overview the section default when a chosen
  architecture exists; **until that lands, the section default is the map** (FR-1's "no chosen
  architecture" case).
- **Edit** `web/src/components/layout/AppSidebar.vue`: add an **Architecture** item
  (`{ label: 'Architecture', to: \`/p/${p}/architecture\`, icon: <lucide icon e.g. Boxes/Shapes> }`)
  to the same section array the `Map`/`Roadmap` items live in (AppSidebar.vue:124–129), following the
  existing item shape and tooltip/collapse behaviour.

### Acceptance criteria

- An **Architecture** entry appears in the left nav and, activated, lands on the relationship map at
  its own route `/p/:project/architecture/map` (FR-1).
- The map route is directly reachable by URL and highlights the Architecture nav item as active.

---

## Milestone 3 — ArchitectureMapView with 2D/3D switch (reusing existing engines)

### Description

The view shell: hosts the graph in 2D (default) or 3D, with a switch equivalent in node identity,
edges, encoding, legend, and click-through across modes (FR-9). 2D opens by default; last-used engine
is remembered per user as a nicety (resolved question).

### Files to change

- **Add** `web/src/views/project/ArchitectureMapView.vue`: modelled on `MapView.vue` — a
  `view` ref defaulting to `'2d'`, a 2D/3D toggle (reuse MapView.vue:100–114 markup), lazy-loaded
  `Map2DView` for 2D and `ForceGraph3D` for 3D, fed from `useArchitectureMap`. Persist last engine in
  `localStorage` (per-user nicety); on phones force 2D (mirror MapView.vue:57–65).
- **Reuse** `web/src/components/map/Map2DView.vue` and `web/src/components/map/ForceGraph3D.vue` — no
  new graph library (NFR-1). Styling extensions land in Milestones 4–5.

### Acceptance criteria

- The view opens in 2D by default; toggling to 3D and back preserves the same nodes, edges, and
  selection (FR-9).
- The chosen engine is restored on next visit; a phone viewport is forced to 2D.
- No new graph-rendering dependency is added to `package.json` (NFR-1).

---

## Milestone 4 — Decision-signal node encoding (colour + glyphs, data-driven)

### Description

Encode decision signals on each node from its `labels`: colour by scale (fixed mapping) plus glyphs
for offline-capable and mobile; unlabelled nodes render neutral (FR-5). Encoding must not rely on
colour alone (NFR-4).

### Files to change

- **Add** `web/src/components/map/archMapStyle.ts`: pure helpers —
  `scaleColour(labels): string` (any label starting `low` → green, `medium` → yellow, `high` →
  orange, else **blue**; case-insensitive; first matching label wins), and `signalGlyphs(labels)`
  returning glyph markers for offline-capable and mobile signals derived from labels (e.g.
  `offline*`, `mobile*`). Neutral default when no relevant labels.
- **Edit** `web/src/components/map/Map2DView.vue`: accept an optional `nodeStyle?(node) =>
  { color, glyphs, borderShape }` prop; when provided, apply it to Cytoscape node style; when absent,
  keep current styling (no impact on `MapView`). Render glyphs as node-badge labels/icons alongside
  the title so the signal is legible without colour (NFR-4).
- **Edit** `web/src/components/map/ForceGraph3D.vue`: accept the same optional `nodeStyle` prop and
  apply node colour + a text/sprite glyph in 3D; default unchanged when absent.
- **Edit** `web/src/views/project/ArchitectureMapView.vue`: pass `nodeStyle` built from
  `archMapStyle.ts`.

### Acceptance criteria

- A node whose labels include `high-scale` renders orange; `low-*` green; `medium-*` yellow; a node
  with no scale label renders blue; a node with no relevant labels renders in the neutral default
  (FR-5).
- Offline-capable and mobile signals show a distinct glyph/icon plus legend entry, conveying the
  signal without relying on colour (NFR-4).
- `MapView.vue` (the main map) renders exactly as before — the new props are optional and unset there.

---

## Milestone 5 — Typed & generic relationship edge styling + labels

### Description

Style edges by relationship kind (FR-4): `evolves_into` directed arrow, `alternative_to` dashed,
`composed_with` thick/overlay, generic `related` a plain undirected edge; label edges by kind in 2D.
In 3D, edges are unlabelled (legend-only is acceptable — resolved question). Missing typed fields fall
back to generic edges without error (FR-4/NFR-5).

### Files to change

- **Edit** `web/src/components/map/archMapStyle.ts`: add `edgeStyle(kind) =>
  { lineStyle, arrow, width, label }` implementing the mapping above; unknown kinds → generic.
- **Edit** `web/src/components/map/Map2DView.vue`: accept an optional `edgeStyle?(edge)` prop; when
  provided, apply Cytoscape edge style + kind label; default unchanged when absent.
- **Edit** `web/src/components/map/ForceGraph3D.vue`: accept the same optional `edgeStyle`, applying
  colour/width/arrow but **no** per-edge label (legend-only in 3D).
- **Edit** `web/src/views/project/ArchitectureMapView.vue`: pass `edgeStyle` from `archMapStyle.ts`.

### Acceptance criteria

- With typed fields present, each edge is visibly distinct by kind and labelled by kind in 2D (FR-4).
- With no typed fields, the same pairs render as generic edges and the view does not error (FR-4/NFR-5).
- 3D omits per-edge labels but keeps kind-distinct edge styling; the legend carries the key (FR-9,
  resolved 3D question).

---

## Milestone 6 — Legend (present-only), click-through, stack-reveal toggle, entry points

### Description

Complete the interactive surface: a legend keyed to what the current map actually contains (FR-6),
click-through to the underlying artifact (FR-7), the off-by-default related-stack reveal (FR-8), strict
read-only (FR-10), and entry points from the catalog README and onboarding (FR-11).

### Files to change

- **Add** `web/src/components/map/ArchMapLegend.vue` (or extend `MapLegend.vue` with an arch mode):
  two keys — relationship kinds (edge styles) and the decision-signal colour/glyph key — each rendered
  **only for kinds/signals present in the current nodes/edges** (FR-6). Compute presence from the
  loaded payload.
- **Edit** `web/src/views/project/ArchitectureMapView.vue`:
  - Click/activate a node → open the architecture artifact via the existing `ArtifactModal`
    (MapView.vue:172–178 pattern) or navigate to the artifact editor route (FR-7).
  - Add a **"Show related stacks"** toggle, default **off** (FR-8). When on with a selected
    architecture, set `showStacks` + `selectedArchId` and `reload()` (Milestone 1) so the stack ring
    appears; toggling off or deselecting returns to the architecture-only map.
  - Expose **no** create/edit/delete/persist-layout affordance; drag/pan/zoom remain ephemeral (FR-10).
- **Edit** `lifecycle/architecture/README.md`: add a line linking into the relationship-map view
  (FR-11). *(Content/docs link — coordinate with catalog seed owner [[architecture-templates]].)*
- **Edit** the project-create / onboarding entry component (grep for the onboarding/project-create
  flow that references [[onboarding-architecture-selection]]) to add a "Browse the architecture map"
  link into the map route (FR-11). If that flow is not yet present, add the link from the Architecture
  section landing and note the dependency on [[onboarding-architecture-selection]].

### Acceptance criteria

- The legend shows both keys and lists **only** the relationship kinds and decision signals present in
  the current map (FR-6).
- Clicking a node opens that architecture's artifact (FR-7).
- The stack-reveal toggle defaults to off; enabling it for a selected architecture reveals its
  compatible stacks via `related_to`; disabling it / deselecting hides them (FR-8).
- The view offers no editing/persist affordance; layout interactions never write back (FR-10).
- The catalog README and the onboarding/project-create flow both link into the map (FR-11).
