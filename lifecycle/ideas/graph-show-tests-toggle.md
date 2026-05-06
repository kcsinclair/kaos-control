---
title: Graph Show Tests Toggle
type: idea
status: approved
lineage: graph-show-tests-toggle
created: "2026-05-06T13:37:33+10:00"
priority: normal
labels:
    - frontend
    - feature
    - vue
    - test
---

# Graph Show Tests Toggle

Add a 'Show Tests' checkbox to both the 2D and 3D graph views, unchecked by default. When unchecked, artefacts of type `test` are excluded from the graph entirely; when checked, they are rendered as nodes with their appropriate edges to parent artefacts.

This follows the same UX pattern already established for 'done' artefacts — a simple boolean toggle that filters a specific artefact category out of the default view to reduce noise. The checkbox should sit alongside the existing visibility controls in the graph toolbar or filter panel.

The filter should apply to both node rendering and edge rendering: suppressing a test node must also suppress any edges that connect exclusively to suppressed nodes, consistent with how other filtered artefact types are handled.
