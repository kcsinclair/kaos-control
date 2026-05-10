---
title: Rename Graph to Map in UI and Routing
type: idea
status: clarifying
lineage: rename-graph-to-map
created: "2026-05-10T09:02:57+10:00"
priority: medium
labels:
    - frontend
    - usability
    - enhancement
    - vue
release: KC-Release0
---

# Rename Graph to Map in UI and Routing

The current navigation uses the term 'Graph' for the visualisation view, which is technically accurate but less intuitive for everyday users. People naturally refer to these visualisations as 'maps' or 'diagrams', and 'map' fits well in the context of this tool.

The menu item label should be changed from 'Graph' to 'Map', and the associated route path and name should be updated to reflect the new terminology. Any component filenames, internal references, or page titles that use 'graph' in a user-facing capacity should also be updated.

All related tests — including any Playwright end-to-end tests that navigate to or assert on the graph route or menu item — should be updated to match the new naming.
