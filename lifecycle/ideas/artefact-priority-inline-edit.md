---
title: Inline Priority Display and Editing on Artefact View
type: idea
status: done
lineage: artefact-priority-inline-edit
created: "2026-05-07T11:48:13+10:00"
priority: high
labels:
    - enhancement
    - frontend
    - artefacts
    - usability
    - vue
release: KC-Release0
---

# Inline Release and Priority Display including Editing on Artefact View

When viewing an artefact, the priority and release fields should be displayed prominently alongside other metadata such as status. Currently priority is not surfaced in the artefact detail view, making it invisible to users who need to assess or adjust urgency at a glance.

The priority and release field should support single-click inline editing, matching the interaction pattern already established by the status field. Clicking the displayed priority value should reveal a dropdown or selector allowing the user to change it without navigating away or opening a separate edit mode.

This change improves consistency across the artefact metadata UI and reduces friction for product owners and analysts who routinely triage and reprioritise items during planning sessions.
