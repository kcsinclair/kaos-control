---
title: 'Frontend Plan: Agent Usage Analytics Report'
type: plan-frontend
status: done
lineage: agent-usage-analytics-report
parent: lifecycle/requirements/agent-usage-analytics-report-2.md
release: KC-Release3
---

# Frontend Plan: Agent Usage Analytics Report

relates-to: [[agent-usage-analytics-report]]

## Overview

Add a **Reports** entry to the left navigation, a new `ReportsView.vue` route
at `/p/:project/reports`, and the supporting API client, store, types, and
ECharts-based charts. Depends on the JSON payload defined in
[[agent-usage-analytics-report-3-be]]. The
[[agent-usage-analytics-report-5-test]] plan verifies the UI behaviour.

ECharts is already a dependency (`web/package.json: "echarts": "^6.0.0"`) so
no new package install is required.

---

## Milestone 1 — Types and API client

### Description

Add TypeScript interfaces for the report response and an API client function
that fetches `GET /api/p/:project/reports/agent-usage` with query
parameters.

### Files to change

- `web/src/types/api.ts` — add:
  ```ts
  export interface AgentUsageBucketPoint {
    bucket_start: string  // RFC3339
    run_count: number
    success_count: number
    failure_count: number
    mean_duration_ms: number | null
    mean_cost_usd: number | null
    mean_output_tokens_per_second: number | null
    mean_ttft_ms: number | null
    cache_hit_ratio: number | null
  }

  export interface AgentUsageGroupSummary {
    run_count: number
    success_count: number
    failure_count: number
    metrics_unavailable_count: number
    total_cost_usd: number
    total_input_cost_usd: number
    total_output_cost_usd: number
    total_duration_ms: number
    total_input_tokens: number
    total_cache_creation_tokens: number
    total_cache_read_tokens: number
    total_output_tokens: number
    mean_duration_ms: number | null
    median_duration_ms: number | null
    p95_duration_ms: number | null
    mean_cost_usd: number | null
    mean_output_tokens_per_second: number | null
    mean_ttft_ms: number | null
    p95_ttft_ms: number | null
    cache_hit_ratio: number | null
  }

  export interface AgentUsageSummary {
    overall: AgentUsageGroupSummary
    per_model: (AgentUsageGroupSummary & { model: string })[]
    per_agent: (AgentUsageGroupSummary & { agent_name: string })[]
  }

  export interface AgentUsageReport {
    summary: AgentUsageSummary
    series: AgentUsageBucketPoint[]
    series_by_model: Record<string, AgentUsageBucketPoint[]>
    series_by_agent?: Record<string, AgentUsageBucketPoint[]>
  }

  export interface AgentUsageFilter {
    from?: string         // RFC3339
    to?: string           // RFC3339
    agent?: string[]
    status?: string[]
    bucket?: 'hour' | 'day' | 'week'
    tz?: string           // IANA name; default = browser TZ
  }
  ```

- `web/src/api/reports.ts` — new file:
  ```ts
  export async function getAgentUsageReport(
    project: string,
    filter: AgentUsageFilter,
  ): Promise<AgentUsageReport>
  ```
  - Build a `URLSearchParams` instance. Append `from`, `to`, `bucket`, `tz`
    once if present. Append `agent` and `status` once per array element
    (repeated params, matching the backend contract).
  - Default `tz` to `Intl.DateTimeFormat().resolvedOptions().timeZone` when
    the caller doesn't supply one.
  - Use the project's existing `apiClient` wrapper so auth, error normalisation,
    and base URL handling all behave consistently with other API modules
    (`web/src/api/agents.ts`).

### Acceptance criteria

- Types exactly mirror the backend struct tags from
  [[agent-usage-analytics-report-3-be]] Milestone 5.
- `getAgentUsageReport` constructs the URL with repeated query params and
  always includes a `tz` value.
- `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 2 — Pinia store for reports

### Description

A small store that owns the current filter, the last response, and a
`loading`/`error` flag. Keeping it in Pinia (rather than local component
state) means subsequent navigation back to Reports preserves the user's
last filter selections inside the session.

### Files to change

- `web/src/stores/reports.ts` — new file:
  - State: `filter: AgentUsageFilter`, `report: AgentUsageReport | null`,
    `loading: boolean`, `error: string | null`.
  - Action `fetch(project: string)`:
    1. Set `loading = true`, clear `error`.
    2. Call `getAgentUsageReport(project, filter)`.
    3. On success, store `report`. On failure, set `error` to the message.
    4. Always clear `loading` at the end.
  - Action `setFilter(patch: Partial<AgentUsageFilter>)` that merges and
    triggers a debounced `fetch` (300 ms) so rapid multi-select clicks
    coalesce.
  - Action `reset()` that restores defaults (last 30 days, day bucket,
    browser TZ, no agent/status filters).

### Acceptance criteria

- The store survives view re-mount within the same SPA session — filter
  values persist when leaving and returning to `/reports`.
- Debounce coalesces three rapid `setFilter` calls into a single request.
- `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 3 — Navigation entry and route

### Description

Add the **Reports** menu item to the existing left-nav, wire the route into
the Vue Router, and add a placeholder `ReportsView.vue` so the navigation
works before charts land.

### Files to change

- `web/src/router/index.ts` (or wherever project sub-routes are registered —
  same place as `agents` and `kanban`) — add:
  ```ts
  {
    path: 'reports',
    name: 'reports',
    component: () => import('@/views/project/ReportsView.vue'),
  }
  ```

- The component that renders the project left nav (search for where the
  Agents link is rendered, e.g. `SideNav.vue` / `WorkspaceView.vue`) — add a
  new `<RouterLink :to="...reports...">` with the `BarChart3` icon from
  `lucide-vue-next`. Match existing item markup and class names exactly so
  active-state styling, icon sizing, and collapse behaviour come "for free".

- `web/src/views/project/ReportsView.vue` — placeholder SFC that renders a
  title and "Loading…" while the rest of the view is built. (Replaced by the
  full implementation in Milestone 7.)

### Acceptance criteria

- The **Reports** entry appears in the left nav for all authenticated users
  who can already see **Agents**.
- Clicking it navigates to `/p/:project/reports` and the URL is project-scoped.
- The active-state styling of the nav item matches existing entries.
- No new role gating is introduced (NFR-3).

---

## Milestone 4 — FilterBar component

### Description

A standalone filter bar component that owns the controls for agent (multi-
select), date range (preset shortcuts + custom range), status (multi-select),
and bucket. It emits an `update` event whose handler calls the store's
`setFilter`.

### Files to change

- `web/src/components/reports/ReportsFilterBar.vue` — new SFC:
  - Props: `agents: string[]` (the universe of agent names, supplied by the
    parent from the project config), `filter: AgentUsageFilter`.
  - Emits: `update(patch: Partial<AgentUsageFilter>)`.
  - Date range presets: "Last 24h", "Last 7d", "Last 30d", "Last 90d",
    "Custom" — the first four set both `from` and `to`; "Custom" reveals two
    native `<input type="datetime-local">` controls.
  - Agent multi-select: a popover with checkboxes; "all" when no checkboxes
    are ticked. Reuse the styling pattern from existing multi-select chips
    (e.g. `KanbanBoardView.vue`'s release filter).
  - Status multi-select: chips for `done`, `failed`, `killed`, `killed-timeout`.
  - Bucket: segmented control `hour | day | week`.
  - All controls render inline so the bar is horizontally scrollable on
    narrow viewports.

### Acceptance criteria

- Each control emits an `update` patch matching the affected `AgentUsageFilter`
  field; no other fields are mutated.
- Preset buttons set `from`/`to` in RFC3339, computed relative to `Date.now()`.
- Custom-range inputs are pre-populated from the current `filter.from`/`to`.
- The component is keyboard-navigable and has labels for screen readers.
- `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 5 — Summary tiles + per-model table

### Description

The static, non-chart parts of the dashboard: a row of six summary tiles
and a sortable per-model table with CSV export.

### Files to change

- `web/src/components/reports/SummaryTiles.vue` — new SFC:
  - Props: `summary: AgentUsageSummary`.
  - Renders six tiles in a responsive grid: total runs, success rate %,
    total cost (`$`, 2 dp), mean output tokens/sec (1 dp), mean TTFT
    (`Xm Ys` or `Xs` format), cache hit ratio (%, 1 dp).
  - `null` metric values render as `"—"`.
  - Tile styling matches the `.dash-tile` pattern from `DashboardView.vue`
    (NFR-4).

- `web/src/components/reports/PerModelTable.vue` — new SFC:
  - Props: `rows: (AgentUsageGroupSummary & { model: string })[]`.
  - Columns per FR-5: model, runs, success %, total cost, mean cost/run,
    input cost, output cost, mean output tokens/sec, mean TTFT, cache hit
    ratio, metrics unavailable.
  - Sortable on every column. Default sort: total cost desc.
  - "Export CSV" button: serialises the current sorted rows to CSV
    (`type=text/csv`) and triggers a download via a `Blob` URL. CSV headers
    are the on-screen column labels (OQ-5: per-model summary only).

### Acceptance criteria

- All six tiles render and display `"—"` when the matching summary field is
  `null`.
- Sorting toggles ascending/descending on column header click; sort indicator
  is visible.
- CSV download produces a file whose rows match the on-screen order after
  sorting.
- Component is responsive: tiles wrap to two rows on narrow viewports.

---

## Milestone 6 — ECharts chart components

### Description

Five chart components, each a thin wrapper around ECharts. Centralising the
ECharts lifecycle in a shared composable avoids reinventing init/dispose
logic in each chart.

### Files to change

- `web/src/composables/useECharts.ts` — new file:
  - `useECharts(container: Ref<HTMLElement | null>, option: Ref<EChartsOption>)`:
    initialises the chart on mount, resizes via `ResizeObserver`, disposes on
    unmount, and watches `option` to re-apply via `setOption(option, true)`.
  - Honours the project's existing dark/light theme switch by reading the
    same store/composable used by other components.

- `web/src/components/reports/charts/RunsOverTimeChart.vue` — new SFC.
  - Props: `series: AgentUsageBucketPoint[]`.
  - Renders a stacked bar of success vs. failure per bucket.
  - X-axis labels formatted in the browser TZ (use `Intl.DateTimeFormat` per
    OQ-4).

- `web/src/components/reports/charts/OutputTokensPerSecChart.vue` — new SFC.
  - Props: `seriesByModel: Record<string, AgentUsageBucketPoint[]>`.
  - Renders one line per model using `mean_output_tokens_per_second`.

- `web/src/components/reports/charts/TtftChart.vue` — new SFC.
  - Props: `seriesByModel: Record<string, AgentUsageBucketPoint[]>`.
  - Renders one line per model using `mean_ttft_ms`. Null bucket values are
    rendered as gaps via ECharts `null`-handling.

- `web/src/components/reports/charts/CostPerRunChart.vue` — new SFC.
  - Props: `seriesByModel: Record<string, AgentUsageBucketPoint[]>`.
  - Renders one line per model using `mean_cost_usd`.

- `web/src/components/reports/charts/CostDurationScatter.vue` — new SFC.
  - Props: `points: { run_id, started_at, agent_name, model, duration_ms, total_cost_usd, output_tokens_per_second }[]`.
  - One point per run, coloured by model. Hover reveals run id, agent, model,
    cost, duration, output tokens/sec. Click emits a `select(runId)` event so
    the parent can navigate to `AgentsRunsView` with the run detail open
    (matches FR-5's scatter linking requirement).
  - The raw-run payload is not in the aggregate response, so the parent
    fetches it separately (Milestone 7 includes a small `/agents/runs?limit`
    call to back this chart; the backend handler reuses the existing run
    list endpoint with the same time filter).

### Acceptance criteria

- Each chart renders the requirement-specified data with correct units on the
  y-axis (FR-5 ordering and titles).
- Browser-TZ formatting matches OQ-4: a bucket at `2026-06-12T00:00:00+10:00`
  shows as "Jun 12" for an AEST browser.
- Empty data renders the chart's empty-state placeholder rather than a JS
  error.
- Charts resize to fit their container.
- Theme switching toggles ECharts colours without a page reload.

---

## Milestone 7 — ReportsView wiring

### Description

Compose the navigation entry, filter bar, summary tiles, charts, table, and
loading/error states into the full `ReportsView.vue`. Replace the placeholder
from Milestone 3.

### Files to change

- `web/src/views/project/ReportsView.vue` — full implementation:
  1. `onMounted`: fetch the agent universe from the project config (reuse
     whichever store/composable `AgentsRunsView.vue` uses for the same
     purpose), then call `reportsStore.fetch(project)` if no report is
     cached yet.
  2. Watch `route.params.project` and re-fetch on change.
  3. Layout (top-to-bottom):
     - `<ReportsFilterBar>` bound to the store's filter.
     - `<SummaryTiles :summary="report.summary">`.
     - Five charts in the order specified in FR-5.
     - `<PerModelTable :rows="report.summary.per_model">`.
  4. Loading state: show skeleton placeholders for tiles + charts while
     `loading` is true.
  5. Empty state: when `report` is loaded and `report.summary.overall.run_count`
     is 0, render a centred "No agent runs in this window" message instead
     of charts/table.
  6. Error state: when `error` is set, render a non-blocking alert above the
     content with a "Retry" button that calls `reportsStore.fetch`.
  7. Scatter chart `select(runId)` handler navigates to
     `/p/:project/agents?run=:runId` — `AgentsRunsView.vue` already opens
     the run detail modal when this query param is present (verify before
     wiring; if it doesn't, raise a defect rather than expanding scope).

### Acceptance criteria

- The page renders all six tiles, five charts (in the FR-5 order), and the
  per-model summary table.
- Changing any filter re-fetches and re-renders the dashboard within 2 s on
  a 10k-run dataset (NFR-1).
- Scatter-point click navigates to the matching run detail
  (cross-references [[agent-run-summary-panel]]).
- Empty state renders without console errors or broken canvases.
- The view respects the project's dark/light theme (NFR-4).
- `pnpm build` succeeds with no new TypeScript errors.
