---
title: Make Pipelines Editable
type: defect
status: approved
lineage: pipeline-editing
created: "2026-05-13T17:01:56+10:00"
priority: normal
labels:
    - defect
    - feature
    - enhancement
    - frontend
---

# Make Pipelines Editable

## Reproduction Steps

1. Open the application and navigate to the Pipelines section.
2. Add a new pipeline using the existing add-pipeline functionality.
3. Attempt to modify or edit the newly created pipeline's configuration.

## Expected Behaviour

An existing pipeline should be editable — the user should be able to click into a pipeline and modify its name, configuration, stages, or other properties, then save the changes.

## Actual Behaviour

No edit capability exists for pipelines. Once a pipeline has been created it can only be viewed; there is no UI affordance (e.g. edit button, inline editing) to update its configuration.
