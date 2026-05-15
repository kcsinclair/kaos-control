---
title: Artifact Relationship Links Not Clickable in Right Panel
type: defect
status: in-development
lineage: artifact-relationship-links-not-clickable
created: "2026-05-16T08:09:06+10:00"
priority: normal
labels:
    - defect
    - frontend
    - artefacts
    - vue
release: KC-Release2
---

# Artifact Relationship Links Not Clickable in Right Panel

## Reproduction Steps

1. Open the application and navigate to any artifact detail view.
2. Observe the right panel which displays parent and children artifact relationships.
3. Note that artifact names are shown following the recent naming changes introduced in `artefact-relationship-labels-and-links.md`.
4. Attempt to click on a parent or child artifact link in the right panel.

## Expected Behaviour

Parent and child artifact entries in the right panel should be rendered as clickable links that navigate to the respective artifact detail view when clicked.

## Actual Behaviour

The parent and child artifact names are displayed in the right panel with the updated naming/labels, but they are not clickable — they appear as plain text rather than navigable links, preventing users from traversing the artifact relationship graph via the UI.
