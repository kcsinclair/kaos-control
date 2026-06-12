---
title: Release Artefacts in Markdown
type: requirement
status: done
lineage: release-artefacts
priority: high
parent: ideas/release-artefacts.md
labels:
    - releases
    - persistence
    - portability
release: KC-Release4
assignees:
    - role: product-owner
      who: agent
---

# Release Artefacts in Markdown

## Problem

Releases are currently first-class entities stored only in the project's
SQLite index (see [[releases-and-roadmaps]]). The SQLite file is local-only
and is not committed to git, so a project cloned onto a second machine
starts with an empty `releases` table even though artifact frontmatter
already references release names. This produces orphaned `release:` fields
on ideas/defects, breaks the Roadmap and Gantt views until a human re-creates
each release by hand, and means release schedule/status data is lost on
fresh checkouts, on `~/.kaos-control` reset, and during disaster recovery.

The repo already treats `lifecycle/` markdown as authoritative and SQLite
as a rebuildable cache for artifacts (see `CLAUDE.md` → "The SQLite index
is a cache; disk is authoritative"). Releases must follow the same model
so that pulling the repo on a fresh machine reconstitutes the full release
plan automatically.

## Goals / Non-goals

### Goals

1. **Disk-authoritative release storage** — every release is materialised
   as a markdown file under `lifecycle/releases/<slug>.md` with YAML
   frontmatter holding name, status, start_date, end_date.
2. **DB→disk sync on every release mutation** — `POST`, `PUT`, `DELETE` on
   `/releases` and rename propagation write the corresponding markdown
   file (or delete it) inside the same request, before responding success.
3. **Disk→DB rehydrate on empty table** — at project load, if the
   `releases` table has zero rows for the project but `lifecycle/releases/`
   contains release markdown files, those files are parsed and inserted
   into the table before the project is marked ready.
4. **Live watcher integration** — `fsnotify` changes inside
   `lifecycle/releases/` cause incremental upsert/delete in the `releases`
   table and emit the same WebSocket events used for artifact changes.
5. **Cross-machine reproducibility** — checking out the repo on a second
   machine and starting kaos-control yields the identical set of releases
   (name, status, dates) seen on the originating machine, with no manual
   step.
6. **Round-trip safety** — a release written by the API and then re-read
   from disk into the DB produces a byte-for-byte equivalent record (modulo
   timestamps).
7. **Backfill of pre-existing projects** — on first startup of this version
   against a project that has DB releases but no `lifecycle/releases/`
   files, every existing release is written to disk so the next commit
   captures them.

### Non-goals

- Splitting `lifecycle/releases/` into per-release subdirectories.
- Storing per-release member ticket lists inside the release markdown
  (membership stays expressed as `release:` on the member artifact).
- Editing release content (rich body / notes) — the body of the release
  markdown is reserved for a free-form description but no UI surface is
  added in this iteration beyond what already exists for releases.
- Conflict resolution UX for simultaneous DB+disk writes by two users
  (the existing single-user assumption from v1 still applies).
- Migrating releases between projects.
- Auto-committing release file changes to git (a separate git-context
  feature; this requirement only writes files).

## Detailed Requirements

### DR-1: Release markdown file format

- Each release is stored at `lifecycle/releases/<release-slug>.md` where
  `release-slug` is the kebab-cased release name (lowercase, spaces →
  `-`, strip characters outside `[a-z0-9-]`). Filenames are unique per
  project; collisions are rejected at create/rename time with HTTP 409.
- The file MUST start with YAML frontmatter containing at minimum:

  ```yaml
  ---
  title: <release name as entered by the user>
  type: release
  status: <planned|active|shipped|unscheduled>
  start_date: <YYYY-MM-DD or omitted when unscheduled>
  end_date:   <YYYY-MM-DD or omitted when unscheduled>
  ---
  ```

- `title` carries the human-readable release name; the filename slug
  carries the URL-safe identifier. A rename changes both.
- The body below the frontmatter is preserved on read/write; new files
  are created with an empty body.
- The `type: release` value MUST be added to `KnownTypes` in
  `internal/artifact/artifact.go` so the artifact parser does not flag
  release files as having an unknown type.
- `lineage` and `parent` are NOT required on release files (releases do
  not participate in lineage chains).

### DR-2: DB→disk on every mutation

- On `POST /api/v1/projects/:project/releases` the server MUST write
  `lifecycle/releases/<slug>.md` after the row is inserted and before
  the HTTP response is sent. A failed write rolls the DB insert back.
- On `PUT /releases/:id` the file is rewritten with the updated
  frontmatter. If the name changed, the old file is renamed/deleted and
  the new filename created in one atomic operation (rename on same
  filesystem; otherwise write-new-then-delete-old).
- On `DELETE /releases/:id` the corresponding file is removed from disk
  after the row is deleted. A failed disk delete logs a warning but does
  not undo the DB delete (the next watcher tick re-inserts a stub).
- Rename propagation against artifact frontmatter (already implemented
  in [[releases-and-roadmaps]]) is unchanged; this requirement only adds
  the release file rename alongside it.

### DR-3: Disk→DB rehydrate on empty table

- At project load, after schema migration, the server checks
  `SELECT COUNT(*) FROM releases WHERE project_id = ?`.
- If the count is zero AND `lifecycle/releases/` exists and contains at
  least one `.md` file, the server parses every file, validates its
  frontmatter against DR-1, and inserts a row per file before the
  project's `ready` flag is set.
- Files with invalid frontmatter (missing required fields, unknown
  status, invalid dates, `end_date < start_date`) are skipped with a
  WARN log; project load does not fail.
- If the count is non-zero, this step is skipped (no merge/diff in v1 —
  disk is read only when DB is empty).

### DR-4: Live watcher

- The existing `fsnotify` watcher must include
  `lifecycle/releases/*.md` in its scope.
- `CREATE`/`WRITE` events trigger upsert into `releases` keyed by
  filename slug; the slug→ID mapping is derived from the
  `lifecycle/releases/<slug>.md` filename.
- `REMOVE`/`RENAME` events trigger delete (or rename) of the matching
  row.
- Watcher events emit `release.changed` WebSocket messages with the
  same envelope shape as `artifact.indexed` so the Roadmap UI live-
  refreshes.
- Watcher-driven changes MUST NOT re-trigger a disk write (loop
  prevention): mutations originating from the API set an in-memory
  "expect-this-event" mark that the watcher consumes silently.

### DR-5: Backfill on first run with existing DB

- On startup, if the `releases` table has rows for the project but
  `lifecycle/releases/` is missing or contains zero release files, the
  server writes one markdown file per existing row before the project is
  marked ready. This is a one-shot migration; subsequent starts find
  both populated and skip the backfill.
- Backfill failures (e.g. unwritable directory) MUST log ERROR but not
  prevent the project from loading.

### DR-6: API surface

- No new REST endpoints are introduced. The existing
  `GET/POST/PUT/DELETE /releases` endpoints gain the disk-sync behaviour
  in DR-2.
- The `Release` JSON response gains a `file_path` field (relative to
  project root, e.g. `lifecycle/releases/q1-2026.md`) so the UI can
  link to the source file.

### DR-7: Non-functional

- **Atomicity** — file writes use the existing safe-write pattern (write
  to `*.tmp` then `rename`) so a crashed write never leaves a half-
  written release file.
- **Idempotency** — rehydrate and backfill are safe to run repeatedly;
  re-running them produces the same DB and disk state.
- **Performance** — rehydrating a project with ≤ 200 release files at
  startup must complete in < 250 ms on the developer reference machine.
- **Path safety** — release file paths must pass through the existing
  `internal/sandbox` resolver; an attempt to write outside
  `lifecycle/releases/` is rejected.
- **Permissions** — file mode follows the project's existing artifact
  write convention (0644 for files, 0755 for `lifecycle/releases/` when
  created).
- **Logging** — every DB→disk and disk→DB action is logged at INFO with
  `project_id`, `release_id`, `slug`, and the originating trigger
  (`api`, `rehydrate`, `backfill`, `watcher`).

## Acceptance Criteria

- [ ] Creating a release via the UI writes
      `lifecycle/releases/<slug>.md` with the correct frontmatter and
      the API response includes a `file_path` field.
- [ ] Editing a release's name renames the file on disk and updates
      `title` and frontmatter; the old file no longer exists.
- [ ] Editing a release's dates or status rewrites the existing file in
      place without changing its filename.
- [ ] Deleting a release removes the corresponding `.md` file from
      `lifecycle/releases/`.
- [ ] On a project whose SQLite DB has been wiped but whose
      `lifecycle/releases/` directory contains valid release files,
      restarting kaos-control reconstitutes all releases with identical
      names, dates, and statuses.
- [ ] On a project with existing DB releases but no
      `lifecycle/releases/` directory, the first startup of this version
      creates one file per release; a subsequent startup makes no
      further changes.
- [ ] Manually editing a release file on disk and saving it triggers a
      WebSocket `release.changed` event and the corresponding DB row is
      updated within one watcher debounce window.
- [ ] Manually deleting a release file removes the row from the DB.
- [ ] A release file with invalid frontmatter (e.g. `end_date` before
      `start_date`, unknown status) is skipped during rehydrate with a
      WARN log; the project still loads.
- [ ] Two releases with names that slug to the same filename
      (e.g. "Q1 2026" and "Q1-2026") cannot both be created; the second
      request returns HTTP 409.
- [ ] `type: release` is accepted by the artifact parser (no
      `unknown type "release"` parse error appears for release files
      in the artifact index).
- [ ] Round-trip test: create a release via API → read the file → wipe
      DB → restart → release reappears with identical fields.
- [ ] Related artifacts: [[releases-and-roadmaps]] continues to work —
      the Roadmap page, Gantt view, kanban filter, and rename
      propagation all behave identically after this change.

## Resolved Questions

1. Should the markdown body of a release file be exposed as an editable
   "release notes" field in the existing release edit modal, or stay
   reserved for a later iteration?

> reserved for a later iteration.

2. When a manual disk edit conflicts with a concurrent API update,
   should the watcher's last-write-wins behaviour be replaced with an
   explicit version field in frontmatter (e.g. `updated_at`) for
   conflict detection?

> Yes

3. Should rehydrate also be triggered manually via an admin endpoint
   (e.g. `POST /releases/rehydrate`) for recovery scenarios, or is the
   empty-table-on-startup trigger sufficient?

> include an API and create a CLI option for this too.

4. For the slug derivation: if the user names a release with characters
   that produce an empty slug (e.g. all emoji), should creation fail
   with HTTP 400 or fall back to a generated `release-<id>` slug?

> fall back

5. Does the watcher need to debounce release-file events on a different
   interval from artifact events, or can it reuse the existing 150 ms
   debounce?

> reuse existing.
