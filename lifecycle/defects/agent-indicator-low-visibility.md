---
title: Running Agent Indicator Has Insufficient Visual Contrast
type: defect
status: done
lineage: agent-indicator-low-visibility
created: "2026-05-09T17:51:30+10:00"
priority: normal
labels:
    - defect
    - frontend
    - usability
    - agent
release: KC-Release0
assignees:
    - role: frontend-developer
      who: agent
---

# Running Agent Indicator Has Insufficient Visual Contrast

## Reproduction Steps

1. Start the application and open the menu bar.
2. Trigger an agent run so that an agent is actively running.
3. Observe the running agent indicator in the menu bar.

## Expected Behaviour

The running agent indicator should be rendered in a high-contrast colour (e.g. blue or green) that clearly distinguishes it from surrounding UI elements, making it immediately obvious that an agent is active.

## Actual Behaviour

The running agent indicator uses a colour that blends in with the menu bar, making it difficult to notice at a glance that an agent is currently running.
