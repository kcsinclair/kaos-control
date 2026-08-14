---
title: "Backend Plan — Architecture Relationship Map"
type: plan-backend
status: done
lineage: architecture-relationship-map
parent: lifecycle/requirements/architecture-relationship-map-2.md
labels:
    - backend
    - architecture
    - visualization
    - graph
release: KC-Release5
---

# Backend Plan — Architecture Relationship Map

Implements the server-side data source for the read-only relationship map defined in
[[architecture-relationship-map-2]]: a purpose-built, derived graph payload whose **nodes are the
`type: architecture` catalog artifacts**, whose **edges are the architecture↔architecture links
already present in the catalog** (body wiki-links today; typed relationship fields when they later
exist), and which can optionally include a selected architecture's compatible **tech-stack** nodes
via `related_to`.

**Scope boundary.** This plan owns only the *derived read model + endpoint*. It performs **no**
writes, adds **no** relationship-authoring API, and defines **no** colour/glyph mapping (that is
presentation, owned by [[architecture-relationship-map-4-fe]]). It relies on architecture/tech-stack
artifacts already being indexed as first-class artifacts — that indexing and the type vocabulary are
owned by [[architectural-artefacts-3-be]]; this plan consumes them read-only. Typed relationship
frontmatter fields (`evolves_into`, `alternative_to`, `composed_with`) are made *parse-ready* here so
the map degrades gracefully, but the authoring UX for them is out of scope
([[artefact-relationship-labels-and-links]], [[architecture-templates]]).

Cross-references:
- [[architecture-relationship-map-4-fe]] — Frontend plan (consumes the endpoint below).
- [[architecture-relationship-map-5-test]] — Test plan (asserts endpoint shape & degradation).
- [[architectural-artefacts-3-be]] — indexing of architecture/tech-stack artifacts & type vocab (dependency).
- [[onboarding-architecture-selection]] — downstream consumer that links into the browse surface.

---

## Milestone 1 — Parse-ready typed relationship fields (graceful readiness, FR-4/NFR-5)

### Description

Teach the artifact parser to recognise the three optional typed architecture-relationship frontmatter
fields so that, *if and when* they appear on an artifact, they surface as classified links — and their
**absence produces no error** (FR-4 "degrade gracefully to FR-3 generic edges"; NFR-5). This milestone
adds no requirement that any artifact carry them; it only makes them first-class if present.

### Files to change

- **Edit** `internal/artifact/artifact.go`:
  - Add edge-kind constants next to `EdgeKindRelatedTo` (artifact.go:39):
    `EdgeKindEvolvesInto = "evolves_into"`, `EdgeKindAlternativeTo = "alternative_to"`,
    `EdgeKindComposedWith = "composed_with"`.
  - Add three optional slice fields to `Frontmatter` (next to `Related`, artifact.go:89):
    `EvolvesInto []string \`yaml:"evolves_into,omitempty"\``, `AlternativeTo []string \`yaml:"alternative_to,omitempty"\``,
    `ComposedWith []string \`yaml:"composed_with,omitempty"\``. All `omitempty`, all optional — no
    validation, no ParseErr when missing.
  - In `extractLinks`, add `addFM(EdgeKindEvolvesInto, fm.EvolvesInto)` etc. alongside the existing
    `addFM(EdgeKindRelatedTo, …)` calls (artifact.go:367). Targets are normalised through the existing
    `normaliseLinkTarget`, so both path and slug forms resolve like `related_to` does today.

### Acceptance criteria

- An architecture artifact with none of the three fields parses with zero new ParseErrs and produces
  exactly the links it produces today (regression-safe).
- An artifact carrying `evolves_into: [architecture/architectures/foo.md]` yields one link with
  `Kind == "evolves_into"` whose `To` resolves to the same node id the wiki-link/`related_to` forms
  resolve to.
- Malformed values (e.g. a scalar instead of a list) fail only that artifact's parse via the existing
  frontmatter-unmarshal error path and do not panic or abort the index scan.

---

## Milestone 2 — Derived architecture-map read model in the index

### Description

Add a dedicated read function that returns the map payload, so classification/scoping lives in one
tested place rather than being re-derived by the client (FR-2, FR-3, FR-8, FR-12). It reuses the
existing indexed `artifacts`, `labels_index`, and `links` tables — no schema change, so freshness is
automatic via the existing watcher/re-index (FR-12).

The function scopes nodes to `type = 'architecture'`, then derives edges by walking the `links` table
and keeping only links whose **both** endpoints are architecture nodes. Each surviving edge is
classified by kind: a typed kind (`evolves_into` / `alternative_to` / `composed_with`) when present;
otherwise a generic kind (`wiki` body-link or `related_to`) surfaced as `related` (FR-3/FR-4). When
`include_stacks` is requested for a given architecture id, the payload additionally includes that one
architecture's `related_to` tech-stack nodes and the `related_to` edges connecting them (FR-8) —
nothing else, to keep the base map legible (NFR-2).

### Files to change

- **Edit** `internal/index/index.go`:
  - Add `type ArchMapData struct { Nodes []*GraphNode; Edges []*GraphEdge }` (reuse the existing
    `GraphNode`/`GraphEdge` types at index.go:781–799 so `labels` ride along for decision-signal
    encoding, and `Kind`/`Label` carry the relationship classification).
  - Add `func (idx *Index) ArchitectureMap(stackFor string) (*ArchMapData, error)`:
    - Select architecture nodes: reuse the `Graph` node-loading path but constrain to
      `type = 'architecture'` (a scoped query mirroring index.go:802–861, including the
      `labels_index` population loop so each node's `Labels` is filled; unlabelled → `[]string{}`).
    - Build `archIDs` set. Query `links`; keep rows where `archIDs[src] && archIDs[dst]`. Map kind:
      typed kinds pass through with `Label` = human kind ("evolves into" / "alternative to" /
      "composed with"); `wiki` and `related_to` between two architectures collapse to
      `Kind = "related"`, `Label = ""` (generic, FR-3). De-duplicate an unordered arch pair that is
      linked both by a wiki body-link and a `related_to` down to a single generic edge unless a typed
      kind exists (typed wins).
    - When `stackFor != ""` and it is a known architecture id: add its `related_to` targets that are
      `type = 'tech-stack'` as nodes (with their labels), and add the `related_to` edges
      (`Kind = "related_to"`) from that architecture to those stacks. Ignore `stackFor` values that
      are not architecture nodes (return base map, no error — NFR-5).
  - Do **not** invent edges: if an artifact's wiki-link target does not resolve to an architecture
    node (typo, removed file), the link is simply dropped from the map (NFR-5), never rendered as a
    dangling node.

### Acceptance criteria

- `ArchitectureMap("")` on the shipped catalog returns exactly one node per
  `type: architecture` file and **no** tech-stack or other-type base nodes (FR-2).
- Every arch↔arch body wiki-link in the current catalog appears as exactly one edge; a pair linked
  only by a wiki body-link appears with `Kind == "related"` (FR-3).
- Given a fixture architecture with `evolves_into` set, the corresponding edge has
  `Kind == "evolves_into"` and a non-empty `Label`; removing that field degrades the same pair to a
  generic `related` edge with no error (FR-4).
- `ArchitectureMap("<archId>")` adds that architecture's `related_to` tech-stacks as nodes plus the
  connecting `related_to` edges, and adds nothing for any other architecture (FR-8/NFR-2).
- A malformed/partial artifact (missing labels, unresolved wiki target) yields a valid payload with
  the affected node in neutral shape / the affected edge omitted — never an error return (NFR-5).

---

## Milestone 3 — HTTP endpoint & route wiring

### Description

Expose Milestone 2 as a read-only JSON endpoint the frontend consumes, following the existing
`handleGraph` conventions (project-from-context, `writeJSON`, `apiError`). Read-only: `GET` only,
behind the same project auth as `/graph` (FR-10).

### Files to change

- **Edit** `internal/http/graph.go`: add `func (s *Server) handleArchitectureMap(w, r)`:
  - `p := projectFromCtx(...)`; nil-guard as in `handleGraph` (graph.go:21–25).
  - Read optional `?stack_for=<artifactId>` query param; pass to `p.Idx.ArchitectureMap(stackFor)`.
  - On error `writeJSON(… apiError("db_error", err.Error()))`; on success `writeJSON(200, data)`.
- **Edit** `internal/http/server.go`: register `GET /api/p/:project/architecture-map` on the same
  project-scoped, auth-protected router group as the existing `graph`/`labels`/`lineages` routes
  (grep `handleGraph` registration and mirror it). No new middleware.

### Acceptance criteria

- `GET /api/p/:project/architecture-map` returns `200` with `{ "nodes": [...], "edges": [...] }`;
  nodes carry `labels`, edges carry `kind` (and `label` for typed kinds).
- `?stack_for=<archId>` includes that architecture's stack ring; omitting it returns the
  architecture-only base map (FR-8 default-off is enforced client-side by simply not sending the param).
- The route requires the same authentication/authorisation as `GET /api/p/:project/graph`; there is
  **no** POST/PUT/DELETE/PATCH variant (FR-10).
- Requesting an unknown project → the same error contract as `handleGraph` (no panic).

---

## Milestone 4 — Regression & freshness verification hooks

### Description

Guarantee the map reflects on-disk changes without a manual rebuild (FR-12) by confirming the endpoint
reads live index state, and confirm the change is inert for the existing lineage/artifact graph
(non-goal: not touching the main graph). No new indexing path is added — this milestone is about
asserting the reuse is correct and adding the seams the test plan needs.

### Files to change

- **Edit** `internal/index/index.go` (only if needed): ensure `ArchitectureMap` runs off the same
  live DB handle as `Graph` (no cached snapshot), so a re-index triggered by the watcher is visible on
  the next call.
- No change to `Graph`, `handleGraph`, or the artifact/lineage graph payload — verified by leaving
  their tests green.

### Acceptance criteria

- After a test writes a new `type: architecture` fixture into the indexed tree and the index re-scans
  (same mechanism the watcher uses), the next `ArchitectureMap("")` call includes the new node with no
  process restart (FR-12).
- The existing `/graph` response for a non-architecture project is byte-identical before/after this
  lineage's changes (main graph untouched — requirement non-goal).
- `make test-unit` and `make lint` pass.
