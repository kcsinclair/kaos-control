---
title: "ArtifactListView TC6: listArtifacts not called after text input — TypeError on mock.calls"
type: defect
status: done
lineage: releases-and-roadmaps
parent: lifecycle/tests/releases-and-roadmaps-6-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
release: KC-Feature-Sprint
---

# ArtifactListView TC6: listArtifacts not called after text input — TypeError on mock.calls

`tests/web/ArtifactListView.releaseFilter.test.ts` TC6 fails with a `TypeError` because `artifactsApi.listArtifacts` is not invoked after a text-input event, so `mock.calls.at(-1)` returns `undefined`.

## Reproduction Steps

1. `cd tests/web`
2. `pnpm exec vitest run ArtifactListView.releaseFilter.test.ts`
3. Observe failure in:
   - `ArtifactListView — Release filter dropdown › TC6: composing release filter with text search passes both to fetchList`

## Expected Behaviour

After selecting a release filter (`v1.0`) and then typing `'login'` into the text search input, `artifactsApi.listArtifacts` is called with a filter object containing both `release: 'v1.0'` and `q: 'login'`.

## Actual Behaviour

`vi.mocked(artifactsApi.listArtifacts).mock.calls.at(-1)` returns `undefined` — the API was not called after the text input event. Accessing index `[1]` on `undefined` throws:

```
TypeError: Cannot read properties of undefined (reading '1')
 ❯ ArtifactListView.releaseFilter.test.ts:282:22
    281|       const lastCall = vi.mocked(artifactsApi.listArtifacts).mock.calls.at(-1)!
    282|       const filter = lastCall[1] as Record<string, unknown>
```

This indicates that `ArtifactListView` does not re-fetch when the text search composable emits its event while a release filter is already active — the two filter signals are not composed together and forwarded to `listArtifacts`.

## Logs / Output

```
 FAIL  ArtifactListView.releaseFilter.test.ts > ArtifactListView — Release filter dropdown > TC6: composing release filter with text search passes both to fetchList
TypeError: Cannot read properties of undefined (reading '1')
 ❯ ArtifactListView.releaseFilter.test.ts:282:22
    281|       const lastCall = vi.mocked(artifactsApi.listArtifacts).mock.calls.at(-1)!
    282|       const filter = lastCall[1] as Record<string, unknown>
    283|       expect(filter.release).toBe('v1.0')
    284|       expect(filter.q).toBe('login')

 Test Files  1 failed (1)
      Tests  1 failed | 8 passed (9)
```
