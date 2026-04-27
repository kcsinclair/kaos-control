---
title: "Kanban View — Frontend Plan"
type: plan-frontend
status: draft
lineage: kanban-view
parent: requirements/kanban-view-3.md
---

# Kanban View — Frontend Plan

This plan implements the Kanban board view, navigation restructure, and UI label rename described in [[kanban-view]]. It depends on the backend endpoint from [[kanban-view-4-be]] to fetch parsed kanban configuration. All board logic (grouping, filtering, virtual fields) is client-side, built on the existing artifact list API.

## Milestone 1 — Rename "Artifacts" to "Artefacts" in user-facing text

### Description

Update all user-facing occurrences of "Artifacts" to "Artefacts" in the sidebar, page titles, and headings. Internal code identifiers (component names, store names, API paths, route names) remain unchanged per requirement.

### Files to change

- `web/src/components/layout/AppSidebar.vue` — Change the `label: 'Artifacts'` string in `navItems` to `'Artefacts'`.
- `web/src/views/project/ArtifactListView.vue` — Change the `<h2>` heading text from "Artifacts" to "Artefacts".

### Acceptance criteria

- [ ] The sidebar displays "Artefacts" instead of "Artifacts".
- [ ] The list view heading reads "Artefacts".
- [ ] No route paths or component/store names are changed.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 2 — Restructure sidebar navigation with sub-menu

### Description

Convert the flat "Artefacts" sidebar entry into an expandable item with two sub-entries: "List" (existing `ArtifactListView`) and "Board" (new Kanban view). The sub-menu is always expanded when the user is on either sub-route.

### Files to change

- `web/src/components/layout/AppSidebar.vue` — Refactor `NavItem` interface and `navItems` to support children. Render a nested `<ul>` for sub-items under "Artefacts". Apply indentation styling for child entries. The parent "Artefacts" item is not itself a link; it acts as a group label.

### Implementation detail

Update the `NavItem` interface:

```typescript
interface NavItem {
  label: string
  to?: string        // undefined for group headers
  children?: NavItem[]
}
```

The navItems function returns:

```typescript
{ label: 'Artefacts', children: [
    { label: 'List', to: `/p/${p}/artifacts` },
    { label: 'Board', to: `/p/${p}/artifacts/board` },
  ]},
{ label: 'Graph', to: `/p/${p}/graph` },
// ... rest unchanged
```

The group header is always expanded (no collapse/expand toggle needed). Child links get left padding to indicate nesting.

### Acceptance criteria

- [ ] Sidebar shows "Artefacts" as a group with "List" and "Board" sub-entries.
- [ ] Clicking "List" navigates to the existing artifact list.
- [ ] Clicking "Board" navigates to the board route.
- [ ] Other nav items (Graph, Agents, etc.) are unaffected.
- [ ] Active-state highlighting works correctly for both sub-entries.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 3 — Add board route and create KanbanBoardView shell

### Description

Register a new route for the board view and create the view component with a loading/empty state.

### Files to change

- `web/src/router/index.ts` — Add a route `{ path: 'artifacts/board', name: 'kanban-board', component: () => import('@/views/project/KanbanBoardView.vue') }`. This must be registered **before** the `artifacts/:pathMatch(.*)+` catch-all so it takes precedence.
- `web/src/views/project/KanbanBoardView.vue` — New file. Initial shell that fetches kanban config from `GET /api/p/:project/config/kanban` and shows either the board or the empty-state message ("No Kanban configuration found. Add a `kanban` section to your project's config.yaml.").

### Implementation detail

The view fetches kanban config on mount. If `kanban` is null, display the empty-state message. Otherwise, proceed to render columns (completed in Milestone 5).

### Acceptance criteria

- [ ] Navigating to `/p/:project/artifacts/board` renders the new view.
- [ ] When no kanban config exists, the empty-state message is displayed.
- [ ] The `artifacts/:pathMatch(.*)+` route still works for artifact editor paths.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 4 — Fetch all artifacts and build kanban API composable

### Description

Create a composable that fetches all artifacts (using the existing list API with a high limit or paginating to completion) and the kanban config, then provides reactive computed data grouping artifacts into columns.

### Files to change

- `web/src/composables/useKanbanBoard.ts` — New file. Composable that:
  1. Fetches kanban config from the backend endpoint (from [[kanban-view-4-be]]).
  2. Uses the artifacts store to fetch the full unfiltered artifact list.
  3. Exposes reactive `columns` (each with name, statuses, and filtered artifact list), an `uncategorised` column when enabled, and a `cardFields` array.
  4. Accepts reactive filter parameters (stage, status, type, label, priority) and applies them client-side.
  5. Computes virtual fields: `age` = days since `created` date, formatted as e.g. "12d".

### Implementation detail

The composable returns:

```typescript
{
  loading: Ref<boolean>
  hasConfig: Ref<boolean>
  columns: ComputedRef<KanbanColumn[]>  // each has { name, statuses, cards }
  cardFields: Ref<string[]>
  applyFilters(filters: Partial<ArtifactFilter>): void
  refresh(): Promise<void>
}
```

Each `KanbanColumn.cards` is an array of artifacts whose status is in that column's `statuses` list, post-filter. The uncategorised column (if enabled) collects artifacts not matched by any configured column.

Virtual field computation: for `age`, calculate `Math.floor((Date.now() - Date.parse(artifact.created)) / 86400000)` and format as `"Nd"`. Unknown card_fields entries are silently dropped.

### Acceptance criteria

- [ ] Artifacts are correctly grouped into columns by status.
- [ ] Uncategorised column appears when enabled and contains artifacts with unmapped statuses.
- [ ] Artifacts with unmapped statuses are excluded when uncategorised is disabled.
- [ ] Filters narrow the set of displayed cards without hiding empty columns.
- [ ] The `age` virtual field computes correctly.
- [ ] Unknown card_fields entries are silently ignored.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 5 — Render Kanban columns and cards

### Description

Build the main board UI: horizontal column layout with vertical card stacks, card rendering with configurable fields, and click-to-navigate.

### Files to change

- `web/src/views/project/KanbanBoardView.vue` — Wire up the `useKanbanBoard` composable. Render the filter bar (reusing the same filter controls pattern from `ArtifactListView`), columns, and cards.
- `web/src/components/artifact/KanbanCard.vue` — New file. Renders a single card: title (always), plus configured `card_fields` (type badge, priority indicator, labels as small tags, age, lineage slug). Clicking navigates to the artifact editor.

### Implementation detail

**Board layout** — A flex container with `overflow-x: auto` for horizontal scrolling. Each column is a flex child with a fixed min-width (~280px), a header (column name + count badge), and a scrollable card area (`overflow-y: auto`, `flex: 1`).

**Card rendering** — The `KanbanCard` component receives the artifact row and the `cardFields` array. It iterates `cardFields` and renders each known field:
- `title` — always shown as heading (regardless of list).
- `type` — rendered as a badge (reuse `.badge` styles).
- `priority` — rendered as a small coloured indicator.
- `labels` — rendered as small tags.
- `age` — rendered as text like "12d".
- `lineage` — always shown as a subtle slug at the bottom.
- Unknown fields — skipped silently.

Cards are focusable (`tabindex="0"`) and navigable via keyboard (Enter opens the artifact editor).

**Column headers** — Show column name and a count badge of currently visible cards.

**Empty column state** — Columns with zero cards show a subtle "No artefacts" placeholder.

### Acceptance criteria

- [ ] Each configured column renders as a vertical lane with header showing name and count.
- [ ] Cards display title plus fields from `card_fields` config.
- [ ] Lineage slug is shown on each card.
- [ ] Clicking a card navigates to the artifact editor.
- [ ] Cards are focusable via Tab and openable via Enter.
- [ ] Columns scroll horizontally when they overflow viewport width.
- [ ] Each column scrolls vertically independently.
- [ ] Empty columns display a placeholder.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 6 — Filter bar on the board view

### Description

Render the same filter controls (stage, status, type, label, priority, reset) above the board, wired to the composable's `applyFilters`.

### Files to change

- `web/src/views/project/KanbanBoardView.vue` — Add the filter bar markup and state (matching `ArtifactListView`'s filter bar pattern). Filters call `applyFilters` on the composable, which recomputes column card distributions.

### Implementation detail

The filter bar is structurally identical to the one in `ArtifactListView`: `<select>` elements for stage, status, type, label, priority, and a Reset button. Label and priority options are fetched from the artifacts store (`fetchLabels`, `fetchPriorities`). When the status filter is applied, it restricts which artifacts appear on cards but does **not** hide empty columns — columns always remain visible.

### Acceptance criteria

- [ ] All five filter dropdowns (stage, status, type, label, priority) appear above the board.
- [ ] Selecting a filter narrows displayed cards across all columns.
- [ ] The status filter does not hide empty columns.
- [ ] Reset clears all filters and shows all artifacts.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 7 — Column drag-to-reorder

### Description

Allow users to drag column headers to reorder columns within the current session using the native HTML drag-and-drop API (no external library).

### Files to change

- `web/src/views/project/KanbanBoardView.vue` — Add `draggable="true"` to column headers. Implement `dragstart`, `dragover`, `dragenter`, `dragleave`, and `drop` event handlers that reorder the columns array.
- `web/src/composables/useKanbanBoard.ts` — Expose a `reorderColumns(fromIndex, toIndex)` method that mutates the reactive columns array.

### Implementation detail

On `dragstart`, store the source column index in `dataTransfer`. On `drop` over a different column header, splice the source column out and insert at the target index. A CSS class highlights the drop target during `dragover`. The reorder is purely in-memory — refreshing the page resets to config order.

### Acceptance criteria

- [ ] Dragging a column header to a new position reorders columns visually.
- [ ] Card contents move with their column.
- [ ] Reordering is transient — page refresh resets to config order.
- [ ] Drop target is visually highlighted during drag.
- [ ] No new runtime dependencies are introduced.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 8 — Real-time updates via WebSocket

### Description

The board listens for `artifact.indexed` WebSocket events and re-renders affected cards, matching the invalidation pattern used by the list view.

### Files to change

- `web/src/views/project/KanbanBoardView.vue` — Use the existing `useWebSocket` composable to listen for `artifact.indexed` events and call `refresh()` on the kanban composable.

### Acceptance criteria

- [ ] When an artifact's status changes (triggering `artifact.indexed`), the board updates to reflect the card moving to the correct column.
- [ ] No full page reload is needed.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 9 — Responsive layout and accessibility

### Description

Ensure the board is usable on narrow viewports and meets the accessibility requirements.

### Files to change

- `web/src/views/project/KanbanBoardView.vue` — Add responsive CSS: on viewports < 768px, columns become horizontally scrollable (they already are via `overflow-x: auto`, but ensure the min-width shrinks or columns stack if needed).
- `web/src/components/artifact/KanbanCard.vue` — Ensure cards have appropriate `tabindex`, `role`, and `aria-label` attributes.
- `web/src/views/project/KanbanBoardView.vue` — Each column wrapper gets `role="region"` and `aria-label` matching the column name.

### Acceptance criteria

- [ ] On viewports < 768px, columns are horizontally scrollable and cards remain legible.
- [ ] Column landmarks use `role="region"` with `aria-label` matching the column name.
- [ ] Cards are focusable via Tab and openable via Enter.
- [ ] 500 cards render without perceptible jank.
- [ ] No new runtime dependencies are introduced.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.
