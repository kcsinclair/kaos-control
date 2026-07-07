---
title: "Missing integration test TestSupervisor_RecordsTTFTUnderLoad"
type: defect
status: in-development
lineage: agent-usage-analytics-report
parent: lifecycle/tests/agent-usage-analytics-report-10-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Reproduction Steps

1. Read the test suite specification artifact at `lifecycle/tests/agent-usage-analytics-report-10-test.md`.
2. Observe that `TestSupervisor_RecordsTTFTUnderLoad` is listed under "What was covered" (item 6) as part of the integration test suite.
3. Check `tests/integration/agent_metrics_test.go` for any definition or implementation of `TestSupervisor_RecordsTTFTUnderLoad`.
4. Run the integration tests matching `TestSupervisor_RecordsTTFTUnderLoad`:
   ```bash
   go test -v -tags=integration ./tests/integration -run TestSupervisor_RecordsTTFTUnderLoad
   ```

## Expected Behaviour

The test file `tests/integration/agent_metrics_test.go` should contain the `TestSupervisor_RecordsTTFTUnderLoad` integration test function to verify load conditions handling, specifically how CPU contention affects TTFT measurements.

## Actual Behaviour

The test function `TestSupervisor_RecordsTTFTUnderLoad` does not exist in `tests/integration/agent_metrics_test.go` or anywhere else in the codebase. Running `go test` for this test reports that no tests were run.

## Logs / Output

```
$ go test -v -tags=integration ./tests/integration -run TestSupervisor_RecordsTTFTUnderLoad
testing: warning: no tests to run
PASS
ok  	github.com/kaos-control/kaos-control/tests/integration	0.181s
```
