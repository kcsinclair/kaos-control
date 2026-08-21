---
title: Pure-Go SQLite for the artifact index (modernc.org/sqlite, no cgo)
type: adr
status: approved
lineage: adr-pure-go-sqlite-index
created: "2026-08-21T11:45:00+10:00"
labels:
    - adr
    - architecture
    - index
    - build
related_to:
    - adr-0004-embedded-spa-single-binary
---

# ADR-0003: Pure-Go SQLite for the artifact index (modernc.org/sqlite, no cgo)

## Context

kaos-control indexes the markdown artifacts under `lifecycle/` into a
queryable cache (`internal/index/`) that backs list/board/graph/search. A
relational store with indexes and ad-hoc queries is the natural fit, and
SQLite is the obvious embedded choice.

The canonical Go SQLite binding, `mattn/go-sqlite3`, is a cgo wrapper around
the C amalgamation. cgo would compromise the project's defining constraint —
a single, self-contained, cross-compilable binary (see
[[adr-0004-embedded-spa-single-binary]]): it requires a C toolchain on every
build host, makes cross-compilation painful, slows builds, and complicates
static linking.

## Decision

Use **`modernc.org/sqlite`**, a pure-Go transpilation of SQLite, as the index
driver. No cgo; `CGO_ENABLED=0` builds work everywhere.

The index remains a **rebuildable cache**, never a system of record — disk is
authoritative (see [[index-is-a-cache]]). This bounds the cost of the pure-Go
driver's trade-offs: if the schema or driver ever needs to change, the cache is
dropped and rebuilt from disk rather than migrated as precious data.

## Consequences

- Single binary, `CGO_ENABLED=0`, trivial cross-compilation — no C toolchain in
  CI or on contributor machines.
- The pure-Go driver is somewhat slower than the cgo binding under heavy write
  load. Acceptable: the index is a single-node cache over a modest number of
  markdown files, updated incrementally by the watcher, not a hot OLTP store.
- Because disk is authoritative and the index is disposable, driver/schema
  churn is low-risk — a schema-version mismatch triggers a full rebuild from
  disk (`internal/index`), not a data-loss event.
- Keeps the door open to the same binary running on any OS/arch the Go
  toolchain targets, with no per-platform native artifacts.
