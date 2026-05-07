---
title: 'Graph: Show Releases Overlay'
type: requirement
status: draft
lineage: graph-show-releases-overlay
created: "2026-05-07T00:00:00+10:00"
priority: normal
parent: lifecycle/ideas/graph-show-releases-overlay.md
labels:
    - feature
    - frontend
    - releases
    - vue
assignees:
    - role: product-owner
      who: agent
---

# Graph: Show Releases Overlay

## Problem

The 2D (Cytoscape.js) and 3D (3d-force-graph) graph views currently show ideas, requirements, plans, tests, and defects but provide no visibility into how those artifacts map to releases. Users cannot answer "what shipped together?" or "what is planned for the next release?" without leaving the graph and inspecting individual artifact frontmatter or the releases list.

## Goals / Non-goals

### Goals

- Let users toggle a releases overlay on both graph views to see release groupings in context.
- Render releases as a chronological timeline spine (linear chain) so temporal ordering is immediately visible.
- Show edges from each release node to the ideas and defects assigned to that release.
- Include a synthetic "Backlog" node (first in chain) for artifacts with no release assignment and an "Unscheduled" node (last in chain) for artifacts explicitly marked unscheduled or with an undefined release.
- Keep the overlay opt-in (off by default) to preserve the uncluttered default graph.

### Non-goals

- Editing release assignments from within the graph (read-only overlay).
- Filtering or hiding non-release nodes when the overlay is active.
- Adding release information to the artifact detail panel or editor.
- Supporting custom timeline layouts or user-controlled positioning of release nodes.

## Detailed Requirements

### Functional

1. **Toggle control** — A "Show Releases" checkbox (or equivalent toggle) must appear in the graph toolbar/controls area of both the 2D and 3D views. Default state: unchecked.
2. **Release nodes** — When the overlay is enabled, one node per release artifact (`type: release`) is added to the graph. Each release node displays its title and, if available, its version or date label.
3. **Timeline spine** — Release nodes are connected in chronological order (by `created` or a `date` frontmatter field) forming a linear chain. The "Backlog" synthetic node anchors the start; the "Unscheduled" synthetic node anchors the end.
4. **Release-to-artifact edges** — Each release node is connected via directed edges to every idea and defect whose frontmatter `release` field matches that release's identifier (slug or title).
5. **Backlog/Unscheduled grouping** — Ideas and defects with no `release` field connect to "Backlog". Ideas and defects with `release: unscheduled` (or equivalent sentinel) connect to "Unscheduled".
6. **Visual distinction** — Release nodes and timeline-spine edges must use a distinct visual style (colour, shape, or icon) differentiating them from regular artifact nodes and lineage edges. The style must be legible in both light and dark themes.
7. **Toggle reactivity** — Enabling or disabling the overlay must not trigger a full graph re-render; nodes and edges should be added/removed incrementally.
8. **Data source** — Release data is sourced from the existing index (SQLite cache / REST API). No new backend endpoint is required if the current `/artifacts` or `/graph` endpoint already includes release artifacts; otherwise a query-parameter filter (e.g., `?include_releases=true`) may be added.

### Non-functional

- **Performance** — Adding the overlay to a graph with up to 500 artifacts and 20 releases must not degrade frame rate below 30 fps on a mid-range laptop (2D) or noticeably stall the force simulation (3D).
- **Consistency** — The toggle state should persist across page navigations within the same session (e.g., via Pinia store or URL query param) but need not persist across browser sessions.
- **Accessibility** — The toggle must be keyboard-accessible and labelled for screen readers. Release nodes should have distinguishable shape or pattern, not rely solely on colour.

## Acceptance Criteria

- [ ] A "Show Releases" toggle is visible in both 2D and 3D graph views.
- [ ] Toggle is off by default; enabling it adds release nodes and timeline edges to the graph.
- [ ] Release nodes are connected chronologically in a linear chain (Backlog → R1 → R2 → … → Unscheduled).
- [ ] Each release node has directed edges to its assigned ideas and defects.
- [ ] Artifacts without a release field appear connected to the "Backlog" node.
- [ ] Release nodes and spine edges are visually distinct from artifact nodes and lineage edges.
- [ ] Disabling the toggle removes all release overlay nodes and edges without a full re-render.
- [ ] Overlay renders correctly in both Cytoscape.js 2D and 3d-force-graph 3D views.
- [ ] No perceptible performance degradation with ≤ 20 releases and ≤ 500 artifacts.
- [ ] Toggle is keyboard-accessible and has an accessible label.

## Resolved Questions

1. What frontmatter field on ideas/defects indicates their release assignment? Is it `release: <slug>`, `release: <title>`, or something else? Does this field already exist in the schema?

> It is the 'release: <title>'

2. How should release chronological order be determined — by a `date` field, `created` timestamp, or explicit `order` field in release frontmatter?

> Releases have a defined start date, or they are unscheduled, the start date should be used.

3. Should the "Unscheduled" node be shown if no artifacts are explicitly marked unscheduled, or should it always appear as the chain terminus?

> Do not display an Unscheduled release unless it has artifacts associated.  The Unscheduled release may NOT be called Unscheduled, it might be called "Future-Release" for example.

4. Are there existing colour/shape conventions for node types in the graph that the release nodes should follow or intentionally contrast with?

> There is a defined colour scheme, lets make Releases light blue.
