---
title: 'Test Plan: Artefacts List Release & Priority Columns'
type: plan-test
status: draft
lineage: artefacts-list-release-priority-columns
parent: requirements/artefacts-list-release-priority-columns-2.md
---

# Test Plan: Artefacts List Release & Priority Columns

## Summary

Integration and end-to-end tests validating the new Priority and Release columns, priority sort order, and release filter in the artifact list view. Tests exercise the backend API contract and the frontend rendering/interaction behaviour described in the [[artefacts-list-release-priority-columns]] backend and frontend plans.

---

## Milestone 1 — Backend API filter tests

### Description

Write integration tests that exercise the `release` query parameter on the artifact list endpoint. These tests use the Go test framework against a running or in-process server with seeded test artifacts.

### Files to change

- `tests/` — new test file (e.g. `tests/artifact_release_filter_test.go`) or extend existing artifact list test file

### Test cases

1. **Release filter — exact match**: Seed artifacts with releases `v1.0` and `v2.0`. Request `GET /api/p/:project/artifacts?release=v1.0`. Assert only `v1.0` artifacts returned.
2. **Release filter — unassigned**: Seed artifacts with and without release. Request `?release=__unassigned__`. Assert only artifacts with empty/null release returned.
3. **Release filter — composition**: Request `?release=v1.0&status=draft`. Assert results match both conditions.
4. **Release filter — no match**: Request `?release=nonexistent`. Assert empty result set with `total: 0`.
5. **Priority in response**: Verify that artifact responses include `frontmatter.priority` with correct values.
6. **Release in response**: Verify that artifact responses include `frontmatter.release` with correct values.

### Acceptance criteria

- [ ] All 6 test cases pass.
- [ ] Tests are idempotent (set up and tear down their own data).
- [ ] Tests run as part of `make test-unit` or a dedicated integration target.

---

## Milestone 2 — Priority column display tests

### Description

Write tests (or extend existing test infrastructure) verifying that the Priority column renders correctly in the artifact list view, including pill styling and empty-state handling.

### Files to change

- `tests/` — new or extended test file for list view rendering
- Alternatively: `web/src/views/project/__tests__/` if component tests exist

### Test cases

1. **Priority pill rendering**: Given an artifact with `priority: high`, the list table row contains a priority pill element with text "high" and appropriate styling class.
2. **Priority empty state**: Given an artifact with no priority field, the cell displays `—`.
3. **All priority values**: Verify each of `critical`, `high`, `normal`, `low` renders with its designated colour class.
4. **Priority column position**: The Priority column header appears after Status and before Release.

### Acceptance criteria

- [ ] All 4 test cases pass.
- [ ] Tests verify DOM structure, not just data presence.

---

## Milestone 3 — Release column display tests

### Description

Verify the Release column renders correctly with plain text values and dash fallback.

### Files to change

- `tests/` — same test file as Milestone 2 or a new companion

### Test cases

1. **Release value display**: Given an artifact with `release: "v1.0"`, the list row shows "v1.0" in the Release column cell.
2. **Release empty state**: Given an artifact with no release, the cell displays `—`.
3. **Release column position**: The Release column header appears after Priority.

### Acceptance criteria

- [ ] All 3 test cases pass.

---

## Milestone 4 — Priority sort-order tests

### Description

Verify that sorting by the Priority column uses logical severity order, not alphabetical order.

### Files to change

- `tests/` — sort behaviour test file

### Test cases

1. **Descending sort**: Click Priority header once (descending). Verify row order: critical, high, normal, low, (empty).
2. **Ascending sort**: Click Priority header twice (ascending). Verify row order: (empty), low, normal, high, critical.
3. **Sort cycle**: Click Priority header three times. Verify sort resets to default (unsorted) order.
4. **Null handling**: Artifacts with no priority always appear last regardless of sort direction.

### Acceptance criteria

- [ ] All 4 test cases pass.
- [ ] Sort does not use alphabetical comparison (which would place `critical` before `high` before `low` before `normal`).

---

## Milestone 5 — Release sort tests

### Description

Verify that sorting by the Release column uses alphabetical (case-insensitive) order with null-last behaviour.

### Files to change

- `tests/` — sort behaviour test file (same as Milestone 4)

### Test cases

1. **Alphabetical sort ascending**: Given releases `v1.0`, `v2.0`, `alpha`. Ascending order: `alpha`, `v1.0`, `v2.0`, (empty).
2. **Alphabetical sort descending**: Descending order: `v2.0`, `v1.0`, `alpha`, (empty).
3. **Null-last**: Artifacts with no release appear at the end in both ascending and descending sorts.
4. **Case insensitivity**: `Alpha` and `alpha` sort adjacently, not separated by case.

### Acceptance criteria

- [ ] All 4 test cases pass.

---

## Milestone 6 — Release filter interaction tests

### Description

End-to-end tests verifying the release filter dropdown in the List view, including composition with other filters and sort reset behaviour.

### Files to change

- `tests/` — filter interaction test file

### Test cases

1. **Filter population**: The release dropdown contains all distinct release values from the dataset plus "All Releases" and "Unassigned".
2. **Filter selection**: Selecting "v1.0" reduces the table to only `v1.0` artifacts.
3. **Unassigned filter**: Selecting "Unassigned" shows only artifacts with no release.
4. **All Releases**: Selecting "All Releases" clears the filter and shows all artifacts.
5. **Composition with status filter**: Set status filter to "draft" AND release to "v1.0". Verify only draft v1.0 artifacts appear.
6. **Composition with text search**: Set release to "v1.0" and type a search term. Verify results match both.
7. **Sort reset on filter change**: Sort by title ascending, then change release filter. Verify sort resets to default.
8. **Reset button**: Click reset-all-filters. Verify release filter returns to "All Releases".
9. **Empty state**: Set release filter to a value with no matching artifacts after other filters applied. Verify empty-state message appears.

### Acceptance criteria

- [ ] All 9 test cases pass.
- [ ] Filter changes trigger re-fetch or re-filter with correct parameters.

---

## Milestone 7 — Responsive layout tests

### Description

Verify that the new columns and filter dropdown do not break layout at key viewport widths.

### Files to change

- `tests/` — responsive/visual test file

### Test cases

1. **1280 px width**: All columns visible, no horizontal scrollbar.
2. **1024 px width**: Table remains usable; new columns may be hidden per responsive rules.
3. **768 px width**: No horizontal overflow; filter bar wraps correctly.
4. **Filter dropdown accessibility**: Release dropdown has accessible label, is keyboard-navigable (Tab, Enter, Arrow keys).

### Acceptance criteria

- [ ] All 4 test cases pass.
- [ ] No new runtime dependencies introduced by test infrastructure.
