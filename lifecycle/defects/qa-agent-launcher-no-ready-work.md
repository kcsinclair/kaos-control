---
title: QA Agent Launcher Panel Shows No Ready Work Despite Approved Tests
type: defect
status: done
lineage: qa-agent-launcher-no-ready-work
created: "2026-04-27T20:27:28+10:00"
priority: normal
labels:
    - defect
    - agent
    - qa
    - frontend
release: April2026
---

# QA Agent Launcher Panel Shows No Ready Work Despite Approved Tests

## Reproduction Steps

1. Navigate to the agent launcher in the UI.
2. Open or view the QA agent panel.
3. Ensure there are test artifacts with `status: approved` present in the `lifecycle/tests/` directory.

## Expected Behaviour

The QA agent panel in the agent launcher should display all approved test artifacts that are ready for the QA agent to action.

## Actual Behaviour

The QA agent panel displays no work items, even though multiple approved test artifacts exist and are ready. This issue is related to the `agent-launcher-panels` lineage.

## Same behaviour for frontend

kanban-view-5-fe.md is approved but not showing when selecting frontend developer.
