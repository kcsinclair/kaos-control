---
title: "Frontend Plan — Inline Priority Display and Editing"
type: plan-frontend
status: draft
lineage: artefact-priority-inline-edit
parent: lifecycle/requirements/artefact-priority-inline-edit-2.md
---

# Frontend Plan — Inline Priority Display and Editing

## Overview

Add a `PriorityDropdown.vue` component modelled on the existing `StatusDropdown.vue` (`web/src/components/artifact/StatusDropdown.vue`) and integrate it into `FrontmatterPanel.vue` as a new "Priority" row immediately after the "Status" row. The component uses the `priorityColors` palette from `useGraphTheme()` for colour-coded badges, performs optimistic updates via `patchPriority()` from `@/api/artifacts`, and stays in sync via the existing WebSocket `artifact.indexed` flow in `ArtifactEditorView.vue`.

---

## Milestone 1 — Create `PriorityDropdown.vue` component

### Description

Create a new SFC at `web/src/components/artifact/PriorityDropdown.vue` following the same structural patterns as `StatusDropdown.vue`:

- **Props**: `project: string`, `path: string`, `priority: string` (current value; empty/missing treated as `"normal"`).
- **Emits**: `changed(newPriority: string)`, `error(message: string)`.
- **Local state**: `optimisticPriority` ref initialised from `props.priority || 'normal'`; `isOpen` ref; `focusedIndex` ref.
- **Options**: constant array `['high', 'medium', 'normal', 'low']`.
- **Trigger element**: a `<button>` with `role="button"`, `aria-haspopup="listbox"`, `:aria-expanded`, `tabindex="0"`. Renders the current priority as a colour-coded badge using `palette.priorityColors[optimisticPriority]` (with `+'33'` for background, `+'66'` for border — matching the pattern in `ArtifactModal.vue:172`).
- **Dropdown panel**: a `<div>` with `role="listbox"`, anchored below the trigger. Each option is a `<div role="option">` with `:aria-selected`, colour indicator dot, and label text.
- **Selection**: on select, if value differs from current, apply optimistic update → call `patchPriority(project, path, value)` → emit `changed` on success / revert + emit `error` on failure.
- **No-change guard**: skip API call if selected value equals `optimisticPriority`.
- **Dismiss**: close on selection, outside click (`v-click-outside` or manual `pointerdown` listener on `document`), or `Escape`.
- **Keyboard nav**: `ArrowDown`/`ArrowUp` move focus through options, `Enter`/`Space` select, `Escape` dismisses and returns focus to trigger.
- **Prop watch**: watch `props.priority` — when it changes externally (WebSocket update flows down from `ArtifactEditorView`), update `optimisticPriority` if the dropdown is closed.
- **Read-only mode**: accept a `readonly: boolean` prop (default `false`). When true, render the badge but disable click, remove `aria-haspopup`, set `tabindex="-1"`.

### Files to change

- `web/src/components/artifact/PriorityDropdown.vue` — **new file**

### Acceptance criteria

- [ ] Component renders a colour-coded priority badge using `useGraphTheme().palette.priorityColors`.
- [ ] Clicking the badge opens a listbox dropdown with four options, each with a colour indicator.
- [ ] Selecting a different value triggers optimistic update + `patchPriority()` API call.
- [ ] On API failure, badge reverts to previous value and `error` event is emitted.
- [ ] No API call when re-selecting the current value.
- [ ] Dropdown closes on selection, outside click, or `Escape`.
- [ ] Full keyboard navigation: `ArrowDown`/`ArrowUp`, `Enter`/`Space`, `Escape`.
- [ ] ARIA attributes: `role="listbox"`, `role="option"`, `aria-selected`, `aria-expanded`, `aria-activedescendant`.
- [ ] When `readonly` is true, badge displays but is not interactive.
- [ ] Unknown priority values (e.g. `"critical"`) render with a neutral grey colour and the raw string as label.

---

## Milestone 2 — Integrate into `FrontmatterPanel.vue`

### Description

Add a "Priority" row to `FrontmatterPanel.vue` immediately after the "Status" row (after line ~59). Follow the same conditional pattern: render `PriorityDropdown` when `project` and `targetPath` are provided, otherwise render a static badge.

### Files to change

- `web/src/components/artifact/FrontmatterPanel.vue` — add priority row, import `PriorityDropdown`

### Acceptance criteria

- [ ] "Priority" row appears directly after "Status" in the metadata sidebar.
- [ ] When `project` and `targetPath` are present, renders interactive `PriorityDropdown` with `:priority="artifact.frontmatter?.priority || 'normal'"`.
- [ ] When `project`/`targetPath` are absent, renders a static colour-coded badge (read-only).
- [ ] `@changed` event is handled (emit upward or invalidate as needed).
- [ ] `@error` event is handled (emit upward for toast display).
- [ ] Layout is consistent with existing rows at all supported viewport widths.

---

## Milestone 3 — WebSocket sync and lock-awareness

### Description

Ensure the priority badge updates in real time when the artifact is modified externally, and respects lock state for read-only rendering.

The existing flow in `ArtifactEditorView.vue` (lines 214-238) already handles `artifact.indexed` events by invalidating the store cache and re-fetching. The re-fetched `artifact.frontmatter.priority` flows down as a prop to `FrontmatterPanel` → `PriorityDropdown`. The `PriorityDropdown`'s prop watcher (from M1) picks up the change and updates `optimisticPriority`.

For lock awareness, `ArtifactEditorView` already tracks whether the artifact is locked by another user. Pass this as the `readonly` prop to `PriorityDropdown` via `FrontmatterPanel`.

### Files to change

- `web/src/components/artifact/FrontmatterPanel.vue` — accept and forward a `readonly` prop to `PriorityDropdown`
- `web/src/views/project/ArtifactEditorView.vue` — pass lock state as `readonly` prop to `FrontmatterPanel` (if not already passed)

### Acceptance criteria

- [ ] Changing priority from another session (or filesystem edit) updates the badge in the open detail view without a page refresh.
- [ ] When the artifact is locked by another user, `PriorityDropdown` renders in read-only mode (badge visible, not clickable).
- [ ] When the lock is released, the dropdown becomes interactive again on the next WebSocket update.

---

## Milestone 4 — Styling and visual consistency

### Description

Ensure the `PriorityDropdown` badge and dropdown visually match the established `StatusDropdown` patterns and existing priority badge styles used in `ArtifactModal.vue` and `KanbanCard.vue`. Use scoped styles within the component.

### Files to change

- `web/src/components/artifact/PriorityDropdown.vue` — scoped `<style>` block

### Acceptance criteria

- [ ] Badge size, font, border-radius, and padding are consistent with the status badge.
- [ ] Dropdown panel shadow, border, and spacing match the status dropdown.
- [ ] Colour-coded dots/indicators in the dropdown options are clearly visible in both light and dark themes (using `useGraphTheme()` palette switching).
- [ ] Priority row integrates cleanly into `FrontmatterPanel` at all viewport widths without layout shift.
