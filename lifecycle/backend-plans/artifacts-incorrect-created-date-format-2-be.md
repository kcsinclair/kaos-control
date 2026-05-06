---
title: "Fix Incorrect Created Date Format — Backend Plan"
type: plan-backend
status: done
lineage: artifacts-incorrect-created-date-format
parent: lifecycle/defects/artifacts-incorrect-created-date-format.md
---

# Backend Plan: Fix Incorrect Created Date Format

The `created` frontmatter field on agent-written artifacts uses a plain date (`"2026-04-27"`) instead of full RFC3339 with timezone (`"2026-04-27T00:00:00+10:00"`). This plan addresses three root causes: agent prompts omit the `created` field spec, the indexer silently swallows malformed dates, and the idea-chat path never stamps `created` at all.

## Milestone 1: Normalise dates during indexing

**Description:** Add a date-normalisation step in `index.Upsert` that accepts both plain-date (`2006-01-02`) and RFC3339 formats, converting plain dates to RFC3339 using the server's local timezone before storing `createdUnix`.

**Files to change:**
- `internal/index/index.go` — extend the `createdUnix` parsing block (~line 394) to try `time.Parse("2006-01-02", ...)` as a fallback when `time.Parse(time.RFC3339, ...)` fails, then convert the result to RFC3339 using `time.Now().Location()`.

**Acceptance criteria:**
- An artifact with `created: "2026-04-27"` is indexed with a correct non-zero `createdUnix` value.
- An artifact with `created: "2026-04-27T00:00:00+10:00"` continues to index correctly.
- An artifact with no `created` field still falls back to git/mtime.
- A warning is logged when a plain-date fallback is used.

## Milestone 2: Stamp `created` on the idea-chat path

**Description:** The `writeIdeaArtifact` function in the idea-chat handler builds frontmatter without setting `Created`. Align it with `handleCreateArtifact` which correctly stamps `time.Now().Format(time.RFC3339)`.

**Files to change:**
- `internal/http/idea_chat.go` — set `fm.Created = time.Now().Format(time.RFC3339)` before calling `buildMarkdown` (~line 165).

**Acceptance criteria:**
- Idea artifacts created via the chat confirmation flow have a valid RFC3339 `created` field in their on-disk frontmatter.
- Existing artifacts without `created` are unaffected.

## Milestone 3: Add `created` format guidance to agent prompt templates

**Description:** Update the agent prompt templates in `lifecycle/config.yaml` so agents either (a) are explicitly told **not** to include `created` (the server will stamp it), or (b) are given the exact RFC3339 format to use. Option (a) is preferred because agents cannot reliably determine the server timezone.

**Files to change:**
- `lifecycle/config.yaml` — in each agent's `prompt_template` that specifies required frontmatter fields (`requirements-analyst`, `planning-analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`), add a line: `Do NOT include the 'created' field — the server sets it automatically.`

**Acceptance criteria:**
- Each of the six agent prompt templates contains an explicit instruction about the `created` field.
- New artifacts produced by agents after this change do not contain a hand-written `created` field (verified by running an agent and inspecting output).

## Milestone 4: Backfill existing malformed dates

**Description:** Write a one-time migration or startup routine that scans all indexed artifacts, detects plain-date `created` values, and rewrites them on disk to RFC3339 format.

**Files to change:**
- `internal/index/index.go` — add a `NormaliseDates(rootDir string)` function called after full scan at startup that reads each artifact, checks if `created` matches `2006-01-02` only, and rewrites the frontmatter with the corrected format (using midnight in server local timezone).
- Alternatively, this could be a standalone CLI command or part of `internal/artifact/` — implementation should be minimal.

**Acceptance criteria:**
- After a server restart, all existing artifacts with plain-date `created` values are rewritten to RFC3339 on disk.
- Git diff shows only the `created` field changed on affected files.
- Artifacts that already have correct RFC3339 dates are untouched.

## Cross-links

- [[artifacts-incorrect-created-date-format]] — the originating defect
- The [[artifacts-incorrect-created-date-format-3-fe|frontend plan]] should ensure the UI gracefully handles both date formats during the transition period.
- The [[artifacts-incorrect-created-date-format-4-test|test plan]] covers integration tests for all milestones.
