---
created: "2026-08-11T15:29:18+10:00"
title: GUI-triggered archiving that moves completed artifacts into subfolders and rewrites references
type: idea
status: draft
lineage: artifact-archiving-action
priority: medium
labels:
    - lifecycle
    - artifacts
    - archiving
    - index
    - ui
assignees:
    - role: product-owner
      who: agent
---

# GUI-triggered archiving that moves completed artifacts into subfolders and rewrites references

## Context

The [idea-archiving](../requirements/idea-archiving-2.md) lineage shipped the
**mechanism** for organising artifacts in folders: recursive discovery, watcher
recursion, and relative-path indexing, so a markdown artifact is first-class at
any nesting depth. That requirement deliberately listed two things as
**non-goals / follow-ups**:

> *"No automatic moving of artifacts into subdirectories on status change … Movement remains user- or agent-driven."*
> *"No UI redesign of the artifact list … folder-based grouping/filtering UI may be specified in a follow-up."*

This idea is that follow-up: the **archiving operation** itself — a way to move
completed artifacts into per-stage archive subfolders (e.g. `ideas/done/`,
`requirements/done/`) in bulk, **triggered from the GUI and processed by the
kaos-control engine**, so the active listings stay small without hand-moving
files or breaking the lineage graph.

## The problem to solve

The stage folders have accumulated a large backlog of completed artifacts —
measured 2026-08-11: roughly **680 `status: done` artifacts** (ideas 95,
requirements 87, backend/frontend/test-plans ~92 each, defects 180, tests 38,
docs 4). Manually relocating them is not viable, because **moving an artifact
silently breaks the lineage graph.**

Parent and link edges are resolved **by path string**, not by lineage+index:

- [internal/statuscheck/statuscheck.go L88](../../internal/statuscheck/statuscheck.go#L88)
  builds `children[a.FM.Parent] = append(…, a.Path)` — a child is matched to its
  parent by the parent's **path**.
- [internal/artifact/artifact.go L357-360](../../internal/artifact/artifact.go#L357)
  emits the parent link via `normaliseLinkTarget(fm.Parent, fromPath)` — again a
  path.

So when a target artifact moves, every inbound `parent:` reference (and every
markdown link to it) still points at the **old** path, and the edge is lost.
Measured scope of what a bulk archive must rewrite: **658 `parent:` lines**
across 667 files (in two formats, `lifecycle/<stage>/x.md` and `<stage>/x.md`),
**19 markdown cross-links**, plus `related_to:` blocks.

This was confirmed empirically: a single manual test move of
`ideas/release-artefacts.md` → `ideas/ideas-done/release-artefacts.md` left
**three dangling references** — the `parent:` in
[release-artefacts-2.md](../requirements/release-artefacts-2.md) and two markdown
links in [release-artefacts-9.md](../requirements/release-artefacts-9.md) — all
still pointing at the pre-move path.

**The engine must move the files AND rewrite every inbound reference
atomically** — the same class of operation that `release.PropagateRename`
([internal/release/propagate.go](../../internal/release/propagate.go)) already
performs for the `release:` frontmatter field when a release is renamed.

## Sketch

1. **GUI trigger.** A control on the artifact list / a stage view — "Archive
   completed" (bulk) and/or a per-artifact "Archive" action — POSTs an archive
   request to the engine. Processed server-side like a devops run / agent run,
   not client-side file shuffling.
2. **Engine operation.** For each artifact to archive:
   - `git mv` it into the stage's archive subfolder (convention TBD — see
     open questions).
   - Rewrite every inbound reference to its new path: `parent:` frontmatter
     (both path formats), markdown links, `related_to:` entries.
   - Re-index the moved file and every rewritten file; broadcast the usual
     `artifact.indexed` / `file.changed` WS events.
   - Commit as one atomic git commit (like `PropagateRename`), so the move and
     the reference rewrite land together and are revertable as a unit.
3. **Selection.** Which artifacts to archive is a filter (status, stage, age),
   surfaced in the GUI so the user reviews the set before committing.
4. **Verification.** After the operation, no reference may point at a
   now-nonexistent path; a post-op check (reuse the `statuscheck` parent map)
   confirms the graph has zero newly-dangling edges.

## Open questions (to think about)

- **Subfolder convention.** `<stage>/done/` (semantically-neutral, matches the
  idea-archiving example) vs `<stage>-done/` (the sibling-folder shape the
  initial test used) vs a single top-level `archive/` tree. idea-archiving
  requirement §Non-goals says folder names carry **no** backend meaning, so this
  is purely a UI/convention choice — but it should be consistent and
  configurable.
- **Which statuses archive.** `done` only, or terminal states too
  (`abandoned`, `rejected`), or `approved` (e.g. the 78 approved tests)? Should
  this be a per-request filter rather than a fixed rule?
- **Split lineages.** A lineage is often partly done and partly active. Do we
  archive only the done members (lineage then spans active + archived folders —
  fine for the graph once references are rewritten), or hold a lineage back
  until *all* members are terminal?
- **Un-archive / move-back.** The same reference-rewrite must run in reverse
  when an artifact is pulled back out of the archive (e.g. a done idea reopened).
  Is un-archive in scope, or a later addition?
- **Agent vs. user trigger.** idea-archiving kept movement "user- or
  agent-driven." Should an agent be able to invoke archive as a workflow step,
  or is this GUI/human-only for now?
- **Reference resolution — fix the root cause too?** An alternative (or
  complement) to rewriting paths on every move is resolving `parent:`/links by
  **lineage+index** rather than path, so moves stop breaking edges at all
  (idea-archiving §FR7 already keys artifact *identity* by lineage+index). Worth
  deciding whether archiving rewrites references, or whether edge resolution
  should become path-independent — or both.

## Related

- [idea-archiving-2.md](../requirements/idea-archiving-2.md) — the shipped
  recursive-subdirectory support this builds on; archiving was its named
  follow-up.
- [internal/release/propagate.go](../../internal/release/propagate.go) — the
  existing move-and-rewrite-references pattern (`PropagateRename`) to model the
  engine operation on.
