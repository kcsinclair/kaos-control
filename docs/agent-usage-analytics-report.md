# Agent Usage Analytics Report

The agent usage analytics report gives operators a fleet-wide view of how the agent fleet is performing across cost, throughput, latency, and reliability. It is scoped to the currently selected project and accessible via the **Reports** entry in the left navigation menu.

## Contents

- [Accessing the dashboard](#accessing-the-dashboard)
- [Filter bar](#filter-bar)
- [Summary tiles](#summary-tiles)
- [Charts](#charts)
- [Per-model summary table](#per-model-summary-table)
- [API reference](#api-reference)
  - [Query parameters](#query-parameters)
  - [Response shape](#response-shape)
  - [Metric definitions](#metric-definitions)
  - [Error responses](#error-responses)
- [Backfilling historical runs](#backfilling-historical-runs)
- [Data availability and `metrics_unavailable`](#data-availability-and-metrics_unavailable)

---

## Accessing the dashboard

Navigate to any project, then click **Reports** (bar chart icon) in the left navigation. The dashboard loads the last 30 days of terminal runs grouped by day.

The route is `/p/:project/reports`. All authenticated users who can access the Agents view can access Reports — no additional role is required.

---

## Filter bar

The filter bar is at the top of the page. All controls update the dashboard without a page reload; rapid changes are debounced (300 ms) so multiple selections coalesce into a single request.

| Control | Options | Default |
|---|---|---|
| **Date range** | Last 24h · Last 7d · Last 30d · Last 90d · Custom | Last 30 days |
| **Agents** | Multi-select; any configured agent | All agents |
| **Status** | `done` · `failed` · `killed` · `killed-timeout` | All four (all terminal statuses) |
| **Bucket** | Hour · Day · Week | Day |

**Custom date range** — selecting *Custom* reveals two date-time pickers. Times are interpreted in the browser's local timezone; the API receives them as RFC 3339 timestamps.

**Agent filter** — when one or more agents are selected, the response also includes per-agent trend series so you can compare agents directly on the time-series charts.

---

## Summary tiles

Six tiles appear below the filter bar, showing fleet-wide aggregates for the filtered window:

| Tile | Description |
|---|---|
| **Total runs** | Count of all runs matching the current filter |
| **Success rate** | `success_count / run_count × 100`, shown as a percentage |
| **Total cost** | Sum of `total_cost_usd` across all runs with metrics, in USD |
| **Mean output tokens/sec** | Average generation throughput; the primary speed signal |
| **Mean TTFT** | Average time to first token (streaming runs only) |
| **Cache hit ratio** | `cache_read_tokens / (input_tokens + cache_creation_tokens + cache_read_tokens)` averaged over runs with metrics |

Tiles show `—` when the metric is unavailable (e.g. no runs with parseable result data in the window).

---

## Charts

Five charts appear in this order below the tiles.

### 1. Runs over time

A stacked bar chart with one bar per bucket. Green bars show successful runs (`done`); red bars show failures (`failed`, `killed`, `killed-timeout`). Use this to spot reliability regressions or sudden spikes in run volume.

### 2. Output tokens/sec over time

A line chart with one series per model. This is the primary signal for detecting backend slowdowns — a drop in output tokens/sec for a given model indicates increased latency on the Anthropic API or a change in output length patterns.

X-axis labels are formatted in the browser's local timezone.

### 3. Time to first token (TTFT) over time

A line chart with one series per model, showing `mean_ttft_ms` per bucket. TTFT captures queuing and cold-start latency independently from generation speed. Null buckets (no streaming runs) render as gaps in the line.

### 4. Cost per run over time

A line chart with one series per model, showing `mean_cost_usd` per bucket. Use this to track whether average spend per run is trending up or down after prompt or model changes.

### 5. Cost vs. duration scatter

One point per run in the filtered window, coloured by model. Hovering a point shows the run ID, started-at time, agent name, model, cost, duration, and output tokens/sec. Clicking a point navigates to the run detail view for that run.

The scatter chart data is sourced from a separate fetch of the most recent completed runs (up to 20 points) rather than the aggregate response; it refreshes when the project changes.

---

## Per-model summary table

Below the charts, a sortable table lists one row per model present in the filtered window. Columns:

| Column | Description |
|---|---|
| Model | Model identifier (e.g. `claude-opus-4-8`) |
| Runs | `run_count` |
| Success % | `success_count / run_count × 100` |
| Total cost | `total_cost_usd` (USD) |
| Mean cost/run | `mean_cost_usd` (USD) |
| Input cost | `total_input_cost_usd` — covers uncached input tokens + cache writes + cache reads |
| Output cost | `total_output_cost_usd` — covers output tokens |
| Mean tokens/sec | `mean_output_tokens_per_second` |
| Mean TTFT | `mean_ttft_ms` in ms |
| Cache hit ratio | Fraction of token calls served from cache |
| Metrics unavail. | `metrics_unavailable_count` — runs counted but without token/cost data |

Click any column header to sort; click again to reverse. The table defaults to descending `total_cost_usd`.

**Export CSV** — downloads the current sorted table as a CSV file. Rows match the on-screen sort order; columns match the table headers.

---

## API reference

```
GET /api/p/:project/reports/agent-usage
```

Returns aggregated analytics for agent runs in the given project. Requires an authenticated session with access to the project.

### Query parameters

All parameters are optional.

| Parameter | Type | Default | Description |
|---|---|---|---|
| `from` | RFC 3339 timestamp | 30 days before `to` | Start of the window (inclusive) |
| `to` | RFC 3339 timestamp | now | End of the window (inclusive) |
| `agent` | string (repeatable) | all agents | Filter to runs whose `agent_name` matches one of the supplied values. Repeat the parameter to specify multiple: `?agent=qa&agent=backend-developer` |
| `status` | string (repeatable) | `done,failed,killed,killed-timeout` | Filter to runs with a matching status. Valid values: `done`, `failed`, `killed`, `killed-timeout`, `running` |
| `bucket` | `hour` \| `day` \| `week` | `day` | Time-series granularity |
| `tz` | IANA timezone name | `UTC` | Timezone for bucket boundaries. The frontend always sends the browser timezone. Example: `Australia/Sydney` |

**Example — last 7 days for a single agent, bucketed by hour:**

```
GET /api/p/myproject/reports/agent-usage?bucket=hour&agent=qa&from=2026-06-20T00:00:00Z&to=2026-06-27T00:00:00Z&tz=Australia/Sydney
```

### Response shape

```jsonc
{
  "summary": {
    "overall": { /* AgentUsageAggregate */ },
    "per_model": [
      { "model": "claude-opus-4-8", /* AgentUsageAggregate fields */ },
      { "model": "claude-sonnet-4-6", /* AgentUsageAggregate fields */ }
    ],
    "per_agent": [
      { "agent_name": "backend-developer", /* AgentUsageAggregate fields */ },
      { "agent_name": "qa", /* AgentUsageAggregate fields */ }
    ]
  },
  "series": [
    /* one AgentUsageSeriesPoint per bucket across [from, to] */
  ],
  "series_by_model": {
    "claude-opus-4-8": [ /* AgentUsageSeriesPoint[] */ ],
    "claude-sonnet-4-6": [ /* AgentUsageSeriesPoint[] */ ]
  },
  "series_by_agent": {
    /* present only when the `agent` filter is active */
    "qa": [ /* AgentUsageSeriesPoint[] */ ]
  }
}
```

**`AgentUsageAggregate` object:**

```jsonc
{
  "run_count": 142,
  "success_count": 138,
  "failure_count": 4,
  "metrics_unavailable_count": 2,
  "total_cost_usd": 18.42,
  "total_input_cost_usd": 3.11,
  "total_output_cost_usd": 15.31,
  "total_duration_ms": 9843200,
  "total_input_tokens": 1048576,
  "total_cache_creation_tokens": 204800,
  "total_cache_read_tokens": 819200,
  "total_output_tokens": 614400,
  "mean_duration_ms": 69318.3,
  "median_duration_ms": 62000.0,
  "p95_duration_ms": 190000.0,
  "mean_cost_usd": 0.1311,
  "mean_output_tokens_per_second": 8.9,
  "mean_ttft_ms": 1240.0,
  "p95_ttft_ms": 3100.0,
  "cache_hit_ratio": 0.38
}
```

**`AgentUsageSeriesPoint` object:**

```jsonc
{
  "bucket_start": "2026-06-20T00:00:00+10:00",
  "run_count": 12,
  "success_count": 11,
  "failure_count": 1,
  "mean_duration_ms": 71400.0,
  "mean_cost_usd": 0.128,
  "mean_output_tokens_per_second": 9.1,
  "mean_ttft_ms": 1180.0,
  "cache_hit_ratio": 0.41
}
```

Buckets with zero runs are included with `run_count: 0` and all means at `0`, so the `series` array always spans the full `[from, to]` window with no gaps.

### Metric definitions

| Field | How it is computed |
|---|---|
| `success_count` | Runs with `status = "done"` |
| `failure_count` | Runs with `status` ∈ `{failed, killed, killed-timeout}` |
| `metrics_unavailable_count` | Runs whose log produced no `type:result` line, or whose log file was absent. These runs contribute to `run_count` and status counts only. |
| `total_input_cost_usd` | Uncached input tokens + cache-write tokens + cache-read tokens, priced at model list rates and scaled so the sum reconciles to the API-reported `total_cost_usd`. |
| `total_output_cost_usd` | Output tokens priced at model list rates, scaled to reconcile with `total_cost_usd`. |
| `mean_output_tokens_per_second` | `output_tokens / (duration_api_ms / 1000)` computed per run, then averaged over runs with metrics. |
| `mean_ttft_ms` | Time from process spawn to first streamed assistant token, averaged over streaming runs. Batch-mode runs contribute `null` and are excluded from the average. |
| `p95_ttft_ms` | 95th percentile TTFT over runs where TTFT was recorded. |
| `cache_hit_ratio` | `cache_read_tokens / (input_tokens + cache_creation_tokens + cache_read_tokens)` summed across runs with metrics, then divided (a weighted ratio, not a mean of per-run ratios). |
| `median_duration_ms` | 50th percentile of `duration_api_ms` over runs with metrics. |
| `p95_duration_ms` | 95th percentile of `duration_api_ms` over runs with metrics. |

**Input cost vs. output cost split** — the split uses Anthropic list prices by model family prefix:

| Model prefix | Input ($/MTok) | Output ($/MTok) | Cache write ($/MTok) | Cache read ($/MTok) |
|---|---|---|---|---|
| `claude-opus-4` | 5.00 | 25.00 | 6.25 | 0.50 |
| `claude-sonnet-4` | 3.00 | 15.00 | 3.75 | 0.30 |
| `claude-haiku-4` | 1.00 | 5.00 | 1.25 | 0.10 |

The split is computed from these rates and then scaled so the two components always sum exactly to the API-reported `total_cost_usd`. For models not matching any prefix, `total_input_cost_usd` and `total_output_cost_usd` are both `0` — the undivided total is still accurate.

### Error responses

All errors use the shared `apiError` shape:

```jsonc
{ "error": "bad_request", "message": "to before from" }
```

| Status | Condition |
|---|---|
| `400 Bad Request` | `to` is before `from`; invalid `bucket` value; unknown `status` value; unparseable `from` or `to`; invalid `tz` name |
| `500 Internal Server Error` | Database query failure |

---

## Backfilling historical runs

When the server first starts after the schema migration, existing runs have `metrics_available = 0`. The backfill command reads each run's log file, parses the `type:result` line, and persists the metrics to the database. Runs without a log file or result line are marked as permanently unavailable and appear in `metrics_unavailable_count`.

```
kaos-control backfill agent-run-metrics --project <project-name>
```

**Flags:**

| Flag | Default | Description |
|---|---|---|
| `--project <name>` | (required) | Project name as registered in `~/.kaos-control/projects/` |
| `--config <path>` | `~/.kaos-control/config.yaml` | App config file location |
| `--dry-run` | false | Parse and count runs but do not write to the database |

**Dry run — preview without writing:**

```
kaos-control backfill agent-run-metrics --project myproject --dry-run
```

Sample output:
```
  would backfill 67b65204a12f0c2c (model=claude-opus-4-8 cost=0.193200)
  would backfill 75fae7880d34faf7 (model=claude-sonnet-4-6 cost=0.041700)
  skip     901a03e37d03c90d (no result line)

Scanned 3 runs: would backfill 2 / skipped 1 / errors 0
```

The command is safe to re-run: it only processes rows where `metrics_available = 0`, so previously backfilled runs are skipped. If errors occur for individual runs they are printed to stderr and do not stop the rest of the batch; the command exits non-zero if any errors occurred.

---

## Data availability and `metrics_unavailable`

Token counts and cost figures come from the `type:result` JSON line that Claude Code emits at the end of a run. Runs produced by drivers that do not emit this line (e.g. Ollama, or runs that were killed before completion) have no token or cost data. Such runs:

- Are counted in `run_count` and the appropriate status counter.
- Increment `metrics_unavailable_count`.
- Contribute `0` to all cost and token sums — they do not inflate or distort averages.
- Are excluded from means, percentiles, and ratios (`mean_cost_usd`, `p95_duration_ms`, `cache_hit_ratio`, etc. all divide only by the count of runs *with* metrics).

The `metrics_unavailable_count` column in the per-model table tells you how many data points were excluded from the numeric columns for that model row.
