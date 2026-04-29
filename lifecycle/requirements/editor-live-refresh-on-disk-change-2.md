---
title: Auto-Refresh Editor Content on External Disk Change
type: requirement
status: approved
lineage: editor-live-refresh-on-disk-change
parent: lifecycle/ideas/editor-live-refresh-on-disk-change.md
created: "2026-04-29"
priority: high
labels:
    - frontend
    - enhancement
    - watcher
    - usability
    - vue
---

# Auto-Refresh Editor Content on External Disk Change

## Problem

When an artifact's Markdown file is modified outside the editor (e.g. by an agent run, a git operation, or a separate text editor), the current UI shows a static banner requiring the user to manually click "Reload from disk" or "Keep editing." This creates unnecessary friction: the user must notice the banner, interrupt their reading flow, and click a button to see content that could have been displayed automatically. During agent runs that touch many files in quick succession the banner can feel especially noisy and slow.

The existing `useExternalChange` composable (`web/src/composables/useExternalChange.ts`) already listens for `file.changed` WebSocket events and exposes a reactive `hasExternalChange` flag, so the plumbing is in place — the missing piece is the automatic re-fetch and a lightweight notification replacing the current blocking banner.

## Goals / Non-goals

### Goals

1. **Automatic content refresh** — when the currently-open artifact changes on disk, the editor re-fetches and re-renders the file without user intervention.
2. **Unobtrusive notification** — a brief, auto-dismissing toast (or equivalent inline indicator) informs the user that the content was refreshed, so changes are never silent.
3. **Dirty-editor safety** — if the user has unsaved local edits, the auto-refresh must NOT silently overwrite them; the current manual banner ("Reload from disk" / "Keep editing") must be preserved for this case.
4. **Save-grace window** — continue to suppress reloads triggered by the user's own recent save (the existing `SAVE_GRACE_MS` logic in `useExternalChange`).

### Non-goals

- Real-time collaborative editing or OT/CRDT merging of concurrent changes.
- Extending auto-refresh to views other than the artifact editor (list view, graph view, kanban board already react to `artifact.indexed` events independently).
- Changing the backend watcher debounce interval or WebSocket event schema.
- Adding user-facing settings to toggle auto-refresh on/off (can be considered later).

## Detailed Requirements

### Functional

| ID | Requirement |
|----|-------------|
| FR-1 | When `ArtifactEditorView` is open in **read mode** (not editing) and a `file.changed` event arrives for the displayed artifact, the component shall automatically re-fetch the artifact from the API and re-render the content. |
| FR-2 | After an automatic refresh, a toast notification shall appear with the message "File updated on disk" (or similar concise text). The toast shall auto-dismiss after 3–5 seconds and be manually dismissable. |
| FR-3 | When the editor is in **edit mode** with unsaved local changes (dirty buffer) and a `file.changed` event arrives, the component shall **not** auto-refresh. Instead it shall display the existing conflict banner ("This file was changed externally — Reload from disk / Keep editing"). |
| FR-4 | If the user clicks "Reload from disk" in the conflict banner the editor shall re-fetch, re-render, and clear the dirty state. No additional toast is needed in this case. |
| FR-5 | The save-grace window (`SAVE_GRACE_MS`, currently 3 000 ms) shall continue to suppress both auto-refresh and the conflict banner for events caused by the user's own save. |
| FR-6 | Rapid successive `file.changed` events for the same artifact (e.g. multiple saves within a short window) shall coalesce so that at most one re-fetch and one toast are triggered per burst. A debounce of 300–500 ms on the re-fetch is acceptable. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-1 | The auto-refresh round-trip (WS event → re-fetch → DOM update) should complete within 500 ms on localhost under normal load. |
| NFR-2 | The toast component must be accessible: include an appropriate ARIA live region (`role="status"` or `aria-live="polite"`) so screen readers announce the update. |
| NFR-3 | Toast styling must respect the existing light/dark theme variables and not introduce new colour tokens. |
| NFR-4 | No new runtime dependencies may be added for the toast; use existing project primitives or a lightweight inline implementation. |

## Acceptance Criteria

- [ ] Opening an artifact in read mode, then modifying its file on disk, causes the editor to display the new content automatically within ~500 ms.
- [ ] A toast notification appears after the automatic refresh and auto-dismisses after 3–5 seconds.
- [ ] Opening an artifact in edit mode, making a local change, then modifying the file on disk, shows the conflict banner instead of auto-refreshing.
- [ ] Clicking "Reload from disk" in the conflict banner loads the latest disk content and clears the dirty flag.
- [ ] Clicking "Keep editing" in the conflict banner dismisses it and preserves the user's local edits.
- [ ] Saving the artifact from the editor does not trigger a spurious auto-refresh or toast within the grace window.
- [ ] Two rapid disk changes within 500 ms result in a single re-fetch and a single toast.
- [ ] The toast is announced by screen readers (verify with an accessibility audit tool or manual VoiceOver check).
- [ ] Existing `useExternalChange` unit/integration tests continue to pass; new tests cover the auto-refresh and dirty-guard paths.
- [ ] Related: [[editor-live-refresh-on-disk-change]]

## Open Questions

1. Should the toast include a one-click "Undo" or "Show diff" action for users who want to see what changed, or is a plain informational message sufficient for v1?
2. If a future iteration adds a user preference to disable auto-refresh globally, should the composable accept a reactive `enabled` flag now to make that easier, or defer the design?
