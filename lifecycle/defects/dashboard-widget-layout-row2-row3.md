---
title: Incorrect Dashboard Widget Layout in Rows 2 and 3
type: defect
status: draft
lineage: dashboard-widget-layout-row2-row3
created: "2026-05-09T18:14:16+10:00"
priority: normal
labels:
    - defect
    - frontend
    - vue
---

# Incorrect Dashboard Widget Layout in Rows 2 and 3

## Reproduction Steps

1. Navigate to the dashboard.
2. Observe the widget arrangement across all rows.

## Expected Behaviour

- Row 1: Current layout (correct).
- Row 2: Three columns — "Stages Distribution" (col 1), "Status Distribution" (col 2), "Recent Ideas & Defects" (col 3).
- Row 3: Two columns — "Completion Velocity" spanning the first two columns, "Recent Activity" in the third column (beneath "Recent Ideas & Defects").

## Actual Behaviour

Row 2 and Row 3 widgets are not arranged as specified. The second row does not correctly display the three-column layout of Stages Distribution, Status Distribution, and Recent Ideas & Defects, and the third row does not correctly show Completion Velocity spanning two columns alongside Recent Activity in the third column.
