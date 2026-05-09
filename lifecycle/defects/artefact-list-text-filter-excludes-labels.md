---
title: Artefact List Text Filter Does Not Include Labels
type: defect
status: draft
lineage: artefact-list-text-filter-excludes-labels
created: "2026-05-09T16:32:20+10:00"
priority: low
labels:
    - defect
    - artefacts
    - frontend
    - usability
---

# Artefact List Text Filter Does Not Include Labels

## Reproduction Steps

1. Open the artefact list view.
2. Assign one or more labels to an artefact (e.g. `frontend`, `v1`).
3. Type one of those label names into the text filter input.
4. Observe the results.

## Expected Behaviour

Artefacts whose labels match the text filter query should appear in the filtered results, just as artefacts matching on title or other indexed fields do.

## Actual Behaviour

The text filter does not search against labels; artefacts are not returned when the query matches a label value but not the title or other currently-searched fields. A dedicated label filter exists as a workaround, but text search omitting labels is inconsistent and reduces discoverability.
