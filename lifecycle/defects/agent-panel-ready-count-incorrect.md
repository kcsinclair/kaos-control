---
title: Agent Panel Ready Count Includes Non-Approved Items
type: defect
status: approved
lineage: agent-panel-ready-count-incorrect
created: "2026-05-10T17:10:11+10:00"
priority: normal
labels:
    - defect
    - frontend
    - agent
    - vue
---

# Agent Panel Ready Count Includes Non-Approved Items

## Reproduction Steps

1. Open the application and navigate to the Agent Panels view.
2. Ensure there are artifacts in various states (e.g. `draft`, `planning`, `in-development`, `approved`).
3. Observe the "Ready" count displayed on one or more Agent Panels.

## Expected Behaviour

The ready count on each Agent Panel should only include artifacts with a status of `approved`. Artifacts in any other status (e.g. `draft`, `planning`, `in-development`) must not be counted.

## Actual Behaviour

The ready count includes artifacts that are not in the `approved` status, causing the count to be higher than expected and misleading to users about how many items are genuinely ready for the agent to process.
