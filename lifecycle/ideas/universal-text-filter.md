---
title: Universal Text Filter Across All Views
type: idea
status: done
lineage: universal-text-filter
created: "2026-04-28T09:29:36+10:00"
priority: high
labels:
    - feature
    - frontend
    - usability
    - enhancement
---

# Universal Text Filter Across All Views

Add a free-text search input to every screen that displays data — tables, Kanban boards, and graph views. As the user types, rows, cards, or nodes that do not match the entered text are hidden or dimmed in real time, allowing fast narrowing of large artifact sets without requiring a full search submission.

The filter should work alongside existing dropdown filters, with both applied simultaneously using AND logic. Matching should be case-insensitive and cover the most relevant fields for each view (e.g. title, status, type, lineage slug), with matched text highlighted where practical.

The filter input should be consistent in placement and behaviour across all screens so users develop a single mental model. State should be local to each screen session and reset on navigation, unless a later decision is made to persist filters in the URL for shareability.
