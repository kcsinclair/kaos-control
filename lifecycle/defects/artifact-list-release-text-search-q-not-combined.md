---
title: ArtifactListView — Release Filter and Text Search Not Combined; q Param Missing from fetchList Call
type: defect
status: in-development
lineage: artifact-list-release-text-search-q-not-combined
created: "2026-05-16T14:00:00+10:00"
priority: normal
labels:
    - defect
    - frontend
    - artifacts
    - filter
release: KC-Release2
assignees:
    - role: frontend-developer
      who: agent
---

# ArtifactListView — Release Filter and Text Search Not Combined; q Param Missing from fetchList Call

## Reproduction Steps

1. Navigate to `/p/testproject/artifacts`.
2. Select a release from the release dropdown (e.g. `v1.0`).
3. While the release filter is active, type a search term (e.g. `login`) in the text search field.
4. Observe the API call sent to `listArtifacts`.

## Expected Behaviour

The combined filter call to `listArtifacts` includes both `{ release: 'v1.0', q: 'login' }`.

## Actual Behaviour

The Vitest unit test `ArtifactListView — Release filter dropdown > TC6: composing release filter with text search passes both to fetchList` fails:

```
AssertionError: expected undefined to be 'login'
filter.release === 'v1.0'  ✓
filter.q       === undefined  ✗  (expected 'login')
```

When a release filter is active and the user types in the text search box, the `q` parameter is omitted from the `fetchList` / `listArtifacts` call. Only the release filter is forwarded; the text search term is silently dropped.

Test file: `tests/web/ArtifactListView.releaseFilter.test.ts:287`

## Notes

The `TextFilter` sub-component emits `update:modelValue` with the search string. The handler in `ArtifactListView` likely rebuilds the filter object from the release selector's state but does not read the current text-search model value, causing `q` to be omitted when both filters are active simultaneously. The fix should ensure the combined filter payload always includes both `release` (if set) and `q` (if non-empty).
