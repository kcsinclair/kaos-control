---
title: Artifacts View 'All Stages' Control Missing Defects and Releases Stages
type: defect
status: in-development
lineage: artifacts-view-all-stages-missing-defects-releases
priority: normal
labels:
    - defect
    - artefacts
    - frontend
    - vue
---

# Artifacts View 'All Stages' Control Missing Defects and Releases Stages

## Reproduction Steps

1. Open the Artifacts view in the application.
2. Locate the stage filter control labelled 'All stages' (or equivalent dropdown/selector).
3. Expand or inspect the list of available stages.

## Expected Behaviour

The 'All stages' control should list every stage supported by the lifecycle, including **defects** and **releases**, so users can filter artifacts by those stages.

## Actual Behaviour

The **defects** stage is absent from the list of stages in the 'All stages' control. Additionally, the **releases** stage is no longer present (it appears to have been removed or omitted), even though releases remain a valid lifecycle stage.
