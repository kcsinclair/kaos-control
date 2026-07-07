---
title: Flaky TestQueue_HappyPath_SingleProject test fails due to false truncated stream marking
type: defect
status: draft
lineage: agent-rate-limit-queue
parent: lifecycle/tests/agent-rate-limit-queue-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Reproduction Steps

1. Run the integration test suite concurrently or under CPU load: `go test ./tests/... -tags integration`
2. Observe that `TestQueue_HappyPath_SingleProject` occasionally fails on the second queued job (`qd1-b-1.md`).
3. The supervisor logs a warning that the run exited cleanly without emitting a terminal result event (truncated stream) even though the stub was configured to write the terminal success event.

## Expected Behaviour

The test should run reliably. Stub execution and output processing should be synchronized such that all output events (specifically the terminal `result` event) are fully read and processed before the run is considered finished and checked for completeness.

## Actual Behaviour

The test is flaky. A race condition exists between the execution/exit of the fake `claude` stub process and the supervisor's stdout scanner reading the pipe. If the process exits extremely fast, the watcher goroutine reaps the process and can close the pipes or exit before the scanner finishes reading, causing the supervisor to false-positive flag a truncated stream and mark the run as `failed` rather than `completed`.

## Logs / Output

```
2026/07/07 15:10:41 INFO queue: run started job_id=f469f6c6a0128077 run_id=a371ac713f2580f1 agent=requirements-analyst artifact=lifecycle/ideas/qd1-b-1.md
2026/07/07 15:10:41 INFO http method=GET path=/api/queue status=200 bytes=1074 duration=376.5µs request_id=loki.local/qudfCq79JV-002696
2026/07/07 15:10:41 INFO http method=GET path=/api/queue status=200 bytes=1074 duration=193.5µs request_id=loki.local/qudfCq79JV-002697
2026/07/07 15:10:41 WARN agent: stream-json run exited cleanly without emitting a terminal result event — marking failed (truncated stream) run_id=a371ac713f2580f1 agent=requirements-analyst driver=claude-code-cli
...
--- FAIL: TestQueue_HappyPath_SingleProject (0.40s)
    queue_dispatch_test.go:63: job[1] ("lifecycle/ideas/qd1-b-1.md") state: got failed, want completed
```
