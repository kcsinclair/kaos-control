---
title: Agent Panel Ready Counts Show Same Value Instead of Role-Specific Counts
type: defect
status: done
lineage: agent-panel-ready-count-not-role-specific
created: "2026-05-11T11:38:12+10:00"
priority: normal
labels:
    - defect
    - agent
    - frontend
    - vue
---

# Agent Panel Ready Counts Show Same Value Instead of Role-Specific Counts

## Reproduction Steps

1. Navigate to the Agents screen in the application.
2. Observe the ready-work count displayed on each agent panel.

## Expected Behaviour

Each agent panel should display a count of artefacts that are ready for that specific role:
- **requirements-analyst** — count of ideas with status `approved`
- **planning-analyst** — count of requirements with status `approved`
- **backend-developer** — count of tickets/plans with status `approved` and type `plan-backend`
- Other roles should similarly show counts filtered to their relevant artefact types and statuses.

## Actual Behaviour

All agent panels display the same count regardless of role (currently showing "50 Ready" for every agent). The counts are not filtered by role, artefact type, or status — suggesting a shared or hardcoded value is being used rather than per-role queries.
