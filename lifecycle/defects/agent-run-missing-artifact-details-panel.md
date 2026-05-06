---
title: Agent Run Not Visible at Bottom of Artifact Details Panel
type: defect
status: done
lineage: agent-run-missing-artifact-details-panel
created: "2026-04-29T15:31:57+10:00"
priority: normal
labels:
    - defect
    - frontend
    - artefacts
    - vue
---

# Agent Run Not Visible at Bottom of Artifact Details Panel

## Reproduction Steps

1. Navigate to the artifact editing screen at `/artifacts/` for any artifact in the kaos-control project.
2. Open the details panel for an artifact.
3. Trigger or observe an agent run associated with that artifact.
4. Scroll to the bottom of the details panel.

## Expected Behaviour

The current or most recent agent run for the artifact should be visible at the bottom of the details panel in the artifact editing screen, allowing users to monitor agent activity without leaving the editing context.

## Actual Behaviour

No agent run information is displayed at the bottom of the details panel. Users must navigate elsewhere to check agent run status, breaking the editing workflow.
