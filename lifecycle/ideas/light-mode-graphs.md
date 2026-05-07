---
title: Light Mode Support for Graphs
type: idea
status: planning
lineage: light-mode-graphs
created: "2026-05-07T15:44:31+10:00"
priority: medium
labels:
    - enhancement
    - frontend
    - usability
    - vue
release: May2026
---

# Light Mode Support for Graphs

Currently the 2D and 3D graph visualisations are rendered in dark mode regardless of the application-level theme setting. When a user selects light mode, the graphs should reflect that choice by switching to an appropriate light colour scheme.

This includes updating background colours, node colours, edge colours, label text, and any other visual elements within both the Cytoscape.js (2D) and 3d-force-graph/three.js (3D) renderers so they are legible and visually consistent with the rest of the UI in light mode.

The implementation should react dynamically to theme changes without requiring a page reload, aligning with the existing theme-switching mechanism used by the surrounding Vue application.
