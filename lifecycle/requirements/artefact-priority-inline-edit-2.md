---
title: Inline Priority Display and Editing on Artefact Detail View
type: requirement
status: approved
lineage: artefact-priority-inline-edit
created: "2026-05-07T00:00:00+10:00"
priority: high
parent: lifecycle/ideas/artefact-priority-inline-edit.md
labels:
    - enhancement
    - frontend
    - artefacts
    - usability
    - vue
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

# Inline Priority Display and Editing on Artefact Detail View

## Problem

The artefact detail view (`ArtifactEditorView` → `FrontmatterPanel`) displays status with an inline dropdown for quick editing, but does not surface the priority field at all. Users — especially product owners and analysts during triage — must open the raw markdown or a separate edit mode to see or change an artefact's priority. This creates friction during planning sessions and breaks consistency with the status field's interaction model.

## Goals / Non-goals

### Goals

1. Display the current priority value prominently in the `FrontmatterPanel` metadata sidebar, visually consistent with the status row.
2. Allow single-click inline editing of priority via a dropdown, matching the established `StatusDropdown` interaction pattern.
3. Persist priority changes via the existing `PATCH /api/p/:project/artifacts/:path/priority` endpoint.
4. Provide clear visual differentiation between priority levels (colour-coded badges using the existing `PRIORITY_COLORS` mapping from `graphConstants.ts`).

### Non-goals

- Bulk-editing priority across multiple artefacts (separate feature).
- Adding priority as a sortable/filterable column to the artefact list view (separate feature).
- Enforcing a closed vocabulary on the backend — the backend accepts any string; the frontend offers the standard set (`high`, `medium`, `normal`, `low`) but must gracefully handle unknown values already present in frontmatter.
- Changing the priority field semantics, ordering, or adding new priority levels.

## Detailed Requirements

### Functional

1. **Priority row in FrontmatterPanel** — A new row labelled "Priority" must appear in the `FrontmatterPanel` component, positioned immediately after the "Status" row.
2. **Badge display** — The current priority value must render as a coloured badge. Colours must use the `PRIORITY_COLORS` map already defined in `web/src/components/graph/graphConstants.ts` (`high`, `medium`, `normal`, `low`). Unknown values must render with a neutral/default colour and the raw string as label.
3. **Inline dropdown** — Clicking the priority badge must open a `role="listbox"` dropdown anchored below the badge, listing the four standard options: `high`, `medium`, `normal`, `low`. Each option must display its corresponding colour indicator.
4. **Selection behaviour** — Selecting an option must:
   - Apply an optimistic UI update (update the badge immediately).
   - Call `PATCH /api/p/:project/artifacts/:path/priority` with `{ priority: <value> }`.
   - Revert to the previous value on API failure, displaying a toast or inline error.
5. **WebSocket sync** — The component must watch for `artifact.indexed` WebSocket events and update the displayed priority if the artifact's priority changed externally (e.g. another user, agent run, or filesystem edit).
6. **Dismiss behaviour** — The dropdown must close on:
   - Selection of an option.
   - Click outside the dropdown.
   - Press of `Escape`.
7. **No-change guard** — If the user selects the value already set, no API call should be made.
8. **Read-only mode** — When the artefact is locked by another user or the current user lacks write permission, the priority badge must display but not be clickable; no dropdown should appear.

### Non-functional

1. **Consistency** — The new `PriorityDropdown` component should follow the same structural patterns as `StatusDropdown` (optimistic update, listbox role, anchor positioning, keyboard support).
2. **Accessibility** — The dropdown must be keyboard-navigable (`ArrowUp`/`ArrowDown` to move, `Enter`/`Space` to select, `Escape` to dismiss) and expose correct ARIA attributes (`role="listbox"`, `aria-activedescendant`, `aria-expanded`).
3. **Performance** — No additional API calls on page load; priority is already included in the artifact frontmatter payload.
4. **Responsiveness** — The priority row and dropdown must render correctly at all supported viewport widths, consistent with the existing `FrontmatterPanel` layout.

## Acceptance Criteria

- [ ] `FrontmatterPanel` displays a "Priority" row with a colour-coded badge showing the artefact's current priority value.
- [ ] Clicking the badge opens an inline dropdown with the four standard priority options (`high`, `medium`, `normal`, `low`), each colour-indicated.
- [ ] Selecting a different priority updates the badge immediately (optimistic) and persists via `PATCH .../priority`.
- [ ] On API failure the badge reverts and an error is shown to the user.
- [ ] An artefact with an unknown priority value (e.g. `"critical"`) renders with a neutral badge and the raw string; the dropdown still offers the standard four options.
- [ ] Real-time updates via WebSocket: changing priority from another session or the filesystem updates the badge without a page refresh.
- [ ] Dropdown closes on outside click, `Escape`, or selection.
- [ ] Keyboard navigation works: `ArrowDown`/`ArrowUp` to move, `Enter`/`Space` to select.
- [ ] When the artefact is locked or the user lacks write access, the badge is visible but non-interactive.
- [ ] No duplicate API call when re-selecting the current value.
- [ ] Visual style and interaction feel consistent with the existing [[artefact-priority-inline-edit]] status inline-edit pattern.

## Resolved Questions

1. Should the dropdown include a "none" / "unset" option to allow clearing priority, or is priority always required?

> Default prioirity is normal so if not set it should display normal

2. Should there be a visual indicator (e.g. animation or flash) when priority changes via WebSocket to draw the user's attention?

> No
