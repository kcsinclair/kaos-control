---
title: Hide Done Items by Default Across All Screens
type: idea
status: done
lineage: hide-done-items-by-default
created: "2026-04-28T10:20:46+10:00"
priority: normal
labels:
    - feature
    - frontend
    - usability
release: April2026
---

# Hide Done Items by Default Across All Screens

All list and table screens should filter out artifacts with `status=done, rejected, abandoned` by default, reducing noise and keeping the focus on active work. A checkbox in the page header — labelled something like "Show done" — should control this filter and be unchecked by default (i.e. done items are hidden).

When a user wants to review completed work they can check the box to reveal done items inline alongside active ones. The preference does not need to persist across sessions; resetting to hidden on each page load is the intended behaviour.

This applies consistently across every screen that displays artifacts — the kanban board, table/list views, graph view, and any other surfaces that render artifacts by status.
