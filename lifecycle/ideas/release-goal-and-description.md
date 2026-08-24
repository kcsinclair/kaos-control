---
title: Release Goal and Description Fields
type: idea
status: draft
lineage: release-goal-and-description
created: "2026-08-24T18:24:34+10:00"
priority: normal
parent: lifecycle/ideas/frontend-lint-gap.md
labels:
    - releases
    - feature
    - enhancement
    - ui
    - frontend
    - backend
release: KC-Release6
---

# Release Goal and Description Fields

Release artifacts should support two optional fields: a short `goal` (e.g. "More and better LLM support") and a longer `description` providing narrative context. Both fields are optional so existing releases remain valid without migration.

The `goal` serves as a one-line intent statement visible in release lists and kanban views, giving stakeholders a quick read on what a release is working toward. The `description` allows free-form markdown for richer context — scope rationale, non-goals, links to related requirements, or delivery notes.

The fields should be surfaced in the release editor in the frontend and included in the release artifact's YAML frontmatter, indexed in SQLite alongside existing release metadata, and rendered in the release detail view.
