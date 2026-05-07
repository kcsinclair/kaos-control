---
title: Status Checker Button
type: idea
status: done
lineage: status-checker-button
created: "2026-04-29T15:48:39+10:00"
priority: normal
labels:
    - feature
    - artefacts
    - workflow
release: April2026
---

# Status Checker Button

Add a status checker button to the UI that inspects the current state of all artifacts within a lineage and identifies where statuses are stale or inconsistent — for example, where an idea is still marked as `clarifying` but its child requirement is already in `planning` and all development plans are `done`.

When stale statuses are detected, the tool should surface a summary of the discrepancies and offer to automatically advance the affected artifacts to their correct status based on the actual state of their descendants. This brings parent artifacts in line with the real progress of the lineage without requiring manual edits.

The feature should be accessible from the artifact detail panel or the graph view, and could run either on a single selected artifact or across all artifacts in a project to produce a full staleness report.
