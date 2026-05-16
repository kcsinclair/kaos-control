---
title: TC2 Runs column header locator fails — CSS text-transform:uppercase breaks case-sensitive regex
type: defect
status: done
lineage: artefacts-agent-run-count-column
parent: lifecycle/tests/artefacts-agent-run-count-column-6-test.md
labels:
    - defect
release: KC-Release2
assignees:
    - role: test-developer
      who: agent
---

# TC2 Runs column header locator fails — CSS text-transform:uppercase breaks case-sensitive regex

## Reproduction Steps

1. Ensure `dist/kaos-control` is current (`make build`).
2. Run `cd tests/e2e && pnpm exec playwright test flows/10-artefact-run-count-column.spec.ts --reporter=list`
3. Observe TC2 fails after the table loads (18 rows are visible).

## Expected Behaviour

`page.locator('th.sort-th, th[role="columnheader"]').filter({ hasText: /^Runs$/ })` finds
the Runs column header, which is confirmed present in the ARIA snapshot as
`columnheader "Runs"`.  `toBeVisible()` succeeds within the 5 s timeout.

## Actual Behaviour

The locator returns zero elements and the test fails with:

```
Error: expect(locator).toBeVisible() failed
Locator: locator('th.sort-th, th[role="columnheader"]').filter({ hasText: /^Runs$/ })
Expected: visible
Timeout: 5000ms
Error: element(s) not found
```

The ARIA page snapshot confirms the `columnheader "Runs"` element **is** in the DOM and the
Runs data cells render correctly (each row has a `"0"` cell for runs).

## Root Cause

`ArtifactListView.vue` applies `text-transform: uppercase` to all table header cells:

```css
/* ArtifactListView.vue line 504 */
.artifact-table th {
  ...
  text-transform: uppercase;
}
```

Playwright's `hasText` with a `RegExp` matches against `element.innerText`, which
Chrome computes **after** applying CSS transforms.  The `innerText` of the Runs `<th>`
is therefore `"RUNS"`, not `"Runs"`.

The test regex `/^Runs$/` is case-sensitive and does not match `"RUNS"`, so the filter
eliminates all candidates and the locator finds nothing.

Diagnostic confirmation — running a one-shot debug spec:
```
innerText="RUNS" textContent="Runs"    # Runs column header
innerText="PATH" textContent="Path"    # Path column header
```

`allTextContents()` (used in TC1) returns `textContent` (pre-transform), which is why
TC1's `startsWith('Runs')` check would have succeeded.  Only `hasText` with a `RegExp`
is affected.

## Fix Required

In `tests/e2e/flows/10-artefact-run-count-column.spec.ts`, change the Runs header locator
regex to be case-insensitive (two occurrences: TC1 line ~115 and TC2 line ~152):

```typescript
// Before
.filter({ hasText: /^Runs$/ })

// After
.filter({ hasText: /^runs$/i })
```

Alternatively, use the role-based locator which matches against the accessible name
(independent of CSS transforms):

```typescript
const runsHeader = page.getByRole('columnheader', { name: 'Runs', exact: true })
```

## Logs / Output

```
  ✘  2 flows/10-artefact-run-count-column.spec.ts:142:3 › TC2: Runs column is sortable (5.3s)

Error: expect(locator).toBeVisible() failed
Locator: locator('th.sort-th, th[role="columnheader"]').filter({ hasText: /^Runs$/ })
Expected: visible
Timeout: 5000ms
Error: element(s) not found
```
