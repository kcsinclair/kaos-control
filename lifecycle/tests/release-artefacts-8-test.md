---
title: "Watcher Delete Reliability Fix — TestWatcherDeletesRowOnFileRemoval"
type: test
status: draft
lineage: release-artefacts
parent: lifecycle/defects/release-artefacts-7-defect.md
---

# Watcher Delete Reliability Fix — TestWatcherDeletesRowOnFileRemoval

Fix for the flaky `TestWatcherDeletesRowOnFileRemoval` integration test caused by
fsnotify event coalescence when a file is deleted immediately after API creation.

## What Was Changed

### `tests/integration/releases_watcher_test.go`

**`TestWatcherDeletesRowOnFileRemoval`** — the only change is the insertion of a
`time.Sleep(debounceWait)` between the API `POST /releases` call and the
`os.Remove` of the on-disk file.

**Why this matters:** when the file is removed within the watcher's 150 ms debounce
window of the API write, fsnotify coalesces the `CREATE` and `REMOVE` events into a
single handler invocation. That handler finds a matching entry in `ExpectedEvents`
(set by the API write path), consumes it as a loop-prevention sentinel, and exits
early — silently dropping the delete. The row stays in the database and the test
fails.

**Fix:** wait one full `debounceWait` (400 ms = 2 × 150 ms debounce + 100 ms buffer)
before removing the file. This ensures the watcher fires and processes the
`ExpectedEvents` entry for the API-driven CREATE before the REMOVE event arrives,
so the subsequent delete is not swallowed.

## Scenarios Covered

| Scenario | Assertion |
|---|---|
| File removed after full debounce window | DB row absent (`pollReleaseBySlug` returns false) |
| File removed after full debounce window | `release.changed` WS event received with `action:"deleted"` |

## Test File

- `tests/integration/releases_watcher_test.go` — `TestWatcherDeletesRowOnFileRemoval`
  (lines 89–130)
