---
title: "Test Suite: Backfill Metrics Test Helper Fixes"
type: test
status: draft
lineage: agent-usage-analytics-report
parent: lifecycle/defects/agent-usage-report-backfill-test-failures-7-defect.md
---

This artifact documents the fixes applied to the backfill metrics integration test helpers in `tests/integration/backfill_metrics_test.go` to address the failures described in the parent defect.

## What was fixed

### `createAgentRunsTable` schema (already resolved in prior commit)

The mock `agent_runs` table now includes all analytics columns that the backfill command writes — `model`, `total_cost_usd`, `duration_api_ms`, `num_turns`, `input_tokens`, `cache_creation_tokens`, `cache_read_tokens`, `output_tokens`, `ttft_ms`, and `metrics_available`. This resolves the `SQL logic error: no such column: model` failure.

### `writeLogFile` — added `modelUsage` to result line

The fake log file written by `writeLogFile` (used when `hasResult=true`) now includes a `"modelUsage"` block alongside the other result fields:

```json
"modelUsage":{"claude-sonnet-4-6":{"outputTokens":80,"costUSD":0.015}}
```

This ensures the backfill tool extracts and persists the model name, exercising the `dominantModel` code path in `agent.ParseResultLine`. The fix also future-proofs the idempotency test against any regression that reintroduces a `model IS NULL` condition in the backfill query.

## Scenarios covered

File: `tests/integration/backfill_metrics_test.go`

- `TestBackfill_PopulatesUnparseableRuns` — 5 done runs with valid log files (including `modelUsage`); asserts all 5 have `metrics_available=1` after backfill.
- `TestBackfill_SkipsMissingLogs` — 1 run with no log file; asserts `metrics_available=0` and command exits 0.
- `TestBackfill_Idempotent` — 1 run; first backfill sets `metrics_available=1` and persists model; second backfill reports "backfilled 0" because the idempotency marker is already set.
- `TestBackfill_DryRun` — `--dry-run` flag; asserts output contains "would backfill" and `metrics_available` remains 0.
