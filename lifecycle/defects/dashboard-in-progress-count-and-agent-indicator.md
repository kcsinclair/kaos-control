---
title: Dashboard In-Progress Count Shows 0 and Running Agent Not Displayed
type: defect
status: done
lineage: dashboard-in-progress-count-and-agent-indicator
created: "2026-05-15T18:19:03+10:00"
priority: normal
labels:
    - defect
    - frontend
    - agent
    - queue
release: KC-Release2
assignees:
    - role: frontend-developer
      who: agent
---

# Dashboard In-Progress Count Shows 0 and Running Agent Not Displayed

## Reproduction Steps

1. Start the application with one or more artifacts in an in-progress state and/or an agent actively running.
2. Navigate to the Dashboard view.
3. Observe the "In Progress" statistic and the running agent indicator.

## Expected Behaviour

- The "In Progress" count on the dashboard should reflect the actual number of artifacts currently in an in-progress state or queued for processing.
- When an agent is actively running, the dashboard should display a visible indicator showing that an agent is running.

## Actual Behaviour

- The "In Progress" count displays 0 even when there are items in the queue and agents are running.
- No running agent indicator is shown on the dashboard when an agent is actively executing.
