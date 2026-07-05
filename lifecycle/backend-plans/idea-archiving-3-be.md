---
title: "Backend Plan: Recursive Subdirectory Support for Artifact Directories"
type: plan-backend
status: approved
lineage: idea-archiving
parent: lifecycle/requirements/idea-archiving-2.md
created: "2026-07-05T16:00:00+10:00"
priority: high
labels:
    - backend
    - watcher
    - index
    - artifacts
---

# Backend Plan: Recursive Subdirectory Support for Artifact Directories

Implements the backend half of [idea-archiving-2](../requirements/idea-archiving-2.md).
Pairs with the frontend plan [[idea-archiving-4-fe]] (which consumes the new
`rel_path` field and breadcrumb) and the test plan [[idea-archiving-5-test]].

## Context from the code

Most of the machinery is *already* recursion-capable and this plan leans on that:

- `internal/index/index.go` `Scan` uses `filepath.WalkDir` (index.go:346) — the
  startup scan already descends subdirectories.
- `internal/watcher/watcher.go` `addDirRecursive` (watcher.go:260) already
  walks and adds a watch per directory, and re-adds on `fsnotify.Create`
  (watcher.go:180) so new dirs get watched at runtime.
- `internal/sandbox/sandbox.go` `Resolve` (sandbox.go:19) already handles
  multi-segment relative paths and rejects `..` / absolute / symlink escapes,
  including walking up to the nearest existing ancestor for not-yet-created
  intermediate dirs (sandbox.go:50-65).
- Artifacts are keyed in SQLite by the full project-relative `path`
  (`artifacts.path` PRIMARY KEY, index.go:1716), and lineage index allocation
  (`NextIndexForLineage`, index.go:927) queries by `lineage` across **all**
  rows regardless of folder.

The **actual flat assumptions** to fix are narrow:

1. No stored/exposed root-relative path field (req #4).
2. `artifact.parsePathComponents` (artifact.go:310) derives `stage` from a
   fixed `lifecycle/<stage>/<file>` split and breaks the "root = top-level dir"
   rule (req #6) once nesting deepens.
3. `handleCreateArtifact` (write.go:38-89) hardcodes
   `lifecycle/<stageDir>/<file>` with no subdirectory support.
4. `handleGetArtifact` (artifacts.go:99-111) joins the user path directly
   instead of going through `sandbox.Resolve` (req #8/#13).
5. Dot-directory skipping, the 5000-watch cap, and the new-subdir-with-contents
   race are not yet handled explicitly.

---

## Milestone 1 — Root-relative path derivation in the artifact model

**Description.** Give every parsed artifact a canonical, forward-slash
root-relative path derived purely from its location under `lifecycle/<stage>/`,
and make stage assignment depend only on the top-level directory under
`lifecycle` (req #5, #6, #14). Flat files yield the bare filename (req #12).

**Files to change.**
- `internal/artifact/artifact.go`
  - Add `RelPath string` to `type Artifact` (artifact.go:55).
  - Rewrite `parsePathComponents` (artifact.go:310): split the project-relative
    path on `/`; require `parts[0] == "lifecycle"`; set `Stage = parts[1]`
    (the top-level dir, never a nested one); set
    `RelPath = strings.Join(parts[2:], "/")` (empty-dir-component collapses to
    bare filename for flat files). Always use `/` regardless of host OS —
    callers pass `filepath.ToSlash`-normalised paths.
  - Ensure `Path` remains the full `lifecycle/<stage>/<sub>/<file>.md` and
    `Slug`/`Index`/`StageSuffix` continue to come from `ParseFilename` on the
    filename stem only (artifact.go:324) — unaffected by depth.
- `internal/index/index.go`
  - In `IndexFile` (index.go:392), normalise `relPath` with `filepath.ToSlash`
    before `artifact.Parse` so Windows separators never reach the model.

**Acceptance criteria.**
- `Parse` of `lifecycle/ideas/done/login.md` yields `Stage == "ideas"`,
  `RelPath == "done/login.md"`, `Slug`/`Index` identical to the same file at
  `lifecycle/ideas/login.md`.
- `Parse` of `lifecycle/ideas/login.md` yields `RelPath == "login.md"`.
- `Parse` of `lifecycle/ideas/2026/q3/release-x.md` yields `Stage == "ideas"`,
  `RelPath == "2026/q3/release-x.md"`.
- `RelPath` always uses `/`; a path containing OS-native `\` separators still
  produces forward-slash `RelPath` (unit test with a synthetic path).

---

## Milestone 2 — Persist and expose `rel_path` (lazy back-fill, no rebuild)

**Description.** Store the root-relative path in the index and surface it on
every API artifact representation, migrating existing indexes lazily without a
schema-version bump or full rebuild (req #4; Resolved Q5 = "back-filled
lazily").

**Files to change.**
- `internal/index/index.go`
  - `createSchema` (index.go:1708): add `rel_path TEXT NOT NULL DEFAULT ''` to
    the `artifacts` table DDL.
  - `checkSchema`/migration path (index.go:1518, 1572): add an idempotent
    `ALTER TABLE artifacts ADD COLUMN rel_path TEXT NOT NULL DEFAULT ''`,
    reusing the existing "duplicate column name discarded silently" pattern
    (index.go:1572). Do **not** bump `schemaVersion` (index.go:36) and do **not**
    trigger `dropAndRecreate`. Rows written before migration keep `''` until
    their next re-index (the startup `Scan` and any write re-index back-fill it).
  - `Upsert` (index.go:542): write `artifact.RelPath` into the new column.
  - `ArtifactRow` (index.go:639): add `RelPath string` with JSON tag
    `"rel_path"`. Populate it in every row-scanning query (`Get` index.go:705,
    list, `ListByLineage`, `ListAllGroupedByLineage`, etc.).
- `internal/http/artifacts.go`
  - Ensure the list response (artifacts.go:84) and single-get response
    (artifacts.go:123) carry `rel_path` (automatic once `ArtifactRow` has it;
    verify the single-get DTO includes it).

**Acceptance criteria.**
- A fresh index stores `rel_path` for both flat and nested artifacts.
- Opening a pre-existing index built before this change does **not** rebuild
  (schema version unchanged); after the next `Scan` every row has a populated
  `rel_path`.
- `GET /p/{project}/artifacts` returns `rel_path` on each item; a flat file's
  `rel_path` equals its bare filename; a nested file's is its `sub/dir/file.md`.
- `GET /p/{project}/artifacts/{path}` returns `rel_path` in the detail payload.

---

## Milestone 3 — Recursion hardening: dot-dirs, new-subdir race, watch cap

**Description.** Make recursive discovery and watching robust: skip dotfiles and
dot-directories in both scan and watcher (req #10), index files that already
exist inside a subdirectory created in a single atomic operation (AC:
"brand-new subdirectory at runtime"), and cap watched directories at 5000 with
a logged fallback (req #11; Resolved Q3 = 5000, Q4 = dot-prefix only).

**Files to change.**
- `internal/index/index.go`
  - In the `Scan` `WalkDir` callback (index.go:346): when `d.IsDir()` and the
    base name starts with `.`, return `filepath.SkipDir`; skip dot-*files* too.
- `internal/watcher/watcher.go`
  - `addDirRecursive` (watcher.go:260): skip directories whose base name starts
    with `.` (return `filepath.SkipDir`); do not `fsw.Add` them.
  - Create handling (watcher.go:180): when a `Create` event names a **directory**,
    call `addDirRecursive` on it (not just `fsw.Add`) and enqueue re-index of any
    `*.md` already inside it, so a `mkdir sub && cp file sub/` that races the
    watcher is not missed. Keep the 150 ms debounce (`fire`, watcher.go:113).
  - Add a package-level cap `const maxWatchedDirs = 5000`. Track the count in
    `addDirRecursive`; when adding a watch would exceed it, stop adding new
    watches and `log` a single warning (per requirement, degradation is
    acceptable above the cap — do not crash). `shouldProcess` (watcher.go:275)
    already filters dotfiles for the change path; keep that.

**Acceptance criteria.**
- `.git`, `.trash`, and any `.`-prefixed directory under an artifact root are
  neither indexed nor watched (verified by scan result + no watch added).
- A dotfile `lifecycle/ideas/.draft.md` is not indexed.
- Creating `lifecycle/ideas/newdir/` containing an artifact in one operation at
  runtime results in the artifact being indexed and `newdir` being watched,
  without restart, within the debounce window.
- With >5000 directories the watcher logs the cap warning and continues serving;
  the process does not panic or exit.

---

## Milestone 4 — Create with subdirectory + safe GET + move identity

**Description.** Let `POST /artifacts` place a new artifact into an optional
subdirectory, route the GET handler through the sandbox, and confirm that moving
a file between folders preserves identity via lineage+index (req #7, #8, #9,
Resolved Q1 = "type+lineage+index is unique").

**Files to change.**
- `internal/http/write.go`
  - `handleCreateArtifact` (write.go:38-89): accept an optional `subdir` (or
    `rel_dir`) field on the create request. Build
    `relPath := path.Join("lifecycle", stageDir, subdir, filename)` using
    forward slashes, then **validate through `sandbox.Resolve(p.Entry.Path,
    relPath)`** before writing (matching the update path at write.go:234) so a
    `..` in `subdir` is rejected (req #8). Empty `subdir` preserves today's
    flat behaviour exactly (req #12). `buildFilename` (write.go:669) is unchanged
    — it still produces the leaf filename; only the directory prefix changes.
- `internal/http/artifacts.go`
  - `handleGetArtifact` (artifacts.go:99-111): replace the direct
    `filepath.Join(p.Entry.Path, relPath)` with
    `sandbox.Resolve(p.Entry.Path, relPath)` so nested GETs are traversal-safe
    and consistent with PUT/DELETE (req #8, #13).
- No new move endpoint (Resolved Q1). Moving is a filesystem operation the
  watcher reflects: the move-away path is removed via `idx.DeletePath` on the
  index error (watcher.go:221-225) and the move-in path is indexed fresh. The
  artifact's *logical* identity (type + lineage + index, from frontmatter and
  filename) is unchanged; graph edges keyed on lineage/parent survive. This
  milestone adds no code for move beyond confirming the above holds and is
  covered by [[idea-archiving-5-test]].

**Acceptance criteria.**
- `POST` with `subdir: "archive"` creates `lifecycle/<stage>/archive/<slug>-<n>...md`
  and it is immediately indexed and returned by the list API.
- `POST`/`PUT` with a `subdir`/path containing `..` that escapes the root is
  rejected with the same error surface as the existing sandbox rejection.
- `GET /p/{project}/artifacts/archive/foo.md` succeeds for a nested artifact and
  a traversal attempt is rejected.
- Moving `lifecycle/ideas/foo.md` → `lifecycle/ideas/archive/foo.md` on disk
  results in exactly one indexed artifact at the new path with unchanged
  `type`, `status`, `lineage`, `index`, `parent`; the old path is gone from the
  index; no duplicate lineage/index row exists.

---

## Milestone 5 — Cross-folder lineage-uniqueness verification

**Description.** Confirm (and, if needed, tighten) that lineage-index uniqueness
is enforced across the whole root, not per folder (req #9). `NextIndexForLineage`
(index.go:927) already selects by `lineage` across every row regardless of
`path`, so allocation is inherently cross-folder; a genuine collision (two files
sharing lineage+index in different subfolders) must be surfaced identically to a
flat collision today.

**Files to change.**
- `internal/index/index.go`
  - Audit `NextIndexForLineage` (index.go:927) and `Upsert` (index.go:542) to
    confirm neither is scoped by directory. Because `artifacts.path` is the
    primary key, two different subfolder files with the same filename collide on
    disk anyway; two files with the same lineage+index but different filenames
    coexist in the index exactly as they would flat — so no folder-specific
    branch is introduced. If a collision-surfacing log/metric exists for the
    flat case, ensure the nested case flows through the same path (no code
    change expected beyond a regression guard).

**Acceptance criteria.**
- `NextIndexForLineage("login")` returns the same next index whether the
  existing lineage members live in the root or scattered across subfolders.
- Two artifacts sharing lineage `x` and index `3` placed in
  `lifecycle/ideas/a/` and `lifecycle/ideas/b/` are surfaced as a collision by
  the same mechanism as two flat files would be — no special-casing by folder.

---

## Out of scope (from requirement Non-goals)

- No auto-move on status change; no folder-name semantics (`done`/`archive` are
  neutral); no symlink traversal or non-markdown files; no per-folder
  permissions; no lineage filename-convention changes.
- UI folder grouping/filtering beyond exposing `rel_path` is deferred — the
  frontend consumer is scoped in [[idea-archiving-4-fe]].
