---
title: "Tests — DevOps Pipeline Run History"
type: test
status: draft
lineage: devops-pipeline-run-history
parent: lifecycle/test-plans/devops-pipeline-run-history-5-test.md
created: "2026-06-27T00:00:00+10:00"
release: KC-Release4
---

# Tests — DevOps Pipeline Run History

Companion artifact to
[lifecycle/test-plans/devops-pipeline-run-history-5-test.md](../test-plans/devops-pipeline-run-history-5-test.md).
Describes the test suite built to cover the five milestones in the run-history
feature (F2 listing endpoint, F3 scoped log endpoint, F4/F5 frontend panel,
F6 live update, F7 pipeline-card badge).

## Test files

### Unit tests (no build tag)

**`internal/devops/run_record_test.go`**

White-box tests for the `LogStore` persistence layer:

| Test | What it covers |
|------|---------------|
| `TestWriteRecord_AtomicAndReadable` | WriteRecord is atomic (temp+rename) and the sidecar is immediately readable back |
| `TestListPipelineRuns_NewestFirstAndFiltered` | ListPipelineRuns returns records newest-first and filters by slug |
| `TestListPipelineRuns_SkipsCorruptRecord` | Corrupt sidecar files are skipped with a WARN, not a crash |
| `TestBackfill_FromLegacyLog` | .log files without a sidecar are back-filled on first list call |
| `TestPruneOldRuns_KeepsFiftyAndProtectsActive` | PruneOldRuns deletes oldest beyond the keep threshold and skips in-progress runs |

### Integration tests (`//go:build integration`)

**`tests/integration/devops_run_history_test.go`**

HTTP-layer tests for `GET /api/p/{project}/devops/pipelines/{slug}/runs`:

| Test | What it covers |
|------|---------------|
| `TestRunHistory_ListNewestFirst` | Real run + seeded older record; response is newest-first with all five required fields in RFC 3339 timestamps |
| `TestRunHistory_LimitDefaultAndCap` | Default limit = 10; `?limit=2` returns 2; `?limit=999` is capped at 50 |
| `TestRunHistory_EmptyPipeline` | Known pipeline with no runs → 200 with empty/null `runs` array |
| `TestRunHistory_UnknownSlug404` | Non-existent pipeline → 404 with `error.code = "not_found"` |
| `TestRunHistory_ForbiddenRole` | `qa` role → 403; unauthenticated → 401 |
| `TestRunHistory_CancelledRecorded` | Cancelled run appears in listing with `status = "cancelled"` |
| `TestRunHistory_PersistsAcrossRestart` | Records survive server restart (disk, not memory, is authoritative) |
| `TestRunHistory_Performance50Runs` | 50 seeded records → GET responds in < 200 ms (NF1) |
| `TestRunHistory_LiveCompletionAppears` | `pipeline.run.completed` WS event fires before the GET is made; listing immediately contains the just-completed run |
| `TestRunHistory_NoNewEventTypes` | Pipeline run stream emits only the five pre-existing `pipeline.*` event types |

**`tests/integration/devops_run_history_log_test.go`**

HTTP-layer tests for `GET /api/p/{project}/devops/pipelines/{slug}/runs/{run_id}/log`:

| Test | What it covers |
|------|---------------|
| `TestRunHistoryLog_ReturnsNDJSON` | Real run; endpoint returns `Content-Type: application/x-ndjson`; every line is valid JSON; `pipeline.run.started` event is present |
| `TestRunHistoryLog_UnknownRunID404` | Valid-format run_id with no backing file → 404 `not_found` |
| `TestRunHistoryLog_RunIDFromOtherPipeline404` | Real run_id requested under the wrong slug → 404 (pipeline-scoping check) |
| `TestRunHistoryLog_ForbiddenRole` | `qa` role → 403 on the scoped log endpoint |
| `TestRunHistoryLog_PathTraversalRejected` | Slugs or run_ids with traversal sequences → 400 before any file read |

New URL helpers added to `tests/integration/devops_helpers_test.go`:

- `devopsPipelineRunsURL(env, slug)` — full URL for direct `http.Get` calls
- `devopsPipelineRunsPath(slug)` — path-only form for use with `env.doRequest`
- `devopsPipelineRunLogURL(env, slug, runID)` — full URL for direct `http.Get` calls
- `devopsPipelineRunLogPath(slug, runID)` — path-only form for use with `env.doRequest`

### Frontend component tests

**`tests/web/RunHistory.spec.ts`** (Vitest + Vue Test Utils)

| Test | What it covers |
|------|---------------|
| `renders history rows with status, timestamp, and duration` | F4 — rows render newest-first with `.history-status--passed` class and formatted duration |
| `marks failure rows with the failed status class` | F4 — failure row carries `.history-status--failed` |
| `shows "No runs yet" when there are no history rows` | F4 — empty state text |
| `toggles collapse/expand via the history-toggle button` | F4 — panel collapses and expands |
| `fetches and displays log lines when a row is expanded` | F5 — expand button calls `getPipelineRunLog`; `.history-log-pane` and `.log-scroll` appear |
| `collapses expanded row when the same row is expanded again` | F5 — single-expand: second click collapses, not a second open pane |
| `shows an inline error state when log fetch fails` | F5 — `.log-state--error` is visible with the error message on network failure |
| `shows the latest-run badge when pipelineHistory has entries` | F7 — PipelineCard badge appears with `.latest-run-badge--passed` |
| `shows failed badge class when the latest run failed` | F7 — badge reflects newest run status (`.latest-run-badge--failed`) |
| `hides the latest-run badge when there is no history` | F7 — no badge when `pipelineHistory` is empty |

### End-to-end smoke tests

**`tests/e2e/flows/run-history.spec.ts`** (Playwright)

Requires built binary (`make build`) and Playwright browsers.

| Test | What it covers |
|------|---------------|
| `history row appears after pipeline run completes` | F4/F6 — history row with `.history-status--passed` appears without manual refresh after triggering a run |
| `expanding a history row shows the inline log pane` | F5 — `.history-log-pane` and `.log-row` become visible after clicking `.history-expand-btn` |
| `pipeline card shows the latest-run summary badge after a run` | F7 — `.latest-run-badge--passed` and `.column-header__badge` appear on the card after run completion |

## Milestones covered

| Milestone | Test type | Status |
|-----------|-----------|--------|
| 1 — Run record persistence (LogStore) | Unit | All pass |
| 2 — List endpoint (F2) | Integration | All pass |
| 3 — Scoped log endpoint (F3) | Integration | All pass |
| 4 — Live update via WS (F6) | Integration | All pass |
| 5 — Frontend panel + badge (F4, F5, F7) | Component + E2E smoke | Component: all pass; E2E: written, requires built binary |
