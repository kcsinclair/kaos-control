---
title: Table Pagination Controls
type: idea
status: clarifying
lineage: table-pagination
priority: normal
labels:
    - feature
    - frontend
    - vue
    - usability
---

# Table Pagination Controls

All tables throughout the application should support pagination to handle large datasets without overwhelming the user. Each table should include controls for selecting the number of rows to display per page (e.g. 10, 25, 50, 100), a direct page-jump input, previous/next navigation buttons, and a summary showing the total number of pages and current position.

This applies universally across every table in the UI — artifact lists, index views, agent run history, and any other tabular data. The pagination state for each table should be preserved where practical (e.g. surviving navigation within the same session).

The implementation should use a reusable Vue component so that pagination behaviour and styling remain consistent across all usage sites, and adding it to future tables requires minimal effort.
