---
title: DevOps Log Steps Show Empty Output
type: defect
status: in-development
lineage: devops-log-steps-empty-output
created: "2026-05-08T15:56:34+10:00"
priority: normal
labels:
    - defect
    - frontend
    - vue
    - operability
assignees:
    - role: frontend-developer
      who: agent
---

# DevOps Log Steps Show Empty Output

## Reproduction Steps

1. Navigate to the DevOps section of the application.
2. Click the log button for a DevOps run.
3. Observe the Steps section in the log output panel.

## Expected Behaviour

The Steps section should display the log output text for each step of the DevOps run.

## Actual Behaviour

The Steps section renders an empty space where the step text should appear. The log button click is now functional (regression from a previous fix), but the step output content is not displayed — the text is missing while the containing element or whitespace is visible.
