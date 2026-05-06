---
title: "Universal Text Filter — Frontend Plan"
type: plan-frontend
status: draft
lineage: universal-text-filter
parent: lifecycle/requirements/universal-text-filter-2.md
---

# Universal Text Filter — Frontend Plan

This plan covers all frontend changes required for the universal text filter feature ([[universal-text-filter]]). The work spans a new reusable component, integration into four views, store updates, and keyboard shortcut handling.

## Milestone 1 — Create the `TextFilter` component

### Description

Build a reusable `TextFilter.vue` component that renders a single-line text input with a search icon (lucide `Search`), a clear button (lucide `X`), and emits debounced change events. This component will be placed on every data view.

### Files to change

- `web/src/components/TextFilter.vue` (new file)

### Implementation detail

Props:
- `modelValue: string` — v-model binding for the search text.
- `placeholder?: string` — defaults to `"Filter by text…"`.
- `debounceMs?: number` — defaults to `200`.

Emits:
- `update:modelValue` — debounced, fires after `debounceMs` of inactivity.

Behaviour:
- Show `Search` icon on the left (lucide-vue-next).
- Show `X` clear button on the right when `modelValue` is non-empty; clicking it emits `""` immediately (no debounce on clear).
- Clear button must have `aria-label="Clear filter"` and be keyboard-accessible.
- Input must have `aria-label="Filter artifacts by text"`.
- The component does NOT own focus management for the `/` shortcut — that is handled at the view level (Milestone 6).
- On small viewports (≤ 480px), the input collapses behind a search icon toggle button; tapping it expands the full input.

### Acceptance criteria

- [ ] Component renders a text input with search icon and clear button.
- [ ] `v-model` binding works with debounced emission at ~200 ms.
- [ ] Clicking the clear button immediately resets the value.
- [ ] `aria-label` attributes are present on both the input and the clear button.
- [ ] On viewports ≤ 480px the input collapses behind a toggle icon.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 2 — Integrate TextFilter into the Artifact List view

### Description

Add the `TextFilter` component to the Artifact List view's filter bar and wire it to the artifacts store so that typing sends the `q` parameter to the backend ([[universal-text-filter]] backend plan, Milestone 2). Highlight matched substrings in the title column.

### Files to change

- `web/src/views/project/ArtifactListView.vue` — add `TextFilter` to the filter bar; bind its value to a reactive `searchText` ref; pass `searchText` as filter `q` to `store.fetchList()`; reset pagination to page 1 on change.
- `web/src/stores/artifacts.ts` — add optional `q?: string` to the filter interface; pass it as a query parameter in `fetchList()`.
- `web/src/views/project/ArtifactListView.vue` (or a small helper) — add a `highlightMatch` function that wraps matched substrings in `<mark>` tags for the title column cell.

### Acceptance criteria

- [ ] TextFilter appears in the filter bar, visually aligned with existing dropdowns.
- [ ] Typing a search string filters the artifact table in real time via the backend `q` parameter.
- [ ] Changing the text resets pagination to page 1.
- [ ] Text filter composes with existing dropdown filters (AND logic) — enabling both narrows results.
- [ ] Clearing the text input restores the full (dropdown-filtered) result set.
- [ ] Matched substrings in the title column are highlighted with `<mark>`.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 3 — Integrate TextFilter into the Kanban Board view

### Description

Add the `TextFilter` component to the Kanban Board view. Filtering is client-side: cards whose `title`, `lineage`, `type`, or `status` do not contain the search text (case-insensitive substring) are hidden.

### Files to change

- `web/src/views/project/KanbanBoardView.vue` — add `TextFilter` to the filter area; add a computed that filters cards within each column by the search text.
- `web/src/composables/useKanbanBoard.ts` (if filtering logic lives here) — accept an optional `searchText` parameter and apply client-side substring filtering after existing dropdown filters.

### Acceptance criteria

- [ ] TextFilter appears on the Kanban view, visually consistent with the list view placement.
- [ ] Cards not matching the text filter are hidden (not dimmed).
- [ ] Empty columns after filtering show a "No matching items" indicator.
- [ ] Text filter composes with dropdown filters using AND logic.
- [ ] Clearing the text input restores all cards (subject to dropdown filters).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 4 — Integrate TextFilter into the Graph view

### Description

Add the `TextFilter` component to the Graph view's filter panel. Non-matching nodes are dimmed (reduced opacity), not removed. Matched nodes retain full opacity and receive a highlight outline. Edges follow node visibility rules. When matches exist, the camera re-centres on the first matched node using the `focusNode` pattern from the requirement's OQ-1 answer.

### Files to change

- `web/src/views/project/GraphView.vue` — add `TextFilter` to the graph filter sidebar/toolbar.
- `web/src/stores/graph.ts` — add a `searchText` ref to the store; add a computed `matchedNodeIds: Set<string>` that tests each node's title, lineage, type, and status against `searchText`.
- `web/src/components/ForceGraph3D.vue` — use `matchedNodeIds` to set node opacity (full for matched, ~0.15 for unmatched); apply a highlight ring/outline to matched nodes; set edge opacity based on whether at least one endpoint is matched. On search text change, animate the camera to the centroid of matched nodes (or the first matched node if only one).
- `web/src/components/Graph2DView.vue` — apply equivalent opacity/highlight styling via Cytoscape style rules; fit the viewport to matched nodes on change.

### Acceptance criteria

- [ ] TextFilter appears on the Graph view alongside existing graph filters.
- [ ] Non-matching nodes are dimmed to ~0.15 opacity; matched nodes retain full opacity.
- [ ] Matched nodes have a visible highlight outline or ring.
- [ ] Edges between two dimmed nodes are dimmed; edges touching at least one matched node are visible.
- [ ] Camera re-centres on matched nodes when filter text changes (3D: animated fly-to; 2D: viewport fit).
- [ ] Clearing the text restores all nodes to full opacity and removes highlight outlines.
- [ ] Text filter composes with existing graph filters using AND logic.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 5 — Integrate TextFilter into the Project Feed view

### Description

Add the `TextFilter` component to the Project Feed view. Filtering is client-side: feed entries whose displayed text (artifact title, event summary) does not contain the search text are hidden.

### Files to change

- `web/src/views/project/ProjectFeedView.vue` — add `TextFilter` to the feed filter bar; add a computed that filters `events` by substring match on `summary` and `artifact_path` (title is derived from path) fields.

### Acceptance criteria

- [ ] TextFilter appears on the Project Feed view, visually consistent with other views.
- [ ] Feed entries not matching the text filter are hidden.
- [ ] Text filter composes with existing feed type toggles using AND logic.
- [ ] Clearing the text input restores all entries (subject to type toggles).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 6 — Keyboard shortcut (`/` to focus, `Escape` to clear)

### Description

Add a global keyboard listener on each view that focuses the `TextFilter` input when the user presses `/` (only when no other input/textarea is focused). Pressing `Escape` while the filter is focused clears its value and blurs the input.

### Files to change

- `web/src/components/TextFilter.vue` — expose a `focus()` method via `defineExpose`; add an internal `onKeydown` handler for `Escape` (clear value, blur).
- `web/src/composables/useTextFilterShortcut.ts` (new file) — a composable that accepts a template ref to a `TextFilter` instance, registers a `keydown` listener for `/` on `document`, and calls `focus()` when appropriate. Cleans up on unmount.
- Each view file (`ArtifactListView.vue`, `KanbanBoardView.vue`, `GraphView.vue`, `ProjectFeedView.vue`) — use the composable with a ref to their `TextFilter` instance.

### Acceptance criteria

- [ ] Pressing `/` when no input is focused moves focus to the TextFilter input on every view.
- [ ] Pressing `/` while typing in another input (e.g. the CodeMirror editor) does NOT steal focus.
- [ ] Pressing `Escape` while the filter is focused clears the value and blurs the input.
- [ ] The shortcut listener is cleaned up on component unmount (no leaks on route change).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Notes

- Filter state is NOT persisted in the URL or across route changes (NG2). The `searchText` ref resets naturally when the view component unmounts.
- Performance: client-side filtering (Kanban, Graph, Feed) uses `String.prototype.includes()` on lowercased strings, which is O(n) and well within the < 16 ms budget for 500 artifacts.
- The `<mark>` highlight in the list view must not rely solely on colour (NFR-2); the default `<mark>` element provides a visible background change that satisfies this.
