---
title: "Fix Incorrect Created Date Format — Test Plan"
type: plan-test
status: in-development
lineage: artifacts-incorrect-created-date-format
parent: lifecycle/defects/artifacts-incorrect-created-date-format.md
---

# Test Plan: Fix Incorrect Created Date Format

This plan covers integration and unit tests verifying that the created-date format defect is resolved across the backend indexing, API, agent prompt, and frontend display layers.

## Milestone 1: Unit tests for date normalisation in indexing

**Description:** Test the enhanced date-parsing logic in `index.Upsert` to confirm it correctly handles plain-date, RFC3339, missing, and malformed `created` values.

**Files to change:**
- `internal/index/index_test.go` — add test cases for `Upsert` (or the extracted normalisation helper) covering:
  - RFC3339 input → correct `createdUnix` stored.
  - Plain-date input (`"2026-04-27"`) → correct `createdUnix` stored with local timezone.
  - Empty string → fallback to `CreatedAt` (git/mtime).
  - Garbage string → fallback to `CreatedAt`, warning logged.

**Acceptance criteria:**
- All four cases pass.
- The plain-date case produces a `createdUnix` value equivalent to midnight local time on that date.

## Milestone 2: Unit test for idea-chat `created` stamping

**Description:** Verify that `writeIdeaArtifact` now includes a valid RFC3339 `created` field in the generated frontmatter.

**Files to change:**
- `internal/http/idea_chat_test.go` (create if needed, or add to existing handler tests) — call the idea-chat confirmation endpoint and assert the written file contains a `created` field matching the RFC3339 pattern.

**Acceptance criteria:**
- The test confirms the on-disk artifact has a `created` value parseable by `time.Parse(time.RFC3339, ...)`.

## Milestone 3: Integration test for API-created artifacts

**Description:** Verify that artifacts created via `POST /api/p/:project/artifacts` continue to have correct RFC3339 `created` values, and that `PUT` preserves the existing `created` value.

**Files to change:**
- `tests/` — add or extend an integration test that:
  - Creates an artifact via POST, reads it back, and asserts `created` is valid RFC3339.
  - Updates the artifact via PUT, reads it back, and asserts `created` is unchanged.

**Acceptance criteria:**
- POST-created artifact has RFC3339 `created`.
- PUT-updated artifact retains original `created` value unchanged.

## Milestone 4: Integration test for date backfill

**Description:** Verify the startup backfill routine correctly rewrites plain-date `created` values to RFC3339 on disk.

**Files to change:**
- `tests/` — add an integration test that:
  - Seeds an artifact file with `created: "2026-04-27"`.
  - Triggers a full index (or server restart).
  - Reads the file back and asserts `created` is now RFC3339 format.
  - Confirms that an artifact already in RFC3339 format is not modified.

**Acceptance criteria:**
- Plain-date values are rewritten to RFC3339 with timezone.
- Correct RFC3339 values are left untouched.
- The test artifact's other frontmatter fields are unchanged.

## Milestone 5: Frontend date display tests

**Description:** Verify the frontend date helper and components handle all `created` format variants without errors.

**Files to change:**
- `web/src/utils/date.test.ts` (or relevant test file) — unit tests for the date normalisation helper covering:
  - RFC3339 input → valid Date object.
  - Plain-date input → valid Date object.
  - Empty/undefined input → returns null or fallback.
- Component tests (if the project uses component testing) for artifact detail views rendering each format.

**Acceptance criteria:**
- The date helper returns correct Date objects for both formats.
- No "Invalid Date" is rendered in any scenario.
- Missing `created` shows a defined fallback, not a crash.

## Cross-links

- [[artifacts-incorrect-created-date-format]] — the originating defect
- [[artifacts-incorrect-created-date-format-2-be|backend plan]] — milestones 1–4 test the backend changes
- [[artifacts-incorrect-created-date-format-3-fe|frontend plan]] — milestone 5 tests the frontend changes
