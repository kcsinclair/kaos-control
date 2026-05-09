---
title: Status Checker Button Does Not Update Artifact Statuses
type: defect
status: done
lineage: status-checker-button-no-status-update
created: "2026-05-06T12:35:38+10:00"
priority: normal
labels:
    - defect
    - frontend
    - backend
    - workflow
    - artefacts
release: KC-Feature-Sprint
---

# Status Checker Button Does Not Update Artifact Statuses

## Reproduction Steps

1. Open the kaos-control UI.
2. Navigate to a lineage where work is completed but upstream artifacts (idea, requirement) remain in an in-progress status — e.g. `lifecycle/ideas/agents-indicator-in-menu-bar.md` and `lifecycle/requirements/agents-indicator-in-menu-bar-2.md`.
3. Click the Status Checker button.
4. Observe the network tab — confirm the API call is made and returns a response.
5. Refresh or observe the artifact list.

## Expected Behaviour

After clicking the Status Checker button, the tool should evaluate the current state of artifacts in the lineage and update their statuses accordingly — e.g. transitioning completed ideas and requirements from an in-progress status (such as `in-development`) to `done` or the appropriate terminal status.

## Actual Behaviour

The API call fires successfully (visible in network/browser tooling), but artifact statuses are not updated. The idea and requirement artifacts remain in their previous in-progress status despite the associated work being complete. No visible error is shown to the user. It is unclear whether logging can be enabled to diagnose the root cause of the failed status propagation.
