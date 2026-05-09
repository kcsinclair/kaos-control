---
title: Kanban View
type: idea
status: done
lineage: kanban-view
priority: normal
labels:
    - artefacts
    - workflow
release: KC-OG-Sprint
---

Rename artifacts to artefacts.

Current left menu item Artefacts will include a sub-menu of List which is the current view

Create a new Artefact view which is Kanban of the artefacts, with the same filters included so you can dynamically refine what you are seeing. The result then is Kanban cards displayed in columns.

Create a Kanban view configuration in YAML so you can change which items are displayed on the card.

Configuration should define the kanban columns and which status fits into which column.

Lets start with:

kanban:
  - column: Backlog
    status: [draft]
  - column: Approved
    status: [approved]
  - column: In-Progress
    status: [in-progress]
  - column: Blocked
    status: [blocked,rejected]
  - column: Done
    status: [done]

Do we need an undefined column, which catches anything not matching the above.
