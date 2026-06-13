---
title: "Release Artefacts in Markdown — Integration Test Suite"
type: test
status: approved
lineage: release-artefacts
parent: lifecycle/test-plans/release-artefacts-5-test.md
---

# Release Artefacts in Markdown — Integration Test Suite

Integration tests that exercise the disk ↔ DB sync behaviour for release markdown files (DR-2 through DR-7). All tests use `t.TempDir()` for isolation via `newTestEnv`. The CLI tests spawn the compiled binary.

## Test Files

### `tests/integration/releases_disksync_test.go` — Milestone T1

Package `integration`, build tag `//go:build integration`.

Exercises the HTTP API ↔ disk round-trip per verb:

- **`TestReleaseCreate_WritesFile`** — `POST /releases` with name `"Q1 2026"` creates `lifecycle/releases/q1-2026.md` with correct frontmatter (title, type, status).
- **`TestReleaseCreate_FallbackSlug`** — emoji-only name produces `lifecycle/releases/release-<id>.md`.
- **`TestReleaseCreate_CollisionReturns409`** — duplicate slug returns 409 with `"release slug already in use"`; only one file on disk.
- **`TestReleaseRenamePropagatesToDisk`** — `PUT` with new name removes old file, creates new file with updated frontmatter title.
- **`TestReleaseInPlaceEditDoesNotRenameFile`** — `PUT` changing only `start_date` keeps filename; frontmatter dates updated.
- **`TestReleaseDeleteRemovesFile`** — `DELETE` removes the `.md` file.
- **`TestReleaseUpdateWithStaleUpdatedAt_Returns409`** — stale `updated_at` returns 409 with `"release was modified"`.
- **`TestReleaseStatusUnscheduledAccepted`** — `status: unscheduled` accepted; frontmatter matches.

### `tests/integration/releases_rehydrate_test.go` — Milestone T2 (API)

- **`TestRehydrateOnEmptyDB`** — 3 files on disk, `POST /releases/rehydrate`, response `inserted==3`, `GET /releases` returns 3 rows.
- **`TestRehydrateOnEmptyDB_200FilesPerformance`** — 200 files, direct `release.Rehydrate` call, asserts `< 250 ms`. Skipped under `-short`.
- **`TestRehydrateSkipsInvalidFrontmatter`** — 2 valid + 1 invalid (`end_date < start_date`): `inserted==2`, `skipped==1`, WARN log captured.
- **`TestRehydrateIdempotent`** — `POST /rehydrate` twice; DB holds exactly 2 rows (ON CONFLICT DO UPDATE).
- **`TestRehydrateAPIRequiresAuth`** — anonymous `POST` returns 401.

### `tests/cli_releases_rehydrate_test.go` — Milestone T2 (CLI)

Package `cli_test`, build tag `//go:build integration`.

- **`TestReleasesRehydrateCLI_Success`** — 3 release files, `kaos-control releases rehydrate --project … --config …`, stdout is valid JSON with `{"inserted":3,"skipped":0,"errors":[]}`.
- **`TestReleasesRehydrateCLI_ProjectNotFound`** — unknown project name, exit code ≠ 0, stderr contains `"not found"`.

### `tests/integration/releases_backfill_test.go` — Milestone T3

- **`TestBackfillWritesFilesForExistingRows`** — 4 DB rows seeded via `Store.Create` (nil sync), `release.Backfill` call, 4 `.md` files with correct frontmatter on disk.
- **`TestBackfillIdempotent`** — Backfill once → snapshot mtimes; no second Backfill call (startup sync no-op condition) → mtimes unchanged.
- **`TestBackfillFailureDoesNotBlockLoad`** — `lifecycle/releases/` chmod `0o555`, Backfill returns non-fatal result, ERROR log captured, project still responds to `GET /releases`. Skipped as root.

### `tests/integration/releases_watcher_test.go` — Milestone T4

- **`TestWatcherUpsertsFromDiskEdit`** — `os.WriteFile` a release file directly → after debounce, `GET /releases` shows new row and hub emits `release.changed`.
- **`TestWatcherDeletesRowOnFileRemoval`** — API create release → `os.Remove` file → row gone, `release.changed{action:"deleted"}` WS event.
- **`TestWatcherRenameUpdatesSlug`** — `os.Rename` file → old slug absent, new slug present. (Note: fsnotify RENAME+CREATE gives the new row a new ID; the test does not assert ID preservation.)
- **`TestAPIWriteDoesNotProduceWatcherEvent`** — `POST /releases` then wait 2× debounce window; exactly 1 `release.changed` event received (loop prevention via `ExpectedEvents`).
- **`TestWatcherIgnoresInvalidFrontmatter`** — file with `status: badstatus` → no DB row, WARN log `"release handler: skipping invalid release file"`.

### `tests/integration/releases_roundtrip_test.go` — Milestone T5

- **`TestCreateWipeRestartRehydrates`** — 3 releases via API (files on disk), open second `project.Open` instance with fresh data dir, poll until startup rehydrate inserts 3 rows, verify names match.
- **`TestArtifactParserAcceptsReleaseType`** — seed `lifecycle/releases/parser-check.md`, startup scan indexes it, `GET /parse-errors` has no `"unknown type \"release\""` error.

### `tests/integration/releases_roadmap_regression_test.go` — Milestone T5 (Roadmap)

- **`TestReleaseListOrdering`** — scheduled releases (with `start_date`) appear before unscheduled ones; within each group ordered by `start_date` then name.
- **`TestReleaseRenamePropagatesArtifactFrontmatter`** — rename a release; seeded idea's on-disk `release:` frontmatter field is rewritten to the new name.
- **`TestKanbanFilterByRelease`** — `GET /artifacts?release=<name>` returns only ideas assigned to that release.

## Helpers Introduced

- `makeReleaseMD(title, status)` — builds minimal valid release frontmatter string.
- `seedReleaseRow(t, store, name, status)` — inserts a DB row without disk write.
- `fileStatMap(t, dir)` — maps filename → `os.FileInfo` for `.md` files in a directory.
- `pollReleaseBySlug(env, slug, wantPresent, timeout)` — polls `GET /releases` until slug is present/absent or timeout.
- `debounceWait` constant — `400 ms` (2 × 150 ms debounce + 100 ms buffer).
- `urlEscape(s)` — minimal space→`%20` encoder for query params.
