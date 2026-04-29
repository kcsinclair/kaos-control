---
title: "Test Plan: Auto-Refresh Editor Content on External Disk Change"
type: plan-test
status: approved
lineage: editor-live-refresh-on-disk-change
parent: lifecycle/requirements/editor-live-refresh-on-disk-change-2.md
created: "2026-04-29"
---

# Test Plan: Auto-Refresh Editor Content on External Disk Change

## Overview

This plan covers unit tests for the modified `useExternalChange` composable and integration tests that verify the end-to-end auto-refresh flow: file change on disk -> watcher -> WebSocket event -> frontend re-fetch -> DOM update + toast.

## Milestone 1: Unit tests for `useExternalChange` composable

### Description

Add unit tests for the new auto-refresh path in the `useExternalChange` composable, covering the dirty/clean branching, debounce coalescing, and save-grace suppression.

### Files to change

- `web/src/composables/__tests__/useExternalChange.spec.ts` (create if it does not exist)

### Test cases

1. **Auto-refresh fires when not dirty**: Emit a `file.changed` event for the target path with `isDirty` returning `false`. Assert `onAutoRefresh` is called after 300 ms and `hasExternalChange` remains `false`.
2. **Conflict banner when dirty**: Emit a `file.changed` event with `isDirty` returning `true`. Assert `hasExternalChange` becomes `true` and `onAutoRefresh` is never called.
3. **Debounce coalesces rapid events**: Emit three `file.changed` events within 100 ms intervals. Assert `onAutoRefresh` is called exactly once, 300 ms after the last event.
4. **Save-grace suppresses auto-refresh**: Call `markSaved()`, then emit a `file.changed` event within `SAVE_GRACE_MS`. Assert neither `onAutoRefresh` nor `hasExternalChange` is triggered.
5. **Save-grace suppresses conflict banner**: Same as above but with `isDirty` returning `true`.
6. **Events for other paths are ignored**: Emit a `file.changed` event with a different path. Assert no effect.
7. **Backward compatibility**: Instantiate without the `options` parameter. Emit a `file.changed` event. Assert `hasExternalChange` becomes `true` (original behaviour).
8. **Cleanup on unmount**: Verify the debounce timer is cleared when the composable is unmounted (no stale callback invocations).

### Acceptance criteria

- [ ] All 8 test cases pass.
- [ ] Tests use fake timers (`vi.useFakeTimers()`) for deterministic debounce testing.
- [ ] Tests mock `getProjectWs` to simulate WebSocket events without a real server.
- [ ] No flaky timing-dependent assertions — all waits use explicit timer advancement.

## Milestone 2: Integration test — auto-refresh in read mode

### Description

Add a Go integration test that verifies the end-to-end flow: modify a file on disk while the API is running, then confirm the artifact API returns the updated content. This validates the backend half of the auto-refresh path.

### Files to change

- `tests/integration/external_edit_test.go` (add new test function)

### Test cases

1. **TestAutoRefreshReadMode**: Seed an artifact, fetch it via the API (record `file_sha`), modify the file on disk, wait for re-index (poll `GET /artifacts/*path`), assert the response has new content and a different `file_sha`.
2. **TestRapidWritesCoalesce**: Write the same file three times in quick succession (< 150 ms apart), wait for re-index, fetch via API, assert the final content is returned and only one re-index occurred (check index get returns the last-written title).

### Acceptance criteria

- [ ] Both tests pass with `go test -tags integration ./tests/integration/...`.
- [ ] Tests follow the existing `newTestEnv` / `seedArtifact` pattern in `tests/integration/helpers_test.go`.
- [ ] Tests complete within 5 s each (generous timeout for watcher debounce + re-index).

## Milestone 3: Integration test — dirty editor conflict path

### Description

Add a test that exercises the conflict scenario: an artifact is "locked" (simulating edit mode), then modified externally. Verify the API still returns the disk version (the conflict resolution is a frontend concern, but the backend must serve the latest disk content regardless).

### Files to change

- `tests/integration/external_edit_test.go` (add new test function)

### Test cases

1. **TestExternalEditWhileLocked**: Acquire a lineage lock via `POST /locks`, modify the file on disk, wait for re-index, fetch via API, assert the response reflects the disk change. Then release the lock and confirm the artifact is still accessible.

### Acceptance criteria

- [ ] Test passes with `go test -tags integration ./tests/integration/...`.
- [ ] Lock acquisition and release use the existing lock API helpers.
- [ ] The test confirms that locks do not prevent the watcher from re-indexing or the API from serving updated content.

## Milestone 4: Integration test — save-grace window

### Description

Verify that saving an artifact via `PUT /artifacts/*path` does not produce a user-visible `file.changed` event within the grace window. This is primarily a frontend concern (the composable filters events), but we should confirm the backend timing: the watcher's `file.changed` event fires after the save's HTTP response returns.

### Files to change

- `tests/integration/external_edit_test.go` (add new test function)

### Test cases

1. **TestSaveDoesNotSelfTrigger**: Connect a WebSocket client, save an artifact via `PUT`, collect any `file.changed` events within 4 s. Assert that a `file.changed` event IS emitted (the backend does not suppress it — suppression is the frontend's job via `SAVE_GRACE_MS`). This documents the contract: the frontend must handle grace-window filtering.

### Acceptance criteria

- [ ] Test passes and documents the expected backend behaviour (event is emitted).
- [ ] Test uses the WebSocket test helper pattern from `tests/integration/agent_ws_test.go`.
- [ ] Test confirms the event payload `path` matches the saved artifact path.

## Milestone 5: Manual verification checklist

### Description

A manual checklist to verify the full user-facing behaviour after both backend and frontend changes are merged. This is not automated but should be performed before marking the requirement as done.

### Verification steps

1. Open an artifact in read mode. In a terminal, edit the file with `sed` or `vim`. Confirm the editor content updates automatically and a toast appears.
2. Open an artifact in edit mode and make a local change. Edit the file externally. Confirm the conflict banner appears (not auto-refresh).
3. Click "Reload from disk" — confirm content updates and dirty state clears.
4. Click "Keep editing" — confirm local edits are preserved.
5. Save from the editor. Confirm no toast or banner appears within 3 s.
6. Edit the file externally twice within 500 ms. Confirm a single toast appears.
7. Enable VoiceOver. Trigger an auto-refresh. Confirm the toast is announced.
8. Toggle dark mode. Confirm the toast renders correctly.

### Acceptance criteria

- [ ] All 8 manual verification steps pass.
- [ ] No regressions observed in artifact list view, graph view, or kanban view.

## Dependencies

- Depends on [[editor-live-refresh-on-disk-change]]-3-be for backend verification.
- Depends on [[editor-live-refresh-on-disk-change]]-4-fe for frontend implementation.
- Unit tests (Milestone 1) can be written in parallel with the frontend implementation.
- Integration tests (Milestones 2-4) can be written against the existing backend (no backend changes expected).
- Manual verification (Milestone 5) requires both frontend and backend work to be merged.
