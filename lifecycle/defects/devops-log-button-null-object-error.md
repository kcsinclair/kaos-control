---
title: Devops Screen Log Button Throws 'null is not an object' Error
type: defect
status: done
lineage: devops-log-button-null-object-error
created: "2026-05-08T15:09:02+10:00"
priority: high
labels:
    - defect
    - frontend
    - vue
release: May2026
assignees:
    - role: frontend-developer
      who: agent
---

# Devops Screen Log Button Throws 'null is not an object' Error

## Reproduction Steps

1. Navigate to the Devops screen in the application.
2. Locate the Log button.
3. Click the Log button.
4. Observe the error in the console or UI.

## Expected Behaviour

Clicking the Log button on the Devops screen should open or display the relevant log output without any errors.

## Actual Behaviour

Clicking the Log button produces a JavaScript error: `null is not an object`. The log view fails to render or open, likely due to a null reference being accessed before it is initialised.
