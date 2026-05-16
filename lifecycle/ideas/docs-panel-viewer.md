---
title: Documentation Panel Viewer
type: idea
status: clarifying
lineage: docs-panel-viewer
created: "2026-05-16T09:18:03+10:00"
priority: normal
labels:
    - feature
    - frontend
    - vue
    - usability
release: KC-Release3
---

# Documentation Panel Viewer

Add a "Documentation" option to the left navigation panel that renders the contents of the `docs/` folder as a browsable list of cards. Each card displays the document title and a short summary extracted from the top of the file. Cards are sorted alphabetically by title.

A search box at the top of the panel filters the visible cards in real time, showing only documents whose title or summary matches the query. This allows users to quickly locate relevant documentation without scrolling through the full list.

Documents should be openable in the existing markdown editor, enabling users to read and edit documentation in place using the same editing experience already available for lifecycle artifacts.
