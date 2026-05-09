---
title: Sortable Table Columns
type: idea
status: done
lineage: sortable-table-columns
created: "2026-04-27T16:59:33+10:00"
priority: normal
labels:
    - enhancement
    - frontend
    - usability
    - vue
release: KC-OG-Sprint
---

# Sortable Table Columns

All data tables in the UI should support column sorting when a column header is clicked. Clicking a header once should sort the column ascending; clicking again should sort descending; a third click should reset to the default order.

A visual indicator (e.g. an arrow icon) should show the current sort column and direction so users can orient themselves at a glance. The sort state should be local to the table component and reset when the user navigates away.

This improves usability when browsing artifact lists, sprint tables, and any other tabular views where users need to locate or compare rows quickly.
