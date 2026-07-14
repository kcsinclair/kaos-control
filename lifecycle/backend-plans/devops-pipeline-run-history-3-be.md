---
title: "Backend Plan — DevOps Pipeline Run History"
type: plan-backend
status: done
lineage: devops-pipeline-run-history
parent: lifecycle/requirements/devops-pipeline-run-history-2.md
created: "2026-06-26T00:00:00+10:00"
release: KC-Release4
---

# Backend Plan — DevOps Pipeline Run History

Implements the persistence, REST, retention, and security requirements (F1–F3,
F6 server side, NF1–NF5) of the requirement
[lifecycle/requirements/devops-pipeline-run-history-2.md](../requirements/devops-pipeline-run-history-2.md).
Pairs with the frontend plan and test plan in the
[[devops-pipeline-run-history]] lineage.

## Context from existing code

- Run logs already persist as NDJSON at `~/.kaos-control/devops/<project>/<run_id>.log`
  via `internal/devops/logger.go` (`LogStore`). `<project>` is the **project id**
  (resolved-question 1).
- `LogStore.ListRuns(projectName)` already exists but reconstructs each run by
  parsing the whole `.log` file and returns a thin `RunSummary{RunID, Pipeline,
  StartTime, Status}` — no end time, duration, per-pipeline filter, and
  `cancelled` is not distinguished from `failed`.
- Existing route `GET /api/p/{project}/devops/runs/{run_id}` (`handleGetRunLog`,
  `internal/http/devops.go:195`) already returns a run log as NDJSON via
  `ReadLogNDJSON`.
- Event hook in `internal/project/project.go:198` already forwards every run
  event (`pipeline.run.started` … `pipeline.run.completed`) to the `LogStore`.
- Role gating uses `requireRole(w, r, p, RolesDevopsOrAdmin...)`
  (`internal/http/permissions.go`; `RolesDevopsOrAdmin = {product-owner, devops}`).
- Path safety: `internal/sandbox` `Resolve(root, userPath)` rejects traversal
  and absolute paths.

### Route-prefix decision

The requirement writes endpoints as `/api/projects/:id/devops/...`. The
established codebase convention is `/api/p/{project}/devops/...` (chi param
`{project}`, project **id**). **This plan follows the existing convention**; the
two notations describe the same endpoints. New routes are nested under the
existing `/api/p/{project}/devops` block in `internal/http/server.go:334-340`.

---

## Milestone 1 — Structured run record + persistence (F1, NF2)

**Description.** Persist a structured, self-contained record for every finished
run alongside the existing `.log` file, so history can be listed cheaply
(NF1) and includes end time, duration, and a true `cancelled` status. Write the
record when a run reaches a terminal state (`passed` | `failed` | `cancelled`),
driven off the existing `pipeline.run.completed` event already flowing through
the `LogStore` event hook. Cancelled runs (the `cancel` path,
`runner.Cancel`) emit `pipeline.run.completed` with `status: cancelled`, so the
same hook covers them.

Record shape (sidecar JSON, one file per run, next to the log):

```
~/.kaos-control/devops/<project>/<run_id>.meta.json
```

```go
type RunRecord struct {
    RunID      string `json:"run_id"`
    Slug       string `json:"slug"`        // pipeline slug
    StartedAt  string `json:"started_at"`  // RFC 3339
    EndedAt    string `json:"ended_at"`    // RFC 3339
    DurationMs int64  `json:"duration_ms"`
    Status     string `json:"status"`      // passed | failed | cancelled
    LogRef     string `json:"log_ref"`     // "<run_id>.log"
}
```

A sidecar file (not a single shared index) is chosen so a corrupt record skips
exactly one run (NF3) and so back-fill (Milestone 5) is per-run idempotent.

**Files to change**
- `internal/devops/logger.go` — add `RunRecord`; add
  `WriteRecord(projectName string, rec RunRecord) error` (atomic write: temp
  file + rename); derive `StartedAt` from the `pipeline.run.started` payload /
  first log entry and `EndedAt`/`DurationMs` from the `run.completed` payload
  (`RunCompletedPayload.DurationSeconds`). Keep `StartTime` parsing logic reuse.
- `internal/project/project.go` — in the event hook (≈ line 198), when
  `eventType == "pipeline.run.completed"`, assemble and `WriteRecord(...)` in
  addition to the existing `WriteEvent(...)`. Track per-run start time/slug from
  the earlier `pipeline.run.started` event (small in-memory map keyed by run id,
  cleared on completion) so the completion handler has start data.
- `internal/devops/events.go` — no shape change; confirm `RunStartedPayload`
  carries `Pipeline` (slug) and `RunCompletedPayload` carries `Status` +
  `DurationSeconds` (both already present).

**Acceptance criteria**
- After a passing run, `<run_id>.meta.json` exists with `status:"passed"`,
  non-empty `started_at`/`ended_at` (valid RFC 3339), and `duration_ms > 0`.
- After a failing run, the record has `status:"failed"`.
- After cancelling a run, the record has `status:"cancelled"`.
- Record write is atomic: a killed server mid-write never leaves a half-written
  `.meta.json` that breaks listing (temp+rename verified by unit test).
- The pre-existing `.log` NDJSON file is still written unchanged (no regression
  to [[devops-pipeline-log-streaming]]).

---

## Milestone 2 — Per-pipeline history listing in the store layer (F1, NF1, NF3)

**Description.** Add a store method that returns finished runs for **one**
pipeline slug, newest-first, reading the sidecar records (fast path) and
tolerating missing/corrupt records (NF3). This is the data source for the F2
endpoint and is independent of HTTP.

```go
// newest-first; limit<=0 means "all", caller caps at 50.
func (s *LogStore) ListPipelineRuns(projectName, slug string, limit int) ([]RunRecord, error)
```

Behaviour:
- Glob `<project>/*.meta.json`, decode each, filter by `slug`, sort by
  `StartedAt` descending, apply `limit`.
- A record that fails to decode is **skipped** with a `WARN` log
  (`project`, file name) — never aborts the listing (NF3).
- Unknown project dir (never run) returns empty slice, nil error.

**Files to change**
- `internal/devops/logger.go` — add `ListPipelineRuns`; add a small private
  helper to read+decode one record returning `(RunRecord, ok bool)`.
- (Optional) deprecate/retain `ListRuns` — keep for back-compat; `ListPipelineRuns`
  is the new path.

**Acceptance criteria**
- Returns only records for the requested slug, newest-first.
- A deliberately corrupted `.meta.json` is skipped and the remaining valid
  records are still returned; a warning is logged.
- With 50 valid records the call completes well within the NF1 budget
  (benchmarked / asserted < 200 ms in the test plan).
- Empty/missing project directory → empty result, no error.

---

## Milestone 3 — History listing endpoint (F2, NF4, NF5)

**Description.** Add `GET /api/p/{project}/devops/pipelines/{slug}/runs`
returning recent history newest-first.

Response:
```json
{ "runs": [ { "run_id": "...", "status": "passed",
             "started_at": "...", "ended_at": "...", "duration_ms": 7200 } ] }
```

Rules:
- `limit` query param: default **10**, hard max **50** (clamp, never error on
  over-max).
- Unknown pipeline slug → `404` (validate slug exists via the same discovery
  the existing handlers use, e.g. `handleGetPipeline` lookup).
- Roles other than `product-owner`/`devops` → `403` (`requireRole(... RolesDevopsOrAdmin)`).
- No runs → `200` with `{"runs": []}`.
- Validate `slug` against `pipelineSlugRe` (`internal/http/devops.go:19`) and
  reject traversal before any filesystem access (NF4).
- `DEBUG` log with `project`, `slug`, resolved `limit` (NF5).

**Files to change**
- `internal/http/devops.go` — add `handleListPipelineRuns`. Reuse the project
  lookup + `requireRole` preamble pattern from `handleGetRunLog`/`handleGetPipeline`.
  Map `[]RunRecord` → response DTO (omit `log_ref` from the wire response).
- `internal/http/server.go` — register the route inside the existing
  `/api/p/{project}/devops` block (near lines 334-340):
  `r.Get("/pipelines/{slug}/runs", s.handleListPipelineRuns)`.

**Acceptance criteria**
- `GET …/pipelines/{slug}/runs` returns runs newest-first with exactly the five
  fields (`run_id`, `status`, `started_at`, `ended_at`, `duration_ms`).
- `?limit=3` returns ≤ 3; `?limit=999` is capped at 50; missing param → 10.
- Unknown slug → `404`; non-devops role → `403`; valid slug, no runs → `200`
  with empty array.
- A `slug` containing `../` is rejected (`400`/`404`) before touching disk.

---

## Milestone 4 — Single-run log retrieval, pipeline-scoped (F3, NF1, NF4, NF5)

**Description.** Add `GET /api/p/{project}/devops/pipelines/{slug}/runs/{run_id}/log`
returning the full stored log as NDJSON for one past run, scoped to the
pipeline. Reuse the existing NDJSON reader; add scoping + validation.

Rules:
- Returns `ReadLogNDJSON(project, run_id)` output, `Content-Type:
  application/x-ndjson`, streamed/flushed progressively (NF1 — do not buffer the
  whole 50k-line log in memory before first byte).
- The `run_id` must belong to `slug` — cross-check against the run record from
  Milestone 2 (record's `Slug == slug`); mismatch or unknown `run_id` → `404`.
- `run_id` and `slug` validated against traversal; log path resolved only within
  the project's run store via `sandbox.Resolve` (NF4).
- Same role gate as F2 (`403` otherwise).
- `DEBUG` log with `project`, `slug`, `run_id` (NF5).

**Files to change**
- `internal/http/devops.go` — add `handleGetPipelineRunLog`. Validate `run_id`
  format (hex, matches `newRunID` shape — 16 hex chars), confirm record exists
  and belongs to slug, then stream via the existing NDJSON path. Factor the
  streaming body out of `handleGetRunLog` if it helps avoid duplication.
- `internal/http/server.go` — register
  `r.Get("/pipelines/{slug}/runs/{run_id}/log", s.handleGetPipelineRunLog)`.
- `internal/devops/logger.go` — if needed, expose a `runID`-format validator and
  a record lookup `Record(project, runID) (RunRecord, bool)`.

**Acceptance criteria**
- Returns the full stored log as NDJSON (one JSON object per line) for a known
  run of the pipeline.
- Unknown `run_id`, or a `run_id` that belongs to a different pipeline → `404`.
- Traversal in `run_id`/`slug` is rejected; resolution stays inside the project
  run store (sandbox unit/integration test).
- Non-devops role → `403`.
- Response begins streaming before the entire log is read into memory.

---

## Milestone 5 — Back-fill of pre-feature runs (resolved-question 4)

**Description.** Runs created before this feature have a `.log` file but no
`.meta.json`. Back-fill: on first `ListPipelineRuns` (or at startup scan) derive
a `RunRecord` from any `.log` lacking a sidecar, by reading its first/last
entries (`pipeline.run.started` → start/slug; `pipeline.run.completed` →
status/duration/end). Write the derived record so subsequent lists are fast.
Logs with no `run.completed` (interrupted/legacy) are listed best-effort with a
derived end time = last entry time and `status` inferred from the last step, or
skipped if unparseable (NF3).

**Files to change**
- `internal/devops/logger.go` — add `backfillRecord(project, runID) (RunRecord,
  bool)` that parses the `.log`; call it lazily inside `ListPipelineRuns` when a
  `.meta.json` is absent, persisting the result via `WriteRecord`.

**Acceptance criteria**
- A project seeded with only legacy `.log` files (no sidecars) lists those runs
  with correct slug/status/timestamps.
- After one listing, the sidecar `.meta.json` files exist (back-fill persisted).
- An unparseable legacy log is skipped with a warning, not a 500.

---

## Milestone 6 — Retention / pruning (NF2, NF5)

**Description.** Bound on-disk growth: keep at least the most recent **50** runs
per pipeline; prune older `.meta.json` **and** their `.log` files. Never prune a
run that is currently executing (check `runner.ActiveRunID`/`IsRunning`). Run
pruning after each run completion (and optionally at startup).

**Files to change**
- `internal/devops/logger.go` — add `PruneOldRuns(projectName, slug string,
  keep int, isActive func(runID string) bool) (removed int, err error)`.
- `internal/project/project.go` — invoke pruning from the completion path after
  `WriteRecord`, passing a closure over `DevopsRunner` to protect active runs.
- Log removed count at `INFO` (NF5).

**Acceptance criteria**
- With > 50 finished runs for one pipeline, only the newest 50 remain on disk;
  both `.meta.json` and matching `.log` are removed for pruned runs.
- A run that is mid-execution is never deleted even if it would fall outside the
  window.
- Pruning logs `INFO` with the number removed; pruning one pipeline does not
  affect another pipeline's history.

---

## Out of scope (per requirement Non-goals)

Live-streaming UI, cross-pipeline analytics, log search, retention-config UI,
and re-run-from-history are explicitly out of scope. Live history updates are
delivered by **reusing** the existing `pipeline.run.completed` WebSocket event
(no new transport) — the frontend ([[devops-pipeline-run-history]] frontend
plan) consumes it; no backend event change is required for F6.
