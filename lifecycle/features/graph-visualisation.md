---
title: Graph & visualisation
type: feature
status: approved
lineage: feature-graph-visualisation
created: "2026-08-21T15:03:00+10:00"
summary: 3D and 2D artifact graphs with filtering and a roadmap Gantt/graph view, coloured and grouped by lifecycle metadata.
function: Visualization
labels:
    - feature
    - graph
    - visualization
related_to:
    - lifecycle/ideas/roadmap-gantt-period-options.md
    - lifecycle/requirements/roadmap-gantt-period-options-2.md
---

# Graph & visualisation

See the whole project as a graph, not just a list.

## What it does

- **3D force graph.** Three.js + 3d-force-graph; nodes coloured by type,
  rings for priority + active status (in-development = green pulse, in-qa =
  amber pulse, etc.); zoom, pan, click-to-edit.
- **2D graph.** Cytoscape.js + fcose; layout selector
  (fcose / circle / breadthfirst / dagre); filter relayout uses the current
  layout; light/dark theme-aware palette.
- **Graph filters.** By stage, status, type, label, priority, release, or
  full-text search; matched nodes highlight, others dim, no relayout.
- **Graph show-toggles.** Show / hide releases overlay, show / hide tests,
  hide done.
- **Roadmap graph.** Time-ordered chain of releases with assigned artifacts
  shown as children; "Backlog" anchor for unassigned items; "Unscheduled"
  terminus for releases without dates.
- **Roadmap Gantt chart.** Releases as bars on a timeline, with a period
  mode toggle: autoscale (granularity — week / month / quarter / half-year /
  year — chosen from the data span) or a fixed calendar period anchored to
  today.

Reachable at **Map** (2D/3D) and **Roadmap** (graph/Gantt).
