---
created: "2026-08-11T16:37:29+10:00"
title: A reindex command/action to rebuild the artifact cache on demand
type: idea
status: draft
lineage: artifact-reindex-command
priority: medium
labels:
    - index
    - cli
    - ops
    - maintainability
assignees:
    - role: product-owner
      who: agent
---

# A reindex command/action to rebuild the artifact cache on demand

## Context

The SQLite index is a cache; disk is authoritative. But there is **no supported
way to force a rebuild / re-evaluation on demand.** This bites whenever the
*derivation logic* changes rather than a file: the startup scan skips
already-indexed, unchanged files via the mtime guard
([index.go Scan](../../internal/index/index.go)) and the SHA+status guard in
`IndexFile`, so a logic change never reaches the existing corpus.

Concrete case (2026-08-11): the open-questions heading detector was made
case-insensitive so `## Open questions` is recognised
([fix f0d5d17e](../../internal/artifact/artifact.go)). But every already-indexed
artifact was skipped on the next start, so the fix had **no effect** — a draft
with a lowercase heading stayed `draft` / `has_open_questions=0` and never
surfaced in "Awaiting Answers". The only way to make it land was to **bump
`schemaVersion`** ([bc8bb12d](../../internal/index/index.go)) to trigger
`dropAndRecreate`, forcing a full rebuild from disk.

That workaround is a blunt instrument:

- It requires a **code change + recompile** — useless as an operational tool.
- It is a **global** rebuild bundled into a release; you can't run it ad-hoc.
- The other options are worse: deleting `index.db` destroys `agent_runs` (the
  cost/token history) and `events`, which are **not** rebuildable from disk;
  editing each file by hand is tedious and mutates content.

Every future derivation change hits the same wall — new frontmatter fields,
status-vocab changes, auto-block rule tweaks, sentinel changes, link
normalisation. A reindex command is the standard escape hatch.

## The problem to solve

Provide a safe, first-class way to **rebuild the artifact cache from disk for a
project on demand** — re-running parsing, indexing, and the derived transitions
(auto-block/unblock, open-questions flag) over every artifact — **without**
destroying the non-cache tables (`agent_runs`, `events`, `scheduler_*`).

The machinery already exists: `dropAndRecreate()`
([index.go](../../internal/index/index.go)) already drops only the
disk-rebuildable tables (artifacts, links, labels, releases, …) and **explicitly
excludes `agent_runs`**, and `Scan(stages)` repopulates from disk. This idea is
mostly about exposing that as a supported entry point.

## Sketch

1. **CLI subcommand** — `kaos-control reindex --project <id>` (following the
   `backfill` subcommand shape in [cmd/kaos-control](../../cmd/kaos-control/backfill.go)
   and the `main.go` dispatch switch). Runs `dropAndRecreate` + `Scan` against
   the project's index, then reports a summary (files scanned, statuses changed,
   artifacts newly blocked/unblocked). `--all` for every registered project;
   `--dry-run` to report what *would* change without writing.
2. **GUI admin action** — a "Rebuild index" button (admin-only) that triggers
   the same operation **in-process** on the running server and broadcasts the
   usual `artifact.indexed` events so open clients refresh live. This is the more
   convenient path for the exact scenario that motivated this (a logic fix that
   needs to reach existing artifacts without a restart).
3. **Preserve non-cache state** — reuse `dropAndRecreate`'s existing exclusion of
   `agent_runs` / `events` / `scheduler_*`; releases rehydrate from
   `lifecycle/releases/*.md` (already disk-authoritative).
4. **Re-run derived transitions** — because the rebuild goes through `IndexFile`,
   `applyOpenQuestionTransition` fires per artifact, so auto-block/unblock and
   the `has_open_questions` flag are recomputed — which is the whole point.

## Open questions (to think about)

- **CLI (server stopped) vs in-process (live) vs both?** SQLite is single-writer,
  so an offline CLI must run with the server down. An in-process API/GUI action
  avoids that but must coordinate with the watcher and broadcast progress. Both
  have value; which ships first?
- **Full drop vs in-place re-evaluate.** `dropAndRecreate` is simplest and safe
  (agent_runs excluded), but a lighter "re-run detection/transitions over
  existing rows without dropping" would avoid rebuilding link/label tables.
  Worth it, or is the full rebuild fine given the tables are cheap to repopulate?
- **Scope granularity.** Whole project only, or also `--path <glob>` /
  single-artifact reindex for targeted fixes?
- **Keep the schema-bump mechanism too?** `schemaVersion` bumps are the right
  trigger for *shipped* migrations (automatic on upgrade); `reindex` is the
  *ad-hoc/manual* tool. They coexist — this idea doesn't remove the former.
- **Guardrails.** Reindex is disruptive (drops + rescans, may flip statuses via
  auto-block). Admin-only; confirmation in the GUI; a `--dry-run` summary first.

## Related

- [index.go `dropAndRecreate` / `Scan`](../../internal/index/index.go) — the
  existing rebuild machinery to expose.
- [idea-archiving-2.md](../requirements/idea-archiving-2.md) — recursive
  subdirectory support; reindex is the natural operational companion to moving
  artifacts around.
