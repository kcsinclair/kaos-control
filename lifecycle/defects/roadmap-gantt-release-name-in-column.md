---
title: 'Roadmap Gantt: Release Name Displayed in First Column Misaligns Rows'
type: defect
status: approved
lineage: roadmap-gantt-release-name-in-column
created: "2026-05-07T11:33:33+10:00"
priority: normal
labels:
    - defect
    - frontend
    - roadmaps
    - vue
---

# Roadmap Gantt: Release Name Displayed in First Column Misaligns Rows

## Reproduction Steps

1. Navigate to the Roadmap view.
2. Switch to the Gantt chart layout.
3. Observe the first column of the Gantt chart.

## Expected Behaviour

The first column should not display the release name as text. The release name should appear only as a label on the bar within the chart area, keeping all columns aligned.

## Actual Behaviour

The release name is rendered as text in the first column, causing the column widths to shift and the Gantt chart rows to fall out of alignment with their corresponding headers and bars.
