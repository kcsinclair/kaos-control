---
title: Kanban board
type: feature
status: approved
lineage: feature-kanban-board
created: "2026-08-21T15:08:00+10:00"
summary: Configurable Kanban columns mapped from statuses, plus a dedicated testing board for test artifacts.
function: Boards & views
labels:
    - feature
    - kanban
related_to:
    - lifecycle/ideas/kanban-view.md
    - lifecycle/requirements/kanban-view-3.md
---

# Kanban board

A configurable board view over the same artifacts as the list and graph.

## What it does

- **Configurable columns.** Per-project `kanban.columns` config maps
  statuses to columns (e.g. "In Progress" can collapse `clarifying`,
  `planning`, `in-development`, `in-qa`).
- **Card fields.** Title, type, priority, labels, age — configurable.
- **Column-aware testing board.** Separate Kanban view for tests with text
  filter and approved-count badge.

Reachable at **Board**; test-specific view at **Testing Board**.
