---
title: Flaky TestSupervisor_PersistsMetricsOnFinish test fails due to false truncated stream marking
type: defect
status: done
lineage: agent-usage-analytics-report
parent: lifecycle/tests/agent-usage-analytics-report-10-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Resolution (2026-07-07)

Fixed — this was a **real supervisor race**, not a test bug. In
`startCommandProcess` (`internal/agent/agent.go`), `cmd.Wait()` ran concurrently
with the stdout scanner and closes the `StdoutPipe` FDs. The Go docs warn it is
"incorrect to call Wait before all reads from the pipe have completed": a
fast-exiting process (the stub prints the terminal `result` line and exits in the
same instant) loses that final line before the scanner reads it →
`resultEventSeen=false` → false `truncated_stream` failure → metrics not
recorded. Real runs rarely hit it because they have a gap between the result
event and process exit; the stub maximises the window.

Fix: the reaper now waits for the readers to drain (they reach EOF on the
process's own exit) before calling `cmd.Wait()`, bounded by `readerDrainGrace`
(5s) so a detached grandchild holding a write end can't hang the supervisor.
Verified: the flaky pair (`TestSupervisor_PersistsMetricsOnFinish`,
`TestQueue_HappyPath_SingleProject`) passed 15/15 (previously ~1-in-5 failed).
Same fix resolves `queue-happypath-singleproject-flaky` (same root cause).

Note: the separate `WARN … hooks API detected="{…}"` log is cosmetic noise — the
version-check invocation runs the arg-ignoring stub and logs its NDJSON as the
"version". It does not cause the truncation (separate process/stdout).

## Reproduction Steps

1. Run the integration test suite sequentially: `go test ./tests/integration -tags integration`
2. Observe that `TestSupervisor_PersistsMetricsOnFinish` occasionally fails.
3. The supervisor logs a warning: `WARN agent: Claude Code version may not support hooks API` (with the stdout of the stub printed as the version string).
4. The supervisor logs a warning that the run exited cleanly without emitting a terminal result event (truncated stream), resulting in 0 metrics recorded.

## Expected Behaviour

The test should pass reliably. The stub execution output should be completely read by the agent supervisor, and the background version check should not run against, or interfere with, the test-specific stub binary in `PATH`, or at least not cause the test to fail.

## Actual Behaviour

The test is flaky.
1. The background version checking goroutine `go checkClaudeVersion()` executes `exec.Command("claude", "--version")`, which resolves to the test-specific stub on `PATH`. Because the stub ignores arguments and prints the test's metric events immediately, the version check consumes and logs them.
2. A race exists between the stub process exit and the supervisor's stdout scanner. If the scanner doesn't read the events before the process is reaped and pipes closed/terminated, or if they are consumed/interfered with, the supervisor marks the run as a `truncated_stream` failure, failing the test.

## Logs / Output

```
2026/07/07 15:14:00 INFO http method=POST path=/api/p/testproject/agents/requirements-analyst/run status=202 bytes=30 duration=6.822083ms request_id=loki.local/DWo0PhUdeu-000046
2026/07/07 15:14:01 WARN agent: Claude Code version may not support hooks API detected="{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"hello\"}]}}\n{\"type\":\"result\",\"subtype\":\"success\",\"total_cost_usd\":0.01,\"duration_ms\":1000,\"duration_api_ms\":900,\"num_turns\":1,\"usage\":{\"input_tokens\":100,\"cache_creation_input_tokens\":0,\"cache_read_input_tokens\":50,\"output_tokens\":200}}" min_required=1.9.0 hint="upgrade Claude Code to enable the claude-mediated driver"
2026/07/07 15:14:01 WARN agent: stream-json run exited cleanly without emitting a terminal result event — marking failed (truncated stream) run_id=2972ad0cc5530094 agent=requirements-analyst driver=claude-code-cli
...
--- FAIL: TestSupervisor_PersistsMetricsOnFinish (0.52s)
    agent_metrics_test.go:72: MetricsAvailable: got 0, want 1
    agent_metrics_test.go:75: TotalCostUSD should be non-nil after run with result line
    ...
```
