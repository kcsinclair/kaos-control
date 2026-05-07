---
title: 2D Graph Layout Selector
type: idea
status: draft
lineage: 2d-graph-layout-selector
created: "2026-05-07T15:48:11+10:00"
priority: normal
labels:
    - frontend
    - feature
    - usability
    - vue
---

# 2D Graph Layout Selector

Add a control to the 2D graph view that allows users to switch between different layout algorithms (e.g. fcose, breadthfirst, dagre, concentric, circle). The control should be accessible within the graph panel, such as a dropdown or button group, without requiring a page reload.

Different layout algorithms surface different structural patterns in the artifact graph — hierarchical layouts clarify lineage chains while force-directed layouts reveal clustering. Giving users the ability to switch layouts on demand improves exploration and comprehension of complex graphs.

The selected layout should be persisted per-session so users don't need to reselect after navigating away and returning to the graph view.
