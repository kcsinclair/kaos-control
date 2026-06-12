---
title: 'Backend Plan: Agent Usage Analytics Report'
type: plan-backend
status: done
lineage: agent-usage-analytics-report
parent: lifecycle/requirements/agent-usage-analytics-report-2.md
release: KC-Release3
---

# Backend Plan: Agent Usage Analytics Report

relates-to: [[agent-usage-analytics-report]]

## Overview

Add a project-scoped aggregation endpoint that returns summary + time-series
analytics for `agent_runs`. To meet NFR-1 (≤ 2s on 10k-run projects), token
and cost metrics are **persisted on `agent_runs`** at run finish (per the
resolved OQ-1 answer) rather than parsed from log files on every request. A
one-off backfill command reprocesses historical runs.

The supervisor is also extended to record `ttft_ms` (time to first streamed
content token) so the API can return TTFT trends without re-reading logs.

The [[agent-usage-analytics-report-4-fe]] frontend plan consumes the JSON
shape defined here. The [[agent-usage-analytics-report-5-test]] test plan
verifies parser, persistence, aggregation, and the HTTP contract.

---

## Milestone 1 — Schema migration: add result/metric columns to `agent_runs`

### Description

Add nullable columns to `agent_runs` for the metric fields needed by the
report. Columns are added via `ALTER TABLE` in `ensureAgentRunsTable` so
existing databases migrate forward without a schema rebuild (consistent with
how `denied_tool_calls_json` was added — see `internal/index/index.go:1413`).

New columns (all nullable so legacy rows remain valid):

- `model TEXT`                          — e.g. `claude-opus-4-7`.
- `total_cost_usd REAL`                 — from `type:result`.
- `duration_api_ms INTEGER`             — from `type:result`.
- `num_turns INTEGER`                   — from `type:result`.
- `input_tokens INTEGER`                — from `type:result.usage`.
- `cache_creation_tokens INTEGER`       — from `type:result.usage`.
- `cache_read_tokens INTEGER`           — from `type:result.usage`.
- `output_tokens INTEGER`               — from `type:result.usage`.
- `ttft_ms INTEGER`                     — recorded by the supervisor on first
  streamed content token; `NULL` for batch runs and non-streaming drivers.
- `metrics_available INTEGER NOT NULL DEFAULT 0` — `1` when a parsed
  `type:result` line populated the cost/token columns; `0` otherwise. Used to
  cheaply count `metrics_unavailable_count` in aggregation queries.

### Files to change

- `internal/index/index.go` — `ensureAgentRunsTable`:
  - Append `ALTER TABLE agent_runs ADD COLUMN …` calls for each new column,
    each wrapped in `_, _ = idx.db.Exec(…)` so re-running is idempotent (the
    underlying SQLite driver returns "duplicate column name" which can be
    safely discarded — match the existing `denied_tool_calls_json` pattern).
  - Add a covering index for the report's primary filter dimensions:
    `CREATE INDEX IF NOT EXISTS idx_agent_runs_started_at ON agent_runs(started_at)`.
  - Add a secondary index:
    `CREATE INDEX IF NOT EXISTS idx_agent_runs_agent_name ON agent_runs(agent_name)`.

- `internal/index/index.go` — `AgentRunRow` struct: add the new fields as
  `*float64` / `*int64` (pointers so `nil` distinguishes "not recorded" from
  zero). Update `InsertAgentRun` and `UpdateAgentRun` to round-trip them.

### Acceptance criteria

- Running the binary against an existing database adds the new columns without
  data loss and without erroring on re-run.
- `agent_runs` rows inserted before the migration return `nil`/`NULL` for the
  new fields when read back via `AgentRunRow`.
- `idx_agent_runs_started_at` and `idx_agent_runs_agent_name` exist after
  startup.
- `go vet ./...` and `make test-unit` pass.

---

## Milestone 2 — Persist parsed result on run finish

### Description

When the supervisor records a run as terminal, parse the log via the existing
`agent.ParseResultLine` helper (delivered by [[agent-run-summary-panel-3-be]])
and write the extracted fields onto the `agent_runs` row. This is the
"cache into the database on finish" approach from OQ-1.

Recording `model` happens earlier: at run start, the driver knows which model
was requested and stamps it on the row alongside `agent_name`. (The
`type:result` line also includes a `model` field; if present, prefer the
result value because it reflects the actual model used after fallbacks.)

### Files to change

- `internal/agent/agent.go` — supervisor finish path (the section around the
  existing `agent.finished` / `agent.failed` broadcast, near the
  `ParseResultLine` call introduced by [[agent-run-summary-panel-3-be]]):
  1. After `ParseResultLine` returns a non-nil `RunResult`, populate a new
     `AgentRunMetrics` struct (fields mirror the new columns).
  2. Call `index.UpdateAgentRunMetrics(runID, metrics)` before broadcasting.
  3. If parsing returns `errNoResultLine`, set `metrics_available=0` (default)
     and continue — do not block the broadcast.
  4. Persistence errors are logged but never abort the finish flow.

- `internal/agent/agent.go` — driver setup: capture the model name passed to
  the driver invocation (e.g. `--model claude-opus-4-7`) and call
  `index.SetAgentRunModel(runID, model)` immediately after the row is
  inserted. For drivers with no model concept (Ollama spawn from local
  config), use the model identifier from their config.

- `internal/index/index.go` — add:
  ```go
  type AgentRunMetrics struct {
      Model                 string
      TotalCostUSD          float64
      DurationApiMs         int64
      NumTurns              int
      InputTokens           int64
      CacheCreationTokens   int64
      CacheReadTokens       int64
      OutputTokens          int64
  }
  func (idx *Index) UpdateAgentRunMetrics(runID string, m AgentRunMetrics) error
  func (idx *Index) SetAgentRunModel(runID string, model string) error
  func (idx *Index) SetAgentRunTTFT(runID string, ttftMs int64) error
  ```

### Acceptance criteria

- A Claude Code run that completes with a `type:result` line ends with all
  metric columns populated and `metrics_available=1`.
- An Ollama (or any other) run that produces no result line ends with
  `metrics_available=0` and `NULL` metric columns.
- The supervisor never blocks or fails when the metrics write fails (the
  failure is logged via the existing logger).
- Existing run lifecycle behaviour (status, exit_code, stderr_tail,
  artifacts_produced_json) is unchanged.

---

## Milestone 3 — TTFT capture in the driver

### Description

Record the wall-clock time between process start and the first streamed
content token from the driver. Persist as `ttft_ms` on the run.

The Claude Code driver streams NDJSON; the first event with
`type == "assistant"` and a non-empty `text` content block marks "first
token". Other drivers (Ollama streaming, Codex CLI streaming) have analogous
hooks — capture them where the existing log writer reads stdout.

### Files to change

- `internal/agent/claudecode.go` (and the equivalent stream readers for
  any other streaming driver):
  - In the stdout reader loop, track a `bool firstTokenSeen` and a
    `time.Time runStart` captured before `exec.Cmd.Start()`.
  - When the first qualifying event is observed, compute
    `ttft := time.Since(runStart)`, call
    `index.SetAgentRunTTFT(runID, ttft.Milliseconds())`, and set the flag so
    we never re-record.
  - Errors writing TTFT are logged but never abort the stream.
- For batch-mode drivers (no streaming), do not call `SetAgentRunTTFT`. The
  column stays `NULL` and the aggregator treats those runs as
  "TTFT unavailable".

### Acceptance criteria

- A streaming Claude Code run has `ttft_ms` populated with a positive integer
  matching the time between process spawn and first assistant token (±50 ms).
- Drivers that don't stream leave `ttft_ms` `NULL`.
- A failure to write `ttft_ms` does not affect the rest of the run.

---

## Milestone 4 — Backfill command for historical runs

### Description

Provide a one-off command that walks every row in `agent_runs` with
`metrics_available=0` and a present log file, parses the log via
`ParseResultLine`, and writes the metrics. Designed to be safe to re-run.

A subcommand on the existing CLI binary (e.g.
`kaos-control backfill agent-run-metrics --project <slug>`) is preferred
over a hidden admin endpoint.

### Files to change

- `cmd/kaos-control/main.go` — add a `backfill` subcommand router.

- `cmd/kaos-control/backfill.go` — new file:
  - Load app config, open the named project's SQLite index.
  - Query rows where `metrics_available=0 AND status IN ('done','failed','killed','killed-timeout')`.
  - For each row:
    - Read `p.Agents.LogPath(runID)`.
    - Call `agent.ParseResultLine(logContent)`.
    - On success: call `UpdateAgentRunMetrics`.
    - On `errNoResultLine` or missing log file: leave `metrics_available=0`
      and log "skipped: no result" — these runs will permanently show as
      "metrics unavailable" in the report.
  - Print a summary line at the end: `backfilled N / skipped M / errors E`.
  - Honour a `--dry-run` flag that parses and counts but does not write.

### Acceptance criteria

- Running the command against a project with mixed legacy data updates all
  parseable runs to `metrics_available=1` and leaves Ollama/missing-log runs
  alone.
- Running the command a second time is a no-op (zero rows match the filter).
- `--dry-run` prints the same counts without writing.
- No project file or log file is modified.

---

## Milestone 5 — Aggregation package

### Description

Add `internal/reports/` with a pure-Go aggregation function that accepts a
filter spec and returns the `summary` + `series` JSON tree described in FR-3
and FR-4. The HTTP handler is a thin marshaller around this package, which
keeps the heavy logic unit-testable without spinning up the HTTP server.

### Files to change

- `internal/reports/agent_usage.go` — new file:
  ```go
  type AgentUsageFilter struct {
      From, To  time.Time
      Agents    []string         // empty = all
      Statuses  []string         // empty = default terminal set
      Bucket    string           // "hour" | "day" | "week"
      Loc       *time.Location   // browser TZ for bucket boundaries; required
  }

  type AgentUsageReport struct {
      Summary AgentUsageSummary           `json:"summary"`
      Series  []AgentUsageSeriesPoint     `json:"series"`
      SeriesByModel map[string][]AgentUsageSeriesPoint `json:"series_by_model"`
      SeriesByAgent map[string][]AgentUsageSeriesPoint `json:"series_by_agent,omitempty"`
  }

  func BuildAgentUsageReport(idx *index.Index, f AgentUsageFilter) (*AgentUsageReport, error)
  ```
- The function:
  1. Validates the filter (`to >= from`; `bucket` ∈ {hour,day,week}; unknown
     statuses rejected). Returns a typed error for HTTP layer to map to 400.
  2. Runs **one** SELECT pulling the minimum columns needed
     (`run_id, agent_name, model, started_at, finished_at, status, ttft_ms,
     total_cost_usd, duration_api_ms, input_tokens, cache_creation_tokens,
     cache_read_tokens, output_tokens, metrics_available`) filtered by
     `started_at BETWEEN ? AND ?` and the status/agent IN clauses.
  3. Streams rows into:
     - an `overall` accumulator,
     - a `per_model` map,
     - a `per_agent` map,
     - a `bucket → series-point` map keyed by `bucketStart(startedAt, bucket, loc)`,
     - parallel `series_by_model` and `series_by_agent` maps.
  4. Computes `median` and `p95` by sorting the per-accumulator duration
     slices (sufficient at 10k rows; revisit with t-digest only if NFR-1 is
     missed).
  5. Computes derived fields (`success_rate`, `mean_*`, `cache_hit_ratio`,
     `mean_output_tokens_per_second`) at the end so accumulators only carry
     sums and counts.
  6. Fills zero-run buckets across `[from, to]` so the series is continuous
     (FR-4).

- `internal/reports/agent_usage.go` — also expose `BucketStart(t time.Time, bucket string, loc *time.Location) time.Time` for unit testing.

### Acceptance criteria

- Returns the JSON-tagged shape from FR-3 and FR-4 verbatim.
- A run with `metrics_available=0` increments `run_count` and
  `metrics_unavailable_count` but contributes `0` to cost/token sums and is
  excluded from `mean_cost_usd`, `mean_output_tokens_per_second`,
  `cache_hit_ratio`, and TTFT means (which divide by the count of runs that
  *do* have metrics).
- Bucket boundaries respect the supplied `*time.Location` so a "day" bucket
  starts at local midnight (browser TZ).
- For a 10,000-run project the function returns in well under 2 s on a
  developer laptop (NFR-1).
- Zero-run windows return `run_count=0` and the series array spans the
  requested window without panicking on empty inputs.

---

## Milestone 6 — HTTP endpoint and routing

### Description

Add `GET /reports/agent-usage` under the project-scoped router. The handler
parses query params, builds an `AgentUsageFilter`, calls the aggregator,
and JSON-encodes the result. Auth + project scoping flow through the
existing chi middleware chain (same chain as
`/agents/runs/{run_id}/result`).

### Files to change

- `internal/http/reports.go` — new file with
  `(s *Server) handleGetAgentUsageReport(w http.ResponseWriter, r *http.Request)`:
  1. Parse `from`/`to` with `time.Parse(time.RFC3339, …)`. Default to
     `now-30d` / `now` if absent.
  2. Parse repeated `agent` and `status` query params.
  3. Parse `bucket` (default `day`); reject anything else with 400.
  4. Parse `tz` query param (IANA name, e.g. `Australia/Sydney`); default UTC.
     Look up via `time.LoadLocation`; on failure return 400
     `apiError("bad_request", "invalid tz")`. (The frontend always sends the
     browser TZ per OQ-4.)
  5. Reject `to < from` with `apiError("bad_request", "to before from")`.
  6. Call `reports.BuildAgentUsageReport(p.Index, filter)`.
  7. `writeJSON(w, 200, report)`.

- `internal/http/routes.go` (or wherever the project sub-router is wired —
  match where `/agents/runs/{run_id}/result` was registered) — register:
  ```go
  r.Get("/reports/agent-usage", s.handleGetAgentUsageReport)
  ```
  under the existing `/api/p/{project}` group.

### Acceptance criteria

- `GET /api/p/{project}/reports/agent-usage` (no params) returns 200 with the
  last-30-days/day report using `tz=UTC`.
- All filter params are honoured and a malformed value returns 400 with the
  shared `apiError(...)` shape (matches existing handlers like
  `handleGetAgentRunResult`).
- The endpoint reuses the existing auth + project-scoping middleware — no new
  middleware is added (NFR-3).
- Endpoint returns within 2 s on a 10k-run dataset (NFR-1); the test plan
  exercises this with a seeded dataset.

---

## Milestone 7 — Documentation touchpoints

### Description

Update repo-level documentation so future contributors can find the new
endpoint and table columns.

### Files to change

- `plans/PROJECT_PLAN.md` — bump "Recent Changes"; add the new endpoint to
  the appropriate Planned/Completed list per the CLAUDE.md commit
  conventions.
- `CLAUDE.md` — if the `internal/` package list grows by `internal/reports/`,
  add a one-line description alongside the existing `internal/agent/` etc.
  entry.

### Acceptance criteria

- `plans/PROJECT_PLAN.md` reflects the new endpoint and backfill command.
- `CLAUDE.md` lists the new package if it was created.
- No other documentation is invented.
