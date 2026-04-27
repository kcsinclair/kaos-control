---
title: "Test Plan: Frontmatter Created Date"
type: plan-test
status: draft
lineage: frontmatter-created-date
parent: ideas/frontmatter-created-date.md
labels:
    - artefacts
    - testing
    - feature
---

# Test Plan: Frontmatter Created Date

Integration and unit tests verifying that the `created` date field is correctly set, persisted, preserved, and displayed. Covers backend changes from [[frontmatter-created-date-2-be]] and frontend behaviour from [[frontmatter-created-date-3-fe]].

## Milestone 1 — Unit tests for Frontmatter parsing with `created` field

**Description**: Verify the artifact parser correctly reads and round-trips the `created` field from YAML frontmatter. Cover the case where `created` is present, absent, and malformed.

**Files to change**:
- `internal/artifact/artifact_test.go` — add test cases:
  - `TestParse_CreatedFieldPresent` — YAML with `created: "2026-04-27T10:00:00+10:00"` → `FM.Created` is populated
  - `TestParse_CreatedFieldAbsent` — YAML without `created:` → `FM.Created` is empty string, no parse error
  - `TestParse_CreatedFieldRoundTrip` — marshal then re-parse; `created` value is preserved exactly

**Acceptance criteria**:
- All three test cases pass
- No regressions in existing artifact_test.go cases
- Tests run in `make test-unit`

## Milestone 2 — SQLite index tests for `created` column

**Description**: Verify the index correctly stores and retrieves the `created` timestamp, including schema migration (version bump triggers rebuild) and backfill behaviour.

**Files to change**:
- `internal/index/index_test.go` (or new file `internal/index/created_test.go`) — add test cases:
  - `TestUpsert_CreatedTimestamp` — upsert an artifact with `created` set; query back via `Get`; verify `Created` field matches
  - `TestUpsert_CreatedZero` — upsert an artifact without `created`; verify `Created` is zero-value
  - `TestSchemaUpgrade` — open an index with old schema version; verify it drops and recreates (existing behaviour, but confirm `created` column exists after)

**Acceptance criteria**:
- Upserted artifacts with `created` values can be retrieved accurately
- Zero-value created dates don't cause errors or panics
- Schema version mismatch triggers rebuild with the new column present

## Milestone 3 — API integration tests for artifact creation

**Description**: End-to-end tests against the HTTP API verifying that `POST /api/p/:project/artifacts` automatically sets `created` and that subsequent `PUT` updates preserve it.

**Files to change**:
- `tests/api_created_date_test.go` (new file) — test cases:
  - `TestCreateArtifact_SetsCreatedDate` — POST to create; verify response includes `created` as a valid ISO 8601 timestamp within 5 seconds of `time.Now()`
  - `TestUpdateArtifact_PreservesCreatedDate` — POST to create, note `created`; PUT to update body; verify `created` is unchanged
  - `TestUpdateArtifact_CannotOverwriteCreated` — POST to create; PUT with a different `created` value; verify the original `created` is preserved
  - `TestGetArtifact_ReturnsCreatedAndMtime` — create an artifact; GET its detail; verify both `created` and `mtime` are present in the JSON response
- `lifecycle/tests/frontmatter-created-date-5.md` — companion test artifact documenting what the test code covers

**Acceptance criteria**:
- All four API test cases pass against a running server (or test harness)
- The `created` field in the response is a valid ISO 8601 string
- The `created` field does not change after updates
- Both `created` and `mtime` are present in detail and list responses

## Milestone 4 — Backfill tests for git-derived created dates

**Description**: Verify that the git history backfill correctly derives a `created` date for existing artifacts that lack the `created:` frontmatter field.

**Files to change**:
- `tests/api_created_date_test.go` — additional test cases:
  - `TestIndexScan_BackfillsCreatedFromGit` — create a file without `created:`, commit it, trigger a re-index; verify the index row has a non-zero `created` value matching the git commit date
  - `TestIndexScan_BackfillFallsBackToMtime` — create an untracked file (no git history) without `created:`; trigger index; verify `created` falls back to file mtime

**Acceptance criteria**:
- Artifacts without `created:` in frontmatter get a `created` value in the index
- Git-derived dates match the first commit's author date for the file
- Untracked files fall back to filesystem mtime
- Backfill does not modify on-disk files

## Milestone 5 — Frontend display tests

**Description**: Verify the Vue components correctly display created and modified dates with the expected formatting and tooltip behaviour.

**Files to change**:
- `web/src/components/artifact/__tests__/FrontmatterPanel.spec.ts` (new or existing) — test cases:
  - `renders created date when present` — mount with artifact having `created`; assert the "Created" row is visible with formatted date
  - `hides created date when empty` — mount with empty `created`; assert "Created" row is absent
  - `shows tooltip with full date-time on hover` — mount; hover over date; assert tooltip text includes time and timezone
  - `renders modified date` — mount; assert "Modified" row shows formatted mtime

**Acceptance criteria**:
- Component tests pass in the Vite test runner (`pnpm test` or equivalent)
- Created date is shown when present, hidden when absent
- Tooltip content includes full date-time with timezone information
