---
title: Architecture Relationship Map
type: requirement
status: blocked
lineage: architecture-relationship-map
parent: lifecycle/ideas/architecture-relationship-map.md
labels:
    - architecture
    - onboarding
    - visualization
    - graph
    - feature
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

# Architecture Relationship Map

## Problem

The architecture catalog (`lifecycle/architecture/`) is a set of nine
independent-looking pages. In reality the architectures form a landscape with
real relationships — one *evolves into* another, some are *simpler alternatives*
to each other, and some *compose* (layer on top of) others. Reading nine pages
does not reveal that shape, so a person choosing an architecture cannot easily
place their instinct ("I want a web app for my team") next to its neighbours and
pick with confidence.

There is no visual surface that shows architectures **as a map of typed
relationships**. The existing artifact graph is scoped to lineage/parent links
across all artifact types, not to architecture-to-architecture relationship
semantics, so it cannot serve this purpose. This requirement defines a dedicated,
read-only relationship map that becomes the visual "browse" surface for
[[onboarding-architecture-selection]].

## Goals / Non-goals

### Goals

- Provide a **dedicated, read-only graph view** of the architecture catalog:
  architectures as nodes, typed relationships as edges.
- Give architecture its **own left-menu entry**; this map is its primary view
  for now.
- Support **both 2D (Cytoscape) and 3D (3d-force-graph)** rendering, reusing the
  existing graph engines, with the user able to switch between them.
- Encode **decision signals at a glance** on each node (e.g. colour by scale,
  icon for offline/mobile) driven by the catalog artifacts' `labels`.
- Show a **legend** for relationship kinds and for the decision-signal colour
  key.
- Let a user **click a node to open** the underlying architecture artifact.
- Offer an **option to reveal related tech stacks** (`related_to` edges) around a
  selected architecture, off by default to keep the base map legible.
- Derive the entire map **from the catalog artifacts** — they are the single
  source of truth.
- Provide **entry points** into the map from the catalog `README` and from the
  project-create / onboarding flow.

### Non-goals

- **No in-app editing** of the map, nodes, edges, or relationships. The map is
  purely derived; maintainers change it by editing artifacts.
- **Not** a replacement for or modification of the main artifact/lineage graph.
- **Not** the guided questionnaire or scoring logic — that is
  [[onboarding-architecture-selection]]; this requirement is only the browse
  surface.
- **No** new relationship-authoring UI. Introducing typed relationship
  frontmatter fields is out of scope here (tracked under
  [[artefact-relationship-labels-and-links]] and [[architecture-templates]]).

## Detailed Requirements

### Functional

**FR-1 — Dedicated view & navigation.** The app adds an **Architecture** entry to
the left navigation. The entry is a *section* that hosts the relationship map,
the Architecture Wizard ([[onboarding-architecture-selection]]), and the
chosen-architecture overview ([[architecture-overview-view]]). The relationship
map is the section's default destination **while the project has no chosen
architecture**; once a selection exists the overview becomes the default and
the map remains directly reachable (own route/URL in both cases).
*(Amended 2026-08-14 to converge with the Architecture menu design — the
map alone was previously "the primary view for now".)*

**FR-2 — Scoped node set.** The map's nodes are exactly the artifacts of
`type: architecture` in the catalog. Non-architecture artifacts do not appear as
base nodes. Adding/removing an architecture artifact changes the node set with no
code change.

**FR-3 — Relationship edges (data source).** Edges between architecture nodes are
derived from the architecture-to-architecture links already present in the
catalog (the body wiki-links / `related_to` among architecture artifacts). The
map MUST render at least these links as undirected/generic edges in v1.

**FR-4 — Typed relationship readiness.** When typed relationship fields
(`evolves_into`, `alternative_to`, `composed_with`) exist on an artifact, the map
MUST render each edge with a distinct style (e.g. directed arrow for
`evolves_into`, dashed for `alternative_to`, thick/overlay for `composed_with`)
and label it by kind. Absence of typed fields MUST degrade gracefully to FR-3
generic edges — the view must not error when typed fields are missing.

**FR-5 — Decision-signal encoding.** Each node visually encodes decision signals
derived from the artifact's `labels`: colour by scale signal and an icon/glyph
for at least offline-capable and mobile. Encoding is data-driven from labels; a
node with no relevant labels renders in a neutral default style.

**FR-6 — Legend.** The view shows a legend with two keys: (a) relationship kinds
and their edge styles, and (b) the decision-signal colour/icon key. The legend
reflects only the kinds/signals actually present in the current map.

**FR-7 — Click-through to artifact.** Clicking (or activating) a node opens the
corresponding architecture artifact in the editor/viewer.

**FR-8 — Related-stack reveal.** A user-toggleable option shows, for the selected
architecture, a second ring of its compatible tech-stack nodes via the
`related_to` edges. This is **off by default**. Deselecting the architecture or
disabling the toggle returns the map to the architecture-only view.

**FR-9 — 2D / 3D switch.** The user can switch the map between a 2D (Cytoscape)
and a 3D (3d-force-graph) rendering. Node identity, edges, decision-signal
encoding, legend, and click-through behaviour are equivalent in both modes.

**FR-10 — Read-only.** The view exposes no affordance to create, edit, delete, or
re-position-persist nodes or edges. Any layout interaction (pan/zoom/drag) is
ephemeral and never written back to artifacts.

**FR-11 — Entry points.** The catalog `README` and the project-create /
onboarding flow each link into the relationship-map view.

**FR-12 — Derived & fresh.** The map is computed from the current catalog
artifacts. When an architecture artifact is added, removed, or its
relationship/label data changes on disk, the map reflects the change without a
manual rebuild step (consistent with the existing index/watcher behaviour).

### Non-functional

**NFR-1 — Reuse existing stack.** The view reuses the existing graph engines
(Cytoscape 2D, 3d-force-graph 3D) and index/API infrastructure; it introduces no
new graph-rendering dependency.

**NFR-2 — Legibility at catalog scale.** With the current curated catalog
(~9 architectures) plus edges, the base map renders legibly with readable node
and edge labels in 2D. The related-stack reveal (FR-8) stays legible for a single
selected architecture's stacks.

**NFR-3 — Performance.** Initial render of the base map completes promptly for
the curated catalog size; toggling 2D/3D or the stack reveal is interactive
(sub-second) at this scale.

**NFR-4 — Accessibility.** Decision-signal encoding does not rely on colour
alone — icons/glyphs and labels/legend convey the same information (aligns with
existing a11y direction for graphs).

**NFR-5 — Graceful degradation.** Missing, malformed, or partial relationship /
label data on an artifact does not break the view; affected nodes/edges fall back
to neutral/generic rendering.

## Acceptance Criteria

- [ ] An **Architecture** entry appears in the left navigation and opens the
  relationship map on its own route (FR-1).
- [ ] The map renders one node per `type: architecture` artifact and no
  non-architecture base nodes (FR-2).
- [ ] Architecture-to-architecture relationships from the current catalog render
  as edges without any hand-drawn/manual edge data (FR-3).
- [ ] When typed relationship fields are present, edges are styled and labelled by
  kind; when absent, edges fall back to generic edges and the view does not error
  (FR-4).
- [ ] Nodes encode decision signals from `labels` (colour-by-scale plus
  offline/mobile icons), with a neutral default for unlabelled nodes (FR-5).
- [ ] A legend shows both the relationship-kind key and the decision-signal key,
  limited to what the current map contains (FR-6).
- [ ] Clicking a node opens the corresponding architecture artifact (FR-7).
- [ ] A toggle reveals the selected architecture's compatible stacks via
  `related_to`, defaulting to off, and hides them again when toggled off /
  deselected (FR-8).
- [ ] The user can switch between 2D and 3D renderings with equivalent nodes,
  edges, encoding, legend, and click-through (FR-9).
- [ ] The view offers no create/edit/delete/persist-layout affordance; artifacts
  remain the sole source of truth (FR-10).
- [ ] The catalog `README` and the onboarding / project-create flow both link into
  the map (FR-11).
- [ ] Adding/removing/changing an architecture artifact on disk is reflected in
  the map without a manual rebuild (FR-12).
- [ ] Missing/partial relationship or label data degrades gracefully rather than
  erroring (NFR-5).
- [ ] Related: consumed by [[onboarding-architecture-selection]] as its "browse"
  surface; typed-edge styling depends on
  [[artefact-relationship-labels-and-links]] and [[architecture-templates]].

## Open Questions

- **Edge label density in 3D.** Labelled relationship edges read well in 2D; in
  3d-force-graph edge labels are harder. Is a legend-only (unlabelled edges)
  presentation acceptable in 3D, or must edge kinds remain individually labelled?

> legend-only in 3D works.

- **Stack ring on click vs. persistent.** FR-8 reveals stacks per selected node.
  Should there be an "expand all stacks" (bipartite) mode as well, or is
  per-selection reveal the only intended mode for v1?

> **Resolved (2026-08-14):** per-selection reveal only for v1, matching the
> idea's resolved "option to show related tech stack". A bipartite mode can be
> a later enhancement if wanted.

- **Scale-signal → colour mapping.** Which exact `labels` define the "scale" axis
  used for node colour (e.g. `high-scale` … small), and what is the fixed colour
  scale? Needs a definitive label→colour table (ideally shared with the
  onboarding decision-signal key).

- **Default engine.** Should the view open in 2D or 3D by default? (2D is likely
  clearer for a small labelled set, but the idea asks for both.)

> **Resolved (2026-08-14):** open in **2D** by default (clearer for a small
> labelled set); the 2D/3D switch (FR-9) covers the rest. Last engine used may
> be remembered per user as a nicety.

- **Relationship extraction in v1.** For FR-3, are the arch↔arch links reliably
  parseable from the current catalog body wiki-links, or is a minimal typed-field
  seed needed before this view is trustworthy?
