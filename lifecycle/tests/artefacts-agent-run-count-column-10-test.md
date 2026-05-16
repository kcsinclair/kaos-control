---
title: "Test fix: TC2 Runs header locator — case-insensitive regex for CSS text-transform"
type: test
status: draft
lineage: artefacts-agent-run-count-column
parent: lifecycle/defects/artefacts-agent-run-count-column-8-defect.md
---

# Test fix: TC2 Runs header locator — case-insensitive regex for CSS text-transform

Addresses defect 8 (`artefacts-agent-run-count-column-8-defect.md`): TC2's Runs column header locator used a case-sensitive regex `/^Runs$/` against `element.innerText`, which Chrome renders as `"RUNS"` after applying `text-transform: uppercase` from `ArtifactListView.vue`. The filter matched nothing and the test failed.

## Changes Made

### `tests/e2e/flows/10-artefact-run-count-column.spec.ts`

Changed both occurrences of the Runs header locator regex from case-sensitive to case-insensitive:

```typescript
// Before (TC1 ~line 115, TC2 ~line 152)
.filter({ hasText: /^Runs$/ })

// After
.filter({ hasText: /^runs$/i })
```

`innerText` reflects the CSS-transformed uppercase value; the `/i` flag makes the filter match regardless of case, independently of any future changes to `text-transform`.

## Scenarios Covered

- **TC2** — Runs column header is found by the updated case-insensitive locator; clicking it once sorts all rows in non-decreasing order; clicking again sorts in non-increasing order.

The TC1 occurrence was updated as a precaution even though TC1 uses `allTextContents()` (which returns `textContent`, pre-transform) for its column-order check; the locator itself also uses `hasText` and would have the same failure mode if relied upon for visibility checks.

## Test Files

| File | Type | Command |
|------|------|---------|
| `tests/e2e/flows/10-artefact-run-count-column.spec.ts` | Playwright E2E | `cd tests/e2e && pnpm exec playwright test flows/10-artefact-run-count-column.spec.ts --reporter=list` |
