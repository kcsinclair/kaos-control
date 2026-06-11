---
title: Test plan — Release artefacts in markdown
type: plan-test
status: in-development
lineage: release-artefacts
parent: requirements/release-artefacts-2.md
---

# Test plan — Release artefacts in markdown

Integration tests that exercise the disk ↔ DB sync behaviour described
in `requirements/release-artefacts-2.md`, covering the contract surface
introduced by [[release-artefacts]]-3-be and consumed by
[[release-artefacts]]-4-fe. Tests live under `tests/integration/` and
follow the existing convention: `testEnv` auto-logins as admin, devops
URL helpers return full URLs, run-log endpoints return NDJSON.

## Milestone T1 — DB→disk round-trip per HTTP verb

**Description.** End-to-end tests that drive the REST API through
`testEnv` and assert the disk side-effects required by DR-2 and DR-6.

**Files to change.**
- `tests/integration/releases_disksync_test.go` (new) — sub-tests:
  - `TestReleaseCreate_WritesFile` — `POST /releases` with name
    `"Q1 2026"` → response contains `file_path:
    "lifecycle/releases/q1-2026.md"`, file exists, frontmatter contains
    `title: Q1 2026`, `type: release`, `status: planned`.
  - `TestReleaseCreate_FallbackSlug` — `POST` with name `"🚀"` →
    response `file_path` matches `lifecycle/releases/release-\d+\.md`.
  - `TestReleaseCreate_CollisionReturns409` — second `POST` with name
    `"Q1-2026"` returns 409, body contains `release slug already in
    use`, no second file created.
  - `TestReleaseRenamePropagatesToDisk` — `PUT` changing name renames
    file: old path absent, new path present, frontmatter `title`
    matches new value.
  - `TestReleaseInPlaceEditDoesNotRenameFile` — `PUT` changing only
    `start_date` keeps filename, updates frontmatter dates.
  - `TestReleaseDeleteRemovesFile` — `DELETE` removes the .md file.
  - `TestReleaseUpdateWithStaleUpdatedAt_Returns409` — concurrent
    update with stale `updated_at` returns 409 with `release was
    modified`.
  - `TestReleaseStatusUnscheduledAccepted` — `POST` with
    `status: unscheduled` succeeds; frontmatter `status: unscheduled`.

**Acceptance criteria.**
- All sub-tests pass under `make test-unit` (or whichever target runs
  `tests/integration`).
- Each assertion reads the actual disk path (under `testEnv`'s
  per-test project root) via `os.ReadFile`, not via the API alone.

## Milestone T2 — Rehydrate (disk→DB) and CLI trigger

**Description.** Covers DR-3 and Resolved Question 3 (admin endpoint +
CLI). Uses the existing `cli_*_test.go` pattern for the CLI form.

**Files to change.**
- `tests/integration/releases_rehydrate_test.go` (new) —
  - `TestRehydrateOnEmptyDB` — populate
    `lifecycle/releases/{a,b,c}.md` then start the project with an
    empty `releases` table → `GET /releases` returns 3 rows with
    matching fields.
  - `TestRehydrateSkipsInvalidFrontmatter` — drop a file with
    `end_date < start_date` alongside 2 valid files → 2 rows inserted,
    one WARN log captured via the project's log buffer, project still
    loads.
  - `TestRehydrateIdempotent` — call `POST /releases/rehydrate` twice
    → DB state unchanged on second call (DR-7 idempotency).
  - `TestRehydrateAPIRequiresAuth` — anonymous `POST` returns 401.
- `tests/cli_releases_rehydrate_test.go` (new) — spawn the binary with
  `releases rehydrate --project <id>` against a temp project, capture
  stdout JSON, assert `{"inserted":N,"skipped":M,"errors":[...]}`
  matches expectation. Exit code 0 on success, non-zero when project
  not found.

**Acceptance criteria.**
- All rehydrate tests pass.
- Performance check inside `TestRehydrateOnEmptyDB`: with 200 files
  the rehydrate completes in `< 250 ms` measured in-process (DR-7).
  Skip the budget check when `-short` is set.

## Milestone T3 — Backfill (DB→disk) on first run

**Description.** Covers DR-5.

**Files to change.**
- `tests/integration/releases_backfill_test.go` (new) —
  - `TestBackfillWritesFilesForExistingRows` — seed DB with 4
    releases via `Store.Create` directly, ensure
    `lifecycle/releases/` is empty, start the project → 4 files on
    disk with frontmatter matching the rows.
  - `TestBackfillIdempotent` — restart the project after the first
    backfill → no further files written (compare directory mtime
    snapshots).
  - `TestBackfillFailureDoesNotBlockLoad` — make
    `lifecycle/releases/` unwritable (chmod 0500) → project still
    becomes `ready`, ERROR log captured.

**Acceptance criteria.**
- All backfill tests pass on macOS and Linux runners (the chmod test
  is skipped on Windows if/when added).
- The unwritable-dir test asserts an ERROR log line containing
  `release_id` and `project_id` fields (DR-7 logging requirement).

## Milestone T4 — Watcher behaviour & WebSocket events

**Description.** Covers DR-4 and the WS-event acceptance criteria.

**Files to change.**
- `tests/integration/releases_watcher_test.go` (new) —
  - `TestWatcherUpsertsFromDiskEdit` — write
    `lifecycle/releases/q2-2026.md` directly → within `2 *
    debounceWindow`, `GET /releases` returns the new row and a WS
    client receives one `release.changed` message.
  - `TestWatcherDeletesRowOnFileRemoval` — `os.Remove` the file →
    row gone, WS `release.deleted` (or `release.changed` with deletion
    semantics — match the envelope chosen in the backend plan).
  - `TestWatcherRenameUpdatesSlug` — `os.Rename` the file → the row
    keeps its ID, gets the new slug, one WS event emitted.
  - `TestAPIWriteDoesNotProduceWatcherEvent` — `POST /releases` then
    wait `2 * debounceWindow`; assert WS clients only saw the API's
    own `release.changed`, not a duplicate from the watcher (loop
    prevention — DR-4).
  - `TestWatcherIgnoresInvalidFrontmatter` — write a file with
    unknown status → no row inserted, WARN log captured.
- Reuse `tests/integration/releases_ws_test.go` helpers for the WS
  client.

**Acceptance criteria.**
- All watcher tests pass.
- Each test uses the helper's NDJSON-aware run-log reader where it
  inspects logs (per the project convention).
- The duplicate-suppression test asserts WS message count == 1 across
  a 2-debounce-window observation period.

## Milestone T5 — Cross-machine reproducibility & artifact-parser sanity

**Description.** Covers the headline Goals 5 & 6 ("Cross-machine
reproducibility", "Round-trip safety") and the artifact-parser
acceptance criterion that `type: release` is accepted by the parser.

**Files to change.**
- `tests/integration/releases_roundtrip_test.go` (new) —
  - `TestCreateWipeRestartRehydrates` — `POST` 3 releases via API,
    close the project, delete the SQLite file, reopen the project,
    `GET /releases` returns the same 3 releases with identical
    fields. This is the headline DR-3 acceptance test.
  - `TestArtifactParserAcceptsReleaseType` — drop a release file in
    place, trigger the artifact indexer, assert no `unknown type
    "release"` parse errors via `GET
    /api/v1/projects/:project/artifacts/parse-errors`.
- `tests/integration/releases_roadmap_regression_test.go` (new) —
  smoke regression for [[releases-and-roadmaps]]:
  - `GET /releases` ordering unchanged (scheduled first, then by
    name).
  - Rename a release; assert artifact frontmatter (`release:` field on
    an idea) is rewritten as before — i.e. the existing propagate
    behaviour is intact.
  - Kanban filter by `release` still returns the right artifact set.

**Acceptance criteria.**
- All round-trip tests pass on a clean DB.
- The roadmap regression test sequences against an existing fixture
  project (extend `tests/fixtures/` if no suitable fixture exists)
  with at least one idea referencing a release.
- No new parse errors are introduced for any pre-existing fixture
  project's artifacts.
