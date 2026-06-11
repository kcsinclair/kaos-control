---
title: Rice Scoring Support for Product Management Prioritisation
type: idea
status: draft
lineage: rice-scoring
priority: medium
labels:
    - feature
    - frontend
    - ux
    - artifacts
    - usability
release: KC-Release3
---

## Raw Idea

## Raw Idea
Each idea and defect can have a RICE score to assist with prioritisation.  Include a User Interface to assist with visualising and updating RICE scores, the score could be viewed in the list view, where default values or N/A are applied if all blank.

## Idea

Each idea and defect artifact should support a RICE score (Reach, Impact, Confidence, Effort) to assist with prioritisation decisions. The score fields would be stored in the artifact frontmatter and treated as optional, so existing artifacts require no migration.

A User Interface should allow users to view and edit RICE scores directly from the list view, displaying the computed RICE score alongside each artifact. Where individual components are missing or all fields are blank, the UI should display a default value or 'N/A' rather than an error or empty cell, keeping the list readable regardless of scoring completeness.

An inline editing experience within the list view (or a dedicated score panel) would let users fill in Reach, Impact, Confidence, and Effort values without navigating away, making it practical to score items during triage or planning sessions.
