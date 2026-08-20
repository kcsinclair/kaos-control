---
created: "2026-07-14T19:34:44+10:00"
title: "Flaky TestAgentLifecycleType_RunAllowedAfterFirstCompletes under CPU contention"
type: test
status: approved
lineage: test-artifact-status-lifecycle
parent: lifecycle/test-plans/test-artifact-status-lifecycle-5-test.md
---

# Integration Tests for Artifact Status Lifecycle

This test suite implements integration tests that cover the artifact status lifecycle behavior, including concurrent run guards and handling of stream truncation issues.

## Files

- `tests/integration/concurrent_run_under_contention_test.go` — Tests for concurrent runs under CPU contention and stream truncation issues

## Scenarios Covered

### Stream Truncation Fix
- The original test `TestAgentLifecycleType_RunAllowedAfterFirstCompletes` was flaky due to improper JSON event emission from fake claude binaries
- Fixed by ensuring that successful agent runs emit the required JSON events (`{"type":"system","subtype":"init",...}` and `{"type":"result","subtype":"success",...}`)
- This prevents the "stream-json run exited cleanly without emitting a terminal result event" error that was causing test failures

### Concurrent Run Tests
- Verifies that multiple concurrent runs work correctly under load
- Ensures that artifact status transitions are handled properly even when multiple agent runs occur
- Tests that the lock system properly releases and allows subsequent runs after the first completes

## Test Execution

The tests in `concurrent_run_under_contention_test.go` can be run with:

```bash
go test -tags=integration ./tests/integration/concurrent_run_under_contention_test.go
```

These tests reproduce and verify the fix for the flaky behavior described in the defect report.