---
title: Markdown Tables Not Rendering Correctly in Artefact View
type: defect
status: draft
lineage: markdown-tables-artefact-view
created: "2026-05-11T13:10:28+10:00"
priority: normal
labels:
    - defect
    - frontend
    - artefacts
    - vue
    - usability
release: KC-Release1
---

# Markdown Tables Not Rendering Correctly in Artefact View

## Reproduction Steps

1. Open an artefact that contains a markdown table in its content.
2. Observe the rendered artefact view.
3. Open the same artefact in the editor and observe the table there.

## Expected Behaviour

Markdown tables should render correctly as formatted HTML tables in the artefact view, consistent with how they appear in the editor.

## Actual Behaviour

Markdown tables are not displayed correctly in the artefact view (e.g. raw pipe characters and dashes shown instead of a formatted table, or layout is broken), while the same tables display correctly in the editor view.
