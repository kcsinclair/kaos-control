---
title: Lineage Filter with Autocomplete
type: requirement
status: approved
lineage: lineage-filter-autocomplete
created: "2026-05-06T10:00:00+10:00"
priority: high
parent: lifecycle/ideas/lineage-filter-autocomplete.md
labels:
    - feature
    - frontend
    - artefacts
assignees:
    - role: product-owner
      who: agent
---

# Lineage Filter with Autocomplete

## Problem

Users working with many artifacts across multiple lineages have no efficient way to narrow the artifact list or board view to a single lineage. They must visually scan or rely on browser find, which is slow and does not integrate with existing filter state (status, type). A dedicated lineage filter with autocomplete would let users jump to a specific feature lineage in seconds.

## Goals / Non-goals

### Goals

- Provide a text-input filter control that narrows displayed artifacts by lineage slug.
- Support substring matching (not prefix-only) so any fragment of a slug surfaces results.
- Show an autocomplete dropdown of matching lineage slugs as the user types.
- Work on both the artifact list view and the board view.
- Compose with all other active filters (status, type, search, etc.) using AND logic.

### Non-goals

- Full-text search across artifact body content (out of scope; this filters on the `lineage` frontmatter field only).
- Regex or glob pattern support in the filter input.
- Persisting the lineage filter across page navigations or sessions.
- Backend API changes — the frontend already receives lineage data via existing endpoints.

## Detailed Requirements

### Functional

1. **Filter control placement** — A text input labelled "Filter by lineage" must appear in the toolbar/filter bar of both the artifact list view and the board (kanban) view.
2. **Autocomplete dropdown** — After the user types at least 1 character, display a dropdown of lineage slugs whose value contains the typed substring (case-insensitive). The dropdown must list at most 10 suggestions, ordered alphabetically.
3. **Substring matching** — Matching must be performed against the full lineage slug using case-insensitive substring containment (e.g., typing `filter` matches `lineage-filter-autocomplete` and `status-filter`).
4. **Selection behaviour** — Clicking a dropdown suggestion or pressing Enter/Tab on a highlighted suggestion populates the input with that slug and applies the filter immediately.
5. **Free-text submission** — Pressing Enter when no suggestion is highlighted applies the current input text as the filter value (substring match against `lineage` field).
6. **Clearing** — A clear button (×) inside the input resets the filter and restores the unfiltered artifact set (respecting other active filters).
7. **Composition with other filters** — The lineage filter must compose with existing status, type, and any future filters via AND logic. Artifacts are shown only if they pass all active filter predicates.
8. **Empty state** — When the applied filter matches zero artifacts, display an inline message such as "No artifacts match lineage ‹value›" in the list/board area.
9. **Slug source** — The set of available lineage slugs for autocomplete must be derived from the indexed artifacts in the current project (i.e., distinct `lineage` values from all artifacts returned by the API or already loaded in the store).

### Non-functional

1. **Performance** — Autocomplete filtering must feel instant (<50 ms perceived latency) for projects with up to 500 distinct lineage slugs.
2. **Accessibility** — The autocomplete must be keyboard-navigable (arrow keys, Enter, Escape to dismiss) and expose appropriate ARIA roles (`combobox`, `listbox`, `option`).
3. **Responsiveness** — The filter input must not overflow or break layout on viewports ≥ 360 px wide.
4. **Debounce** — Input changes should be debounced (150–300 ms) before filtering the autocomplete list, to avoid excessive re-renders during fast typing.

## Acceptance Criteria

- [ ] Artifact list view displays a "Filter by lineage" text input in the filter bar.
- [ ] Board view displays the same filter input in its toolbar.
- [ ] Typing a substring shows an autocomplete dropdown with matching lineage slugs (case-insensitive, max 10 results).
- [ ] Selecting a suggestion filters artifacts to only those with that exact lineage slug.
- [ ] Submitting free text filters artifacts to those whose lineage contains the entered substring.
- [ ] The filter composes with status and type filters (AND logic).
- [ ] Clearing the input removes the lineage filter without affecting other active filters.
- [ ] An empty-state message is shown when no artifacts match.
- [ ] Autocomplete is keyboard-navigable (arrows, Enter, Escape).
- [ ] ARIA attributes are present (`role="combobox"`, `role="listbox"`, `role="option"`).
- [ ] No perceptible lag on autocomplete with ≤ 500 lineage slugs.

## Questions

- Should the filter support multiple lineage selections (OR within lineage, AND with other filters), or is single-lineage sufficient for the initial implementation?

> Single lineage is sufficient.

- Should the autocomplete dropdown show a count of artifacts per lineage slug to help users gauge result size?

> Yes please.
