---
title: 'Frontend Plan: Artefacts List Release & Priority Columns'
type: plan-frontend
status: approved
lineage: artefacts-list-release-priority-columns
priority: high
parent: requirements/artefacts-list-release-priority-columns-2.md
release: May2026
---

# Frontend Plan: Artefacts List Release & Priority Columns

## Summary

Add Release and Priority columns to the artifact list table in `ArtifactListView.vue`, integrate them with the existing sortable-table mechanism, and port the release filter dropdown from the Board view. The backend already serves all necessary data — this is a purely frontend change.

---

## Milestone 1 — Add Priority and Release columns to the table

### Description

Extend the artifact list table with two new columns. Priority displays coloured pills (matching status pill styling) with severity-appropriate colours. Release displays the release name as plain text. Both show a dash (`—`) when the field is absent or empty.

### Files to change

- `web/src/views/project/ArtifactListView.vue`
  - Add `<SortHeader>` entries for "Priority" and "Release" in the table header row
  - Add `<td>` cells in the row template rendering `item.frontmatter.priority` and `item.frontmatter.release`
  - Priority cell: render as a coloured pill/badge (`<span class="priority-pill priority-{value}">`)
  - Release cell: plain text with `—` fallback
  - Column position: Priority after Status, Release after Priority

- `web/src/views/project/ArtifactListView.vue` (scoped styles or new utility class)
  - Add `.priority-pill` styles with colour variants for `critical` (red), `high` (orange), `normal` (blue), `low` (grey)
  - Match the sizing and border-radius of existing status pills

### Acceptance criteria

- [ ] Priority column displays after Status, Release column displays after Priority.
- [ ] Priority values render as coloured pills: critical=red, high=orange, normal=blue, low=grey.
- [ ] Missing priority shows `—` (no pill).
- [ ] Release values render as plain text; missing release shows `—`.
- [ ] Table does not horizontally overflow at 1280 px viewport width.
- [ ] On narrow viewports, new columns follow existing responsive table behaviour (hidden/truncated).

---

## Milestone 2 — Integrate columns with sortable-table mechanism

### Description

Register the two new columns in the `useSortableTable` column definitions. Priority requires custom sort order (not alphabetical). Release uses standard string sort with null-last behaviour.

### Files to change

- `web/src/views/project/ArtifactListView.vue`
  - Extend the `useSortableTable` column config object:
    ```ts
    priority: {
      type: 'number',
      getValue: (row) => priorityOrder(row.frontmatter?.priority),
    },
    release: {
      type: 'string',
      getValue: (row) => row.frontmatter?.release ?? '',
    },
    ```
  - Add a `priorityOrder` helper function mapping: `critical → 4, high → 3, normal → 2, low → 1, '' → 0`

- `web/src/composables/useSortableTable.ts` — no changes expected; the existing `getValue` + type system should handle this. Verify null-last behaviour for release.

### Acceptance criteria

- [ ] Clicking Priority header cycles ascending/descending/unsorted; descending shows critical first.
- [ ] Priority sort order: critical > high > normal > low > (empty).
- [ ] Clicking Release header sorts alphabetically (case-insensitive); artifacts with no release sort last.
- [ ] Sort indicators (arrows) display correctly on new column headers.
- [ ] Existing column sorting is unaffected.

---

## Milestone 3 — Add release filter dropdown to List view toolbar

### Description

Port the release filter from `KanbanBoardView.vue` to `ArtifactListView.vue`. The filter populates from `releasesStore.releases`, includes an "All Releases" default and an "Unassigned" option, and composes with existing filters.

### Files to change

- `web/src/views/project/ArtifactListView.vue`
  - Import and initialise `useReleasesStore`
  - Add `const selectedRelease = ref('')` to filter state
  - Add `<select>` dropdown in the filter bar (after existing filter controls), matching existing dropdown styling:
    ```html
    <select v-model="selectedRelease" @change="applyFilters">
      <option value="">All Releases</option>
      <option v-for="r in releasesStore.releases" :key="r.id" :value="r.name">{{ r.name }}</option>
      <option value="__unassigned__">Unassigned</option>
    </select>
    ```
  - Include `release: selectedRelease.value || undefined` in the `applyFilters()` store call
  - Reset `selectedRelease` in the reset-filters function
  - Fetch releases in `onMounted`: add `releasesStore.fetch(project)` to the `Promise.all`

- `web/src/api/artifacts.ts` (see [[artefacts-list-release-priority-columns]] backend plan Milestone 2)
  - Add `if (f.release) p.set('release', f.release)` to `filterParams()`

### Acceptance criteria

- [ ] Release filter dropdown is visible in the List view toolbar.
- [ ] Dropdown lists all distinct release names from the releases store, plus "All Releases" and "Unassigned".
- [ ] Selecting a release narrows the table to matching artifacts.
- [ ] Selecting "Unassigned" shows only artifacts with no release value.
- [ ] Selecting "All Releases" clears the release filter.
- [ ] Release filter composes (AND) with stage, status, type, label, priority, and text search filters.
- [ ] Changing the release filter resets the active sort to default order (via `resetSort()`).
- [ ] The reset-filters button clears the release filter along with all other filters.
- [ ] Filter dropdown is keyboard-navigable and has an accessible label.

---

## Milestone 4 — Responsive and visual polish

### Description

Verify that the two new columns and the filter dropdown render correctly across viewport sizes and do not break existing layout. Adjust column widths, truncation, and filter bar wrapping as needed.

### Files to change

- `web/src/views/project/ArtifactListView.vue` (styles)
  - Set appropriate `min-width` / `max-width` on new columns
  - Ensure filter bar wraps gracefully with the additional dropdown
  - Verify no horizontal scrollbar appears at 1280 px

### Acceptance criteria

- [ ] At 1280 px width: all columns visible, no horizontal overflow.
- [ ] At < 1024 px width: new columns hidden or truncated per existing responsive pattern.
- [ ] Filter bar wraps cleanly with the additional release dropdown.
- [ ] Priority pills are legible at all supported sizes.
- [ ] No new runtime dependencies introduced.
