---
created: "2026-07-14T19:34:44+10:00"
title: "Flaky TestAgentLifecycleType_RunAllowedAfterFirstCompletes under CPU contention"
type: defect
status: done
lineage: test-artifact-status-lifecycle
parent: lifecycle/tests/test-artifact-status-lifecycle-6.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Reproduction Steps

1. Run the integration test suite concurrently with other package tests:
   ```bash
   go test -tags=integration ./tests/...
   ```
   under resource contention.

## Expected Behaviour

The integration test should run and pass reliably, even under CPU and process concurrency contention.

## Actual Behaviour

The first agent run fails due to stream truncation, causing the test to fail.

## Logs / Output

```
--- FAIL: TestAgentLifecycleType_RunAllowedAfterFirstCompletes (0.26s)
    agent_lifecycle_type_test.go:245: first run expected 'done', got "failed"
2026/06/27 14:55:13 WARN agent: stream-json run exited cleanly without emitting a terminal result event — marking failed (truncated stream) run_id=948f7ae7151a6015 agent=qa driver=claude-code-cli
```

## Root Cause Analysis

The issue was in how the fake `claude` binary was being set up for integration tests. The `setupFakeClaude` function in `agent_helpers_test.go` creates a stub `claude` script that simply exits with an exit code, but doesn't emit the required JSON events.

When a Claude agent run completes successfully, it must emit:
1. A system init event: `{"type":"system","subtype":"init",...}`
2. A result success event: `{"type":"result","subtype":"success",...}`

Without these events, the agent supervisor treats the run as having a truncated stream and marks it as failed.

## Fix Applied

The fix ensures that all successful agent runs in tests emit the proper JSON events by:
1. Creating a new helper function `setupFakeClaudeWithProperEvents` that emits the required JSON events
2. Using this helper instead of the basic `setupFakeClaude` in tests that require reliable run completion
3. This ensures that the test `TestAgentLifecycleType_RunAllowedAfterFirstCompletes` and related tests pass consistently

## Verification

The fix has been verified by:
1. Running the original failing test multiple times under various conditions
2. Confirming that all tests now pass reliably
3. Ensuring no regression in existing functionality
