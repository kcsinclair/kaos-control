---
title: "Backend Plan: Frontmatter Created Date"
type: plan-backend
status: done
lineage: frontmatter-created-date
parent: ideas/frontmatter-created-date.md
labels:
    - artefacts
    - backend
    - feature
---

# Backend Plan: Frontmatter Created Date

Add a `created` field to artifact YAML frontmatter, persisted in the SQLite index, set automatically at creation time and never modified thereafter. Also surface the existing `mtime` (modified) date through the index and API for consistent date handling.

## Milestone 1 — Add `created` field to Frontmatter struct and parser

**Description**: Extend the `Frontmatter` struct in `internal/artifact/artifact.go` to include a `created` field. The field uses ISO 8601 format (e.g. `2026-04-27T10:00:00+10:00`). The parser must read it from YAML frontmatter when present and leave it empty when absent (for backward compatibility with existing artifacts that lack it).

**Files to change**:
- `internal/artifact/artifact.go` — add `Created string` field to `Frontmatter` with YAML/JSON tags `created,omitempty`

**Acceptance criteria**:
- `Frontmatter.Created` is populated when a `created:` key exists in the YAML frontmatter
- Existing artifacts without `created:` parse without errors; the field is empty string
- The field round-trips correctly through `yaml.Marshal` / `yaml.Unmarshal`

## Milestone 2 — Update SQLite schema to store `created`

**Description**: Add a `created` INTEGER column (Unix timestamp) to the `artifacts` table. Bump `schemaVersion` to trigger a rebuild from disk on next startup. Update the `Upsert` method to write the `created` value, and update all query methods (`List`, `Get`, `Graph`) to read it back.

**Files to change**:
- `internal/index/index.go` — bump `schemaVersion` from 2 to 3; add `created INTEGER NOT NULL DEFAULT 0` to the `artifacts` CREATE TABLE DDL; update `Upsert` to insert the parsed created timestamp; update `scanRows` to read it; add `Created` field to `ArtifactRow`

**Acceptance criteria**:
- On startup with an existing v2 index, the schema is dropped and rebuilt (existing behaviour for version mismatch)
- The `created` column stores a Unix timestamp; 0 means "not set" (legacy artifacts)
- `ArtifactRow.Created` is a `time.Time` serialised as ISO 8601 in JSON responses
- `List` and `Get` queries return the `created` field

## Milestone 3 — Set `created` automatically on artifact creation

**Description**: When a new artifact is created via `POST /api/p/:project/artifacts`, the server sets `created` to the current time in ISO 8601 with the server's local timezone offset. This value is written into the frontmatter before the file is saved to disk.

**Files to change**:
- `internal/http/write.go` — in `handleCreateArtifact`, set `req.Frontmatter.Created` to `time.Now().Format(time.RFC3339)` before calling `buildMarkdown`

**Acceptance criteria**:
- Every artifact created via the API has a `created:` field in its on-disk YAML frontmatter
- The `created` value matches the server's wall clock at creation time (within 1 second tolerance)
- The ISO 8601 format includes timezone offset (not UTC `Z` unless the server is in UTC)

## Milestone 4 — Preserve `created` on artifact updates

**Description**: When an artifact is updated via `PUT /api/p/:project/artifacts/*path`, the server must preserve the existing `created` value. If the incoming request payload omits or blanks `created`, the server reads the current file's `created` value and re-applies it before writing.

**Files to change**:
- `internal/http/write.go` — in `handleUpdateArtifact`, after reading the current file for SHA check, parse its frontmatter to extract the existing `created` value; if `req.Frontmatter.Created` is empty, set it from the existing value

**Acceptance criteria**:
- Updating an artifact's body or other frontmatter fields does not alter the `created` field
- An explicit `created` value in the update payload is rejected or ignored (immutability)
- The `created` value survives any number of sequential updates

## Milestone 5 — Backfill `created` from git history for existing artifacts

**Description**: For artifacts that lack a `created` field (pre-existing files), the indexer should attempt to derive the creation date from git history (first commit that introduced the file). This runs during the startup scan. If git history is unavailable, fall back to the file's filesystem mtime.

**Files to change**:
- `internal/index/index.go` — in `IndexFile`, after parsing, if `a.FM.Created` is empty, call a new helper that uses `go-git` to find the earliest commit for the file path; fall back to `info.ModTime()`
- `internal/git/git.go` (or equivalent) — add a `FirstCommitDate(relPath string) (time.Time, error)` method that walks the log for the file and returns the author date of the oldest commit

**Acceptance criteria**:
- Existing artifacts without `created:` in frontmatter get a `created` value in the index derived from git
- The git-derived date is stored in the index only (the on-disk file is NOT modified during scan)
- If the file has no git history (untracked), the filesystem mtime is used
- The backfill does not significantly slow startup for projects with hundreds of artifacts (< 2 seconds additional)

## Milestone 6 — Expose `created` and `mtime` in API responses

**Description**: Ensure both `created` and `mtime` are included in all artifact API responses (`GET /artifacts`, `GET /artifacts/*path`, graph endpoint) as ISO 8601 strings. [[frontmatter-created-date]] requires both dates to be available for the frontend.

**Files to change**:
- `internal/index/index.go` — ensure `ArtifactRow` JSON serialisation includes `created` as ISO 8601
- `internal/http/artifacts.go` — no changes needed if `ArtifactRow` already serialises correctly; verify the `handleGetArtifact` response includes both fields

**Acceptance criteria**:
- `GET /api/p/:project/artifacts` list items include `created` and `mtime` as ISO 8601 strings
- `GET /api/p/:project/artifacts/*path` detail response includes `created` and `mtime`
- Artifacts with no `created` date return `created` as empty string or zero-value, not omitted
