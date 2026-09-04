---
title: "Tests — Release Goal & Description REST API + Round-Trip-From-Disk"
type: test
status: draft
lineage: release-goal-and-description
parent: lifecycle/test-plans/release-goal-and-description-5-test.md
created: "2026-09-03T21:40:00Z"
---

# Tests — Release Goal & Description REST API + Round-Trip-From-Disk

Implements Milestone 3 of
[lifecycle/test-plans/release-goal-and-description-5-test.md](../test-plans/release-goal-and-description-5-test.md)
(DR-3, DR-5, DR-9): integration coverage for the `goal`/`description` fields
on the release REST endpoints, PUT merge-against-current semantics, and the
[[index-is-a-cache]] round-trip-from-disk invariant.

Scope note: this test-developer run covers only the `tests/integration/`
portion of Milestone 3, per the test-developer role's write scope
(`test-developer` writes integration tests in `tests/`; the `internal/release/*_test.go`,
`internal/index/*_test.go`, `internal/http/releases_test.go`, and
`web/src/**/__tests__/*.spec.ts` files called for in Milestones 1, 2, 4, and 5
of the plan belong to `backend-developer`/`frontend-developer` unit/component
test coverage, not this role).

## Scenarios covered

| Scenario | Assertion |
|---|---|
| `POST /releases` with `goal`+`description` | 201; response echoes both; on-disk file contains `goal:`/`description:` lines (file-first write, DR-3) |
| `GET /releases` (list) and `GET /releases/{slug}` | both keys always present; populated when set, `""` when never set |
| `PUT /releases/{id}` omitting `goal`/`description` | stored values unchanged (merge-against-current) |
| `PUT /releases/{id}` with `goal: ""`, `description: ""` | both cleared; disk file drops the `goal:`/`description:` lines (`omitempty` marshal) |
| `PUT /releases/{id}` changing only `goal` (name/status unchanged) | file rewritten with new goal, `description` preserved, `artifacts_renamed == 0`, slug/file path unchanged — no spurious rename |
| Create/update via API, wipe the `releases` **table** only (`Store.PruneExcept` with an empty keep set), `release.Rehydrate` | identical `goal`/`description` reproduced from disk for both a populated and an unset release (DR-9 key acceptance check) |
| Pre-existing release file with neither `goal:` nor `description:` key, `POST /releases/rehydrate` | inserted with empty fields, no error; file bytes and mtime unchanged after rehydrate — indexing is read-only w.r.t. disk (DR-8) |

## Test file

- `tests/integration/releases_goal_description_test.go` (new) — 7 tests:
  `TestReleaseGoalDescription_CreateEchoesAndPersistsToDisk`,
  `TestReleaseGoalDescription_GetAndListReflectValues`,
  `TestReleaseGoalDescription_UpdateOmittingFieldsPreservesValues`,
  `TestReleaseGoalDescription_UpdateEmptyStringClearsValues`,
  `TestReleaseGoalDescription_UpdateOnlyGoalNoSpuriousRename`,
  `TestReleaseGoalDescription_RoundTripFromDisk`,
  `TestReleaseGoalDescription_RehydrateExistingFilesNoNewKeys`.

## Incidental fix

`tests/integration/releases_rehydrate_test.go` —
`TestRehydrateOnEmptyDB_200FilesPerformance` builds its own in-memory SQLite
`releases` table rather than going through `internal/index`'s schema-rebuild
path. The backend's Milestone 2 work (`goal`/`description` columns) had not
been reflected there, so every upsert in that performance test failed with
`table releases has no column named goal` and the test reported
`Inserted = 0, want 200`, unrelated to timing. Added `goal TEXT NOT NULL
DEFAULT ''` / `description TEXT NOT NULL DEFAULT ''` to the hand-rolled
`CREATE TABLE` to match the production schema. Verified this failure predates
this change (reproduced on `git stash`).

## Regression check

Full existing release suite re-run green alongside the new tests:
`TestReleases_*`, `TestReleaseStatus_*`, `TestRehydrate*`, `TestReleasePatch_*`,
`TestCreateWipeRestartRehydrates`, `TestArtifactParserAcceptsReleaseType`.
