---
title: "Frontend Plan: Lineage Filter with Autocomplete"
type: plan-frontend
status: draft
lineage: lineage-filter-autocomplete
parent: lifecycle/requirements/lineage-filter-autocomplete-2.md
---

# Frontend Plan: Lineage Filter with Autocomplete

## Summary

Implement a reusable autocomplete text input component that filters artifacts by lineage slug. The component will appear in the filter bar of both `ArtifactListView` and `KanbanBoardView`, compose with existing filters via AND logic, and source its suggestions from the existing `listLineages` API. Each suggestion displays the lineage slug and an artifact count.

This plan depends on the [[lineage-filter-autocomplete]] backend plan's `total` field enhancement (Milestone 3). If that field is unavailable, the frontend will sum `statuses` values as a fallback.

---

## Milestone 1: Create `LineageAutocomplete` component

### Description

Build a standalone, accessible autocomplete input component that accepts a list of lineage options (slug + count), performs case-insensitive substring matching, and emits the selected/entered value.

### Files to create

- `web/src/components/common/LineageAutocomplete.vue`

### Implementation details

- Props: `options: Array<{ slug: string; count: number }>`, `modelValue: string`, `placeholder?: string`
- Emits: `update:modelValue` (on selection or Enter), `clear` (on × click or Escape with empty input)
- Internal state: `query` (input text), `highlightedIndex`, `isOpen` (dropdown visibility)
- Filtering logic: case-insensitive substring match on `slug`; max 10 results; sorted alphabetically
- Debounce input changes by 200 ms before computing suggestions (use a local `setTimeout`/`clearTimeout` or a `watchDebounced` from VueUse if available)
- Dropdown item format: `<slug> (<count>)` e.g. `lineage-filter-autocomplete (5)`
- Keyboard navigation: ArrowDown/ArrowUp move highlight, Enter selects highlighted or submits free text, Escape closes dropdown, Tab selects highlighted if open
- Clear button (×) visible when input is non-empty; clicking it emits `clear`
- ARIA: input has `role="combobox"`, `aria-expanded`, `aria-controls`; dropdown `ul` has `role="listbox"`, `id` matching `aria-controls`; each `li` has `role="option"`, `aria-selected`

### Acceptance Criteria

- [ ] Component renders a text input with placeholder "Filter by lineage".
- [ ] Typing 1+ characters shows a dropdown of matching slugs (max 10, alphabetical).
- [ ] Each suggestion shows slug and artifact count.
- [ ] Arrow keys navigate suggestions; Enter/Tab selects; Escape dismisses.
- [ ] Clear button resets input and emits `clear`.
- [ ] All required ARIA attributes present (`combobox`, `listbox`, `option`).
- [ ] Debounce of 200 ms on input before filtering.

---

## Milestone 2: Add lineage data to the artifacts store

### Description

Fetch and expose the list of distinct lineage slugs (with counts) so both views can feed it to the autocomplete component.

### Files to change

- `web/src/stores/artifacts.ts` — add `lineages` ref and `fetchLineages()` action that calls `listLineages(project)`.

### Implementation details

- `lineages` ref type: `Array<{ slug: string; count: number }>`, derived by mapping `LineageSummary[]` — use `total` field if present, otherwise sum values of `statuses`.
- `fetchLineages` called alongside `fetchLabels`/`fetchPriorities` in both views' `onMounted`.

### Acceptance Criteria

- [ ] `store.lineages` populated on mount with slug + count pairs.
- [ ] Data stays current — re-fetched on `artifact.indexed` WebSocket event alongside existing invalidation.

---

## Milestone 3: Integrate into `ArtifactListView`

### Description

Add the `LineageAutocomplete` component to the filter bar and wire it into the existing filter logic.

### Files to change

- `web/src/views/project/ArtifactListView.vue`

### Implementation details

- Import and place `<LineageAutocomplete>` after the existing `<select>` filters in `.filter-bar`.
- Bind `v-model` to a new `selectedLineage` ref (string, default `''`).
- On `update:modelValue`: set `selectedLineage`, call `applyFilters()` which now passes `lineage: selectedLineage.value || undefined` to `store.fetchList`.
- On `clear`: reset `selectedLineage` to `''`, call `applyFilters()`.
- `resetFilters()` also resets `selectedLineage`.
- The `visibleItems` computed and client-side filtering remain unchanged — the lineage filter is applied server-side via the existing `lineage` query param on the artifacts endpoint.
- If the server does exact match and user typed free-text substring, add a client-side post-filter on `visibleItems` that checks `row.lineage.toLowerCase().includes(selectedLineage.value.toLowerCase())`.
- Empty state: the existing "No artifacts found." message already covers the zero-match case; update text to "No artifacts match lineage ‹value›" when `selectedLineage` is active and results are empty.

### Acceptance Criteria

- [ ] Lineage autocomplete input appears in the filter bar of the list view.
- [ ] Selecting a suggestion filters the list to that exact lineage.
- [ ] Free-text submission filters by substring.
- [ ] Filter composes with stage, status, type, label, priority (AND logic).
- [ ] Clearing the lineage filter restores full list (respecting other filters).
- [ ] Empty state message mentions the lineage value when applicable.
- [ ] `resetFilters` clears the lineage input.

---

## Milestone 4: Integrate into `KanbanBoardView`

### Description

Add the same `LineageAutocomplete` to the board view's filter toolbar.

### Files to change

- `web/src/views/project/KanbanBoardView.vue`
- `web/src/composables/useKanbanBoard.ts` (if `applyFilters` needs a `lineage` param)

### Implementation details

- Mirror Milestone 3 integration pattern: `selectedLineage` ref, pass to `applyFilters({ ..., lineage })`.
- If `useKanbanBoard.applyFilters` doesn't currently accept `lineage`, extend its filter type to include it.
- Same empty-state and reset behaviour as list view.

### Acceptance Criteria

- [ ] Lineage autocomplete input appears in the board view toolbar.
- [ ] Filtering by lineage narrows board cards to matching artifacts.
- [ ] Composes with other active board filters.
- [ ] Reset clears lineage filter.

---

## Milestone 5: Responsive layout and polish

### Description

Ensure the autocomplete doesn't break layout on small viewports and visually matches the existing filter bar style.

### Files to change

- `web/src/components/common/LineageAutocomplete.vue` (scoped styles)
- `web/src/views/project/ArtifactListView.vue` (minor style adjustments if needed)

### Implementation details

- Input width: `min-width: 160px; max-width: 260px; flex: 1` to fit alongside existing selects.
- Dropdown absolutely positioned below input, `max-height: 240px; overflow-y: auto`.
- On viewports < 480px, input takes full row (leveraging existing `flex-wrap: wrap` on `.filter-bar`).
- Match existing filter bar font-size (`var(--text-sm)`), border style, and spacing.

### Acceptance Criteria

- [ ] No layout overflow on viewports ≥ 360px wide.
- [ ] Dropdown doesn't extend beyond viewport bounds.
- [ ] Visual style consistent with adjacent select filters.
- [ ] No perceptible lag with ≤ 500 lineage slugs in the autocomplete list.
