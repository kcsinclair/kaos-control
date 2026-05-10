---
title: "Fix: MapFilters required props in graph-show-tests-toggle test suite"
type: test
status: draft
lineage: rename-graph-to-map
parent: lifecycle/defects/rename-graph-to-map-7-defect.md
---

# Fix: MapFilters required props in graph-show-tests-toggle test suite

## Overview

Fixes the defect reported in `rename-graph-to-map-7-defect.md` where the
`MapFilters.vue` component was mounted in Milestone 2 tests without providing
three of its required props, causing Vue prop warnings on every test mount.
A secondary issue with the Milestone 3 test 5 stub name is also resolved.

## Changes Made

### File: `tests/web/graph-show-tests-toggle.test.ts`

**Milestone 2 — `defaultProps` fix (lines ~161–181)**

Added the three missing required props to the `defaultProps` constant used
by all seven Milestone 2 component tests for `MapFilters.vue`:

- `showReleases: false` — controls the "Show Releases" toggle
- `showNodeTitles: true` — controls node label visibility
- `showNodeLineage: false` — controls lineage label visibility

All seven tests now mount the component in a production-equivalent state
with no Vue prop warnings.

**Milestone 3 test 5 — stub name fix**

Changed `stubs: { GraphFilters: true }` to `stubs: { MapFilters: true }`.

`MapView.vue` imports `MapFilters.vue` under the local alias `GraphFilters`,
but Vue Test Utils resolves stubs by the component's own registered name
(`MapFilters`), not the parent's local alias. Using `MapFilters: true`
ensures prop validation is fully suppressed before the stub is applied,
eliminating the two spurious `showNodeTitles` / `showNodeLineage` warnings
that appeared for this test.

## Scenarios Covered

All scenarios are inherited from the existing test file — no new test cases
were added; only the fixture completeness was corrected:

- **Milestone 1** (8 tests): GraphStore `hideTests` state and filtering — unchanged.
- **Milestone 2** (7 tests): `MapFilters.vue` checkbox rendering — now runs without prop warnings.
- **Milestone 3** (5 tests): `MapView` integration smoke tests — test 5 no longer emits stub-resolution warnings.

## Test Files

| File | Change |
|------|--------|
| `tests/web/graph-show-tests-toggle.test.ts` | Added `showReleases`, `showNodeTitles`, `showNodeLineage` to Milestone 2 `defaultProps`; fixed Milestone 3 test 5 stub key from `GraphFilters` to `MapFilters` |
