---
title: Universal Text Filter — Test Plan
type: plan-test
status: approved
lineage: universal-text-filter
parent: lifecycle/requirements/universal-text-filter-2.md
---

# Universal Text Filter — Test Plan

This plan covers integration tests for the universal text filter feature ([[universal-text-filter]]). Tests exercise the backend API (`q` parameter) and the frontend UI (text input, filtering, highlighting, keyboard shortcuts) across all four views. Tests are written in the `tests/` directory and target the running application.

## Milestone 1 — Backend API text filter tests

### Description

Write integration tests that verify the `GET /artifacts` endpoint correctly filters by the `q` query parameter, composes with other filters, and handles edge cases.

### Files to change

- `tests/universal_text_filter_api_test.go` (new file)

### Test cases

1. **Basic substring match** — Seed artifacts with known titles. Send `GET /artifacts?q=<substring>`. Assert only matching artifacts are returned and `total` reflects the filtered count.
2. **Case insensitivity** — Seed an artifact titled "Kanban View". Send `q=kanban`. Assert the artifact is returned.
3. **Matches on slug** — Seed an artifact with slug `kanban-view`. Send `q=kanban-view`. Assert it is returned.
4. **Matches on lineage** — Send `q=<lineage-slug>`. Assert all artifacts in that lineage are returned.
5. **Matches on type** — Send `q=requirement`. Assert artifacts of type `requirement` are returned.
6. **Matches on status** — Send `q=draft`. Assert artifacts with status `draft` are returned.
7. **Composition with dropdown filters** — Send `q=kanban&status=draft`. Assert only artifacts matching both conditions are returned.
8. **No matches** — Send `q=zzz_nonexistent_zzz`. Assert an empty `items` array and `total: 0`.
9. **Empty q** — Send `q=` (empty string). Assert all artifacts are returned (same as no `q` parameter).
10. **Special characters** — Send `q=100%25` (URL-encoded `%`). Assert no SQL injection or LIKE-wildcard issues; only literal matches returned.
11. **Pagination reset** — Seed enough artifacts to span multiple pages. Send `q=<term>&offset=0`. Assert results start from the beginning.

### Acceptance criteria

- [ ] All 11 test cases pass against a running instance.
- [ ] Tests clean up any seeded data after completion.
- [ ] Tests are runnable via `go test ./tests/ -run TestUniversalTextFilterAPI`.

---

## Milestone 2 — Artifact List view UI tests

### Description

Write browser-level integration tests (or HTTP + DOM assertion tests, matching the project's existing test approach) that verify the TextFilter component works correctly on the Artifact List view.

### Files to change

- `tests/universal_text_filter_list_test.go` (new file)

### Test cases

1. **Filter input present** — Navigate to the Artifact List view. Assert a text input with `aria-label="Filter artifacts by text"` is present.
2. **Real-time filtering** — Type a known artifact title substring into the filter. Assert the table updates to show only matching rows.
3. **Title highlighting** — Type a search term. Assert the title column of matching rows contains a `<mark>` element wrapping the matched substring.
4. **Clear button** — Type text, then click the clear (×) button. Assert the filter input is empty and the full list is restored.
5. **Composition with dropdowns** — Select a status dropdown value AND type a search string. Assert only artifacts matching both are shown.
6. **Pagination reset** — Navigate to page 2, then type a search term. Assert the view resets to page 1.
7. **Empty results** — Type a string that matches nothing. Assert the table shows an appropriate empty state.

### Acceptance criteria

- [ ] All 7 test cases pass.
- [ ] Tests are runnable via `go test ./tests/ -run TestUniversalTextFilterList`.

---

## Milestone 3 — Kanban Board view UI tests

### Description

Verify the TextFilter works on the Kanban Board view with client-side filtering.

### Files to change

- `tests/universal_text_filter_kanban_test.go` (new file)

### Test cases

1. **Filter input present** — Navigate to the Kanban view. Assert the TextFilter input is present.
2. **Cards hidden on filter** — Type a search term. Assert cards not matching are hidden (not present in the DOM or have `display: none`).
3. **Empty column indicator** — Type a term that removes all cards from at least one column. Assert the column remains visible with a "No matching items" message.
4. **Composition with dropdowns** — Combine text filter with a dropdown filter. Assert AND logic is applied.
5. **Clear restores cards** — Type text, then clear. Assert all cards reappear (subject to dropdown filters).

### Acceptance criteria

- [ ] All 5 test cases pass.
- [ ] Tests are runnable via `go test ./tests/ -run TestUniversalTextFilterKanban`.

---

## Milestone 4 — Graph view UI tests

### Description

Verify the TextFilter works on the Graph view with dimming/highlighting behaviour and camera focus.

### Files to change

- `tests/universal_text_filter_graph_test.go` (new file)

### Test cases

1. **Filter input present** — Navigate to the Graph view. Assert the TextFilter input is present.
2. **Non-matching nodes dimmed** — Type a search term. Assert nodes not matching have reduced opacity (check style or class).
3. **Matching nodes highlighted** — Assert matched nodes retain full opacity and have a highlight class or outline style.
4. **Edge visibility** — Assert edges between two dimmed nodes are dimmed; edges touching a matched node are visible.
5. **Camera focus** — Type a search term matching a single node. Assert the camera animates toward that node (verify position change or animation trigger).
6. **Clear restores** — Clear the filter. Assert all nodes return to full opacity with no highlight outlines.
7. **Composition with graph filters** — Apply a type filter via the graph sidebar AND type text. Assert both are applied (AND logic).

### Acceptance criteria

- [ ] All 7 test cases pass.
- [ ] Tests are runnable via `go test ./tests/ -run TestUniversalTextFilterGraph`.

---

## Milestone 5 — Project Feed view UI tests

### Description

Verify the TextFilter works on the Project Feed view.

### Files to change

- `tests/universal_text_filter_feed_test.go` (new file)

### Test cases

1. **Filter input present** — Navigate to the Project Feed view. Assert the TextFilter input is present.
2. **Entries hidden on filter** — Type a substring of a known event summary. Assert only matching entries are visible.
3. **Composition with type toggles** — Enable only specific feed event types AND type a search string. Assert both filters are applied.
4. **Clear restores entries** — Type text, then clear. Assert all entries reappear (subject to type toggles).

### Acceptance criteria

- [ ] All 4 test cases pass.
- [ ] Tests are runnable via `go test ./tests/ -run TestUniversalTextFilterFeed`.

---

## Milestone 6 — Keyboard shortcut and accessibility tests

### Description

Verify the `/` focus shortcut and `Escape` clear/blur behaviour, as well as accessibility attributes, across all views.

### Files to change

- `tests/universal_text_filter_keyboard_test.go` (new file)

### Test cases

1. **`/` focuses filter** — On each of the four views, press `/` when no input is focused. Assert the TextFilter input gains focus.
2. **`/` does not steal focus** — Focus another input (e.g. the editor), press `/`. Assert the TextFilter does NOT gain focus.
3. **`Escape` clears and blurs** — Focus the filter, type text, press `Escape`. Assert the filter value is empty and the input is no longer focused.
4. **`aria-label` on input** — Assert the filter input has `aria-label="Filter artifacts by text"` on each view.
5. **`aria-label` on clear button** — Assert the clear button has `aria-label="Clear filter"` on each view.
6. **Clear button keyboard-accessible** — Tab to the clear button and press `Enter`. Assert the filter is cleared.

### Acceptance criteria

- [ ] All 6 test cases pass across all four views.
- [ ] Tests are runnable via `go test ./tests/ -run TestUniversalTextFilterKeyboard`.

---

## Milestone 7 — Performance validation

### Description

Verify that client-side filtering meets the < 16 ms budget for datasets of 500 artifacts.

### Files to change

- `tests/universal_text_filter_perf_test.go` (new file)

### Test cases

1. **500-artifact dataset** — Seed 500 artifacts. Navigate to the Kanban view (heaviest client-side filter). Type a search term. Measure the time from input event to render completion. Assert it completes within 16 ms (one animation frame).
2. **Debounce prevents jank** — Type rapidly (simulate 10 characters in < 100 ms). Assert that filtering is invoked at most once after the debounce period, not per-keystroke.

### Acceptance criteria

- [ ] Filtering 500 artifacts completes within 16 ms.
- [ ] Rapid typing does not cause perceptible jank or multiple intermediate renders.
- [ ] Tests are runnable via `go test ./tests/ -run TestUniversalTextFilterPerf`.

---

## Notes

- Test data seeding should use the `POST /artifacts` API endpoint to create artifacts with controlled titles, slugs, and statuses, then clean up via `DELETE /artifacts/*` after each test.
- The [[universal-text-filter]] backend plan (Milestone 3) includes its own unit tests for `buildWhere`. The API tests in this plan's Milestone 1 are integration-level and complement those unit tests.
- Browser-level tests (Milestones 2–7) depend on both the [[universal-text-filter]] backend and frontend plans being implemented first.
