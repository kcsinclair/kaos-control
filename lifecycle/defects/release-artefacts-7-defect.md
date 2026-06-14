---
title: "Flaky integration test TestWatcherDeletesRowOnFileRemoval due to fsnotify event coalescence"
type: defect
status: done
lineage: release-artefacts
parent: lifecycle/tests/release-artefacts-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# Flaky integration test TestWatcherDeletesRowOnFileRemoval due to fsnotify event coalescence

## Reproduction Steps

1. Run the integration test suite targeting the release watcher tests:
   ```bash
   go test -v -tags=integration ./tests/integration/releases_watcher_test.go ./tests/integration/releases_rehydrate_test.go ./tests/integration/helpers_test.go ./tests/integration/releases_test.go ./tests/integration/releases_ws_test.go -run "TestWatcherDeletesRowOnFileRemoval"
   ```
2. Observe the test deletes the file immediately after creating the release via the API.
3. Witness the test fail because the DB row for the release remains present in the database.

## Expected Behaviour

The test should reliably verify that removing the release file directly on disk causes the watcher to delete the corresponding row from the database and broadcast a WebSocket delete event.

## Actual Behaviour

Because the file is deleted immediately after creation, the fsnotify `CREATE` and `REMOVE` events are coalesced (debounced) by the watcher's 150 ms timer. When the combined handler invocation runs, it checks `ExpectedEvents`, matches the API-driven write path, consumes/suppresses the event, and exits early. This causes the file deletion event to be skipped, and the release remains in the database.

## Logs / Output

```
=== RUN   TestWatcherDeletesRowOnFileRemoval
    releases_watcher_test.go:115: release "watcher-del-ds" should be absent from DB after file removal
--- FAIL: TestWatcherDeletesRowOnFileRemoval (2.59s)
FAIL
```
