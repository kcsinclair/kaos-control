---
title: SortHeader.vue uses slot for label text but test plan specifies a label prop
type: defect
status: done
lineage: sortable-table-columns
parent: lifecycle/tests/sortable-table-columns-6-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
release: KC-Feature-Sprint
---

# SortHeader.vue uses slot for label text but test plan specifies a label prop

## Reproduction Steps

1. Run `cd tests/web && pnpm test SortHeader.a11y --reporter=verbose`.
2. Observe one failure:
   ```
   × SortHeader — label text — renders the label text
     → expected '' to contain 'Created'
   ```
3. The test mounts `SortHeader` with `{ props: { label: 'Created', column: 'title', sortColumn: null, sortDirection: null, sortable: true } }`.
4. It then asserts `wrapper.text()` contains `'Created'`.
5. The actual component (`web/src/components/SortHeader.vue`) does not declare a `label` prop. Instead it uses `<slot />` to render the column header text.
6. Since no slot content is provided by the test, `wrapper.text()` returns an empty string (only the icon SVG is rendered).

## Expected Behaviour

`SortHeader` should accept a `label` string prop and render it as the visible column header text. The test plan spec (header comment of `SortHeader.a11y.test.ts`, line 7) explicitly lists `label: string — display text for the column` as a required prop.

## Actual Behaviour

The component renders no visible text when mounted via props alone:

```
× SortHeader.a11y.test.ts > SortHeader — label text > renders the label text
AssertionError: expected '' to contain 'Created'

 ❯ SortHeader.a11y.test.ts:207:28
    205|   it('renders the label text', () => {
    206|     const wrapper = mountSortHeader({ label: 'Created' })
    207|     expect(wrapper.text()).toContain('Created')
             |                            ^
```

## Logs / Output

```
 Test Files  1 failed | ...
       Tests  1 failed (33 tests total in this file)

FAIL  SortHeader.a11y.test.ts > SortHeader — label text > renders the label text
→ expected '' to contain 'Created'
```

Fix: add a `label` prop to `SortHeader.vue` and render it in the template:

```ts
// In <script setup>:
const props = defineProps<{
  label: string        // ← add this
  column: string
  sortColumn: string | null
  sortDirection: SortDirection
  sortable?: boolean
}>()
```

```html
<!-- In <template>: -->
<span class="sort-th__inner">
  {{ label }}          <!-- ← replace or augment <slot /> -->
  ...icons...
</span>
```

All call sites in ArtifactListView, AgentsRunsView, and ParseErrorsView that currently use slot content for the header text will need to be updated to pass the `label` prop instead (or both slot and prop can be supported if backwards compatibility is needed).
