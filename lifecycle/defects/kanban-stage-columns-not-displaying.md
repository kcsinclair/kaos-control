---
title: Kanban Stage Columns Not Displaying and Stages Field Shows Type Values
type: defect
status: abandoned
lineage: kanban-stage-columns-not-displaying
created: "2026-04-28T08:25:28+10:00"
priority: normal
labels:
    - defect
    - frontend
    - vue
---

# POSSIBLY REDUNDANT

## Reproduction Steps

1. Navigate to the Kanban board view in the UI.
2. Observe the column layout — columns representing lifecycle stages are not rendered.
3. Inspect the "stages" field/filter on the board — it displays the same values as the "type" field instead of distinct stage values.

## Expected Behaviour

The Kanban board should display one column per lifecycle stage (e.g. draft, clarifying, planning, in-development, in-qa, approved, done). The stages field should show stage/status values distinct from the artifact type vocabulary.

## Actual Behaviour

The Kanban board does not render the expected stage columns. The stages field is populated with the same values as the artifact type field (e.g. idea, ticket, plan-backend, etc.) rather than the workflow status values.
