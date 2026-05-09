---
title: Status-Checker Button Non-Functional on Graph Views and Missing from Artifacts/Kanban Screens
type: defect
status: done
lineage: status-checker-button-not-working-graph-missing-screens
created: "2026-05-06T12:05:07+10:00"
priority: normal
labels:
    - defect
    - frontend
    - vue
release: KC-Feature-Sprint
---

# Status-Checker Button Non-Functional on Graph Views and Missing from Artifacts/Kanban Screens

## Reproduction Steps

1. Navigate to the 2D graph view.
2. Locate the status-checker button in the UI.
3. Click the status-checker button and observe the result.
4. Repeat steps 2–3 on the 3D graph view.
5. Navigate to the Artifacts screen and observe whether the status-checker button is present.
6. Navigate to the Kanban Board screen and observe whether the status-checker button is present.

## Expected Behaviour

- The status-checker button should function correctly on both the 2D and 3D graph views, triggering the expected status-check behaviour when clicked.
- The status-checker button should be visible and accessible on the Artifacts screen.
- The status-checker button should be visible and accessible on the Kanban Board screen.

## Actual Behaviour

- The status-checker button does not work on the 2D graph view (clicking produces no effect or an error).
- The status-checker button does not work on the 3D graph view (clicking produces no effect or an error).
- The status-checker button is absent from the Artifacts screen.
- The status-checker button is absent from the Kanban Board screen.
