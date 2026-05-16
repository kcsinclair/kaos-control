---
title: Artifact List View — "Runs" Column Header and Cell Missing from Table
type: defect
status: done
lineage: artifact-list-runs-column-not-present
created: "2026-05-16T14:00:00+10:00"
priority: high
labels:
    - defect
    - frontend
    - artifacts
    - runs
release: KC-Release2
assignees:
    - role: frontend-developer
      who: agent
---

# Artifact List View — "Runs" Column Header and Cell Missing from Table

## Reproduction Steps

1. Navigate to `/p/testproject/artifacts`.
2. Inspect the table headers.
3. Check for a "Runs" column header (`th`) and per-row run count cells (`.cell-runs`).

## Expected Behaviour

- A "Runs" column header is present between the "Type" and "Created" columns.
- Each row contains a `.cell-runs` cell showing the integer count of completed agent runs for that artifact (0 if none).
- The column is sortable (clicking the header cycles ascending / descending).

## Actual Behaviour

The E2E tests in `Flow 10 — Artefact run count column` fail with:

```
Error: expect(locator).toBeVisible() failed
Locator: locator('th.sort-th, th[role="columnheader"]').filter({ hasText: /^Runs$/ })
Expected: visible
Error: element(s) not found
```

No "Runs" column header is present in the artifact list table. Both TC1 (column present with correct counts) and TC2 (column sortable) fail because the header element does not exist.

Failing tests:
- `flows/10-artefact-run-count-column.spec.ts:94` — TC1: Runs column is present, positioned correctly, shows correct counts including 0
- `flows/10-artefact-run-count-column.spec.ts:142` — TC2: Runs column is sortable (ascending then descending)

## Notes

The "Runs" column (with `.cell-runs` per-row cells and a sortable header) has not been implemented in the `ArtifactListView` component. The test plan for this feature is in `lifecycle/test-plans/` (Flow 10, Milestones 4–6). The frontend implementation should add: the column header with sort support, per-row run count cells sourced from a new or existing API endpoint, and WebSocket-driven refresh on `agent.finished` events.
