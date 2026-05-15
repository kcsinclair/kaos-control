---
title: Bulk Edit Artifacts (Status, Priority, Release)
type: idea
status: draft
lineage: artefacts-bulk-edit
created: "2026-05-12T12:38:31+10:00"
priority: high
labels:
    - feature
    - artefacts
    - frontend
    - enhancement
    - usability
release: KC-Release3
---

# Bulk Edit Artifacts (Status, Priority, Release)

Should this be bulk actions on tickets, e.g. add to work queue.

From the artifacts screen, users need the ability to select multiple artifacts at once and apply shared changes to status, priority, and release assignment in a single action. This supports quick replanning sessions where teams need to triage or reorganise a set of artifacts without editing each one individually.

The interaction flow is: select one or more artifacts via checkboxes, click a "Bulk Edit" button, and a modal appears with dropdown fields for status, priority, and release. Only fields the user explicitly changes are applied — unmodified dropdowns leave existing values intact. On save, all selected artifacts are updated atomically.

This is a primarily frontend change with a backend endpoint to accept a batch update payload, applying the changes to each artifact's frontmatter and re-indexing. The feature reduces friction during sprint planning and release grooming workflows.
