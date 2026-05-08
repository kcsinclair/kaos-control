---
title: "ArtifactListView Release sort: TC1/TC3 corrected for empty-string-to-end behaviour"
type: test
status: in-qa
lineage: sortable-table-columns
parent: lifecycle/defects/sortable-table-columns-18-defect.md
---

# ArtifactListView Release sort: TC1/TC3 corrected for empty-string-to-end behaviour

Fixes stale test assertions in `ArtifactListView.releaseSort.test.ts` that
pre-dated the `useSortableTable` change which pins empty strings to the **end**
of a sorted list regardless of direction (commit `0e39fc9`).

## File modified

`tests/web/ArtifactListView.releaseSort.test.ts`

## Changes made

### File header comment

Updated the implementation note from:

> Missing release maps to `''` (empty string, not null), so it sorts at the
> **start** of ascending and end of descending via string ordering.

to:

> Missing release maps to `''` (empty string, not null), but `useSortableTable`
> pins empty strings to the **END** of the sorted list in both directions.

### TC1 — ascending sort order

- **Old description:** "ascending sort orders rows alphabetically: (empty), alpha, v1.0, v2.0"
- **New description:** "ascending sort orders rows alphabetically: alpha, v1.0, v2.0, (empty)"
- Expected order changed from `['—', 'alpha', 'v1.0', 'v2.0']` to
  `['alpha', 'v1.0', 'v2.0', '—']`.

### TC3 — empty value placement

- **Old description:** "artifact with no release appears first in ascending and last in descending"
- **New description:** "artifact with no release appears last in both ascending and descending"
- Ascending assertion changed from `expect(values[0]).toBe('—')` to
  `expect(values[values.length - 1]).toBe('—')`.

## Scenarios covered

| Test case | Scenario | Expected outcome |
|-----------|----------|-----------------|
| TC1 | Ascending sort with a missing-release artifact | `alpha, v1.0, v2.0, —` (empty last) |
| TC2 | Descending sort with a missing-release artifact | `v2.0, v1.0, alpha, —` (empty last) — unchanged, was already correct |
| TC3 | Empty release position in ascending | Last row is `—` |
| TC3 | Empty release position in descending | Last row is `—` |
| TC4 | Case-insensitive sort | `alpha` and `Alpha` sort adjacently, both before `beta` |
