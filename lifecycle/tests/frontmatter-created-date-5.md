---
title: "Tests: Frontmatter Created Date"
type: test
status: draft
lineage: frontmatter-created-date
parent: lifecycle/test-plans/frontmatter-created-date-4-test.md
---

# Tests: Frontmatter Created Date

Integration and unit tests verifying that the `created` date field is correctly parsed,
stored, preserved, and backfilled across the artifact lifecycle. All tests pass as of
2026-04-27.

## Test files

### `internal/artifact/artifact_test.go`

Unit tests for the artifact parser (Milestone 1):

- **TestParse_CreatedFieldPresent** — Parses a YAML frontmatter block containing
  `created: "2026-04-27T10:00:00+10:00"` and asserts `FM.Created` is populated with
  the exact value and no parse errors are emitted.
- **TestParse_CreatedFieldAbsent** — Parses a frontmatter block without a `created:`
  key and asserts `FM.Created` is the empty string with no related parse errors.
- **TestParse_CreatedFieldRoundTrip** — Parses a frontmatter block, marshals
  `Frontmatter` back to YAML via `gopkg.in/yaml.v3`, re-parses the result, and asserts
  the `created` value is preserved exactly through the cycle.

### `internal/index/created_test.go`

Unit tests for the SQLite index (Milestone 2):

- **TestUpsert_CreatedTimestamp** — Upserts an artifact whose `FM.Created` is a valid
  RFC3339 string; retrieves the row via `Get` and asserts `row.Created` matches the
  expected Unix timestamp.
- **TestUpsert_CreatedZero** — Upserts an artifact with no `created` frontmatter and no
  `CreatedAt` backfill; asserts `row.Created` is the zero `time.Time`.
- **TestSchemaUpgrade** — Seeds a SQLite DB with `schema_version = 0`, opens it via
  `index.Open`, and verifies the schema was rebuilt with the `created` column present
  and the version bumped to the current constant.

### `tests/integration/api_created_date_test.go`

API integration tests (Milestones 3 and 4). All tests use a real HTTP server started
via `newTestEnv`:

- **TestCreateArtifact_SetsCreatedDate** — POSTs a new artifact; asserts the response
  includes `artifact.created` as a valid ISO 8601 timestamp within 5 seconds of the
  request time, and that `frontmatter.created` is also set.
- **TestUpdateArtifact_PreservesCreatedDate** — Creates an artifact, notes its
  `created` value, then PUTs an updated body; asserts `created` is unchanged in the
  PUT response.
- **TestUpdateArtifact_CannotOverwriteCreated** — Creates an artifact, then PUTs with
  `frontmatter.created` set to a different value (`2000-01-01T00:00:00Z`); asserts the
  server silently discards the overwrite and returns the original `created`.
- **TestGetArtifact_ReturnsCreatedAndMtime** — Creates an artifact, then GETs its
  detail; asserts both `artifact.created` and `artifact.mtime` are present and parse as
  valid timestamps.
- **TestIndexScan_BackfillsCreatedFromGit** — Writes an artifact file without
  `created:`, commits it via go-git with a known timestamp, calls `IndexFile`, and
  asserts the index row has a non-zero `Created` within the commit time window. Also
  asserts the on-disk file was not modified.
- **TestIndexScan_BackfillFallsBackToMtime** — Writes an artifact file without
  `created:` but does not commit it (untracked); calls `IndexFile` and asserts the
  index row `Created` falls back to filesystem mtime. Also asserts the on-disk file was
  not modified.
