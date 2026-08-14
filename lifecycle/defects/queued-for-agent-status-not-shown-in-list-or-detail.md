---
title: Queued-for-Agent status not shown in artefact list or detail view
type: defect
status: draft
lineage: queued-for-agent-status-not-shown-in-list-or-detail
created: "2026-08-15T09:28:41+10:00"
priority: normal
labels:
    - defect
    - artifacts
    - frontend
    - status
    - queue
    - ui
    - ux
release: KC-Release5
assignees:
    - role: frontend-developer
      who: agent
---

# Queued-for-Agent status not shown in artefact list or detail view

## Reproduction Steps

1. Queue an artefact for agent processing (e.g. via the 'Queue for Agent' action in the UI or API).
2. Navigate to the artefact list page.
3. Observe the status column/indicator for the queued artefact.
4. Click into the artefact detail view.
5. Observe the status bar at the top of the detail view.

## Expected Behaviour

- The artefact list page should display a visible "Queued for Agent" status indicator for any artefact that has been queued, making the queue state immediately scannable without opening each artefact.
- The artefact detail view should show "Queued for Agent" prominently in its status bar whenever the artefact is in the queued state, so the current status is instantly visible upon opening the artefact.

## Actual Behaviour

- The artefact list page does not show a "Queued for Agent" status; the queued state is invisible to the user when browsing the list.
- The artefact detail view's status bar does not reflect the queued state, giving no indication that the artefact has already been queued for agent processing.
