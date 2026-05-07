---
title: Inline Status Change on Artefact View
type: idea
status: done
lineage: artefact-inline-status-change
created: "2026-05-06T17:42:43+10:00"
priority: normal
labels:
    - frontend
    - artefacts
    - usability
    - enhancement
    - vue
release: May2026
---

# Inline Status Change on Artefact View

When viewing an artefact, the current status field should be interactive rather than leading to a separate "change status" action. Clicking the displayed status badge or label opens a dropdown listing the valid target statuses for that artefact.

Selecting a status from the dropdown immediately triggers the status transition and saves the change in place, replacing the old two-step flow. The dropdown should only surface statuses that are valid transitions from the current state, enforcing the workflow state machine rules already in place on the backend.

This reduces friction for the most common editing action on an artefact and keeps the user in context rather than navigating away or opening a modal.
