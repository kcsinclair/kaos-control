---
title: 'Watcher tests flaky under full test-suite load: external-edit events not delivered within 2 s window'
type: defect
status: done
lineage: innovation-maker
parent: lifecycle/tests/Innovation Maker - Making Releases from Ideas-5-tests.md
labels:
    - defect
    - watcher
    - backend
assignees:
    - role: backend-developer
      who: agent
release: KC-OG-Sprint
---

# Watcher tests flaky under full test-suite load: external-edit events not delivered within 2 s window

## Reproduction Steps

1. Run the full integration test suite (all tests in `tests/integration/`):
   ```
   go test -tags=integration ./tests/integration/... -timeout 120s
   ```
2. Observe intermittent failures in `TestExternalEditPickedUp` and `TestExternalDeleteRemovesFromIndex`. Failure rate is roughly 1–2 out of every 5 full-suite runs (20–40 %).
3. Both tests pass consistently when run in isolation:
   ```
   go test -tags=integration ./tests/integration/... -run "TestExternalEditPickedUp|TestExternalDeleteRemovesFromIndex" -count 5
   ```

## Expected Behaviour

`TestExternalEditPickedUp`: writing a file directly to `lifecycle/ideas/` is detected by the fsnotify watcher and the artifact appears in the SQLite index within **2 seconds**.

`TestExternalDeleteRemovesFromIndex`: removing a file from `lifecycle/ideas/` is detected by the watcher and the row is removed from the index within **2 seconds**.

Both tests should pass reliably regardless of the number of other tests running in the same binary invocation.

## Actual Behaviour

Under full-suite load the watcher events are not processed within the 2 s deadline, causing the tests to fail:

```
    external_edit_test.go:50: watcher did not pick up externally written file within 2s
--- FAIL: TestExternalEditPickedUp (2.14s)
```

```
    external_edit_test.go:137: expected deleted file to be removed from index
--- FAIL: TestExternalDeleteRemovesFromIndex (2.16s)
```

## Logs / Output

The following log lines appear in the test output immediately before the failure. They indicate that watchers from **completed tests** are still processing fsnotify events (deletes of their temporary directories) while a new test's watcher is trying to start, consuming OS-level fsnotify dispatch capacity:

```
2026/04/27 11:49:06 WARN watcher: delete from index failed path=lifecycle/requirements/approve-done.md err="sql: database is closed"
2026/04/27 11:49:06 WARN watcher: delete from index failed path=lifecycle/requirements/full-lifecycle-2.md err="sql: database is closed"
2026/04/27 11:49:06 WARN watcher: delete from index failed path=lifecycle/ideas/put-valid-prio.md err="sql: database is closed"
...
    external_edit_test.go:50: watcher did not pick up externally written file within 2s
2026/04/27 11:49:13 ERROR watcher stopped with error project=testproject err="lstat /var/folders/.../lifecycle/releases: no such file or directory"
```

The `"sql: database is closed"` warnings come from previous tests' watcher goroutines that are still alive after `t.Cleanup` has closed the database. These stale goroutines hold fsnotify watches on now-deleted temp directories, delaying event delivery to the current test's watcher.

## Root Cause

`newTestEnv` tears down the SQLite index (closing the DB) before the watcher goroutine has been stopped. The watcher goroutine continues to receive events (OS-level `IN_DELETE` / `FSE_DELETE` from the temp directory being removed by `t.TempDir` cleanup), attempts to update the closed DB, and logs the `"sql: database is closed"` warnings. Under load these stale goroutines saturate the fsnotify event loop, delaying event delivery to new tests.

The fix is to ensure the watcher goroutine is fully stopped (its context cancelled and the goroutine joined) **before** the DB is closed in test teardown — or equivalently, before `t.Cleanup` removes the temp directory. The watcher already accepts a `context.Context`; cancelling it and waiting for the goroutine to exit should be sufficient.
