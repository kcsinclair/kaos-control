---
title: Improve Edge Line Contrast in 3D Graph
type: requirement
status: approved
lineage: 3d-graph-edge-contrast
created: "2026-05-10"
priority: normal
parent: lifecycle/ideas/3d-graph-edge-contrast.md
labels:
    - frontend
    - enhancement
    - usability
    - vue
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

# Improve Edge Line Contrast in 3D Graph

## Problem

Edge lines in the 3D force graph are difficult to see, making it hard to trace relationships between nodes. Two factors contribute:

1. **Low link widths** — the default `linkWidth` is 0.5 px, `assigned` is 0.8 px, and only `timeline` edges reach 1.5 px. At typical zoom levels these are near-invisible.
2. **Low-contrast colours against canvas background** — in the dark theme the `related_to` edge colour (`#64748b`) and especially the `assigned` edge colour (`#334155`) have poor contrast against the canvas (`#0f172a`). In the light theme, `assigned` (`#94a3b8`) washes out against the white canvas.

The combined effect is that users cannot reliably read the graph's relationship structure, particularly for `parent` and `related_to` edges which carry the most semantic meaning.

## Goals / Non-goals

### Goals

- G1: All edge kinds are clearly visible against the canvas background in both light and dark themes.
- G2: Different edge kinds remain visually distinguishable from each other.
- G3: Edge hierarchy is preserved — `timeline` edges should still appear most prominent, `assigned` edges least prominent, with `parent`/`depends_on`/`blocks`/`related_to` in between.

### Non-goals

- NG1: User-configurable edge colours or widths at runtime (future work).
- NG2: Changes to the 2D Cytoscape graph — this requirement is scoped to `ForceGraph3D.vue` and `graphConstants.ts` only.
- NG3: Changes to node colours, sizes, or other non-edge visual elements.

## Detailed Requirements

### Functional Requirements

**FR-1: Increase minimum link width**
Increase the default `linkWidth` so that every edge kind is visible at the initial camera distance (post `zoomToFit`). Suggested minimum values:
- `parent`, `depends_on`, `blocks`, `related_to`, `label`: ≥ 1.0
- `assigned`: ≥ 0.6
- `timeline`: ≥ 2.0

The exact values may be tuned during implementation as long as the contrast and hierarchy goals (G1–G3) are met.

**FR-2: Improve edge colour contrast — dark theme**
Update the dark-theme `edgeColors` in `graphConstants.ts` so that every edge kind achieves a minimum WCAG contrast ratio of 3:1 against the dark canvas (`#0f172a`). Specifically:
- `related_to` — shift from `#64748b` to a lighter slate (e.g. `#94a3b8` or equivalent).
- `assigned` / `assignedEdgeColor` — shift from `#334155` to a visible but subdued tone (e.g. `#475569`).
- `parent` — verify contrast of `#94a3b8` is sufficient (it is ~4.5:1 — acceptable).

**FR-3: Improve edge colour contrast — light theme**
Update the light-theme `edgeColors` in `graphConstants.ts` so that every edge kind achieves a minimum WCAG contrast ratio of 3:1 against the light canvas (`#ffffff`). Specifically:
- `assigned` / `assignedEdgeColor` — shift from `#94a3b8` to a darker slate (e.g. `#64748b`).
- `related_to` — verify contrast of `#475569` is sufficient (it is ~5.5:1 — acceptable).

**FR-4: Add link opacity**
Apply a subtle `linkOpacity` (e.g. 0.7–0.9) to non-timeline edges so that dense graphs remain readable without edges fully obscuring nodes. Timeline edges should remain fully opaque.

**FR-5: Keep palette properties in sync**
`edgeColors.timeline`, `edgeColors.assigned`, `timelineEdgeColor`, and `assignedEdgeColor` must remain consistent within each palette — update both the map entry and the named property when changing a value.

### Non-functional Requirements

**NFR-1: No layout or physics changes**
Edge contrast changes must not alter force-simulation parameters, node positions, or camera behaviour.

**NFR-2: Maintain theme reactivity**
The existing `watch(isDark, ...)` handler in `ForceGraph3D.vue` already refreshes `linkColor`. Any new link styling (width, opacity) must also be applied during theme switches.

**NFR-3: Performance**
Adding `linkOpacity` must not introduce measurable frame-rate regression on graphs with ≤ 500 edges.

## Acceptance Criteria

- [ ] In dark theme: every edge kind has ≥ 3:1 contrast ratio against the canvas background `#0f172a`.
- [ ] In light theme: every edge kind has ≥ 3:1 contrast ratio against the canvas background `#ffffff`.
- [ ] Edge widths are increased; hierarchy is `timeline` > `parent`/`depends_on`/`blocks`/`related_to`/`label` > `assigned`.
- [ ] Non-timeline edges have a configurable opacity applied via `linkOpacity` (or per-link alpha channel).
- [ ] `edgeColors.assigned` and `assignedEdgeColor` remain in sync within each palette.
- [ ] `edgeColors.timeline` and `timelineEdgeColor` remain in sync within each palette.
- [ ] Theme toggling (light ↔ dark) updates edge colours, widths, and opacity without requiring a page reload.
- [ ] No visible change to node colours, sizes, or label rendering.
- [ ] No regression in force-simulation layout or camera behaviour.
- [ ] Visual verification on a graph with ≥ 20 nodes and mixed edge kinds (parent, depends_on, related_to, timeline, assigned).
- [ ] Related: [[3d-graph-edge-contrast]]

## Resolved Questions

- Q1: Should edge width increase when the user zooms in (proportional scaling), or remain constant in screen-space? Constant is simpler but very thick at close zoom.

> Yes, proportional scaling.

- Q2: Are there accessibility requirements beyond WCAG AA contrast (e.g. colour-blind–safe edge palettes)?

> No, I will raise an accessibility idea for later.
