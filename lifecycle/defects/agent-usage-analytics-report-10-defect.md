---
title: "Flaky TestSupervisor_RecordsTTFT under load"
type: defect
status: in-development
lineage: agent-usage-analytics-report
parent: lifecycle/tests/agent-usage-analytics-report-9-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Reproduction Steps

1. Run the integration test suite under resource contention:
   ```bash
   go test -tags=integration ./tests/...
   ```

## Expected Behaviour

The supervisor measures Time To First Token (TTFT) and verifies that it falls within a reasonable, realistic range (e.g., [80, 500] ms).

## Actual Behaviour

Under CPU contention, the test process suffers scheduling delays, resulting in a measured TTFT of 1275 ms, which exceeds the upper bound of 500 ms and causes the test to fail.

## Logs / Output

```
--- FAIL: TestSupervisor_RecordsTTFT (1.49s)
    agent_metrics_test.go:144: TtftMs: got 1275 ms, expected in range [80, 500]
```
