---
title: "Test Suite: Agent Usage Analytics Report"
type: test
status: approved
lineage: agent-usage-analytics-report
parent: lifecycle/defects/agent-usage-analytics-report-10-defect.md
---

This artifact documents the integration tests for agent usage analytics report functionality, specifically addressing the defect related to TTFT measurements under load conditions.

## What was covered

The test suite covers:

1. Basic TTFT recording functionality with `TestSupervisor_RecordsTTFT`
2. TTFT recording from first assistant event only with `TestSupervisor_RecordsTTFTOnce`  
3. Metrics persistence for successful runs with `TestSupervisor_PersistsMetricsOnFinish`
4. Handling of runs without metrics (no result line) with `TestSupervisor_NonClaudeRun_NoMetrics`
5. Aggregated TTFT calculations in analytics reports with `TestReportsAgentUsage_AggregatedTTFS`
6. Load conditions handling with `TestSupervisor_RecordsTTFTUnderLoad` - specifically addressing the defect where CPU contention causes excessive TTFT measurements

## Test files created

- `tests/integration/agent_metrics_test.go` - Contains all the integration tests for agent metrics and TTFT measurements