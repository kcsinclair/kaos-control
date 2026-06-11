---
title: Rice Scoring Support for Product Management Prioritisation
type: idea
status: draft
lineage: rice-scoring
priority: medium
labels:
    - feature
    - frontend
    - backend
    - ux
    - artifacts
    - usability
release: KC-Release3
---

## Raw Idea

## Raw Idea
Each idea and defect can have a RICE score to assist with prioritisation.  Include a User Interface to assist with visualising and updating RICE scores, the score could be viewed in the list view, where default values or N/A are applied if all blank.

## Idea

Add RICE (Reach, Impact, Confidence, Effort) scoring to idea and defect artifacts to assist with prioritisation. The score fields would be stored in the artifact frontmatter and treated as optional, so existing artifacts require no migration. The computed score (Reach × Impact × Confidence / Effort) is derived automatically when all four values are present.

In list views, display the RICE score (or 'N/A' if any component is blank) as a sortable column alongside existing metadata. This allows product owners and analysts to quickly compare and rank items without opening each artifact individually.

Provide an inline editing experience within the list view (or a dedicated score panel) — accessible from both the list view and the artifact detail view — that lets users enter or update the four RICE component values without navigating away. The interface should show a live preview of the computed score as values are entered, making it practical to score items during triage or planning sessions.
