---
title: 'Graph: Show Releases Overlay'
type: idea
status: planning
lineage: graph-show-releases-overlay
created: "2026-05-07T11:04:02+10:00"
priority: normal
labels:
    - feature
    - frontend
    - releases
    - vue
---

# Graph: Show Releases Overlay

Add a 'Show Releases' checkbox to both the 2D and 3D graph views, off by default. When enabled, release nodes are rendered on the graph and connected to the ideas and defects that belong to each release, giving a clear picture of which work shipped (or is planned to ship) together.

Releases are arranged chronologically along a timeline spine, with a special 'Backlog' node (representing unassigned/undefined release) as the first anchor and 'Unscheduled' as the final node. Each release is linked to its predecessor in time, forming a linear chain that acts as a backbone across the graph.

This overlay is intentionally opt-in to avoid cluttering the default view. The feature applies equally to the Cytoscape.js 2D graph and the 3d-force-graph 3D view, reusing the existing node/edge rendering pipeline with a distinct visual style for release nodes and timeline edges.
