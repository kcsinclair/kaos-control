---
title: '3D Map Filter Redesign: Top Menu Bar'
type: idea
status: planning
lineage: 3d-map-filter-redesign-top-menu
created: "2026-05-10T09:11:27+10:00"
priority: normal
labels:
    - enhancement
    - frontend
    - usability
    - vue
release: KC-Release4
---

# 2D and 3D Map Filter Redesign: Top Menu Bar

The current 2D and 3D map view suffers from visual overload when viewing large lineages — too many nodes are shown simultaneously with insufficient filtering controls, making it difficult to navigate or reason about the graph structure.

Filters should be moved from their current location into a top menu bar, providing a more prominent and accessible UI pattern. This allows users to quickly scope the visible nodes by type, status, or lineage without the controls competing with the graph canvas itself.

The redesigned filter bar should support multi-select facets (e.g. artifact type, workflow status, lineage slug) and ideally offer a quick-clear action, so users can rapidly toggle between focused and full-graph views without losing their camera position.
