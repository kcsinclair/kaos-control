---
title: 'Fix: TC6 TextFilter debounce bypass in release-filter test suite'
type: test
status: draft
lineage: artefacts-list-release-priority-columns
parent: lifecycle/defects/artefacts-list-release-priority-columns-8-defect.md
---

# Fix: TC6 TextFilter debounce bypass in release-filter test suite

## Summary

Fixes the TC6 regression in `tests/web/ArtifactListView.releaseFilter.test.ts` where
the combined release-filter + text-search test crashed with a `TypeError` because
`TextFilter.vue`'s 200 ms debounce prevented `listArtifacts` from being called after
a synthetic `input` event. `flushPromises()` only drains the microtask queue; it does
not advance `setTimeout` timers.

## Changes

### `tests/web/ArtifactListView.releaseFilter.test.ts`

- Added `import TextFilter from '../../web/src/components/TextFilter.vue'` so the
  component can be located via `wrapper.findComponent(TextFilter)`.
- Rewrote TC6 to emit `update:modelValue` directly on the `TextFilter` sub-component
  instance via `textFilterWrapper.vm.$emit('update:modelValue', 'login')`, bypassing the
  debounce timer entirely. This is the canonical pattern for testing debounced child
  components in Vitest without fake timers.

## Scenarios covered

| Test | Description |
|---|---|
| TC6 (fixed) | Composing a release filter with a text-search query passes both `filter.release` and `filter.q` to the `listArtifacts` call in the same fetch. |

All other TC1–TC5, TC7–TC9 tests in the suite remain unmodified and continue to pass.

## How to run

```sh
cd tests/web && pnpm vitest run ArtifactListView.releaseFilter
```

Expected output: **9 tests passed**.
