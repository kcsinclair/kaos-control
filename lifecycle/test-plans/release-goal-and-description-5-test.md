---
title: Test Plan — Release Goal and Description Fields
type: plan-test
status: done
lineage: release-goal-and-description
parent: lifecycle/requirements/release-goal-and-description-2.md
created: "2026-09-03T11:15:00Z"
---

# Test Plan — Release Goal and Description Fields

Verifies the acceptance criteria of
[lifecycle/requirements/release-goal-and-description-2.md](../requirements/release-goal-and-description-2.md)
across the backend ([[release-goal-and-description]] `-3-be`) and frontend
(`-4-fe`) plans. Each milestone maps to explicit requirement DRs / acceptance
checkboxes. The headline invariant under test is
[[index-is-a-cache]]: disk is authoritative; the cache is rebuildable from disk.

## Test strategy

- **Go unit tests** (`internal/release/*_test.go`) — parser/marshaller round-trip
  and store/rehydrate behaviour, mirroring the existing `file_test.go`,
  `rehydrate_test.go`, `disksync_test.go`.
- **Go HTTP/integration tests** (`internal/http/releases_test.go` and
  `tests/`) — the four endpoints + PUT merge semantics + cache-wipe rebuild,
  using the existing `testEnv` harness (admin auto-login) noted in the project
  memory.
- **Frontend component tests** (`web/src/components/releases/__tests__/`) —
  editor round-trip and detail-view rendering with Vitest + Vue Test Utils,
  following the `ReleaseDropdown.spec.ts` pattern.

---

## Milestone 1 — Parser & marshaller round-trip (DR-1, DR-2)

**Description.** Unit-test the on-disk file model.

**Files to change**
- `internal/release/file_test.go`

**Cases**
- Parse a file with `goal:` and a multi-line `description:` block scalar →
  populated fields; `Marshal` → re-`Parse` reproduces both byte-for-byte
  (newlines preserved).
- Parse a file with **neither** key → empty `Goal`/`Description`, no error.
- Parse `goal:` / `description:` set to `""` or whitespace-only → empty fields,
  no validation error.
- `Marshal` of a `File` with both empty → output contains **no** `goal:` /
  `description:` line (omitempty), and key ordering stays
  `title, type, status, goal, description, start_date, end_date, updated_at`.
- Multi-line `goal` value → accepted verbatim (no newline validation error).

**Acceptance criteria**
- All cases pass; a golden round-trip asserts no data loss and byte-stability for
  the both-empty case.

---

## Milestone 2 — Store, rehydrate & schema rebuild (DR-4, DR-8, DR-9)

**Description.** Unit-test cache population and the rebuild-from-disk path.

**Files to change**
- `internal/release/store_test.go` (new or extend existing store coverage)
- `internal/release/rehydrate_test.go`
- `internal/index/created_test.go` (extend the `TestSchemaUpgrade` family) or a
  new index test asserting the `releases` table has `goal`/`description` columns
  after a version-mismatch rebuild.

**Cases**
- `UpsertBySlug` then `GetBySlug`/`Get`/`List`/`GetByName` → `Goal`/`Description`
  round-trip through the cache.
- `Rehydrate` of a temp `lifecycle/releases/` dir with files carrying
  `goal`/`description` → cache rows carry identical values (DR-9).
- Rehydrate of files with **neither** key → empty fields, `result.Skipped == 0`,
  no error, and **no file rewrite** (assert file mtime/content unchanged) (DR-8).
- Open an index whose stored `schema_version` is stale (< new version) → drop &
  rebuild; `PRAGMA table_info(releases)` includes `goal` and `description`; no
  `ALTER TABLE` executed (assert via the rebuild path, not a migration).

**Acceptance criteria**
- All cases pass. A test explicitly asserts the columns arrive via
  schema-version rebuild, not migration — guarding [[index-is-a-cache]] /
  [[adr-0003-pure-go-sqlite-index]].

---

## Milestone 3 — REST API & round-trip-from-disk (DR-3, DR-5, DR-9)

**Description.** Integration-test the endpoints and the cache-wipe rebuild.

**Files to change**
- `internal/http/releases_test.go`
- `tests/integration/` (new `releases_goal_description_test.go`, following
  existing `cli_releases_rehydrate_test.go` / integration harness conventions).

**Cases**
- `POST /releases` with `goal`+`description` → 201 body echoes both; the on-disk
  file contains them (assert file bytes) — file-first write path (DR-3).
- `GET /releases` and `GET /releases/{slug}` → both keys present; `""` when unset.
- `PUT /releases/{slug}` **omitting** `goal` → stored value unchanged; sending
  `"goal": ""` → cleared. Same matrix for `description` (the merge-against-current
  semantics from the `-3-be` plan Milestone 4).
- `PUT` that changes only `goal`/`description` (name unchanged) → file rewritten
  with new values, cache row updated, no spurious rename/propagation.
- **Round-trip-from-disk (DR-9 / key acceptance check):** create/edit via API,
  then wipe the `releases` **table** (not the files) and re-run `Rehydrate` (or
  restart) → identical `goal`/`description` reproduced from disk.
- A pre-existing releases dir with no new keys loads with empty fields, no error,
  no rewrite (DR-8) — exercised through the HTTP list path.

**Acceptance criteria**
- All cases pass; the round-trip-from-disk case is the explicit
  [[index-is-a-cache]] compliance gate from the requirement's acceptance list.

---

## Milestone 4 — Frontend editor & detail rendering (DR-6, DR-7, DR-10)

**Description.** Component-test the editor round-trip and detail-view rendering.

**Files to change**
- `web/src/components/releases/__tests__/ReleaseFormModal.spec.ts` (new)
- `web/src/components/releases/__tests__/ReleaseDetailModal.spec.ts` (new)

**Cases**
- Editor seeds `goal`/`description` from an existing release in edit mode; submit
  includes both in the payload; clearing a field submits `''` (clear semantics).
- `goal` input enforces `maxlength="120"`.
- Submitting empty goal + description → payload leaves the release without them
  (no error path).
- Detail modal shows the `goal` subtitle only when set (`v-if`), and renders
  `description` through `MarkdownPreview` (assert markdown → HTML, e.g. a heading
  or link renders) only when set; absent → section/subtitle omitted.
- Sanity: a `description` containing raw `<script>`/HTML is **not** executed
  (inherits `MarkdownPreview` `html: false`) (DR-10).

**Acceptance criteria**
- All component tests pass under the existing Vitest setup; `pnpm -C web test`
  green.

---

## Milestone 5 — List / roadmap / kanban subtitle (DR-7)

**Description.** Verify the `goal` subtitle appears in the roadmap/list contexts
and never breaks layout, and that `description` does not leak into these
contexts.

**Files to change**
- `web/src/components/releases/__tests__/GanttChart.spec.ts` (new or extend)
- Optionally a `RoadmapView` / `KanbanBoardView` render assertion if those
  surfaces render release names directly.

**Cases**
- A release with `goal` set → subtitle rendered under the name in the Gantt
  row/list; absent → not rendered.
- `description` is **not** present in any list/roadmap/kanban surface
  (detail-view only — Non-goal).
- Single-line truncation class/style is applied to the subtitle.

**Acceptance criteria**
- Tests pass; confirms the display-only, list-context scope (Resolved Q2) without
  introducing a `description` column.

---

## Regression / non-goals guard

- Existing release tests (`file_test.go`, `rehydrate_test.go`,
  `disksync_test.go`, `releases_test.go`) still pass unchanged — proving
  backward compatibility and no `status`/date/slug/lineage semantic drift.
- No test asserts `goal` sorting/filtering (out of scope) or a `description`
  list column (out of scope).
