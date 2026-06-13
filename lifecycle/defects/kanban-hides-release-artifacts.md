---
title: Release Artifacts Incorrectly Shown in Kanban View
type: defect
status: draft
lineage: kanban-hides-release-artifacts
created: "2026-06-12T16:05:32+10:00"
priority: normal
labels:
    - defect
    - frontend
    - artifacts
    - releases
    - filter
release: KC-Release3
assignees:
    - role: frontend-developer
      who: agent
---

# Release Artifacts Incorrectly Shown in Kanban View

## Reproduction Steps

1. Open the application and navigate to the Kanban view.
2. Observe the list of artifact cards displayed across the Kanban columns.
3. Note that artifacts of type `release` are present among the cards.

## Expected Behaviour

Release artifacts should not appear in the Kanban view. Only artifact types relevant to active work items (e.g. ideas, requirements, plans, defects) should be shown.

## Actual Behaviour

Release artifacts are displayed in the Kanban view alongside other artifact types, cluttering the board and mixing completed/shipped artefacts with in-progress work items.
