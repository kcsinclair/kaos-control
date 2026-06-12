---
title: "Test Suite: Agent Usage Analytics Report"
type: test
status: draft
lineage: agent-usage-analytics-report
parent: lifecycle/test-plans/agent-usage-analytics-report-5-test.md
---

This artifact describes the test files that implement the agent-usage-analytics-report test suite.

## Milestone 1 — BucketStart unit tests

File: `internal/reports/agent_usage_test.go`

Four tests covering the `BucketStart` and `nextBucket` functions in `internal/reports/agent_usage.go`:

- `TestBucketStart_HourUTC` — verifies truncation to the start of the hour in UTC.
- `TestBucketStart_DayBrowserTZ` — verifies UTC 14:00 maps to midnight Sydney on the next calendar day (UTC+10).
- `TestBucketStart_WeekISO` — verifies ISO Monday alignment for a Friday input.
- `TestBucketStart_DSTBoundary` — verifies that `nextBucket` uses `AddDate` (timezone-aware) rather than `Add(24h)`, so a day bucket spanning the Sydney DST end is 25 hours long.

## Milestone 2 — Aggregation unit tests

File: `internal/reports/agent_usage_test.go`

Eleven tests exercising `BuildAgentUsageReport` and the underlying accumulator logic against a live in-memory SQLite index:

- `TestAggregate_AllSuccess` — 10 done runs; assert run_count, success_count, and mean_cost_usd.
- `TestAggregate_MixedStatus` — mixed statuses including "running"; verifies default status filter excludes running.
- `TestAggregate_NoResultLineCounted` — runs without metrics; verifies metrics_unavailable_count and NaN-free mean_cost_usd.
- `TestAggregate_NonClaudeDriver` — Ollama-style runs with no TTFT or metrics; verifies MeanTTFTMs=0 and MeanOutputTokensPerSecond=0.
- `TestAggregate_EmptyWindow` — no runs; verifies run_count=0 and non-empty series (bucket fill).
- `TestAggregate_MultiAgent` — three agents; verifies per_agent count and that run_count sums correctly.
- `TestAggregate_MultiModel` — opus and sonnet runs; verifies per_model count, series_by_model, and cost totals.
- `TestAggregate_StatusFilter` — explicit status=["failed"] filter; verifies success_count=0.
- `TestAggregate_AgentFilter` — explicit agents=["qa"] filter; verifies series_by_agent is populated.
- `TestAggregate_PercentileAccuracy` — 100 runs 1–100 ms; verifies p50 and p95 using the floor formula.
- `TestAggregate_BadFilterTo` — To < From; verifies ErrBadFilter is returned.

## Milestone 3 — Index schema migration unit tests

File: `internal/index/index_test.go` (appended)

Three tests against the `ensureAgentRunsTable` migration function:

- `TestEnsureAgentRunsTable_AddsNewColumns` — fresh index; PRAGMA table_info confirms all analytics columns exist.
- `TestEnsureAgentRunsTable_Idempotent` — second call to ensureAgentRunsTable returns no error.
- `TestEnsureAgentRunsTable_BackwardsCompatible` — recreate table with legacy schema, insert row, run migration, assert row survives with nil analytics columns.

## Milestone 4 — Metrics persistence and TTFT unit tests

File: `internal/index/index_test.go` (appended)

Four tests for the index analytics write methods:

- `TestUpdateAgentRunMetrics_PopulatesColumns` — verifies all metric columns round-trip and metrics_available=1.
- `TestSetAgentRunModel_OverwritesNull` — verifies SetAgentRunModel sets the model column on a null row.
- `TestSetAgentRunTTFT_RecordedOnce` — verifies last write wins at DB level (plain UPDATE); documents that the supervisor enforces single-write via firstTokenSeen.
- `TestUpdateAgentRunMetrics_UnknownRunID` — verifies no panic on non-existent run ID.

## Milestone 5 — Reports API integration tests

File: `tests/integration/reports_api_test.go`

Twelve integration tests for `GET /api/p/:project/reports/agent-usage`:

- Defaults (30-day window, old runs excluded), response shape, from/to filter, agent filter, status filter, bad to/from (400), unknown bucket (400), bad timezone (400), metrics unavailable count, empty project, 10k performance (<2s), and auth required (401).

## Milestone 6 — Supervisor metrics integration tests

File: `tests/integration/agent_metrics_test.go`

Four tests for the supervisor's metrics-persistence path using fake claude scripts:

- `TestSupervisor_PersistsMetricsOnFinish` — result line emitted → metrics_available=1.
- `TestSupervisor_NonClaudeRun_NoMetrics` — no NDJSON output → metrics_available=0.
- `TestSupervisor_RecordsTTFT` — sleep 120ms before first assistant event → ttft_ms in [80, 500].
- `TestSupervisor_RecordsTTFTOnce` — two assistant events emitted; ttft_ms > 0.

## Milestone 7 — Backfill CLI integration tests

File: `tests/integration/backfill_metrics_test.go`

Four tests for the `kaos-control backfill agent-run-metrics` command. Uses a compiled binary (built once via `sync.Once`) and seeds SQLite directly:

- `TestBackfill_PopulatesUnparseableRuns` — 5 runs with valid logs → all metrics_available=1.
- `TestBackfill_SkipsMissingLogs` — run with no log file → metrics_available=0, command succeeds.
- `TestBackfill_Idempotent` — second run reports "backfilled 0".
- `TestBackfill_DryRun` — `--dry-run` outputs "would backfill" but does not modify the DB.

## Milestone 8 — Frontend unit tests

Files: `tests/web/reportsApi.test.ts`, `tests/web/reportsStore.test.ts`, `tests/web/SummaryTiles.test.ts`, `tests/web/PerModelTable.test.ts`, `tests/web/ReportsFilterBar.test.ts`, `tests/web/ReportsView.test.ts`

- **reportsApi**: query param construction, timezone defaulting, omission of unset fields.
- **reportsStore**: loading flag lifecycle, error handling, debounce on setFilter, reset to defaults.
- **SummaryTiles**: 6 tiles rendered, success rate "—" at zero runs, percentage formatting.
- **PerModelTable**: row count, default sort desc by cost, sort toggle aria-sort, CSV export content.
- **ReportsFilterBar**: Last 7d preset from/to, Custom preset reveals datetime inputs, agent checkbox toggle, status chip toggle, bucket segmented control, keyboard accessibility.
- **ReportsView**: empty state, single/multi-agent rendering, error banner with Retry, Retry refetch, scatter select navigation, filter change triggers refetch after debounce.
