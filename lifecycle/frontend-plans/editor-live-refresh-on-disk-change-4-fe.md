---
title: "Frontend Plan: Auto-Refresh Editor Content on External Disk Change"
type: plan-frontend
status: done
lineage: editor-live-refresh-on-disk-change
parent: lifecycle/requirements/editor-live-refresh-on-disk-change-2.md
created: "2026-04-29T00:00:00+10:00"
---

# Frontend Plan: Auto-Refresh Editor Content on External Disk Change

## Overview

Modify `useExternalChange` composable and `ArtifactEditorView` to automatically re-fetch and re-render artifact content when a `file.changed` WebSocket event arrives and the editor is in read mode (no dirty buffer). When the editor has unsaved changes, preserve the existing manual conflict banner. Show a brief toast notification after each auto-refresh using the existing `useUiStore` toast system.

## Milestone 1: Extend `useExternalChange` to support auto-refresh callback

### Description

Add a debounced auto-refresh path to the `useExternalChange` composable. When the caller provides an `onAutoRefresh` callback and the editor is not dirty, the composable should invoke the callback instead of just setting `hasExternalChange = true`. Include a 300 ms debounce to coalesce rapid successive events (FR-6).

### Files to change

- `web/src/composables/useExternalChange.ts`

### Changes

1. Accept an optional `options` parameter: `{ isDirty?: () => boolean; onAutoRefresh?: () => void }`.
2. When a `file.changed` event arrives (and passes the save-grace check):
   - If `isDirty()` returns `true` (or `isDirty` is not provided): set `hasExternalChange.value = true` (current behaviour — shows conflict banner).
   - If `isDirty()` returns `false` and `onAutoRefresh` is provided: debounce 300 ms, then invoke `onAutoRefresh()`. Do NOT set `hasExternalChange`.
3. Use a local `setTimeout`/`clearTimeout` pair for the debounce. Cancel any pending debounce timer in the `onUnmounted` cleanup.
4. Continue to export the existing API (`hasExternalChange`, `markSaved`, `acknowledge`) unchanged so other consumers are unaffected.

### Acceptance criteria

- [ ] When `isDirty()` returns `false`, a `file.changed` event triggers `onAutoRefresh` after 300 ms.
- [ ] When `isDirty()` returns `true`, a `file.changed` event sets `hasExternalChange = true` (conflict banner path).
- [ ] Two `file.changed` events within 300 ms result in a single `onAutoRefresh` call.
- [ ] The save-grace window (`SAVE_GRACE_MS`) suppresses both paths.
- [ ] Existing call sites that do not pass `options` continue to work identically.

## Milestone 2: Wire auto-refresh into `ArtifactEditorView`

### Description

Update `ArtifactEditorView.vue` to use the new `onAutoRefresh` callback from `useExternalChange`. In read mode, auto-refresh re-fetches the artifact and re-renders it. In edit mode with unsaved changes, the conflict banner continues to appear.

### Files to change

- `web/src/views/project/ArtifactEditorView.vue`

### Changes

1. Define an `isDirty` function: returns `true` when `editing.value === true` (the user is in edit mode and may have unsaved changes).
2. Define an `autoRefresh` function that:
   a. Calls `store.invalidate(artifactPath.value)`.
   b. Re-fetches: `artifact.value = await store.fetchOne(project.value, artifactPath.value)`.
   c. Calls `ui.info('File updated on disk')` to show a toast (auto-dismisses via existing 4 s timeout in `useUiStore`).
3. Pass `{ isDirty: () => editing.value, onAutoRefresh: autoRefresh }` to `useExternalChange`.
4. The existing `hasExternalChange` banner in the template remains unchanged — it only appears when `isDirty()` was `true` at event time.

### Acceptance criteria

- [ ] In read mode: disk change triggers automatic re-fetch and content update within ~500 ms (NFR-1).
- [ ] In read mode: an "info" toast "File updated on disk" appears and auto-dismisses after ~4 s (FR-2).
- [ ] In edit mode: disk change shows the conflict banner, not auto-refresh (FR-3).
- [ ] "Reload from disk" in the conflict banner works as before (FR-4).
- [ ] "Keep editing" in the conflict banner works as before.
- [ ] Saving does not trigger a spurious toast within the grace window (FR-5).
- [ ] Two rapid disk changes produce one re-fetch and one toast (FR-6).

## Milestone 3: Verify toast accessibility compliance

### Description

The existing `Toast.vue` component already uses `aria-live="polite"` and `role="status"` on individual toast items. Verify that the auto-refresh "info" toast is announced by screen readers and that no additional ARIA changes are needed (NFR-2).

### Files to review (no changes expected)

- `web/src/components/common/Toast.vue`
- `web/src/stores/ui.ts`

### Changes

- Verify that the `.toast--info` variant is styled using existing theme variables (`--color-surface`, `--color-border`, `--color-text`) and does not introduce new colour tokens (NFR-3).
- Verify the toast container has `aria-live="polite"` and each toast has `role="status"` — already present in the current implementation.

### Acceptance criteria

- [ ] The "File updated on disk" toast is announced by VoiceOver / screen reader when it appears.
- [ ] No new colour tokens or CSS custom properties are introduced.
- [ ] Light and dark theme both render the info toast correctly.
- [ ] The toast is manually dismissable via the close button (already supported).

## Milestone 4: Remove redundant `artifact.indexed` listener (cleanup)

### Description

`ArtifactEditorView` currently has a separate `useWebSocket` listener for `artifact.indexed` events that calls `load()` when not editing (lines 147-152). With the new auto-refresh on `file.changed`, this listener is redundant for the same file — the `file.changed` event always fires before or alongside `artifact.indexed`. However, `artifact.indexed` can also fire for re-index events not triggered by a file change (e.g. startup scan). Evaluate whether to keep, remove, or gate this listener.

### Files to change

- `web/src/views/project/ArtifactEditorView.vue`

### Changes

1. Keep the `artifact.indexed` listener but add a guard: skip if `artifact.value?.file_sha` matches the current SHA (avoiding a duplicate fetch when auto-refresh already handled it).
2. This ensures startup re-index and manual re-index still update the view, while avoiding double-fetch on normal file changes.

### Acceptance criteria

- [ ] After auto-refresh handles a `file.changed` event, a subsequent `artifact.indexed` event for the same file does not trigger a redundant fetch.
- [ ] A standalone `artifact.indexed` event (without a preceding `file.changed`) still refreshes the view.
- [ ] No regressions in the existing artifact editor behaviour.

## Dependencies

- Depends on [[editor-live-refresh-on-disk-change]]-3-be confirming the `file.changed` event payload shape and `file_sha` in the API response.
- The [[editor-live-refresh-on-disk-change]]-5-test test plan will add integration and unit tests covering the auto-refresh paths.
