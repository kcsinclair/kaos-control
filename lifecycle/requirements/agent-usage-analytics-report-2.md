---
title: Agent Usage Analytics Report
type: requirement
status: blocked
lineage: agent-usage-analytics-report
priority: normal
parent: lifecycle/ideas/agent-usage-analytics-report.md
labels:
    - agent
    - runs
    - observability
    - feature
    - frontend
    - backend
release: KC-Release3
assignees:
    - role: product-owner
      who: agent
---

# Agent Usage Analytics Report

## Problem

Operators have no aggregated view of how the agent fleet is performing. Today, `agent_runs` rows are visible per-run in `AgentsRunsView.vue`, and the per-run summary panel (see [[agent-run-summary-panel]]) parses the terminal `type:result` JSON line on demand to show cost, duration, turn count, and token usage for one run at a time. There is no way to answer questions like *"is the qa agent getting slower week-on-week?"*, *"which agent is eating the most spend?"*, or *"are truncated/failed runs trending up?"* without manually opening individual logs. As agent usage scales, the lack of fleet-wide visibility makes cost regressions, performance degradation, and reliability issues invisible until they become severe.

## Goals / Non-goals

### Goals

- Add a new **Reports** entry in the left navigation menu that routes to a single-page analytics dashboard.
- Provide aggregated metrics across all historical agent runs: timing (wall-clock duration, API duration), cost (`total_cost_usd`), token usage (input, cache create, cache read, output), and run counts by status.
- Surface time-series trends so operators can see whether average duration, cost-per-run, or failure rate is improving or degrading over time.
- Allow filtering by agent name, date range, and run status so the data is actionable for a specific investigation.
- Render the dashboard with charts plus a sortable, exportable summary table.

### Non-goals

- Per-run drill-down — the existing run detail modal (see [[agent-run-summary-panel]]) already covers single-run analysis. The dashboard links to that view; it does not replicate it.
- Live streaming updates to the dashboard. A page refresh or explicit "refresh" action is sufficient for v1; WebSocket-driven live tiles are out of scope.
- Cost projections, budget alerts, or billing integration. The dashboard reports observed cost only — no forecasting and no alerting.
- Cross-project aggregation. Reports are scoped to the currently selected project, matching the rest of the app.
- Token metrics for drivers that do not emit `type:result` (e.g. Ollama, codex-cli, gemini-cli where applicable). Those runs are counted for timing and status, but their token/cost cells render as "N/A".
- Persisting historical aggregates indefinitely outside the existing `agent_runs` table and per-run log files. No new long-term retention policy is introduced.

## Detailed Requirements

### Functional

#### FR-1: Reports navigation entry

- Add a new top-level entry labelled **Reports** to the left navigation menu, with an appropriate `lucide-vue-next` icon (e.g. `BarChart3`).
- The entry routes to `/p/:project/reports` and is project-scoped.
- Visible to all authenticated roles that can already see the Agents view; no new role gate is introduced.

#### FR-2: Backend aggregation endpoint

- Add `GET /api/p/:project/reports/agent-usage` returning a JSON document with two top-level objects: `summary` (overall + per-agent aggregates) and `series` (time-bucketed trend data).
- Query parameters (all optional):
  - `from` — RFC3339 timestamp; default = 30 days before `to`.
  - `to` — RFC3339 timestamp; default = now.
  - `agent` — repeated string; filters to runs whose `agent_name` matches one of the values. Omitted = all agents.
  - `status` — repeated string from `{done, failed, killed, killed-timeout, running}`; default = all terminal statuses (`running` excluded by default).
  - `bucket` — one of `hour`, `day`, `week`; default `day`.
- Run metadata (counts, durations, status, agent name, started_at, finished_at) is sourced from the `agent_runs` SQLite table.
- Token and cost data is parsed from each run's log file via `agent.ParseResultLine` (the same helper used by `GET /agents/runs/{run_id}/result`). Runs whose log file is missing, unparseable, or produced by a driver that does not emit `type:result` contribute `null` for token/cost fields and are counted in a `metrics_unavailable_count` field.
- See OQ-1 for the open architectural question on aggregation strategy (on-the-fly log parsing vs. persisted result columns on `agent_runs`).

#### FR-3: Summary aggregates

The `summary` block of the response includes:

- `overall`: `run_count`, `success_count`, `failure_count` (failed + killed + killed-timeout), `metrics_unavailable_count`, `total_cost_usd`, `total_duration_ms`, `total_input_tokens`, `total_cache_creation_tokens`, `total_cache_read_tokens`, `total_output_tokens`, `mean_duration_ms`, `median_duration_ms`, `p95_duration_ms`, `mean_cost_usd`, `mean_tokens_per_minute` (computed as `(input + cache_creation + cache_read + output) / (duration_ms / 60000)` averaged over runs with metrics).
- `per_agent`: an array of objects, one per `agent_name` present in the filtered set, each with the same fields as `overall` plus the `agent_name` key.

#### FR-4: Trend series

The `series` block of the response includes one entry per time bucket within `[from, to]`:

- `bucket_start` — RFC3339 timestamp at the start of the bucket.
- `run_count`, `success_count`, `failure_count`, `mean_duration_ms`, `mean_cost_usd`, `mean_tokens_per_minute`.
- Buckets with zero runs are included with zero values (so charts render a continuous x-axis).
- When `agent` filter contains multiple values, the response also includes `series_by_agent`: a map of `agent_name → [{bucket_start, run_count, mean_duration_ms, mean_cost_usd}, …]` so the frontend can render multi-series charts without re-aggregating.

#### FR-5: Reports dashboard page

- Route `/p/:project/reports` renders a new `ReportsView.vue` page.
- Top of page: filter bar with controls for agent (multi-select), date range (preset shortcuts: Last 24h, Last 7d, Last 30d, Last 90d, Custom), status (multi-select), and bucket size.
- Below the filter bar, a row of summary tiles showing: total runs, success rate %, total cost, total tokens, mean duration, mean tokens/minute.
- Charts (in this order, each with a clear title and unit on the y-axis):
  1. **Runs over time** — stacked bar (success / failure) per bucket.
  2. **Mean run duration over time** — line, one series per selected agent (or one aggregate line when no agent filter).
  3. **Cost per run over time** — line, same series rules as duration.
  4. **Cost vs. duration scatter** — one point per run in the filtered window; colour-coded by agent; hover reveals run id, started_at, agent, cost, duration. Linked: clicking a point navigates to the run detail.
- Below the charts: a sortable summary table with one row per agent (the `per_agent` array from FR-3). Columns: agent, runs, success %, total cost, mean cost/run, mean duration, mean tokens/minute, metrics unavailable.
- An "Export CSV" button downloads the summary table as CSV.
- Empty state: when there are zero runs in the filtered window, charts and table render an empty-state message ("No agent runs in this window") rather than blank canvases.

#### FR-6: Charting library

- Reuse the existing UI dependencies where practical. Three.js and Cytoscape are not appropriate for 2D analytics charts.
- Add a lightweight charting library — recommendation: **Chart.js** (small bundle, well-supported, native dark/light theming via CSS vars). The implementation plan should confirm and pin a version. (See OQ-2.)

### Non-functional

#### NFR-1: Performance

- The aggregation endpoint must return in ≤ 2 seconds for a project with ≤ 10,000 historical runs on a developer laptop.
- For larger projects, the implementation plan must specify the strategy (caching, indexed columns, or background materialization). Worst-case response under any filter combination on a 10k-run project must not exceed 10 seconds.
- The dashboard must render the first chart within 500 ms of the response arriving (no client-side blocking aggregation over the raw run list).

#### NFR-2: Graceful degradation

- Runs with missing or unparseable result lines must not cause the endpoint to fail; they contribute to `run_count` and status counts only, and increment `metrics_unavailable_count`.
- If `to < from`, return HTTP 400 with `apiError("bad_request", …)`.
- If the project has no agent runs at all, the endpoint returns a well-formed response with zero counts; the dashboard renders the empty state.

#### NFR-3: Auth & scoping

- The endpoint reuses the existing project-scoped middleware (same chain as `GET /agents/runs/{run_id}/result`).
- The dashboard route is gated behind the existing authenticated-user check; no new role permissions are added.

#### NFR-4: Visual consistency

- Reuse Tailwind utilities and CSS variables already present in the project. Match spacing, card border radius, and colour palette of `AgentsRunsView.vue` and `DashboardView.vue`.
- Honour the project's existing dark/light theme.

#### NFR-5: Testability

- Backend: unit tests for the aggregation function with fixture runs covering — all-success, mixed status, runs with no result line, runs from a non-Claude driver, empty window, multi-agent.
- Backend: an integration test (under `tests/`, see CLAUDE.md for the test harness) hits `GET /reports/agent-usage` with realistic data and asserts the response shape and key aggregates.
- Frontend: component test for `ReportsView.vue` rendering with a mocked response (covering empty state, single agent, multi-agent, error response).

## Acceptance Criteria

- [ ] A **Reports** entry appears in the left navigation menu and routes to `/p/:project/reports`.
- [ ] `GET /api/p/:project/reports/agent-usage` returns a `summary` and `series` payload conforming to FR-3 and FR-4 for a project with mixed-status historical runs.
- [ ] Filter query parameters (`from`, `to`, `agent`, `status`, `bucket`) correctly restrict the aggregated result; an unfiltered request returns the last 30 days of data with `bucket=day`.
- [ ] Runs without a parseable `type:result` line (e.g. Ollama, missing log file) contribute to `run_count` and `metrics_unavailable_count` but do not pollute cost/token aggregates with `NaN` or zeros.
- [ ] The dashboard renders six summary tiles, four charts, and the per-agent summary table populated from the response.
- [ ] Changing the agent, date range, status, or bucket filter re-fetches and re-renders within 2 seconds for a 10,000-run dataset.
- [ ] The scatter chart links each point to the corresponding run detail view (see [[agent-run-summary-panel]]).
- [ ] The "Export CSV" button downloads a CSV whose rows match the on-screen summary table.
- [ ] An empty result window shows the empty state, not broken charts.
- [ ] Unit, integration, and frontend tests covering the cases listed under NFR-5 pass via `make test-unit` and the integration test target.
- [ ] Backend and frontend implementation plans ([[agent-usage-analytics-report]] be/fe stages) exist and gate `planning → in-development` per project config.

## Open Questions

- **OQ-1 (architecture):** Should token/cost aggregates be computed by parsing each run's log file on every request, or by persisting `total_cost_usd`, `duration_api_ms`, `input_tokens`, `cache_creation_tokens`, `cache_read_tokens`, `output_tokens` columns on the `agent_runs` table at run finish? Persistence is faster at query time but requires a schema migration and a backfill routine for historical runs. The implementation plan should pick one approach and justify it under the NFR-1 budget.
- **OQ-2 (charting library):** Confirm Chart.js as the dependency choice, or propose an alternative. The frontend already ships three.js (3D graph) and Cytoscape (2D graph) — neither is suitable for analytics charts. A separate library is required.
- **OQ-3 (per-agent vs. per-role grouping):** Should the dashboard group by `agent_name` (e.g. `requirements-analyst`) or by `role` (e.g. `analyst`)? `role` collapses multiple agents into one series, which may be more useful at the fleet level. v1 ships per-agent grouping; per-role can be added later if useful.
- **OQ-4 (timezone handling):** Bucket boundaries are calendar-relative (a "day" bucket starts at midnight). Should boundaries use UTC, or the server's local timezone, or the browser's timezone? Default proposal: UTC for the API; the frontend formats bucket labels in the browser timezone for display.
- **OQ-5 (CSV export scope):** Should "Export CSV" emit the per-agent summary only (current proposal), the time-series, or the raw run list? Per-agent summary is sufficient for v1 reporting; if operators want raw run export, that becomes a separate feature.
