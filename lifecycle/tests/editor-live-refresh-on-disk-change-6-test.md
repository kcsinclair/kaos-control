---
title: "Tests: Auto-Refresh Editor Content on External Disk Change"
type: test
status: draft
lineage: editor-live-refresh-on-disk-change
parent: lifecycle/test-plans/editor-live-refresh-on-disk-change-5-test.md
created: "2026-04-29T00:00:00+10:00"
---

# Tests: Auto-Refresh Editor Content on External Disk Change

## Summary

This artifact documents the automated tests built for the `editor-live-refresh-on-disk-change` feature.
Tests cover two layers: Vue composable unit tests and Go integration tests.

## Test files

| File | Layer | Milestones |
|---|---|---|
| `tests/web/useExternalChange.test.ts` | Vitest unit | Milestone 1 |
| `tests/integration/external_edit_test.go` | Go integration | Milestones 2, 3, 4 |

Note: `external_edit_test.go` also contains pre-existing tests (`TestExternalEditPickedUp`, `TestExternalEditUpdateExisting`, `TestExternalDeleteRemovesFromIndex`) that were present before this feature work.

## Scenarios covered

### Milestone 1 — `useExternalChange` composable unit tests (`tests/web/useExternalChange.test.ts`)

1. **Auto-refresh fires when not dirty** — emits `file.changed` with `isDirty: () => false`; asserts `onAutoRefresh` is called after 300 ms and `hasExternalChange` stays `false`.
2. **Conflict banner when dirty** — emits `file.changed` with `isDirty: () => true`; asserts `hasExternalChange` becomes `true` and `onAutoRefresh` is never called.
3. **Debounce coalesces rapid events** — emits three `file.changed` events at 100 ms intervals; asserts `onAutoRefresh` is called exactly once 300 ms after the last event.
4. **Save-grace suppresses auto-refresh** — calls `markSaved()` then emits `file.changed`; asserts neither `onAutoRefresh` nor `hasExternalChange` is triggered.
5. **Save-grace suppresses conflict banner** — same as above but with `isDirty: () => true`.
6. **Events for other paths are ignored** — emits `file.changed` for a different path; asserts no effect.
7. **Backward compatibility** — instantiates without `options`; asserts `hasExternalChange` becomes `true` (original behaviour).
8. **Cleanup on unmount** — unmounts the component before the debounce fires; asserts `onAutoRefresh` is never called.

All tests use `vi.useFakeTimers()` for deterministic timer control and mock `getProjectWs` via `vi.mock('@/api/ws')`.

### Milestone 2 — Backend API returns updated content (`tests/integration/external_edit_test.go`)

- **TestAutoRefreshReadMode** — seeds an artifact, records its `file_sha` via GET, overwrites the file on disk, polls until the API returns the new title and a changed `file_sha`. Confirms the watcher + re-index pipeline delivers fresh content within 5 s.
- **TestRapidWritesCoalesce** — writes the same file three times at 40 ms intervals (< watcher debounce of 150 ms), then confirms the API returns the last-written title after a single re-index round.

### Milestone 3 — External edit while locked (`tests/integration/external_edit_test.go`)

- **TestExternalEditWhileLocked** — acquires a lineage lock, modifies the file on disk, confirms the API serves the updated content regardless of the lock, then releases the lock and verifies the artifact remains accessible.

### Milestone 4 — Backend save-grace contract (`tests/integration/external_edit_test.go`)

- **TestSaveDoesNotSelfTrigger** — registers a hub channel, saves an artifact via PUT, collects WebSocket events for up to 4 s. Asserts that a `file.changed` event with the saved artifact's path IS emitted. Documents the contract: the backend does not suppress this event — that is the frontend's responsibility via `SAVE_GRACE_MS`.

## How to run

```sh
# Milestone 1 — composable unit tests
cd tests/web && pnpm test

# Milestones 2–4 — Go integration tests
go test -tags integration -run 'TestAutoRefreshReadMode|TestRapidWritesCoalesce|TestExternalEditWhileLocked|TestSaveDoesNotSelfTrigger' ./tests/integration/...

# All integration tests
go test -tags integration ./tests/integration/...
```
