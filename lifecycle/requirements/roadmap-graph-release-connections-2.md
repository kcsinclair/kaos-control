---
title: 'Roadmap Graph: Directed Release Chain with Backlog Root and Unscheduled Leaves'
type: requirement
status: planning
lineage: roadmap-graph-release-connections
created: "2026-05-07"
priority: high
parent: lifecycle/ideas/roadmap-graph-release-connections.md
labels:
    - frontend
    - roadmaps
    - feature
    - vue
release: May2026
assignees:
    - role: product-owner
      who: agent
---

# Roadmap Graph: Directed Release Chain with Backlog Root and Unscheduled Leaves

## Problem

The Roadmap graph currently displays releases without a clear directed temporal ordering. There is no visual representation of the chronological sequence between releases, no synthetic "Backlog" root node anchoring unplanned work, and no defined placement for releases that lack a scheduled date. Users cannot quickly understand the release timeline or where unscheduled work sits relative to planned milestones.

## Goals / Non-goals

### Goals

- Display all releases as nodes in a directed graph connected by edges that convey chronological order.
- Provide a synthetic "Backlog" root node (replacing the current "Undefined" label) that anchors unplanned/unassigned work.
- Position unscheduled releases as terminal leaf nodes at the end of the directed chain.
- Give users an immediate, scannable view of the full release timeline from backlog through scheduled releases to unscheduled future work.

### Non-goals

- This requirement does not cover editing releases or changing their dates from the graph view.
- This requirement does not define how individual artifacts (ideas, tickets) are displayed within or attached to release nodes.
- Drag-and-drop reordering of releases is out of scope.
- Filtering or hiding specific releases from the graph is out of scope.

## Detailed Requirements

### Functional

1. **Backlog root node** — The graph MUST render a synthetic node labelled "Backlog" as the first (root) node. This node represents work not yet assigned to any release.

2. **Scheduled release ordering** — All releases that have a `start` date MUST be sorted in ascending chronological order by that date. A directed edge MUST connect each release to the next one in the sorted sequence.

3. **Backlog-to-first-scheduled edge** — A directed edge MUST connect the Backlog node to the earliest scheduled release.

4. **Unscheduled release handling** — Releases without a `start` date MUST appear after all scheduled releases. If multiple unscheduled releases exist, they MUST be sorted alphabetically by release name and connected with directed edges in that order.

5. **Last-scheduled-to-first-unscheduled edge** — A directed edge MUST connect the last (most recent) scheduled release to the first unscheduled release (alphabetically).

6. **Single unscheduled release** — If only one unscheduled release exists, it appears as a single terminal leaf connected from the last scheduled release.

7. **No scheduled releases** — If no releases have a `start` date, the Backlog node connects directly to the first unscheduled release (alphabetically sorted).

8. **Empty state** — If no releases exist at all, only the Backlog node is rendered (no edges).

9. **Edge directionality** — All edges MUST be visually directed (e.g., arrows) indicating flow from earlier to later in the timeline.

### Non-functional

1. The graph MUST render without perceptible delay for up to 50 release nodes.
2. The graph layout MUST remain readable (no overlapping labels) for up to 20 release nodes without user interaction.
3. The implementation MUST use the existing graph library already in use for the Roadmap view (Cytoscape.js with fcose layout, or 3d-force-graph depending on which view mode is active).

## Acceptance Criteria

- [ ] A synthetic "Backlog" node is rendered as the root of the directed graph.
- [ ] The label reads "Backlog" (not "Undefined").
- [ ] Scheduled releases appear in ascending chronological order by `start` date.
- [ ] Directed edges connect each scheduled release to the next in sequence.
- [ ] A directed edge connects Backlog to the earliest scheduled release.
- [ ] Unscheduled releases (no `start` date) appear after all scheduled releases.
- [ ] Multiple unscheduled releases are sorted alphabetically and connected by directed edges.
- [ ] A directed edge connects the last scheduled release to the first unscheduled release.
- [ ] When no scheduled releases exist, Backlog connects directly to the first unscheduled release.
- [ ] When no releases exist, only the Backlog node is shown with no edges.
- [ ] All edges display a directional indicator (arrowhead).
- [ ] Graph renders without perceptible delay (< 200 ms) with 50 release nodes.
- [ ] No label overlap with up to 20 release nodes at default zoom.

## Resolved Questions

1. Should the Backlog node visually differ from release nodes (e.g., different colour, shape, or icon)?

> Yes, releases can be light blue, if a rounded cube shape is possible, that would be awesome.

2. When two scheduled releases share the same `start` date, what is the tiebreaker for ordering — alphabetical by name, or creation date?

> Alphabetical for tiebreaker works.

3. Should edges display any label or tooltip (e.g., time gap between releases)?

> Yes, time between starting dates would be great.

4. Does clicking a release node navigate somewhere (e.g., filtered artifact list for that release), or is interaction out of scope for this requirement?

> Clicking on the release node should display the same modal as displayed in the gantt view.
> Clicking on an idea or defect node should display the same modal as used in the regular graphs.
