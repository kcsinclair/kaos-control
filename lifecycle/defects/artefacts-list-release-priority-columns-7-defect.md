---
title: Release sort pins empty release to tail in both directions instead of first-in-ascending
type: defect
status: done
lineage: artefacts-list-release-priority-columns
parent: lifecycle/tests/artefacts-list-release-priority-columns-6-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
---

# Release sort pins empty release to tail in both directions instead of first-in-ascending

## Reproduction Steps

1. Open the Artifact List view with artifacts that include at least one with no `release` field set.
2. Click the **Release** column header to apply ascending sort.
3. Observe the order of rows in the table.

Alternatively, run the failing unit tests:

```sh
cd tests/web && pnpm vitest run ArtifactListView.releaseSort
```

## Expected Behaviour

Ascending sort: the artifact with no release (empty string `''` via `row.frontmatter?.release ?? ''`) appears **first** — empty string sorts before any non-empty string alphabetically.

Descending sort: the artifact with no release appears **last**.

Spec reference: Milestone 5 TC1 and TC3 of `lifecycle/tests/artefacts-list-release-priority-columns-6-test.md`.

## Actual Behaviour

Ascending sort: the artifact with no release appears **last**. The order seen is `alpha`, `v1.0`, `v2.0`, `—` instead of `—`, `alpha`, `v1.0`, `v2.0`.

The same tail-pinning applies in descending order, so the artifact-without-release stays at the end in both directions rather than moving to first position in ascending.

## Root Cause

`web/src/composables/useSortableTable.ts` lines 101–106 unconditionally pin null and empty-string values to the **tail** regardless of sort direction:

```ts
// Pin nulls and empty strings to end regardless of sort direction
const aIsEmpty = aVal == null || aVal === ''
const bIsEmpty = bVal == null || bVal === ''
if (aIsEmpty && bIsEmpty) return 0
if (aIsEmpty) return 1   // always pushes empty to the end
if (bIsEmpty) return -1
```

This was introduced by commit `e94d17c fix(useSortableTable): pin empty strings and nulls to sort tail`. The release column spec requires natural alphabetical ordering (where `'' < 'alpha'`), meaning the empty-string artifact should float to the **top** in ascending order and to the **bottom** in descending order — the opposite of the tail-pinning behaviour.

The fix should make the empty-pin behaviour direction-aware (or only apply it to columns where tail-pinning is desired, such as Priority where the custom numeric comparator handles ordering).

## Logs / Output

```
FAIL  ArtifactListView.releaseSort.test.ts > ArtifactListView — Release column sort > TC1: ascending sort orders rows alphabetically: (empty), alpha, v1.0, v2.0
AssertionError: expected 'alpha' to be '—' // Object.is equality

- Expected
+ Received

- —
+ alpha

 ❯ ArtifactListView.releaseSort.test.ts:163:23
    161|     expect(values.length).toBe(4)
    162|     // '' (empty string) < 'alpha' < 'v1.0' < 'v2.0' in localeCompare
    163|     expect(values[0]).toBe('—')      // missing release renders as '—'
       |                       ^

FAIL  ArtifactListView.releaseSort.test.ts > ArtifactListView — Release column sort > TC3: artifact with no release appears first in ascending and last in descending
AssertionError: expected 'alpha' to be '—' // Object.is equality

- Expected
+ Received

- —
+ alpha

 ❯ ArtifactListView.releaseSort.test.ts:202:23
    200|     await clickSortHeader(wrapper, 'Release')
    201|     let values = getReleaseValues(wrapper)
    202|     expect(values[0]).toBe('—')
       |                       ^

Test Files  1 failed | 0 passed
      Tests  2 failed | 2 passed (4)
```
