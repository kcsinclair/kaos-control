---
title: "ArtifactListView Release sort: TC1/TC3 expect empty values first in ascending (stale after empty-string-to-end fix)"
type: defect
status: approved
lineage: sortable-table-columns
parent: lifecycle/tests/sortable-table-columns-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# ArtifactListView Release sort: TC1/TC3 expect empty values first in ascending (stale after empty-string-to-end fix)

## Reproduction Steps

1. Run the full integration test suite from `tests/web/`:
   ```sh
   cd tests/web && pnpm test
   ```
2. Observe failures in `ArtifactListView.releaseSort.test.ts`:
   - TC1: "ascending sort orders rows alphabetically: (empty), alpha, v1.0, v2.0"
   - TC3: "artifact with no release appears first in ascending and last in descending"

## Expected Behaviour

Per `lifecycle/tests/sortable-table-columns-6-test.md` (Milestone 1 scenarios):
> "Empty string handling (ascending) | Empty strings sort to the end"

And per the `useSortableTable` implementation (`web/src/composables/useSortableTable.ts` lines 101–106), empty strings and nulls are pinned to the END of the sorted list regardless of sort direction.

TC1 and TC3 should therefore expect: in ascending sort, artifacts with no release (rendered as `—`) appear **last**, not first.

## Actual Behaviour

TC1 and TC3 assert that an artifact with no release value (`frontmatter.release === undefined`, mapping to `''`) sorts **first** in ascending order:

```
// TC1 line 163
expect(values[0]).toBe('—')   // fails — actual first value is 'alpha'
```

The test comment at the top of the file says:
> "Missing release maps to '' (empty string, not null), so it sorts at the start of ascending and end of descending via string ordering."

This comment and the associated assertions reflect the behaviour **before** commit `0e39fc9` ("fix(useSortableTable): sort empty strings to end in ascending order"), which changed empty strings to pin to end in all directions. TC2 (descending) was correctly updated and passes, but TC1 and TC3 were not updated.

## Logs / Output

```
FAIL  ArtifactListView.releaseSort.test.ts
  × TC1: ascending sort orders rows alphabetically: (empty), alpha, v1.0, v2.0
    AssertionError: expected 'alpha' to be '—' // Object.is equality
    - Expected: —
    + Received: alpha
     ❯ ArtifactListView.releaseSort.test.ts:163:23

  × TC3: artifact with no release appears first in ascending and last in descending
    AssertionError: expected 'alpha' to be '—' // Object.is equality
    - Expected: —
    + Received: alpha
     ❯ ArtifactListView.releaseSort.test.ts:202:23

 Test Files  1 failed | 49 passed (50)
      Tests  2 failed | 849 passed (851)
```

## Fix Guidance

In `tests/web/ArtifactListView.releaseSort.test.ts`:

1. Update the file header comment to say empty string sorts to the **end** (not start) in ascending.
2. TC1: change the expected order to `alpha, v1.0, v2.0, —` (empty last).
3. TC3: change the ascending assertion to `expect(values[values.length - 1]).toBe('—')` (empty last in ascending too, consistent with the implementation's "pin to end" behaviour).
