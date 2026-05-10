---
title: Improve Edge Line Contrast in 3D Graph
type: idea
status: done
lineage: 3d-graph-edge-contrast
created: "2026-05-10T08:56:28+10:00"
priority: normal
labels:
    - frontend
    - enhancement
    - usability
    - vue
release: KC-Release0
---

# Improve Edge Line Contrast in 3D Graph

Edge lines in the 3D force graph are currently difficult to distinguish against the background, reducing the readability of the graph and making it hard to trace relationships between nodes.

The edge line colour, opacity, and/or width should be adjusted so that connections are clearly visible against the scene background. This may involve exposing a configurable edge colour or applying a higher-contrast default that works across both light and dark themes.

This is a pure frontend change targeting the `3d-force-graph` rendering configuration in the Vue SPA.
