---
title: 'Test Plan: Agent Usage Analytics Report'
type: plan-test
status: in-development
lineage: agent-usage-analytics-report
parent: lifecycle/requirements/agent-usage-analytics-report-2.md
release: KC-Release3
---

# Test Plan: Agent Usage Analytics Report

relates-to: [[agent-usage-analytics-report]]

## Overview

Unit, integration, and component tests covering the backend aggregation
package, the HTTP endpoint, the TTFT capture path, the backfill command, and
the new `ReportsView` frontend. The implementation work is delivered by
[[agent-usage-analytics-report-3-be]] and
[[agent-usage-analytics-report-4-fe]].

---

## Milestone 1 — Backend unit tests: bucket boundaries

### Description

`BucketStart` is a small but boundary-sensitive function that must respect
the supplied timezone for "day" and "week" buckets (per OQ-4). Cover the
edge cases up front because the rest of the aggregation depends on it.

### Files to change

- `internal/reports/agent_usage_test.go` — new test file:
  1. **`TestBucketStart_HourUTC`** — `2026-06-12T14:37:12Z` with `bucket=hour`,
     UTC → `2026-06-12T14:00:00Z`.
  2. **`TestBucketStart_DayBrowserTZ`** — `2026-06-12T14:00:00Z`,
     `bucket=day`, `loc=Australia/Sydney` → `2026-06-13T00:00:00+10:00`
     (because the UTC instant is already past local midnight).
  3. **`TestBucketStart_WeekISO`** — `bucket=week`, document expected start
     (e.g. Monday 00:00 local) and assert.
  4. **`TestBucketStart_DSTBoundary`** — pick a Sydney DST transition (e.g.
     2026-04-05) and assert that the day bucket length is 25 h on that day
     (a regression caused by naively adding `24 h` would be caught here).

### Acceptance criteria

- All tests pass with `go test ./internal/reports/ -run TestBucketStart`.
- Tests fail if anyone replaces the location-aware logic with naive UTC
  arithmetic.

---

## Milestone 2 — Backend unit tests: aggregation

### Description

Test `BuildAgentUsageReport` against an in-memory SQLite index seeded with
fixture runs, covering the matrix in NFR-5: all-success, mixed status, runs
with no result line, non-Claude driver runs, empty window, multi-agent,
multi-model.

### Files to change

- `internal/reports/agent_usage_test.go` — extend with helper
  `seedRuns(t, idx, runs)` and the following tests:

  1. **`TestAggregate_AllSuccess`** — 10 done runs with valid metrics; assert
     `run_count=10`, `success_count=10`, `failure_count=0`,
     `metrics_unavailable_count=0`, `mean_cost_usd` matches the average.

  2. **`TestAggregate_MixedStatus`** — 5 done, 2 failed, 1 killed,
     1 killed-timeout, 1 running. Default status filter (terminal only)
     drops the running row, so `run_count=9`,
     `failure_count=4` (`failed + killed + killed-timeout`).

  3. **`TestAggregate_NoResultLineCounted`** — 5 runs with `metrics_available=0`
     plus 5 with metrics; assert `metrics_unavailable_count=5`,
     `mean_cost_usd` averages only the 5 with metrics, no NaN appears.

  4. **`TestAggregate_NonClaudeDriver`** — Ollama-style runs (no TTFT, no
     tokens). Assert `mean_ttft_ms = nil` (not zero), `mean_output_tokens_per_second = nil`,
     run still counted in `run_count`.

  5. **`TestAggregate_EmptyWindow`** — zero runs in the filter window; assert
     summary fields are zero/nil and `series` is a continuous array spanning
     the requested window.

  6. **`TestAggregate_MultiAgent`** — three agents; assert `per_agent` has 3
     entries and their `run_count` values sum to `overall.run_count`.

  7. **`TestAggregate_MultiModel`** — `claude-opus-4-7` and
     `claude-sonnet-4-6`; assert `per_model` has both, each with the right
     subset of cost and token totals, `series_by_model` has both keys.

  8. **`TestAggregate_StatusFilter`** — supply `Statuses=["failed"]`; assert
     only failed runs are counted and `success_count=0`.

  9. **`TestAggregate_AgentFilter`** — supply `Agents=["qa"]`; assert
     `series_by_agent` is present and only `qa` runs are counted.

  10. **`TestAggregate_PercentileAccuracy`** — 100 runs with known
      durations; assert `median_duration_ms` and `p95_duration_ms` match a
      reference calculation within ±1 ms.

  11. **`TestAggregate_BadFilterTo`** — `to < from`; assert a typed error is
      returned (mapped to 400 by the HTTP layer).

### Acceptance criteria

- All tests pass with `go test ./internal/reports/`.
- Tests cover every NFR-5 backend case.
- A run with `metrics_available=0` never pollutes any mean/sum it should be
  excluded from.

---

## Milestone 3 — Backend unit tests: schema migration

### Description

The migration must be idempotent and additive — re-running it on a database
that already has the columns must not error or drop data.

### Files to change

- `internal/index/index_test.go` — add:

  1. **`TestEnsureAgentRunsTable_AddsNewColumns`** — open a fresh DB, assert
     the new columns exist (query `PRAGMA table_info(agent_runs)`).

  2. **`TestEnsureAgentRunsTable_Idempotent`** — call `ensureAgentRunsTable`
     twice; second call must succeed with no error.

  3. **`TestEnsureAgentRunsTable_BackwardsCompatible`** — manually create a
     pre-migration `agent_runs` table (only the original columns), insert a
     row, then call `ensureAgentRunsTable`. Assert the row survives and the
     new columns return `NULL` for it.

### Acceptance criteria

- All tests pass with `go test ./internal/index/`.
- Existing rows from before the migration are readable as `AgentRunRow`
  with `nil` pointer fields where appropriate.

---

## Milestone 4 — Backend unit tests: metrics persistence and TTFT

### Description

Test the index methods added in
[[agent-usage-analytics-report-3-be]] Milestone 2 + 3 (`UpdateAgentRunMetrics`,
`SetAgentRunModel`, `SetAgentRunTTFT`).

### Files to change

- `internal/index/index_test.go` — add:

  1. **`TestUpdateAgentRunMetrics_PopulatesColumns`** — insert a run, call
     `UpdateAgentRunMetrics`; read back and assert every field round-trips
     and `metrics_available=1`.

  2. **`TestSetAgentRunModel_OverwritesNull`** — insert with no model, call
     `SetAgentRunModel("claude-opus-4-7")`, assert column updates.

  3. **`TestSetAgentRunTTFT_RecordedOnce`** — call twice; assert the *first*
     value wins (the supervisor uses a flag, but the index does not enforce
     this; document the choice and have the test assert whichever behaviour
     the implementation guarantees).

  4. **`TestUpdateAgentRunMetrics_UnknownRunID`** — call against a non-existent
     run; assert error or zero rows affected (whichever the implementation
     returns), no panic.

### Acceptance criteria

- All tests pass.
- Metric writes never corrupt unrelated columns.

---

## Milestone 5 — Backend integration test: aggregation HTTP endpoint

### Description

End-to-end coverage of `GET /api/p/:project/reports/agent-usage` through the
HTTP layer, including auth, validation, defaults, and response shape. Uses
the existing integration test harness (see CLAUDE.md and the auto-memory
note: testEnv auto-logins as admin; devops URL helpers return full URLs for
http.Get).

### Files to change

- `tests/integration/reports_api_test.go` — new test file:

  1. **`TestReportsAgentUsage_Defaults`** — seed runs spanning 60 days; no
     query params; assert response covers the last 30 days at day-bucket
     resolution, status filter implicit (running excluded).

  2. **`TestReportsAgentUsage_ResponseShape`** — assert the JSON payload
     contains `summary.overall`, `summary.per_model`, `summary.per_agent`,
     `series`, `series_by_model`. `series_by_agent` is present iff the
     `agent` query param is set.

  3. **`TestReportsAgentUsage_FilterFrom_To`** — seed runs at known
     timestamps; query a narrow window; assert only runs in the window are
     counted.

  4. **`TestReportsAgentUsage_FilterAgent`** — multi-agent dataset; query
     `?agent=qa&agent=backend-developer`; assert `summary.per_agent` lists
     only those two and `series_by_agent` has both keys.

  5. **`TestReportsAgentUsage_FilterStatus`** — query `?status=failed`;
     assert `success_count=0` and only failed runs included.

  6. **`TestReportsAgentUsage_BadTo_Returns400`** — `to < from`; assert
     HTTP 400 with `apiError("bad_request", …)` shape.

  7. **`TestReportsAgentUsage_UnknownBucket_Returns400`** — `?bucket=year`;
     assert HTTP 400.

  8. **`TestReportsAgentUsage_BadTz_Returns400`** — `?tz=Mars/Phobos`; assert
     HTTP 400.

  9. **`TestReportsAgentUsage_MetricsUnavailableRuns`** — seed mixed runs
     with and without metrics; assert `metrics_unavailable_count` matches
     and the cost/token totals exclude the unavailable runs.

  10. **`TestReportsAgentUsage_Empty`** — fresh project with zero runs;
      assert 200 with `run_count=0` and a continuous series array.

  11. **`TestReportsAgentUsage_Performance10k`** — seed 10,000 synthetic
      runs; assert the endpoint returns within 2 s (NFR-1). Mark the test
      with a `-short`-skip guard so `make test-unit` stays fast — only run
      under the integration target.

  12. **`TestReportsAgentUsage_AuthRequired`** — anonymous request; assert
      the same auth failure response as other project-scoped endpoints.

### Acceptance criteria

- All tests pass with the project's integration test target.
- The performance test fails the suite if NFR-1 regresses.
- No test depends on real Claude Code binaries — seeded rows are written
  directly to the index.

---

## Milestone 6 — Backend integration test: TTFT capture and metrics persistence

### Description

Exercise the supervisor path that records `ttft_ms` and writes metrics on
run finish. Uses a fake driver binary that emits a scripted NDJSON stream
(same pattern as the [[agent-run-summary-panel]] integration tests).

### Files to change

- `tests/integration/agent_metrics_test.go` — new test file:

  1. **`TestSupervisor_PersistsMetricsOnFinish`** — drive a fake Claude Code
     run that emits a `type:result` line; after the run is reported
     finished, query `agent_runs` directly and assert `metrics_available=1`
     and all metric columns are populated.

  2. **`TestSupervisor_NonClaudeRun_NoMetrics`** — drive a fake Ollama run
     that produces no `type:result` line; assert `metrics_available=0` and
     metric columns are `NULL`.

  3. **`TestSupervisor_RecordsTTFT`** — fake driver pauses ~120 ms before
     emitting the first assistant token; assert `ttft_ms` is recorded in
     the 120–200 ms range.

  4. **`TestSupervisor_RecordsTTFTOnce`** — fake driver emits multiple
     assistant tokens; assert `ttft_ms` is written exactly once
     (the second token does not overwrite it).

### Acceptance criteria

- All tests pass.
- No reliance on real model API calls — uses fake driver fixtures.
- Failures in metric writes do not affect the rest of the run lifecycle
  (verified by checking exit code and `agent.finished` event arrive even
  when the metric write is forced to fail in a fault-injection variant).

---

## Milestone 7 — Backfill command tests

### Description

Verify the `backfill agent-run-metrics` CLI subcommand from
[[agent-usage-analytics-report-3-be]] Milestone 4.

### Files to change

- `tests/integration/backfill_metrics_test.go` — new test file:

  1. **`TestBackfill_PopulatesUnparseableRuns`** — seed 5 runs with
     `metrics_available=0` and valid log files containing a `type:result`
     line. Run the backfill. Assert all 5 now have `metrics_available=1`
     and matching metric columns.

  2. **`TestBackfill_SkipsMissingLogs`** — seed a run with no log file. Run
     the backfill. Assert the row is unchanged and the command reports it
     as skipped.

  3. **`TestBackfill_Idempotent`** — run the backfill twice; the second run
     reports zero rows processed (no work to do).

  4. **`TestBackfill_DryRun`** — run with `--dry-run`; assert log
     parsing happens (count reported) but no rows are updated.

### Acceptance criteria

- All tests pass.
- The backfill never modifies log files or other project state.

---

## Milestone 8 — Frontend unit tests: types, store, API client

### Description

Pure-TypeScript tests covering the API URL construction, the Pinia store
state transitions, and debounce behaviour.

### Files to change

- `tests/web/reportsApi.test.ts` — new test file:

  1. **`builds query params from filter`** — all fields set; assert the URL
     contains `from`, `to`, repeated `agent`, repeated `status`, `bucket`,
     `tz`.

  2. **`defaults tz to browser timezone when not supplied`** — leave `tz`
     undefined; assert the URL contains a non-empty `tz` matching
     `Intl.DateTimeFormat().resolvedOptions().timeZone`.

  3. **`omits unset fields`** — only `bucket` supplied; assert no other
     query params are added.

- `tests/web/reportsStore.test.ts` — new test file:

  1. **`fetch sets loading then clears it`** — mock the API; assert
     `loading` toggles correctly on resolve.

  2. **`fetch stores error on failure`** — mock a rejection; assert
     `error` is populated and `loading` returns to `false`.

  3. **`setFilter debounces multiple calls`** — call `setFilter` three
     times within 100 ms; assert only one `fetch` happens (after 300 ms).

  4. **`reset returns defaults`** — mutate filter, call `reset`, assert
     filter equals the default shape.

### Acceptance criteria

- All tests pass via `pnpm --prefix tests/web test` (or the project's
  Vitest target).
- No real network calls — `fetch` / `apiClient` are mocked.

---

## Milestone 9 — Frontend component tests: FilterBar, SummaryTiles, PerModelTable

### Description

Vue Test Utils + Vitest coverage for the static UI pieces.

### Files to change

- `tests/web/ReportsFilterBar.test.ts` — new test file:

  1. **`preset Last 7d emits matching from/to`**.
  2. **`switching to Custom reveals datetime-local inputs`**.
  3. **`agent checkbox toggle emits update with agent array`**.
  4. **`status chip toggle emits update with status array`**.
  5. **`bucket segmented control emits update with bucket value`**.
  6. **`controls are keyboard-navigable (tab order + Enter activates)`**.

- `tests/web/SummaryTiles.test.ts` — new test file:

  1. **`renders six tiles with correct labels and values`**.
  2. **`renders "—" when a tile's summary value is null`**.
  3. **`success rate is formatted as percentage`**.

- `tests/web/PerModelTable.test.ts` — new test file:

  1. **`renders one row per model`**.
  2. **`sort by total cost desc by default`**.
  3. **`column header click toggles ascending/descending`**.
  4. **`Export CSV produces a blob whose rows match the sorted view`**
     (intercept `URL.createObjectURL` and inspect the blob).

### Acceptance criteria

- All tests pass.
- Cover empty-data and sort-toggle states.

---

## Milestone 10 — Frontend component tests: ReportsView integration

### Description

End-to-end (within Vitest + Vue Test Utils) tests of the `ReportsView`
composition — mocked API response, mocked router, mocked store, and DOM
assertions across the FR-5 scenarios listed in NFR-5.

### Files to change

- `tests/web/ReportsView.test.ts` — new test file:

  1. **`renders empty state when run_count is zero`** — mock the report
     with zero runs; assert "No agent runs in this window" is shown and no
     charts mount.

  2. **`renders single-agent dataset`** — mock a report containing one
     `per_agent` entry; assert the per-model table renders and at least one
     chart is mounted.

  3. **`renders multi-agent dataset`** — mock a report with three agents
     and two models; assert tiles + table + all five charts mount.

  4. **`renders error state on API failure`** — mock a 500 response;
     assert the error banner is visible with a Retry button.

  5. **`Retry triggers another fetch`** — click Retry; assert a second API
     call is made.

  6. **`scatter select emits navigation to AgentsRunsView`** — simulate a
     scatter point click; assert `router.push` is called with
     `/p/<project>/agents?run=<runId>`.

  7. **`changing filter triggers refetch`** — change the bucket from `day`
     to `hour`; assert the API is called with `bucket=hour`.

### Acceptance criteria

- All tests pass.
- Tests cover the explicit NFR-5 frontend states (empty, single, multi,
  error).
- No reliance on a live backend.

---

## Milestone 11 — Manual UI smoke test

### Description

A short manual sweep before sign-off, since charting libraries and theme
switches are easy to regress in ways unit tests miss.

### Files to change

- None — this milestone is a checklist for the developer/QA before marking
  the feature `in-qa → approved`.

### Acceptance criteria

- `make build-web && make build && make run` starts the binary, the
  **Reports** entry appears in the left nav.
- Visit `/p/<project>/reports`:
  - Tiles, all five charts, and the per-model table render.
  - Empty-state message shows when narrowing the filter to a zero-run
    window.
  - Toggling dark/light theme repaints charts without a page reload.
  - Scatter-point click opens the corresponding run detail
    (cross-references [[agent-run-summary-panel]]).
  - "Export CSV" downloads a file matching the table contents.
- No console errors or unhandled promise rejections during the above flow.
