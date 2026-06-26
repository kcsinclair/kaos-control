---
title: "Frontend Plan — DevOps Pipeline Run History"
type: plan-frontend
status: in-development
lineage: devops-pipeline-run-history
parent: lifecycle/requirements/devops-pipeline-run-history-2.md
created: "2026-06-26T00:00:00+10:00"
release: KC-Release4
---

# Frontend Plan — DevOps Pipeline Run History

Implements the UI requirements (F4, F5, F6 client side, F7) of
[lifecycle/requirements/devops-pipeline-run-history-2.md](../requirements/devops-pipeline-run-history-2.md),
consuming the REST endpoints defined in the backend plan of the
[[devops-pipeline-run-history]] lineage. Reuses, rather than replaces, the live
log streaming view from [[devops-pipeline-log-streaming]].

## Context from existing code

- DevOps page: `web/src/views/project/DevOpsView.vue` — a split-pane Kanban
  grid grouped by `pipeline.type`, with a per-type **column header**
  (`.column-header`) and `PipelineCard` per pipeline. Subscribes to all
  `pipeline.*` WS events and forwards them to store handlers.
- `web/src/components/devops/PipelineCard.vue` — card; already shows a run
  status badge (`.run-status--{running|passed|failed|cancelled}`) and already
  renders a `RunHistory` child.
- `web/src/components/devops/RunHistory.vue` — **already exists** but is fed
  from session-only WS state (`devops.runHistory`, last 50 runs); shows status,
  start time, duration, a `log` button. **No persistence, no expand-in-place,
  no fetch-on-load.** This plan upgrades it.
- Store: `web/src/stores/devops.ts` — `runHistory: RunHistoryEntry[]`,
  `historyForPipeline(slug)`, WS handlers (`handleRunStarted`,
  `handleRunCompleted`, …), `fetchRunLog`, `loadRunLog`, log buffer.
- API: `web/src/api/devops.ts` — `devopsApi` with `getRunLog`, `parseRunLog`
  (NDJSON → `LogLine[]`), `listPipelines`, etc. Client: `web/src/api/client.ts`
  (`api.get`/`getText`). WS: `web/src/api/ws.ts` + `composables/useWebSocket.ts`.
- Status colours/icons: `.run-status--*` CSS vars; lucide-vue-next icons.

### API-shape decision

The backend exposes the **project id** as the `{project}` path segment and the
existing `devopsApi` calls already use `/p/{project}/devops/...`. New calls
follow that convention:
`GET /p/{project}/devops/pipelines/{slug}/runs` and
`GET /p/{project}/devops/pipelines/{slug}/runs/{runId}/log`.

---

## Milestone 1 — API client + types for history (supports F4, F5)

**Description.** Add typed client functions and interfaces for the two new
endpoints. Keep NDJSON parsing reuse (`parseRunLog`).

**Files to change**
- `web/src/api/devops.ts` — add:
  - `interface RunHistoryRow { run_id; status; started_at; ended_at; duration_ms }`
    and `interface RunsResponse { runs: RunHistoryRow[] }`.
  - `listPipelineRuns(project, slug, limit?=10): Promise<RunsResponse>`
    → `api.get('/p/{project}/devops/pipelines/{slug}/runs?limit=…')`.
  - `getPipelineRunLog(project, slug, runId): Promise<string>`
    → `api.getText('/p/{project}/devops/pipelines/{slug}/runs/{runId}/log')`.
- Reuse existing `parseRunLog` to turn the NDJSON string into `LogLine[]`.

**Acceptance criteria**
- New functions issue requests to the exact paths above with correct
  query/limit encoding.
- Types compile (`pnpm build` / `vue-tsc`) with no `any` leakage on the new
  response shapes.
- A unit/component test can mock these without touching the live network.

---

## Milestone 2 — Store: persisted history per pipeline (F4, F6)

**Description.** Make history authoritative from the server, kept live by WS.
On panel load (or reconnect) fetch via Milestone 1; while connected, the
existing `handleRunCompleted` prepends the just-finished run — **no polling**.

**Files to change**
- `web/src/stores/devops.ts`:
  - Add per-pipeline history map state, e.g.
    `pipelineHistory: Map<string, RunHistoryRow[]>`, plus
    `historyLoading`/`historyError` keyed by slug.
  - `fetchPipelineHistory(project, slug, limit=10)` → calls
    `devopsApi.listPipelineRuns`, stores newest-first rows.
  - Update `handleRunCompleted(payload)` to **prepend** a normalized row for the
    completed run into `pipelineHistory[slug]` (mapping WS `duration_ms`/`status`
    into a `RunHistoryRow`), de-duplicating by `run_id`, and trim to the display
    max. Keep the existing `runHistory`/active-run behaviour intact so live
    streaming is unaffected ([[devops-pipeline-log-streaming]]).
  - Ensure the cancel path (`handleRunCompleted` with `status:'cancelled'`)
    likewise prepends.

**Acceptance criteria**
- Calling `fetchPipelineHistory` populates `pipelineHistory[slug]` newest-first
  from the API response.
- A `pipeline.run.completed` event prepends exactly one row (no duplicate when
  the same run is later re-fetched) and surfaces `cancelled` correctly.
- No interval/polling timer is introduced; history stays current purely via the
  WS handler while connected.

---

## Milestone 3 — Run History panel rendering (F4)

**Description.** Upgrade `RunHistory.vue` to render the persisted rows: each row
shows relative + absolute start time, human-readable duration (`1m 12s`), and a
colour+icon pass/fail/cancelled indicator (failure visually distinct, red). The
panel is collapsible, shows "No runs yet" when empty, and must not obstruct the
Run/Cancel/Edit controls in `PipelineCard`.

**Files to change**
- `web/src/components/devops/RunHistory.vue`:
  - Source rows from `devops.pipelineHistory[slug]` (via a getter); call
    `fetchPipelineHistory` on mount and on reconnect.
  - Add a collapse toggle (default expanded or collapsed — pick collapsed to
    keep cards compact; lucide chevron icon). Persist nothing server-side.
  - Render relative time ("3m ago") + absolute (`toLocaleString`) and duration
    via a shared formatter; status badge reuses `.run-status--*` classes + a
    lucide icon per status (e.g. check / x / slash-circle).
  - Empty state: "No runs yet".
- `web/src/components/devops/PipelineCard.vue` — ensure the panel sits below the
  card actions and the collapse doesn't shift the Run controls.
- (Optional) `web/src/composables/` — add a small `formatRelativeTime` /
  `formatDurationMs` util (reuse `useNow()` for live relative ticks) to avoid
  duplicating the ad-hoc formatters currently in `RunHistory.vue` /
  `PipelineLogPane.vue` / `StepProgress.vue`.

**Acceptance criteria**
- Panel lists the most recent runs (default 10) newest-first beneath each
  pipeline, each row showing relative+absolute timestamp, duration, and a
  coloured status indicator with failure clearly distinct (red).
- Empty pipeline shows "No runs yet".
- Panel collapses/expands; the Run/Cancel/Edit buttons remain reachable and
  unmoved in their primary position.

---

## Milestone 4 — Expandable run log in place (F5)

**Description.** Make each history row expandable; expanding fetches that run's
full log via Milestone 1 and renders it in a scrollable monospaced pane inline.
Single-expand (expanding one collapses the previously expanded). Fetch failures
show an inline error, not a blank pane.

**Files to change**
- `web/src/components/devops/RunHistory.vue`:
  - Track `expandedRunId` (single value → single-expand behaviour).
  - On expand, call `devopsApi.getPipelineRunLog(project, slug, runId)` →
    `parseRunLog` → render `LogLine[]` in an inline scrollable `<pre>`/monospace
    pane. Reuse the line-rendering style from `PipelineLogPane.vue` (kinds:
    run-start/step-start/output/step-end/run-end, ok/fail colours) — extract a
    presentational sub-component if it reduces duplication.
  - Loading state while fetching; inline error state on failure with a retry.
  - This is **on-demand historical** retrieval — it does **not** touch the live
    `logBuffer` / split-pane streaming view ([[devops-pipeline-log-streaming]]);
    no regression to the live "View log" affordance.

**Acceptance criteria**
- Expanding a row loads and displays that run's full log inline in a scrollable
  monospaced pane.
- Expanding a second row collapses the first (single-expand); layout is
  predictable and non-overlapping.
- A failed log fetch shows an inline error (with retry), never a blank pane.
- The live split-pane streaming view continues to work for active runs.

---

## Milestone 5 — Latest-run summary on card and group header (F7, resolved-question 3)

**Description.** Show a summary indicator derived from each pipeline's most
recent run (status + when it ran) on the pipeline card/detail **and** in the
DevOps type/group column header, so the latest outcome is visible without
expanding history.

**Files to change**
- `web/src/components/devops/PipelineCard.vue` — derive latest-run summary from
  `pipelineHistory[slug][0]` (fallback to the active run if one is in flight);
  render a compact badge (status colour/icon + relative time).
- `web/src/views/project/DevOpsView.vue` — in each type `.column-header`, render
  an aggregate latest-run indicator for that group (e.g. worst/most-recent
  status across the group's pipelines, per resolved-question 3 "yes in the
  devops card grouping"). Keep it lightweight; derive from store state already
  loaded for the group's cards.
- Add a store getter `latestRunForPipeline(slug)` if it simplifies templates.

**Acceptance criteria**
- Each pipeline card shows a latest-run summary (status + when) without
  expanding the history panel.
- Each type/group column header shows a group-level latest-run summary.
- The summary updates live when a run completes (driven by the Milestone 2 WS
  prepend), with no manual refresh.

---

## Out of scope (per requirement Non-goals)

No new trend charts / cross-pipeline analytics, no log search box, no
retention-settings UI, no per-historical-run re-run button. Live updates reuse
the existing `pipeline.run.completed` WS event only — no new transport and no
polling while connected.
