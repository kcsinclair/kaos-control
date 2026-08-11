---
title: 'Agent View: Started Column Must Display Date and Time'
type: defect
status: approved
lineage: agent-view-started-column-missing-time
created: "2026-08-11T17:40:37+10:00"
priority: normal
labels:
    - defect
    - agent
    - frontend
    - ui
    - usability
    - vue
release: KC-Release5
assignees:
    - role: frontend-developer
      who: agent
---

# Agent View: Started Column Must Display Date and Time

## Reproduction Steps

1. Navigate to the agent runs view in the UI.
2. Observe the "Started" column for any agent run entry.

## Expected Behaviour

The "Started" column should display both the date and time (e.g. `2026-08-11 14:32:05`) so the user can see exactly when each agent run began.

## Actual Behaviour

The "Started" column displays only the date (or an incomplete timestamp), omitting the time component.
