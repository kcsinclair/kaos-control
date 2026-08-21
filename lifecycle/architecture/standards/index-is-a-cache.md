---
title: The index is a cache — disk is authoritative
type: doc
status: approved
lineage: standard-index-is-a-cache
created: "2026-08-21T11:50:00+10:00"
labels:
    - standard
    - architecture
    - index
---

# Standard: The index is a cache — disk is authoritative

The SQLite index (`internal/index/`) is a **rebuildable cache** over the
markdown artifacts under `lifecycle/`. The files on disk are the single source
of truth. Any code or agent change must preserve this.

## Rules

- **Never treat the index as the system of record.** Anything in the index must
  be reconstructable from disk by a full re-scan.
- **Writes go to disk, then re-index.** API writes (`PUT/POST /artifacts/*`)
  persist the file and re-index synchronously before responding; the watcher
  (`fsnotify`, 150 ms debounce) re-indexes external edits (e.g. files
  replicated from an Obsidian vault).
- **A schema/driver mismatch triggers a rebuild, not a migration.** Dropping and
  rebuilding the cache from disk is always a valid recovery — see
  [[adr-0003-pure-go-sqlite-index]].
- **Don't stamp derived state back onto files from the index path.** Indexing is
  read-only with respect to file content (e.g. `created:` is backfilled into the
  DB, never written into files by the indexer). File mutations belong to the API
  write path, the `backfill-created` CLI, or agents — not the index.
- **Guard against stale-vs-disk desync.** When a SHA/mtime guard could lock a
  cached status stale against disk, prefer reconciling from disk over trusting
  the cache.
