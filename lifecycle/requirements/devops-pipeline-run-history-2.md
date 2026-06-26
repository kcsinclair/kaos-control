---
title: DevOps Pipeline Run History
type: requirement
status: planning
lineage: devops-pipeline-run-history
created: "2026-06-26T00:00:00+10:00"
priority: normal
parent: lifecycle/ideas/devops-pipeline-run-history.md
labels:
    - feature
    - frontend
    - backend
    - operability
release: KC-Release4
assignees:
    - role: product-owner
      who: agent
---

# DevOps Pipeline Run History

## Problem

The DevOps page lets the Product Owner trigger pipelines and watch a single
run stream live ([[devops-pipelines]], [[devops-pipeline-log-streaming]]), but
once a run finishes its outcome is lost from view. There is no at-a-glance
record of how a pipeline has behaved over its recent executions, and no way to
re-open the log of a past run to diagnose an intermittent failure without
re-triggering the pipeline. Run logs are already persisted on disk at
`~/.kaos-control/devops/<project>/`, yet that history is not surfaced in the UI.
Users cannot answer basic operability questions — "did the last deploy pass?",
"how long does this build usually take?", "what failed yesterday?" — from the
DevOps screen.

## Goals / Non-goals

### Goals

1. Display a recent run history panel beneath each pipeline card/detail,
   listing the last N executions with timestamp, duration, and pass/fail status.
2. Let the user expand any historical run to read its full, stored log output
   in place, without leaving the DevOps screen.
3. Persist run history across server restarts by reading from the existing
   on-disk run store, so history is not limited to the current session.
4. Surface history via a REST endpoint, and keep the panel current during an
   active run by reusing the existing pipeline WebSocket events (no new
   transport, no polling while connected).
5. Make the most recent run's outcome obvious at the pipeline level (a summary
   badge derived from the latest run).

### Non-goals

- **Live streaming UI** — the split-pane live log view is already covered by
  [[devops-pipeline-log-streaming]]; this requirement only adds *historical*
  run listing and on-demand log retrieval.
- **Cross-pipeline analytics** — aggregate dashboards, trend charts, or
  pass-rate reporting across pipelines are out of scope (relates to
  [[agent-usage-analytics-report]]).
- **Log search** — full-text search within or across run logs is out of scope.
- **Retention configuration UI** — pruning policy is a backend concern; no
  user-facing retention settings in this feature.
- **Re-run / replay from history** — triggering a new run is unchanged; this
  feature does not add per-historical-run re-run controls.

## Detailed Requirements

### Functional

#### F1 — Run history persistence

- Each completed run (passed **or** failed) must be recorded in the on-disk run
  store under `~/.kaos-control/devops/<project>/`, keyed by pipeline slug and
  run id.
- Each persisted run record must capture: `run_id`, pipeline `slug`, start
  timestamp (RFC 3339), end timestamp, total `duration`, overall `status`
  (`passed` | `failed` | `cancelled`), and a reference to the stored full log.
- Records must survive a server restart and be readable on next startup.
- Cancelled runs ([[devops-pipelines]] cancel action) must be recorded with
  status `cancelled`.

#### F2 — Run history listing API

- `GET /api/projects/:id/devops/pipelines/:slug/runs` returns the recent run
  history for one pipeline, newest first.
- Response shape:
  `{ runs: [{ run_id, status, started_at, ended_at, duration_ms }] }`.
- The endpoint accepts an optional `limit` query parameter (default **10**,
  maximum **50**); results are capped at the maximum.
- A pipeline with no recorded runs returns `200` with an empty `runs` array.
- An unknown pipeline slug returns `404`.
- Access is restricted to the `product-owner` and `devops` roles (consistent
  with the DevOps page visibility rule); other roles receive `403`.

#### F3 — Single-run log retrieval API

- `GET /api/projects/:id/devops/pipelines/:slug/runs/:run_id/log` returns the
  full stored log for a single past run.
- The log is returned as NDJSON (one JSON line record per output line),
  consistent with the existing run-log endpoint behaviour.
- An unknown `run_id` for the pipeline returns `404`.
- Same role restriction as F2 (`product-owner`, `devops`; else `403`).

#### F4 — History panel UI

- Beneath each pipeline (card and/or detail view) a "Run History" panel lists
  the most recent runs (default 10) newest-first.
- Each row shows, at a glance: relative + absolute start timestamp, duration
  (human-readable, e.g. `1m 12s`), and a pass/fail/cancelled status indicator
  (colour + icon; failure visually distinct, e.g. red).
- The panel must indicate when no runs exist yet ("No runs yet").
- The panel must be collapsible and must not obstruct the existing Run controls.

#### F5 — Expandable run log

- Each history row is expandable; expanding fetches the run's full log via F3
  and renders it in a scrollable, monospaced pane inline within the panel.
- Expanding a second row may collapse the first (single-expand) — a
  predictable, non-overlapping layout is required.
- Log fetch failures must show an inline error state, not a blank pane.

#### F6 — Live update of history during a run

- While the client is connected, completion of a run (the existing
  `pipeline.run.completed` event, plus the cancel path) must cause the history
  panel for that pipeline to prepend the new run without a manual refresh.
- The panel must not poll while a live WebSocket connection is active; on
  initial load or reconnect it fetches via F2.

#### F7 — Latest-run summary

- Each pipeline must display a summary indicator derived from its most recent
  run (status + when it ran), so the latest outcome is visible without
  expanding the history panel.

### Non-functional

- **NF1 — Performance**: F2 must respond within 200 ms for a pipeline with up
  to 50 stored runs. F3 must stream the log progressively for logs up to the
  50,000-line buffer limit defined in [[devops-pipeline-log-streaming]] without
  blocking the response.
- **NF2 — Retention**: The on-disk store must bound growth — retain at least
  the most recent 50 runs per pipeline; older runs may be pruned. Pruning must
  not delete a run that is currently executing.
- **NF3 — Isolation**: A corrupt or unreadable run record must be skipped with
  a server log warning and must not fail the entire F2 listing.
- **NF4 — Security**: `run_id` and `slug` path segments must be validated
  against path traversal; log retrieval must resolve only within the project's
  run store (reuse the sandbox path resolver).
- **NF5 — Observability**: Listing and log-retrieval requests log at DEBUG with
  project, slug, and run id; pruning logs at INFO with the count removed.

## Acceptance Criteria

- [ ] A completed run (pass and fail) is persisted and still listed after a
      server restart.
- [ ] A cancelled run is recorded with status `cancelled`.
- [ ] `GET .../runs` returns runs newest-first with `run_id`, `status`,
      `started_at`, `ended_at`, `duration_ms`.
- [ ] `GET .../runs?limit=N` honours the limit and caps at 50.
- [ ] `GET .../runs` for a pipeline with no runs returns `200` and `[]`.
- [ ] `GET .../runs` and `.../runs/:run_id/log` return `403` for roles other
      than `product-owner`/`devops`, and `404` for unknown slug/run id.
- [ ] `GET .../runs/:run_id/log` returns the full stored log as NDJSON.
- [ ] The history panel renders beneath each pipeline showing timestamp,
      duration, and pass/fail/cancelled status per run.
- [ ] Expanding a history row loads and displays that run's full log inline; a
      fetch error shows an inline error state.
- [ ] Completing a run updates the history panel live (no manual refresh, no
      polling) via the existing WebSocket events. ([[devops-pipeline-log-streaming]])
- [ ] Each pipeline shows a latest-run summary indicator.
- [ ] Path traversal via `slug`/`run_id` is rejected.
- [ ] Per-pipeline retention is bounded (≥ 50 runs retained; an executing run
      is never pruned).
- [ ] No regression to live streaming or run triggering. ([[devops-pipelines]])

## Resolved Questions

1. The on-disk store path is `~/.kaos-control/devops/<project>/`; is the
   `<project>` segment the project **name** or its registration **id**?
   ([[devops-pipelines]] resolved-question wording says name — confirm this is
   still the convention for history lookups.)

> The directory ~/.kaos-control/devops/kaos-control is for the project ID

2. Default history length is specified as 10 with a max of 50 — is 50 an
   acceptable retention ceiling per pipeline, or should retention be larger
   than the display maximum?

> 50 is good.

3. Should the latest-run summary (F7) also appear in the DevOps card grouping
   header, or only on the individual pipeline card/detail?

> Yes in the devops card grouping.

4. For runs that predate this feature (already on disk but possibly without the
   full structured metadata of F1), should they be back-filled/migrated, listed
   best-effort, or ignored?

> back-filled.
