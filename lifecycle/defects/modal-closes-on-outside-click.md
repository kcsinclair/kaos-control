---
title: Modals Dismiss on Outside Click Instead of Explicit Action
type: defect
status: approved
lineage: modal-closes-on-outside-click
created: "2026-05-16T07:50:43+10:00"
priority: normal
labels:
    - defect
    - frontend
    - usability
    - vue
---

# Modals Dismiss on Outside Click Instead of Explicit Action

## Reproduction Steps

1. Open any modal dialog in the application.
2. Click anywhere outside the modal (on the backdrop or surrounding UI).
3. Observe the modal closes immediately.

## Expected Behaviour

Modals should only close when the user explicitly dismisses them — either by clicking a designated close button (e.g. an "×" icon or a "Cancel"/"Close" button within the modal). Clicking outside the modal should have no effect.

## Actual Behaviour

Clicking outside the modal causes it to disappear immediately, bypassing any required confirmation or explicit dismissal action. This can result in accidental data loss or unexpected workflow interruption.
