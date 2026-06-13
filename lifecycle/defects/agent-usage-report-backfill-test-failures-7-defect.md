---
title: Backfill metrics integration tests fail due to outdated database schema and missing modelUsage
type: defect
status: approved
lineage: agent-usage-analytics-report
parent: lifecycle/tests/agent-usage-analytics-report-6-test.md
labels:
  - defect
assignees:
  - role: test-developer
    who: agent
---

# Backfill metrics integration tests fail due to outdated database schema and missing modelUsage

The `kaos-control backfill` command tests fail when run against the mock setup because the mock database schema and the mock log files are out of sync with the production schema/behavior.

## Reproduction Steps

1. Temporarily resolve the duplicate function compiler errors (see defect `agent-usage-report-test-compile-error-7-defect.md`) so that the integration tests compile.
2. Run `go test -v -tags=integration ./tests/integration/ -run="TestBackfill_"` in the repository root.
3. Observe `TestBackfill_PopulatesUnparseableRuns` failing with `SQL logic error: no such column: model`.
4. Observe `TestBackfill_Idempotent` failing with `second backfill run should report 'backfilled 0'`.

## Expected Behaviour

1. The mock database created by `createAgentRunsTable` in `backfill_metrics_test.go` should include all the analytics columns (like `model`, `total_cost_usd`, `duration_api_ms`, etc.) that exist in the production database schema.
2. The mock result line printed to log files by `writeLogFile` should contain the `"modelUsage"` key. This allows the backfill tool to extract the model and save it to the database, ensuring that subsequent backfill runs correctly register the run as already backfilled (reporting `backfilled 0` on the second run).

## Actual Behaviour

1. The mock `agent_runs` table is created without the migrated columns. Running the backfill command queries this table, resulting in:
   `error: querying runs: SQL logic error: no such column: model (1)`
2. The mock log file created by `writeLogFile` does not include `"modelUsage"`. The backfill tool parses the model as `""`, leaving the `model` column NULL in the database. Because `model` remains NULL, the query `WHERE metrics_available=0 OR model IS NULL OR model=''` matches the run again on the second backfill pass, so the command reports backfilling the run again (instead of reporting `backfilled 0`).

## Logs / Output

```
=== RUN   TestBackfill_PopulatesUnparseableRuns
    backfill_metrics_test.go:197: backfill exited 1; stdout: ; stderr: error: querying runs: SQL logic error: no such column: model (1)
--- FAIL: TestBackfill_PopulatesUnparseableRuns (1.23s)

=== RUN   TestBackfill_Idempotent
    backfill_metrics_test.go:273: second backfill run should report 'backfilled 0'; got:   backfilled idempotent-run
        
        Scanned 1 runs: backfilled 1 / skipped 0 / errors 0
--- FAIL: TestBackfill_Idempotent (0.03s)
```
