---
title: "Test Plan — DevOps Pipeline Run History"
type: plan-test
status: draft
lineage: devops-pipeline-run-history
parent: lifecycle/requirements/devops-pipeline-run-history-2.md
created: "2026-06-26T00:00:00+10:00"
release: KC-Release4
---

# Test Plan — DevOps Pipeline Run History

Verifies the requirement
[lifecycle/requirements/devops-pipeline-run-history-2.md](../requirements/devops-pipeline-run-history-2.md)
against the backend and frontend plans in the [[devops-pipeline-run-history]]
lineage. Mirrors each Acceptance Criterion in the requirement.

## Context from existing test infrastructure

- Integration tests live in `tests/integration/` with `//go:build integration`;
  run via `make test-integration` (`go test ./... -tags=integration`).
- Harness: `tests/integration/helpers_test.go` — `newTestEnv`,
  `newTestEnvWithCfgYAML`, `env.login/logout`, `env.doRequest`, `readJSON`,
  `requireStatus`, `makeArtifact`. The default env auto-logs-in as admin.
- DevOps harness: `tests/integration/devops_helpers_test.go` — `newDevopsTestEnv(t,
  map[string]string{"<slug>.yaml": <yaml>})`, `writePipelineFiles`,
  `waitForRunComplete`, URL helpers, and `devopsCfgYAML` (roles
  `product-owner`+`devops`; users `admin@test.local`=product-owner,
  `dev@test.local`=devops, `qa@test.local`=qa).
- NDJSON parsing pattern: `runAndFetchNDJSON` in `devops_run_log_test.go`.
- WS capture pattern: `collectDevopsEvents` in `devops_ws_test.go`
  (`env.proj.Hub.Register`).
- On-disk log path in tests: `filepath.Join(env.dataDir, "devops", "testproject",
  runID+".log")` (see `devops_logs_test.go`). New `.meta.json` records live
  beside it.
- Unit tests: `internal/devops/*_test.go`, table-driven, `Test<Func>_<Scenario>`.
- Frontend component tests under `tests/web/` (pnpm); e2e under `tests/e2e/`
  (Playwright, `make test-e2e`).

New test files to add:
- `tests/integration/devops_run_history_test.go` (F1, F2, F6, NF2, NF3)
- `tests/integration/devops_run_history_log_test.go` (F3, NF4)
- `internal/devops/run_record_test.go` (unit: record write/read, back-fill,
  prune, corrupt-skip)
- `tests/web/RunHistory.spec.ts` (F4, F5, F7 component behaviour)
- `tests/e2e/run-history.spec.ts` (F4–F7 smoke)

A `test` artifact will be written to
`lifecycle/tests/devops-pipeline-run-history-6-test.md` after the test code
lands (it documents which files cover which milestone, per the
[[devops-pipeline-log-streaming]] test artifact precedent).

---

## Milestone 1 — Unit tests: run record, back-fill, prune (backend M1, M2, M5, M6; NF2, NF3)

**Description.** Fast, filesystem-level unit tests for the `LogStore` additions,
using `t.TempDir()` and synthesized `.log`/`.meta.json` files — no HTTP.

**Files to change**
- `internal/devops/run_record_test.go` (new).

**Cases**
- `TestWriteRecord_AtomicAndReadable` — write a `RunRecord`, confirm file exists,
  decodes, fields round-trip; assert atomic (no leftover temp file).
- `TestListPipelineRuns_NewestFirstAndFiltered` — seed records for two slugs,
  assert only the requested slug returns, ordered newest-first.
- `TestListPipelineRuns_SkipsCorruptRecord` — one valid + one truncated/garbage
  `.meta.json`; valid one still returned, no error (NF3).
- `TestBackfill_FromLegacyLog` — seed only a `.log` with `run.started`+
  `run.completed` lines, no sidecar; `ListPipelineRuns` derives the record and
  persists a `.meta.json`; unparseable legacy log is skipped (resolved-Q4, NF3).
- `TestPruneOldRuns_KeepsFiftyAndProtectsActive` — seed 55 records; prune keeps
  newest 50, removes both `.meta.json` and `.log` of the rest; a run reported
  active by the `isActive` closure is never removed (NF2).

**Acceptance criteria**
- All cases pass under `go test ./internal/devops/ -run RunRecord` and under
  `make test-unit` (no `integration` tag needed).
- Corrupt-record and unparseable-log cases assert a non-fatal skip, not a panic
  or error return.

---

## Milestone 2 — Integration: history listing endpoint (F2; NF1, NF5)

**Description.** Drive real runs through the HTTP API, then assert the listing
endpoint contract end-to-end, including persistence across a restart.

**Files to change**
- `tests/integration/devops_run_history_test.go` (new).
- `tests/integration/devops_helpers_test.go` — add URL helpers
  `devopsPipelineRunsURL(env, slug)` and
  `devopsPipelineRunLogURL(env, slug, runID)` alongside the existing ones.

**Cases**
- `TestRunHistory_ListNewestFirst` — run a passing then a failing pipeline; GET
  `…/pipelines/{slug}/runs`; assert 2 runs, newest-first, each with exactly
  `run_id, status, started_at, ended_at, duration_ms`; statuses `passed` then
  `failed`; `started_at` valid RFC 3339 (`isRFC3339`).
- `TestRunHistory_LimitDefaultAndCap` — create several runs; assert default
  returns ≤10, `?limit=2` returns 2, `?limit=999` capped at 50.
- `TestRunHistory_EmptyPipeline` — known pipeline, no runs → `200` and `[]`.
- `TestRunHistory_UnknownSlug404` — unknown slug → `404`.
- `TestRunHistory_ForbiddenRole` — `env.login("qa@test.local", …)` → `403`
  (mirrors `TestDevopsRun_ForbiddenRole`); unauthenticated → `401`.
- `TestRunHistory_CancelledRecorded` — start a long pipeline, POST cancel, wait,
  then list → newest run has `status:"cancelled"` (F1/AC).
- `TestRunHistory_PersistsAcrossRestart` — run a pipeline, tear down and
  re-open the project against the **same** `dataDir`, list again → run still
  present (F1/AC "still listed after a server restart"). Add an env helper to
  reopen a project on an existing data dir if one does not already exist.
- `TestRunHistory_Performance50Runs` — seed 50 records; assert the GET responds
  in < 200 ms (NF1) — measure server handler latency, allow CI slack but assert
  a generous upper bound.

**Acceptance criteria**
- Each requirement Acceptance-Criteria bullet for `…/runs` maps to a passing
  case above.
- Restart case proves disk—not memory—is the source of truth.
- Role/empty/unknown-slug cases return the exact status codes specified.

---

## Milestone 3 — Integration: single-run log retrieval (F3; NF1, NF4)

**Description.** Verify the pipeline-scoped log endpoint returns NDJSON, scopes
by slug, and rejects traversal.

**Files to change**
- `tests/integration/devops_run_history_log_test.go` (new).

**Cases**
- `TestRunHistoryLog_ReturnsNDJSON` — run a pipeline, GET
  `…/pipelines/{slug}/runs/{run_id}/log`; assert `Content-Type` is NDJSON and
  every non-empty line parses as JSON (reuse the `runAndFetchNDJSON` scanner
  pattern); content matches the run's logged steps.
- `TestRunHistoryLog_UnknownRunID404` — unknown `run_id` → `404`.
- `TestRunHistoryLog_RunIDFromOtherPipeline404` — request a real `run_id` under
  the wrong `slug` → `404` (scoping check, backend M4).
- `TestRunHistoryLog_ForbiddenRole` — non-devops role → `403`.
- `TestRunHistoryLog_PathTraversalRejected` — `run_id`/`slug` containing `..%2f`
  / `../` → rejected (`400`/`404`), no file read outside the run store (NF4).

**Acceptance criteria**
- Full stored log returned as NDJSON for a known run.
- Cross-pipeline and unknown `run_id` both `404`; traversal rejected.
- Role gate returns `403` for non-devops/non-owner.

---

## Milestone 4 — Integration: live update via WebSocket (F6)

**Description.** Prove that completing a run makes the run available to the
listing endpoint immediately (the data the frontend prepends), driven by the
existing `pipeline.run.completed` event — with no new event type and no polling.

**Files to change**
- `tests/integration/devops_run_history_test.go` (extend).

**Cases**
- `TestRunHistory_LiveCompletionAppears` — register a Hub channel
  (`collectDevopsEvents` pattern), trigger a run, await `pipeline.run.completed`,
  then immediately GET `…/runs` and assert the just-completed run is present and
  newest — confirming the completion event and the persisted record are
  consistent without a refresh delay.
- `TestRunHistory_NoNewEventTypes` — assert the event stream for a run contains
  only the five existing `pipeline.*` types (guards F6 "reusing existing
  events"; no `pipeline.history.*` invented). ([[devops-pipeline-log-streaming]])

**Acceptance criteria**
- The completed run is listable immediately after `pipeline.run.completed`.
- No new WS event types are introduced.

---

## Milestone 5 — Frontend component + e2e (F4, F5, F7)

**Description.** Component-level tests for the upgraded `RunHistory.vue` plus a
Playwright smoke covering the full panel flow against the built binary.

**Files to change**
- `tests/web/RunHistory.spec.ts` (new) — mount `RunHistory` with a mocked
  `devopsApi`:
  - renders rows newest-first with timestamp + duration + status colour/icon;
  - failure row is visually distinct (red class/icon present);
  - empty history shows "No runs yet";
  - panel collapses/expands; Run controls remain present;
  - expanding a row calls `getPipelineRunLog` and renders parsed log lines;
    a second expand collapses the first (single-expand);
  - a rejected `getPipelineRunLog` shows an inline error state (not blank);
  - latest-run summary badge reflects `pipelineHistory[slug][0]` (F7).
- `tests/e2e/run-history.spec.ts` (new) — with a seeded project + pipeline:
  trigger a run via the UI, confirm a history row appears live after completion
  (no manual refresh), expand it to see the log, and confirm the card and group
  header show the latest-run summary.

**Acceptance criteria**
- Component spec passes under `tests/web` (pnpm test) covering F4/F5/F7 branches
  incl. the inline-error and single-expand behaviours.
- e2e smoke passes under `make test-e2e`: live row appears, log expands, summary
  visible on card and group header.
- No regression to live streaming or run triggering (existing devops e2e/specs
  still green). ([[devops-pipelines]], [[devops-pipeline-log-streaming]])

---

## Coverage map (requirement Acceptance Criteria → tests)

| Requirement AC | Covered by |
|---|---|
| Pass/fail persisted, listed after restart | M2 `…PersistsAcrossRestart`, `…ListNewestFirst` |
| Cancelled recorded as `cancelled` | M1 prune/record + M2 `…CancelledRecorded` |
| `…/runs` newest-first w/ 5 fields | M2 `…ListNewestFirst` |
| `?limit=N` honoured, capped at 50 | M2 `…LimitDefaultAndCap` |
| No runs → 200 + [] | M2 `…EmptyPipeline` |
| 403 for other roles, 404 unknown slug/run | M2 `…ForbiddenRole`/`…UnknownSlug404`, M3 `…404` cases |
| `…/log` returns full NDJSON | M3 `…ReturnsNDJSON` |
| Expand loads log inline; error state | M5 component spec |
| Live update, no polling | M4 `…LiveCompletionAppears`/`…NoNewEventTypes` |
| Latest-run summary indicator | M5 component + e2e |
| Path traversal rejected | M3 `…PathTraversalRejected` |
| Retention bounded; active run never pruned | M1 `…PruneOldRuns…` |
| No regression to streaming/triggering | M5 e2e + existing devops suites |
