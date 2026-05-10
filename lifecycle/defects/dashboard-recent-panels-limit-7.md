---
title: Dashboard Recent Panels Should Show 7 Most Recent Items
type: defect
status: done
lineage: dashboard-recent-panels-limit-7
created: "2026-05-10T08:50:35+10:00"
priority: normal
labels:
    - defect
    - frontend
    - vue
release: KC-Release0
assignees:
    - role: frontend-developer
      who: agent
---

# Dashboard Recent Panels Should Show 7 Most Recent Items

## Reproduction Steps

1. Start the application and navigate to the Dashboard.
2. Observe the "Recent Ideas & Defects" widget.
3. Observe the "Recent Activity" widget.
4. Note the number of items displayed in each panel.

## Expected Behaviour

Both the "Recent Ideas & Defects" panel and the "Recent Activity" panel should each display the 7 most recent items.

## Actual Behaviour

One or both panels do not display exactly 7 items — either showing a different fixed limit or an incorrect number of recent entries.
