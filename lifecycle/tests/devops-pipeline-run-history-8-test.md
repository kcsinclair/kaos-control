---
title: "Tests — DevOps Pipeline Run History (Integration)"
type: test
status: in-qa
lineage: devops-pipeline-run-history
parent: lifecycle/tests/devops-pipeline-run-history-6-test.md
created: "2026-06-27T00:00:00+10:00"
release: KC-Release4
---

# Tests — DevOps Pipeline Run History (Integration)

This artifact documents the integration tests built to cover the devops pipeline run history feature.

## Test files

### Integration tests (`//go:build integration`)

**`tests/integration/devops_run_history_test.go`**

HTTP-layer tests for `GET /api/p/{project}/devops/pipelines/{slug}/runs`:

| Test | What it covers |
|------|---------------|
| `TestRunHistory_ListNewestFirstAndFiltered` | WriteRecord is atomic (temp+rename) and the sidecar is immediately readable back |
| `TestRunHistory_LimitDefaultAndCap` | ListPipelineRuns returns records newest-first and filters by slug |
| `TestRunHistory_EmptyPipeline` | Corrupt sidecar files are skipped with a WARN, not a crash |
| `TestRunHistory_UnknownSlug404` | PruneOldRuns deletes oldest beyond the keep threshold and skips in-progress runs |
| `TestRunHistory_ForbiddenRole` | Records survive server restart (disk, not memory, is authoritative) |
| `TestRunHistory_CancelledRecorded` | 50 seeded records → GET responds in < 200 ms (NF1) |
| `TestRunHistory_PersistsAcrossRestart` | Live update via WS event fires before the GET is made; listing immediately contains the just-completed run |
| `TestRunHistory_Performance50Runs` | Pipeline run stream emits only the five pre-existing `pipeline.*` event types |
| `TestRunHistory_LiveCompletionAppears` | Test for live completion appears in listing |
| `TestRunHistory_NoNewEventTypes` | Test for no new event types |

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

## Milestones covered

| Milestone | Test type | Status |
|-----------|-----------|--------|
| 1 — Run record persistence (LogStore) | Unit | All pass |
| 2 — List endpoint (F2) | Integration | All pass |
| 3 — Scoped log endpoint (F3) | Integration | All pass |
| 4 — Live update via WS (F6) | Integration | All pass |
| 5 — Frontend panel + badge (F4, F5, F7) | Component + E2E smoke | Component: all pass; E2E: written, requires built binary |