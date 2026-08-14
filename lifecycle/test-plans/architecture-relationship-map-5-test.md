---
title: "Test Plan — Architecture Relationship Map"
type: plan-test
status: approved
lineage: architecture-relationship-map
parent: lifecycle/requirements/architecture-relationship-map-2.md
labels:
    - test
    - architecture
    - visualization
    - graph
release: KC-Release5
---

# Test Plan — Architecture Relationship Map

Verifies the read-only relationship map from [[architecture-relationship-map-2]] end to end: the
derived backend read model + endpoint from [[architecture-relationship-map-3-be]] and the browse
surface from [[architecture-relationship-map-4-fe]]. Coverage maps 1:1 to the requirement's FR/NFR
acceptance criteria, with explicit graceful-degradation cases (NFR-5) and a freshness case (FR-12).

**Test surfaces.** Go unit/HTTP tests for the index read model and endpoint (`internal/index`,
`internal/http`); integration tests in `tests/` driving the running server (the `testEnv` harness
auto-logins as admin and exposes URL helpers); component/behaviour tests in `web/src` for the view,
encoding, legend, and toggles. Each area below also produces/updates a `test` artifact under
`lifecycle/tests/` describing what the code covers.

Cross-references:
- [[architecture-relationship-map-3-be]] — backend under test.
- [[architecture-relationship-map-4-fe]] — frontend under test.
- [[architectural-artefacts-5-test]] — sibling that seeds indexed architecture/tech-stack fixtures we reuse.

---

## Milestone 1 — Backend: typed-field parsing & graceful absence (FR-4, NFR-5)

### Description

Unit-test the parser changes so typed relationship fields classify correctly when present and are inert
when absent.

### Files to change

- **Add** `internal/artifact/archmap_links_test.go`.
- **Add/Update** `lifecycle/tests/architecture-relationship-map-parser-test.md` (artifact).

### Test cases / acceptance criteria

- An architecture fixture with none of `evolves_into`/`alternative_to`/`composed_with` parses with
  **zero** new ParseErrs and unchanged link output (regression).
- A fixture with `evolves_into: [architecture/architectures/foo.md]` yields exactly one link with
  `Kind == "evolves_into"` resolving to the same node id as the `related_to`/wiki form.
- A malformed typed field (scalar not list) fails only that artifact's parse and never aborts the scan
  or panics (NFR-5).

---

## Milestone 2 — Backend: `ArchitectureMap` read model (FR-2, FR-3, FR-4, FR-8, NFR-2, NFR-5)

### Description

Table-driven tests over `Index.ArchitectureMap` on a seeded catalog fixture, asserting node scoping,
edge derivation/classification, the stack ring, and degradation.

### Files to change

- **Add** `internal/index/archmap_test.go`.
- **Add/Update** `lifecycle/tests/architecture-relationship-map-readmodel-test.md` (artifact).

### Test cases / acceptance criteria

- `ArchitectureMap("")` returns exactly one node per `type: architecture` fixture and **no**
  tech-stack/other-type base nodes (FR-2).
- Every arch↔arch body wiki-link in the fixture appears as exactly one edge; a wiki-only pair is
  `Kind == "related"` (FR-3). A pair linked by both wiki and `related_to` collapses to a single generic
  edge; a typed kind on that pair wins over generic (FR-4).
- `ArchitectureMap("<archId>")` adds only that architecture's `related_to` tech-stack nodes + connecting
  `related_to` edges and nothing for other architectures (FR-8, NFR-2).
- An unresolved wiki target (typo/removed file) is dropped, not rendered as a dangling node; a node
  with no labels comes back with `labels: []` — no error return (NFR-5).
- `stack_for` naming a non-architecture id returns the base map with no error (NFR-5).

---

## Milestone 3 — Backend: HTTP endpoint contract & read-only guarantee (FR-3, FR-8, FR-10)

### Description

Integration tests through the running server verifying the endpoint shape, params, auth parity with
`/graph`, and absence of any mutating verb.

### Files to change

- **Add** `tests/architecture_map_test.go` (uses `testEnv` admin auto-login + URL helpers).
- **Add/Update** `lifecycle/tests/architecture-relationship-map-endpoint-test.md` (artifact).

### Test cases / acceptance criteria

- `GET /api/p/:project/architecture-map` → `200` with `{nodes, edges}`; nodes carry `labels`, typed
  edges carry `kind` + `label` (FR-3/FR-4).
- `?stack_for=<archId>` includes that architecture's stack ring; omitting the param returns the
  architecture-only base map (FR-8 default-off).
- The route requires the same auth as `GET /api/p/:project/graph`; `POST`/`PUT`/`DELETE`/`PATCH` on
  the path are rejected (no mutating variant exists) — read-only (FR-10).
- Unknown project returns the same error contract as `handleGraph` (no panic/500 leak).

---

## Milestone 4 — Backend: freshness without manual rebuild (FR-12)

### Description

Prove the map reflects on-disk catalog changes via the existing index/watcher path, and that the main
artifact/lineage graph is untouched (requirement non-goal).

### Files to change

- **Add** cases in `tests/architecture_map_test.go`.
- **Add/Update** `lifecycle/tests/architecture-relationship-map-freshness-test.md` (artifact).

### Test cases / acceptance criteria

- Writing a new `type: architecture` file into the indexed tree and letting the index re-scan makes the
  new node appear on the next endpoint call **without** restarting the process (FR-12).
- Removing an architecture file removes its node (and its now-dangling edges) on the next call (FR-12).
- The `/graph` response for the same project is unchanged by this lineage's changes (non-goal: main
  graph untouched).

---

## Milestone 5 — Frontend: nav, route, 2D-default & 2D/3D equivalence (FR-1, FR-9, NFR-1)

### Description

Behaviour tests for the Architecture nav entry, the map route, the default engine, and cross-mode
equivalence.

### Files to change

- **Add** `web/src/views/project/__tests__/ArchitectureMapView.spec.ts`.
- **Add/Update** `lifecycle/tests/architecture-relationship-map-view-test.md` (artifact).

### Test cases / acceptance criteria

- An **Architecture** nav item exists and routes to `/p/:project/architecture/map`, reachable directly
  by URL (FR-1).
- The view opens in **2D** by default; toggling to 3D and back preserves nodes, edges, and selection
  (FR-9). Last-used engine is restored on remount (nicety).
- No new graph-rendering dependency appears in `web/package.json` — engines are the existing Cytoscape
  / 3d-force-graph (NFR-1).

---

## Milestone 6 — Frontend: decision-signal encoding, edge styling, legend (FR-5, FR-4, FR-6, NFR-4)

### Description

Unit-test the pure style helpers and the legend's present-only behaviour; assert colour+glyph encoding
does not rely on colour alone.

### Files to change

- **Add** `web/src/components/map/__tests__/archMapStyle.spec.ts` and
  `web/src/components/map/__tests__/ArchMapLegend.spec.ts`.
- **Add/Update** `lifecycle/tests/architecture-relationship-map-encoding-test.md` (artifact).

### Test cases / acceptance criteria

- `scaleColour`: `low*` → green, `medium*` → yellow, `high*` → orange, otherwise blue; no relevant
  label → neutral default (FR-5).
- Offline-capable and mobile labels yield distinct glyphs plus legend entries — signal is conveyed
  without colour (NFR-4).
- `edgeStyle`: `evolves_into` directed, `alternative_to` dashed, `composed_with` thick, unknown →
  generic (FR-4).
- The legend lists **only** the relationship kinds and decision signals present in the supplied map,
  and nothing absent (FR-6).

---

## Milestone 7 — Frontend: click-through, stack-reveal toggle, read-only, entry points (FR-7, FR-8, FR-10, FR-11)

### Description

Behaviour tests for interaction and the entry-point links.

### Files to change

- **Add** cases in `web/src/views/project/__tests__/ArchitectureMapView.spec.ts`.
- **Add/Update** `lifecycle/tests/architecture-relationship-map-interaction-test.md` (artifact).

### Test cases / acceptance criteria

- Clicking a node opens the corresponding architecture artifact (modal or editor route) (FR-7).
- The **Show related stacks** toggle defaults to **off**; enabling it for a selected architecture
  requests `stack_for` and reveals the stack ring; disabling / deselecting hides it (FR-8).
- The view exposes no create/edit/delete/persist-layout control; pan/zoom/drag never issue a write
  (FR-10).
- The catalog `README` and the onboarding/project-create flow both contain a link into the map route
  (FR-11).

---

## Milestone 8 — Full-catalog smoke (NFR-2, NFR-3)

### Description

A smoke test over the shipped ~9-architecture catalog confirming legible, prompt rendering at real
scale.

### Files to change

- **Add/Update** `lifecycle/tests/architecture-relationship-map-smoke-test.md` (artifact) and, where an
  E2E harness exists, a case in the end-to-end suite ([[end-to-end-smoke-tests-3-test]] style).

### Test cases / acceptance criteria

- The base map renders one legible node per shipped architecture with readable node/edge labels in 2D
  at the current catalog size (NFR-2).
- Initial base-map render and toggling 2D/3D or the stack reveal are interactive at this scale (NFR-3).
