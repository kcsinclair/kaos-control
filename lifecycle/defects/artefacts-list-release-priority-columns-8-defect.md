---
title: TC6 release-filter + text-search test fails because TextFilter debounce is not advanced
type: defect
status: draft
lineage: artefacts-list-release-priority-columns
parent: lifecycle/tests/artefacts-list-release-priority-columns-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# TC6 release-filter + text-search test fails because TextFilter debounce is not advanced

## Reproduction Steps

1. Run the release-filter web component test suite:

```sh
cd tests/web && pnpm vitest run ArtifactListView.releaseFilter
```

2. Observe the TC6 failure.

## Expected Behaviour

TC6 verifies that when a user combines a release filter selection with a text search query, both `filter.release` and `filter.q` are forwarded together in the same `listArtifacts` call. The test should pass and confirm the composed filter is correctly assembled.

## Actual Behaviour

`listArtifacts` is never called after the text-input event, so `mock.calls.at(-1)` returns `undefined`. Accessing `undefined[1]` throws a `TypeError` and the test crashes rather than failing with a meaningful assertion.

## Root Cause

`web/src/components/TextFilter.vue` debounces the `update:modelValue` emit by **200 ms** (the default `debounceMs` prop). The test in `tests/web/ArtifactListView.releaseFilter.test.ts` (lines 275–289) triggers the native `input` event and then calls `flushPromises()`. `flushPromises()` only drains the microtask queue; it does not advance `setTimeout` timers. Therefore the debounce callback never fires, `onSearchText` is never called in the parent, `applyFilters` is not invoked, and no new `listArtifacts` call is recorded.

The fix requires one of the following in the test:

- Add `vi.useFakeTimers()` in a `beforeEach` / `afterEach` pair and call `vi.advanceTimersByTime(250)` after triggering the input event.
- Mount `TextFilter` with `debounceMs: 0` (pass via the component's stub or as a prop on the `TextFilter` sub-component) so the emit is synchronous during tests.
- Emit `update:model-value` directly on the `TextFilter` wrapper component via `wrapper.findComponent(TextFilter).vm.$emit(...)` to bypass the debounce entirely.

## Logs / Output

```
FAIL  ArtifactListView.releaseFilter.test.ts > ArtifactListView — Release filter dropdown > TC6: composing release filter with text search passes both to fetchList
TypeError: Cannot read properties of undefined (reading '1')
 ❯ ArtifactListView.releaseFilter.test.ts:282:22
    280|
    281|       const lastCall = vi.mocked(artifactsApi.listArtifacts).mock.calls.at(-1)!
    282|       const filter = lastCall[1] as Record<string, unknown>
       |                      ^
    283|       expect(filter.release).toBe('v1.0')
    284|       expect(filter.q).toBe('login')

Test Files  1 failed | 5 passed (6)
      Tests  1 failed | 30 passed (31)
```
