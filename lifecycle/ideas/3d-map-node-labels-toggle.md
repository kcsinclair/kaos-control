---
title: Toggleable Node Titles and Lineage in 3D Maps
type: idea
status: draft
lineage: 3d-map-node-labels-toggle
created: "2026-05-10T09:09:43+10:00"
priority: normal
labels:
    - frontend
    - enhancement
    - roadmaps
    - vue
    - usability
---

# Toggleable Node Titles and Lineage in 3D Maps

Add two checkboxes to the 3D map and 3D roadmap views: "Show node titles" and "Show node lineage". These controls should toggle the display of per-node label overlays on the 3D force graph, giving users the option to reduce visual clutter or expose more information as needed.

For node titles, reuse the existing 2D map behaviour: display the title truncated at 15 characters with an ellipsis when the text exceeds that length. For node lineage, display the full lineage string without truncation so users can identify exactly which lineage a node belongs to.

Both checkboxes should default to off (matching the current unlabelled 3D experience) and their state should persist for the session. The implementation should leverage the existing label/overlay mechanism available in the `3d-force-graph` library rather than introducing a separate rendering layer.
