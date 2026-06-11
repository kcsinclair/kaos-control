---
title: Backend plan — Release artefacts in markdown
type: plan-backend
status: in-development
lineage: release-artefacts
parent: requirements/release-artefacts-2.md
---

# Backend plan — Release artefacts in markdown

Implements the DB↔disk synchronisation behaviour described in
`requirements/release-artefacts-2.md`. Releases become disk-authoritative
markdown files under `lifecycle/releases/<slug>.md`, mirrored into the
SQLite cache on rehydrate, backfill, and watcher events. Companion plans:
[[release-artefacts]] frontend ([[release-artefacts]]-4-fe) and tests
([[release-artefacts]]-5-test).

## Milestone B1 — Release markdown file format & parser support

**Description.** Establish the on-disk representation: register
`type: release` in the artifact parser, define a `release.File` model
that owns frontmatter ↔ struct conversion, and add `unscheduled` to the
release status vocabulary so DR-1's full set
(`planned|active|shipped|unscheduled`) is accepted everywhere. Add the
`updated_at` field (Resolved Question 2) to the frontmatter contract for
conflict detection.

**Files to change.**
- `internal/artifact/artifact.go` — add `"release": true` to
  `KnownTypes`.
- `internal/release/release.go` — add `"unscheduled": true` to
  `ValidStatuses`; extend `Validate` error message to mention the new
  value; add an `UpdatedAt`-derived `Version time.Time` field if not
  already serialised.
- `internal/release/file.go` (new) — `type File struct { Title, Slug,
  Status string; StartDate, EndDate *time.Time; UpdatedAt time.Time;
  Body string }`. Functions: `func Parse(path string, raw []byte)
  (*File, error)` (YAML frontmatter + body split, validation matching
  DR-1), `func (f *File) Marshal() ([]byte, error)` (frontmatter + body
  with deterministic key order: `title, type, status, start_date,
  end_date, updated_at`), `func Slugify(name string) string` (lowercase,
  spaces→`-`, strip `[^a-z0-9-]`, collapse runs of `-`, trim leading/
  trailing `-`; returns empty string when no usable chars remain).
- `internal/release/file_test.go` (new) — unit tests for `Slugify`,
  `Parse` error cases (missing fields, unknown status, `end_date <
  start_date`, malformed dates), and round-trip `Marshal` →`Parse`
  equality.

**Acceptance criteria.**
- `go test ./internal/artifact/... ./internal/release/...` passes with
  the new cases.
- Parsing a release `.md` file no longer produces an `unknown type
  "release"` error in the artifact index logs.
- `release.Slugify("Q1 2026")` == `"q1-2026"`;
  `release.Slugify("🚀🚀")` == `""` (caller is responsible for the
  `release-<id>` fallback — see Milestone B2).
- `release.Validate` rejects status `"foo"` and accepts each of
  `planned`, `active`, `shipped`, `unscheduled`.
- A round-trip (`Marshal` → write → read → `Parse`) yields a `File`
  byte-for-byte equal to the original modulo `updated_at` precision
  (DR-7 atomicity / round-trip).

## Milestone B2 — Disk sync on POST / PUT / DELETE

**Description.** Add a `release.DiskSync` collaborator that the HTTP
handlers call inside the same request as the DB mutation. Implements
slug collision detection (HTTP 409), the `release-<id>` emoji-only
fallback (Resolved Question 4), and the safe-write `*.tmp` + atomic
rename pattern (DR-7). Adds a `file_path` field to the JSON `Release`
response (DR-6). Pipes the originating-mutation marker into the project's
in-memory "expected file event" set so the watcher (Milestone B4) treats
the event as a no-op.

**Files to change.**
- `internal/release/disksync.go` (new) — `type DiskSync struct {
  sandbox sandbox.Resolver; expected *ExpectedEvents }`. Methods:
  `Write(projectRoot string, r *Release) (relPath string, err error)`,
  `Rename(projectRoot, oldSlug, newSlug string, r *Release) (string,
  error)`, `Delete(projectRoot, slug string) error`. Each method records
  the absolute target path in `expected` before touching disk so the
  watcher can suppress the resulting fsnotify event.
- `internal/release/disksync_test.go` (new) — unit tests covering atomic
  rename (writes go through `*.tmp`), sandbox rejection of `..`, and the
  fallback `release-<id>` slug path.
- `internal/release/store.go` — extend `Create`, `Update`, `Delete` to
  accept a `*DiskSync` parameter and a project-root path; on `Create`
  if `Slugify(name) == ""`, derive `slug = fmt.Sprintf("release-%d",
  id)` after the row insert (requires a follow-up `UPDATE releases SET
  slug = ?` if a `slug` column is added — see below). On `Update` with
  name change, call `Rename`; otherwise `Write`. On `Delete` call
  `Delete` after the row is gone.
- `internal/index/index.go` — add `slug TEXT NOT NULL DEFAULT ''` to
  the `releases` schema and write a migration; populate slugs from
  existing `name` rows on migrate. Add `UNIQUE(project_id, slug)`
  constraint.
- `internal/http/releases.go` — wire `DiskSync` through the project
  context; on collision (`UNIQUE` violation or `Stat` of target shows
  existing file) return `409 Conflict` with `{"error":"release slug
  already in use"}`. Add `file_path` to the `Release` JSON marshaller
  (computed as `lifecycle/releases/<slug>.md`). On `PUT`, when the body
  contains an `updated_at` older than the row's current `updated_at`,
  return `409 Conflict` with `{"error":"release was modified"}`
  (Resolved Question 2 — conflict detection).
- `internal/project/project.go` — initialise and inject the
  `ExpectedEvents` registry alongside `DiskSync`.

**Acceptance criteria.**
- `POST /api/v1/projects/:project/releases` with `{"name":"Q1 2026"}`
  returns 201, the response body includes `"file_path":
  "lifecycle/releases/q1-2026.md"`, and the file exists with the correct
  frontmatter.
- A second `POST` with name `"Q1-2026"` returns 409 (DR-1 collision).
- `POST` with `name: "🚀"` returns 201 with `file_path:
  "lifecycle/releases/release-<id>.md"`.
- `PUT /releases/:id` changing name from `"Q1 2026"` to
  `"Q1 2026 hotfix"` renames the file (old gone, new present) inside the
  request; `start_date`/`status` edits rewrite the same file in place.
- `DELETE /releases/:id` removes the file from disk.
- `PUT` with stale `updated_at` returns 409.
- A simulated write failure (read-only mount) causes the surrounding
  DB transaction to roll back (DR-2).
- All file writes go through `sandbox.Resolver`; an injected slug
  containing `../etc` is rejected with 400.

## Milestone B3 — Rehydrate, backfill, and CLI/admin trigger

**Description.** Implement startup synchronisation in both directions
plus the manual rehydrate trigger (Resolved Question 3): HTTP
`POST /api/v1/projects/:project/releases/rehydrate` and a CLI command
`kaos-control releases rehydrate --project <id>`. Step runs after schema
migration and before the project's `ready` flag flips.

**Files to change.**
- `internal/release/rehydrate.go` (new) — `Rehydrate(ctx, store,
  projectRoot string) (RehydrateResult, error)` reads
  `lifecycle/releases/*.md`, skips files failing DR-1 validation with a
  WARN log, and upserts via `Store`. Returns counts (`inserted`,
  `skipped`, `errors`).
- `internal/release/backfill.go` (new) — `Backfill(ctx, store, sync,
  projectRoot string) (BackfillResult, error)` writes one markdown file
  per DB row when `lifecycle/releases/` is missing or empty (DR-5).
- `internal/project/project.go` — on load, after schema migrate:
  1. `count := store.Count(projectID)`.
  2. If `count == 0` and `lifecycle/releases/` has `.md` files →
     `Rehydrate`.
  3. If `count > 0` and `lifecycle/releases/` has no `.md` files →
     `Backfill`.
  4. Mark project ready only after both branches return.
- `internal/http/releases.go` — `POST /releases/rehydrate` runs
  `Rehydrate` against the current project regardless of count, returning
  `{"inserted":N,"skipped":N,"errors":[...]}` as JSON.
- `cmd/kaos-control/main.go` (and/or `internal/backfillcmd/`) — register
  a new subcommand `releases rehydrate --project <id>` that opens the
  project, runs `Rehydrate`, prints the result, and exits.
- `internal/release/rehydrate_test.go` (new) — table tests covering:
  valid file inserted, invalid file (end < start) skipped with WARN log
  recorded, idempotency (running twice yields the same DB state).

**Acceptance criteria.**
- Wiping the SQLite DB and restarting against a `lifecycle/releases/`
  directory with 3 valid + 1 invalid file inserts 3 rows and logs one
  WARN. Project still loads.
- Running `Rehydrate` twice in succession is idempotent (DR-7).
- On a project with 5 DB rows and no `lifecycle/releases/` directory,
  first startup produces 5 `.md` files; the second produces no further
  writes (DR-5).
- Rehydrating ≤ 200 files completes in < 250 ms locally (DR-7).
- `POST /releases/rehydrate` returns the result struct and is allowed
  only for authenticated users; CLI form prints the same struct as JSON
  on stdout.
- Backfill failure (unwritable dir) emits ERROR and does not block
  project load (DR-5).

## Milestone B4 — Watcher integration & WebSocket events

**Description.** Extend the existing `fsnotify` watcher to cover
`lifecycle/releases/`, reusing the 150 ms debounce (Resolved Question 5).
Watcher-driven changes upsert/delete via `Store`, broadcast a
`release.changed` WS event (envelope shape matching `artifact.indexed`),
and respect the `ExpectedEvents` mark so API-originating writes do not
loop.

**Files to change.**
- `internal/watcher/watcher.go` — register `lifecycle/releases/` as a
  watched path; dispatch events with `.md` extension to a new
  `releaseHandler` callback that the project supplies.
- `internal/watcher/release_handler.go` (new) — translates fsnotify
  events into store mutations: `CREATE`/`WRITE` → parse file →
  `Store.UpsertBySlug`; `REMOVE` → `Store.DeleteBySlug`; `RENAME` →
  detect old+new event pair within debounce window and emit a rename.
  Before mutating, check `ExpectedEvents.Consume(absPath)`; if
  consumed, return without emitting a WS event.
- `internal/hub/hub.go` — define `MsgReleaseChanged` envelope
  `{"type":"release.changed","project":"<id>","release":{...}}` mirroring
  `artifact.indexed`.
- `internal/release/store.go` — add `UpsertBySlug`, `DeleteBySlug`,
  `Count(projectID)` helpers.
- `internal/watcher/release_handler_test.go` (new) — table tests using
  an in-memory store + temp dir: write file → row inserted + WS event
  emitted; API-originating expected event → row updated, no WS event;
  rename pair → single rename event.

**Acceptance criteria.**
- Manually writing `lifecycle/releases/q2-2026.md` upserts the row
  within one debounce tick and emits exactly one `release.changed`
  WebSocket message.
- Deleting the file deletes the row; renaming the file
  (`q2-2026.md` → `q2-2026-hotfix.md`) updates the row's `slug` in
  place rather than producing delete+insert.
- API-driven `POST` produces zero watcher-emitted WS events (loop
  prevention — DR-4); the API handler still emits its own
  `release.changed` after a successful response.
- A release file with `end_date < start_date` written manually is
  rejected by the watcher with a WARN log; no row is modified.
- Existing `artifact.indexed` events under `lifecycle/ideas/` etc.
  continue to fire unchanged.
