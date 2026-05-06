---
title: 'Test Plan: Lineage Filter with Autocomplete'
type: plan-test
status: approved
lineage: lineage-filter-autocomplete
parent: lifecycle/requirements/lineage-filter-autocomplete-2.md
---

# Test Plan: Lineage Filter with Autocomplete

## Summary

End-to-end and integration tests validating the lineage filter autocomplete feature across both the artifact list view and board view. Tests cover autocomplete behaviour, filter composition, keyboard accessibility, and edge cases.

This plan exercises the components built in the [[lineage-filter-autocomplete]] frontend plan and the API surface documented in the [[lineage-filter-autocomplete]] backend plan.

---

## Milestone 1: Backend integration tests — lineages endpoint

### Description

Verify the `/lineages` endpoint returns correct data including the new `total` field, and that the `lineage` query parameter on `/artifacts` filters correctly.

### Files to create/change

- `tests/lineages_api_test.go` (new or extend existing)

### Test cases

1. **Lineages list returns all distinct slugs** — seed 3 lineages with varying artifact counts; assert response contains all 3 with correct `total` values.
2. **Lineages list includes statuses breakdown** — assert `statuses` map sums to `total` for each lineage.
3. **Artifact list filtered by exact lineage** — `GET /artifacts?lineage=my-feature` returns only artifacts with that lineage.
4. **Lineage filter composes with status filter** — `GET /artifacts?lineage=my-feature&status=draft` returns intersection.
5. **Empty lineage filter returns all** — omitting the `lineage` param returns unfiltered results.

### Acceptance Criteria

- [ ] All 5 test cases pass.
- [ ] Tests use real SQLite index (no mocks).
- [ ] Tests run in < 5 seconds.

---

## Milestone 2: Frontend unit tests — `LineageAutocomplete` component

### Description

Unit test the autocomplete component in isolation using Vue Test Utils, verifying rendering, filtering logic, keyboard interaction, and ARIA attributes.

### Files to create

- `web/src/components/common/__tests__/LineageAutocomplete.spec.ts`

### Test cases

1. **Renders input with placeholder** — mount component, assert input has `placeholder="Filter by lineage"`.
2. **No dropdown on empty input** — assert dropdown `ul` not rendered when input is empty.
3. **Shows suggestions on input** — type "filter", assert dropdown appears with matching slugs.
4. **Case-insensitive matching** — type "FILTER", assert same results as lowercase.
5. **Max 10 suggestions** — provide 15 options all matching; assert only 10 rendered.
6. **Alphabetical ordering** — assert suggestions sorted A-Z by slug.
7. **Shows artifact count** — assert each suggestion item contains `(<count>)`.
8. **Click suggestion emits value** — click a suggestion; assert `update:modelValue` emitted with that slug.
9. **Arrow key navigation** — press ArrowDown twice; assert third item highlighted (index 1 → visual highlight).
10. **Enter selects highlighted** — highlight item via ArrowDown, press Enter; assert emission.
11. **Enter submits free text when nothing highlighted** — type text, press Enter without navigating; assert emission of raw text.
12. **Escape closes dropdown** — open dropdown, press Escape; assert dropdown hidden.
13. **Clear button emits clear** — set value, click ×; assert `clear` emitted and input emptied.
14. **ARIA combobox role** — assert `role="combobox"` on input wrapper, `aria-expanded` toggles.
15. **ARIA listbox and option roles** — assert dropdown has `role="listbox"`, items have `role="option"`.
16. **Debounce** — type rapidly; assert filtering function called only after debounce period (use fake timers).

### Acceptance Criteria

- [ ] All 16 test cases pass.
- [ ] Tests use `vi.useFakeTimers()` for debounce tests.
- [ ] No external API calls in unit tests.

---

## Milestone 3: Integration tests — list view filter behaviour

### Description

Test that the lineage filter integrates correctly with `ArtifactListView`, composing with other filters and updating the displayed artifact list.

### Files to create

- `tests/lineage_filter_list_test.go` or `web/src/views/project/__tests__/ArtifactListView.lineage.spec.ts` (browser-level or component-level as appropriate to existing test patterns)

### Test cases

1. **Filter bar contains lineage input** — assert `LineageAutocomplete` rendered in `.filter-bar`.
2. **Selecting a lineage filters artifacts** — select "my-feature" from autocomplete; assert only artifacts with `lineage === "my-feature"` are shown.
3. **Free-text substring filter** — type "feat"; assert artifacts whose lineage contains "feat" are shown.
4. **Composition with status filter** — set status=draft AND lineage="my-feature"; assert intersection displayed.
5. **Composition with type filter** — set type=idea AND lineage="my-feature"; assert intersection.
6. **Clear restores unfiltered state** — apply lineage filter, then clear; assert all artifacts shown (respecting other active filters).
7. **Reset button clears lineage** — apply lineage filter, click Reset; assert lineage input empty and all artifacts shown.
8. **Empty state message** — filter to a lineage with no matches; assert "No artifacts match lineage" message displayed.
9. **WebSocket refresh preserves filter** — simulate `artifact.indexed` event; assert lineage filter remains active and list re-fetches with it applied.

### Acceptance Criteria

- [ ] All 9 test cases pass.
- [ ] Tests validate DOM state after filter application.
- [ ] No flaky timing issues (proper use of `nextTick` / `waitFor`).

---

## Milestone 4: Integration tests — board view filter behaviour

### Description

Mirror Milestone 3 tests for the `KanbanBoardView`.

### Files to create

- `tests/lineage_filter_board_test.go` or `web/src/views/project/__tests__/KanbanBoardView.lineage.spec.ts`

### Test cases

1. **Board toolbar contains lineage input** — assert `LineageAutocomplete` rendered.
2. **Selecting a lineage filters board cards** — select slug; assert only matching cards visible across columns.
3. **Free-text substring filter on board** — type substring; assert matching cards only.
4. **Composition with other board filters** — combine with stage/type; assert AND logic.
5. **Clear restores full board** — clear lineage; assert all cards return.
6. **Reset clears lineage on board** — click Reset; assert lineage input empty.

### Acceptance Criteria

- [ ] All 6 test cases pass.
- [ ] Board columns correctly reflect filtered state (empty columns may be hidden or show zero cards per existing behaviour).

---

## Milestone 5: Accessibility and performance tests

### Description

Validate keyboard-only operation and performance with large datasets.

### Files to create/change

- `web/src/components/common/__tests__/LineageAutocomplete.a11y.spec.ts`

### Test cases

1. **Full keyboard workflow** — Tab into input, type, ArrowDown to highlight, Enter to select, Tab to next filter — all without mouse.
2. **Screen reader announcement** — assert `aria-activedescendant` updates on navigation, `aria-selected="true"` on highlighted option.
3. **Performance: 500 slugs** — mount component with 500 options; type a character; assert dropdown renders within 50 ms (measure via `performance.now()`).
4. **Performance: rapid typing** — type 10 characters quickly; assert only 1-2 filter computations occur (debounce working).

### Acceptance Criteria

- [ ] All 4 test cases pass.
- [ ] No accessibility violations detectable by automated tools (axe-core if available in test harness).
- [ ] Autocomplete remains responsive with 500 options.
