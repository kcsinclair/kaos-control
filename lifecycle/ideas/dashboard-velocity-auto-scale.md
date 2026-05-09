---
title: Dashboard Completion Velocity Improvements
type: idea
status: planning
lineage: dashboard-velocity-auto-scale
created: "2026-05-09T16:30:53+10:00"
priority: high
labels:
    - enhancement
    - frontend
    - vue
    - usability
release: KC-Release0
---

# Dashboard Completion Velocity Improvements

Improve the Completion Velocity widget on the dashboard to support auto-scaling column widths based on the selected time granularity. The widget should offer three view modes: days (minimum 7 days), weeks (minimum 4 weeks), and months (3 months), with days as the default view.

Columns should automatically scale their width to best utilise available space within each granularity, ensuring data is readable without manual resizing. When the number of columns exceeds the visible area, the widget should become horizontally scrollable so all data remains accessible without truncation.

This improvement makes velocity trends easier to read across different planning horizons, supporting both short-term sprint tracking and longer-term delivery pattern analysis.
