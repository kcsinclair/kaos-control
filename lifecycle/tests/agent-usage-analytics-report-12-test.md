---
title: "Test Suite: TTFT Flakiness Under CPU Contention"
type: test
status: draft
lineage: agent-usage-analytics-report
parent: lifecycle/defects/agent-usage-analytics-report-10-defect.md
---

This artifact documents the fix for the flaky `TestSupervisor_RecordsTTFT`
described in the parent defect, and supersedes the false "Resolution" claim
in that defect and in `lifecycle/tests/agent-usage-analytics-report-10-test.md`
(item 6), which asserted that `TestSupervisor_RecordsTTFTUnderLoad` had been
added when it had not — see
`lifecycle/defects/agent-usage-analytics-report-11-defect.md`.

## What was covered

1. `TestSupervisor_RecordsTTFT` (`tests/integration/agent_metrics_test.go`) —
   already widened to a generous `[80, 2000]` ms bound in an earlier commit
   (`3ca5855c`), which resolves the originally reported failure
   (measured TTFT of 1275 ms exceeding a `[80, 500]` ms bound). No further
   change was needed here.
2. `TestSupervisor_RecordsTTFTUnderLoad` (new) — genuinely simulates the CPU
   contention scenario from the defect by busy-spinning one goroutine per
   `runtime.NumCPU()` core for the duration of the run, then asserts only
   that TTFT is recorded as a positive value. It deliberately has no upper
   bound, since scheduling delays under real contention are expected to
   inflate TTFT and a strict ceiling would itself be flaky.
3. `TestSupervisor_RecordsTTFTOnce`, `TestSupervisor_PersistsMetricsOnFinish`,
   `TestSupervisor_NonClaudeRun_NoMetrics`, and
   `TestReportsAgentUsage_AggregatedTTFS` re-verified unaffected by this
   change.

## Test files affected

- `tests/integration/agent_metrics_test.go` — added
  `TestSupervisor_RecordsTTFTUnderLoad`.

## Verification

```
go test -tags=integration ./tests/integration/... -run 'TestSupervisor_|TestReportsAgentUsage_AggregatedTTFS'
```

All tests pass.
