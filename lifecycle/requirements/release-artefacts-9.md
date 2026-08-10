---
title: Release Artefacts in Markdown — reverse to markdown-authoritative
type: requirement
status: draft
lineage: release-artefacts
priority: high
parent: requirements/release-artefacts-2.md
labels:
    - releases
    - persistence
    - architecture
    - cache
assignees:
    - role: product-owner
      who: agent
---

# Release Artefacts in Markdown — reverse to markdown-authoritative

## Problem

[release-artefacts-2.md](release-artefacts-2.md) stated the correct goal
(§Problem: *"markdown authoritative, SQLite a rebuildable cache… releases must
follow the same model"*) but its **Detailed Requirements shipped the opposite**,
inherited from the originating idea
[release-artefacts.md](../ideas/release-artefacts.md) (*"keep the SQLite DB up to
date… if database table is empty, reload from files. On DB change, sync to
disk"* — DB-authoritative, disk-as-backup):

- **DR-2** — "write the file **after** the row is inserted"; the file is a
  downstream mirror of the DB row.
- **DR-3** — "disk is read only when DB is empty"; whenever the table is
  populated the DB wins and disk is ignored.
- **DR-5** — Backfill writes markdown **from** DB rows, treating the DB as a
  legitimate origin.

The code implemented those DRs faithfully
([internal/release/store.go](../../internal/release/store.go) opens a
transaction, writes the SQL row, then mirrors to disk inside the same tx). The
result is that **writes land in the DB and can fail to reach disk**: observed
2026-08-10, the `releases` table rows read `updated_at: 2026-08-10` while
[kc-release4.md](../../lifecycle/releases/kc-release4.md) still read
`2026-07-07`. This is exactly the drift "disk is authoritative" exists to
prevent, and it is unique to releases — every other artifact type is
disk-authoritative.

A second, unintended consequence: because `releases` is a lifecycle stage
([config.go:619](../../internal/config/config.go#L619)) and `release` is a known
type, each release file is **also** indexed into the generic `artifacts` table.
Every release therefore exists as two cache rows in two tables, parsed by two
independent parsers ([release/file.go](../../internal/release/file.go) and the
generic `artifact.Parse`).

## Goal

Reverse the write path so a release's **markdown file is authoritative** and the
`releases` table is a rebuildable cache, matching every other artifact type.
Collapse to a **single cache** (the dedicated `releases` table); stop caching
release files in the `artifacts` table.

This requirement **supersedes DR-2, DR-3, and DR-5 of
[release-artefacts-2.md](release-artefacts-2.md)**. DR-1 (file format), DR-4
(live watcher), DR-6 (API surface), and DR-7 (non-functional) carry forward
unchanged except where noted below.

## Implementation status (branch `release-markdown-authoritative`, 2026-07-17)

DR-1, DR-2, DR-3, DR-4 are **implemented**. DR-5 (id→slug external key) is
**deferred** — see note below.

- **DR-1** — [store.go](../../internal/release/store.go) `Create`/`Update`/`Delete`
  now write markdown first, then refresh the cache (`Create` conflict-checks and
  returns `ErrConflict`; `Update` refreshes by id; `Delete` removes the file
  before the row). Emoji-only fallback slug is now a name hash (`fallbackSlug`),
  not `release-<id>`, so it needs no DB id pre-insert.
- **DR-2** — [rehydrate.go](../../internal/release/rehydrate.go) `Rehydrate` runs
  unconditionally on load and prunes cache rows with no file (`PruneExcept`); a
  missing dir prunes to empty.
- **DR-3** — `Backfill` deleted; [project.go](../../internal/project/project.go)
  `startupReleaseSyncTyped` always rehydrates.
- **DR-4** — [index.go](../../internal/index/index.go) `IndexFile` skips
  `type: release`; a startup `DELETE FROM artifacts WHERE type='release'` purges
  historical rows.
- **DR-5 deferred.** The `{id}` API routes are unchanged. In normal operation the
  autoincrement id is stable: `Rehydrate`'s `UpsertBySlug` hits
  `ON CONFLICT(project_id,name) DO UPDATE`, preserving each row's id across a
  restart. The id only changes on a **full cache wipe** (DB deleted), which is
  the disaster-recovery case where stale `{id}` links breaking is acceptable.
  Because {id} is stable in normal use, the slug-key migration is a separable
  follow-up rather than a blocker. Tracked as an open question below.

## Detailed Requirements

### DR-1 (revised): Markdown is written first, cache follows
- On `POST` / `PUT` / `DELETE /releases`, the server MUST **write, rename, or
  delete the markdown file first** (via the existing atomic tmp+rename in
  [disksync.go](../../internal/release/disksync.go), which already registers
  `ExpectedEvents` to suppress the watcher echo), and **only then** refresh the
  `releases` cache row by parsing the file just written
  (`UpsertBySlug` / `DeleteBySlug`, already present at
  [store.go L253-279](../../internal/release/store.go#L253)).
- The cache refresh MUST complete synchronously before the HTTP response, so
  read-after-write is consistent (mirrors `PUT /artifacts` re-indexing
  synchronously).
- A failed **disk** write MUST return an error and MUST NOT touch the cache
  (inverse of today's DB-rollback-on-disk-failure at
  [store.go L178-182](../../internal/release/store.go#L178)).

### DR-2 (revised): Disk→DB rehydrate is unconditional at load
- At project load, the `releases` cache is rebuilt from
  `lifecycle/releases/*.md` **regardless of whether the table already has rows**
  (`release.Rehydrate`, [rehydrate.go](../../internal/release/rehydrate.go)).
  Disk is the source; the table is discarded/replaced, not merged.
- The "only when the table is empty" condition of the old DR-3 is removed.

### DR-3 (revised): Remove Backfill (DB→disk)
- Delete the DB→disk backfill branch in `startupReleaseSyncTyped`
  ([project.go L377-408](../../internal/project/project.go#L377)) and
  [backfill.go](../../internal/release/backfill.go). A populated DB with an empty
  `lifecycle/releases/` directory means **no releases** — the DB is never
  allowed to resurrect files.

### DR-4 (new): Single cache — release files leave the `artifacts` table
- The index Scan / `IndexFile` path MUST NOT upsert `type: release` files into
  the `artifacts` table; the `releases` table is the sole cache.
- The idea/defect count join in `Store.Get`
  ([store.go L45-65](../../internal/release/store.go#L45)) counts *other*
  artifacts assigned to a release via their `release:` frontmatter field, not the
  release row itself, and MUST continue to work.

### DR-5 (new): Slug is the durable external key
- The `releases.id` autoincrement PK has **no markdown equivalent** and is
  unstable across a cache rebuild. The external API key for a release MUST become
  its **slug**, not `id`
  ([releases.go L127-359](../../internal/http/releases.go#L127) key reads/updates
  on `{id}` today). Existing `{id}` routes either move to `{slug}` or resolve
  `id` through the slug so links survive a rehydrate.

## Open questions

- **Artifact-list visibility (the main UI ripple).** Removing release files from
  the `artifacts` table drops them from the artifact list stage filter
  ([ArtifactListView.vue:104](../../web/src/views/project/ArtifactListView.vue#L104))
  and anything that enumerates them there;
  [RoadmapView.vue:66](../../web/src/views/project/RoadmapView.vue#L66) already
  filters `type !== 'release'` out of the backlog. Confirm releases should live
  **only** on the Roadmap view, and check `graph-show-releases-overlay-*` does
  not read release rows from the artifacts table.
- **`id`→`slug` migration.** Do we break the `{id}` API contract outright, or
  keep `{id}` resolving via slug for one release for compatibility?
- **Rename ordering.** `PropagateRename`
  ([propagate.go](../../internal/release/propagate.go)) rewrites the `release:`
  field on assigned artifacts. Define ordering: rename the file → propagate to
  assignees → refresh caches.

## Acceptance Criteria

- [ ] Creating/editing/deleting a release writes the markdown file **before** any
      cache write; killing the process between the disk write and the cache
      refresh leaves disk correct and the cache self-heals on next load.
- [ ] Editing a release via the API and then wiping the `releases` **table**
      (not the file) and restarting reproduces the identical release from disk.
- [ ] A release edited **on disk** (e.g. Obsidian) is reflected in the cache and
      API within one watcher debounce, with no DB-origin write clobbering it.
- [ ] A populated `releases` table with an empty `lifecycle/releases/` directory
      results in **zero** releases after load (no backfill resurrection).
- [ ] `type: release` files do **not** appear in the `artifacts` table; the
      Roadmap, Gantt, rename propagation, and release↔artifact assignment all
      still work.
- [ ] Release API links remain valid across a cache rebuild (slug-keyed).

## Related

- [release-artefacts-2.md](release-artefacts-2.md) — superseded DRs (DB-first).
- [release-artefacts.md](../ideas/release-artefacts.md) — originating idea, whose
  DB-authoritative framing this requirement reverses.
