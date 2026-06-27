---
title: "Flaky TestAgentLifecycleType_RunAllowedAfterFirstCompletes under CPU contention"
type: defect
status: in-development
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
