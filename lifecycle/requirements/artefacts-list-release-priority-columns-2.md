---
title: 'Artefacts List: Release & Priority Columns with Releases Filter'
type: requirement
status: blocked
lineage: artefacts-list-release-priority-columns
created: "2026-05-07T00:00:00+10:00"
priority: normal
parent: ideas/artefacts-list-release-priority-columns.md
labels:
    - artefacts
    - frontend
    - enhancement
    - releases
assignees:
    - role: product-owner
      who: agent
---

# Artefacts List: Release & Priority Columns with Releases Filter

## Problem

The artifact list view displays rows without any indication of which release an artifact belongs to or what its priority level is. Users must open individual artifacts to discover this information, making it impossible to scan release scope or triage priorities from the list. Additionally, the Board view already supports a release filter dropdown, but the List view does not — forcing users who prefer the tabular layout to switch views just to narrow results by release.

## Goals / Non-goals

### Goals

- Add a **Release** column to the artifact list table showing each artifact's `release` frontmatter value.
- Add a **Priority** column to the artifact list table showing each artifact's `priority` frontmatter value.
- Port the existing release filter (already present in Board view) to the List view so users can narrow results to a specific release.
- Ensure the new columns participate in the existing sortable-table-columns mechanism.
- Maintain visual consistency with existing columns and filter controls.

### Non-goals

- Adding a priority filter (only the release filter is in scope).
- Inline editing of release or priority values from the list view.
- Changes to the Board view or any other view.
- Adding new API endpoints — the release and priority fields are already present in indexed artifact data.

## Detailed Requirements

### Functional

1. **Release column** — A new column labelled "Release" must be added to the artifact list table. It displays the value of the artifact's `release` frontmatter field. If the field is absent or empty, the cell displays a dash (`—`) or is left blank.

2. **Priority column** — A new column labelled "Priority" must be added to the artifact list table. It displays the value of the artifact's `priority` frontmatter field (e.g. `low`, `normal`, `high`, `critical`). If absent, display a dash (`—`) or leave blank.

3. **Column position** — The Priority column should appear after the Status column. The Release column should appear after Priority. Exact ordering may be adjusted during implementation if it reads better, but both columns must be grouped near Status for scannability.

4. **Sorting support** — Both new columns must integrate with the existing sortable column mechanism ([[sortable-table-columns]]):
   - Priority sorts by logical severity order: `critical > high > normal > low` (descending = most critical first).
   - Release sorts alphabetically (lexicographic, case-insensitive). Artifacts with no release sort last.

5. **Release filter** — A dropdown filter control (matching the style and placement of existing filters in the List view toolbar) must be added that lists all distinct `release` values present in the current dataset. Selecting a release narrows the table to artifacts matching that release. An "All" or empty option clears the filter.

6. **Filter population** — The release filter options must be derived dynamically from the loaded artifact data (not hard-coded). If no artifacts have a `release` value, the filter dropdown should still render but show only the "All" option.

7. **Interaction with existing filters** — The release filter composes with any other active filters (e.g. status, type, lineage). Filters are AND-combined: an artifact must satisfy all active filters to appear.

8. **Interaction with sorting** — Changing the release filter resets the sort state to default order (consistent with existing filter-sort interaction defined in [[sortable-table-columns]]).

9. **Empty-state consistency** — When no artifacts match the combined filters (including the new release filter), the existing "no results" empty state should display.

### Non-functional

1. **Performance** — Adding two columns and one filter must not degrade list rendering for tables up to 1 000 rows beyond the existing performance budget (sort + render < 100 ms).

2. **Responsive layout** — On narrow viewports, the new columns may be hidden or truncated per existing responsive table behaviour. They must not cause horizontal overflow on standard desktop widths (≥ 1280 px).

3. **No new dependencies** — Implementation must use existing UI primitives (filter dropdowns, table column definitions) without introducing new libraries.

4. **Accessibility** — Filter dropdown must be keyboard-navigable and labelled for screen readers. Column headers follow existing accessible sort-header pattern.

## Acceptance Criteria

- [ ] The artifact list table displays a "Priority" column showing each artifact's priority value.
- [ ] The artifact list table displays a "Release" column showing each artifact's release value.
- [ ] Artifacts without a `release` value show a dash or blank in the Release column.
- [ ] Artifacts without a `priority` value show a dash or blank in the Priority column.
- [ ] Clicking the Priority column header sorts by logical severity order (critical → high → normal → low), not alphabetically.
- [ ] Clicking the Release column header sorts alphabetically; artifacts with no release sort last.
- [ ] A release filter dropdown is present in the List view toolbar.
- [ ] The release filter lists all distinct release values from the current dataset dynamically.
- [ ] Selecting a release in the filter narrows the table to only matching artifacts.
- [ ] The release filter composes (AND) with other active filters.
- [ ] Changing the release filter resets the active sort to default order.
- [ ] The release filter and new columns do not introduce horizontal overflow at 1280 px viewport width.
- [ ] No new runtime dependencies are added.

## Open Questions

1. Should the Priority column use coloured badges or icons (similar to status pills) to improve scannability, or plain text?
2. Should the Release column link to a release artifact (if one exists), or remain plain text?
3. Is there a defined ordering for releases beyond alphabetical (e.g. chronological by target date) that should be used for sorting?
