---
title: Test Artifact Management and Test Runner
type: idea
status: draft
lineage: test-artifact-management
created: "2026-05-05T18:38:51+10:00"
priority: normal
labels:
    - feature
    - testing
    - frontend
    - backend
    - qa
    - agent
    - artefacts
    - vue
---

# Test Artifact Management and Test Runner

Tests are a special class of artifact that are run frequently and require dedicated UX treatment. A new left-menu item called "Testing" should display a board of all test artifacts as cards (matching the style of Kanban board cards). Selecting a test card opens the existing `kaos-control/artifacts` detail screen. On the Kanban board, a "Show Tests" checkbox (unchecked by default) controls whether test-type artifacts appear in the board columns, keeping the default view uncluttered.

On the artifact detail screen, a "Run Test" button should appear when the artifact status is `approved`. Clicking it invokes the agent runner using the QA agent against that single test artifact, following the same agent execution flow used elsewhere in the system.

The Testing board should support multi-select: only approved tests are selectable. A "Run Tests" button runs all selected tests serially via the QA agent, waiting for each to complete before starting the next. Non-approved tests should be visually distinguished and excluded from selection.
