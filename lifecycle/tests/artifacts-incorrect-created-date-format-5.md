---
title: "Tests: Fix Incorrect Created Date Format"
type: test
status: approved
lineage: artifacts-incorrect-created-date-format
parent: lifecycle/test-plans/artifacts-incorrect-created-date-format-4-test.md
---

# Tests: Fix Incorrect Created Date Format

Integration and unit-level tests verifying that the created-date format defect is
resolved across the backend indexing, idea-chat, API, and frontend display layers.

## Test files

### `tests/integration/date_normalise_test.go`

Covers Milestone 1 (index upsert edge cases) and Milestone 4 (NormaliseDates startup
backfill). All tests use a real SQLite index via `newTestEnv` or directly via
`env.proj.Idx`.

**Milestone 4 — NormaliseDates on-disk backfill:**

- **TestNormaliseDates_RewritesPlainDate** — Seeds an artifact with
  `created: "2026-04-27"`, opens the project (triggering Scan → NormaliseDates), reads
  the file back, and asserts the `created` field is now a valid RFC3339 value
  representing midnight local time on 2026-04-27.
- **TestNormaliseDates_LeavesRFC3339Untouched** — Seeds an artifact with a valid
  RFC3339 `created` field; asserts the field is unchanged after startup normalisation.
- **TestNormaliseDates_PreservesOtherFields** — Seeds a plain-date artifact; after
  normalisation asserts all other frontmatter fields (title, type, status, lineage)
  and the body are unmodified.
- **TestNormaliseDates_IndexReflectsNormalisedValue** — After startup normalisation,
  re-indexes the rewritten file and asserts the index row `Created` matches midnight
  local time on the original plain-date.

**Milestone 1 — Index Upsert date parsing (exercised via the project index):**

- **TestIndexUpsert_PlainDateCreated** — Upserts an artifact with `created: "2026-04-27"`
  (plain date); asserts the stored `Created` equals midnight local time on that date.
- **TestIndexUpsert_GarbageCreated** — Upserts an artifact with
  `created: "not-a-date-at-all"` and a non-zero `CreatedAt`; asserts the stored
  `Created` falls back to the `CreatedAt` value rather than erroring.
- **TestIndexUpsert_EmptyCreatedUsesCreatedAt** — Upserts an artifact with an empty
  `created` field and a non-zero `CreatedAt`; asserts the stored `Created` equals the
  `CreatedAt` backfill.
- **TestIndexUpsert_RFC3339Created** — Upserts an artifact with a valid RFC3339
  `created` field including timezone offset; asserts the stored `Created` has the
  correct Unix timestamp.

### `tests/integration/idea_chat_created_date_test.go`

Covers Milestone 2 (idea-chat writeIdeaArtifact RFC3339 created stamping).
Both tests require `ANTHROPIC_API_KEY` and are skipped when the key is absent.

- **TestIdeaChatAccept_ArtifactHasRFC3339Created** — Drives the idea-chat conversation
  to proposal state via `convergeToProposal`, sends `__accept__`, reads the written
  on-disk file, and asserts the `created:` frontmatter field is a valid RFC3339
  timestamp within a 10-second window of the request time.
- **TestIdeaChatAccept_IndexReflectsRFC3339Created** — After `__accept__`, GETs the
  artifact from the API and asserts `artifact.created` in the JSON response is a valid
  RFC3339/RFC3339Nano timestamp within the expected time window.

### `tests/integration/api_created_date_test.go`

Covers Milestone 3 (API POST/PUT created date correctness). Pre-existing file;
tests were written and are passing before this defect fix. Included here for
completeness of the test audit trail.

- **TestCreateArtifact_SetsCreatedDate** — POST creates an artifact; asserts RFC3339
  `created` in the response and frontmatter within 5 seconds of the request.
- **TestUpdateArtifact_PreservesCreatedDate** — PUT update; asserts `created` unchanged.
- **TestUpdateArtifact_CannotOverwriteCreated** — PUT with explicit different `created`;
  asserts server preserves original.
- **TestGetArtifact_ReturnsCreatedAndMtime** — GET detail; asserts both `created` and
  `mtime` are valid timestamps.
- **TestIndexScan_BackfillsCreatedFromGit** — Index backfill from git commit date (no
  disk write).
- **TestIndexScan_BackfillFallsBackToMtime** — Index backfill from filesystem mtime (no
  disk write).

### `tests/web/useFormatDate.test.ts`

Covers Milestone 5 (frontend date helper unit tests). Pure unit tests with no DOM
mounting; run via Vitest.

- **formatShortDate** suite (7 tests) — Verifies valid RFC3339, RFC3339 with offset,
  and plain-date inputs all produce non-empty non-"Invalid Date" strings; `undefined`,
  empty string, garbage, and whitespace-only inputs all return the `—` fallback.
- **formatFullDateTime** suite (6 tests) — Same coverage for the long-form formatter.
- **useFormatDate composable** suite (1 test) — Asserts the composable returns the same
  functions as the named exports and they behave identically.
