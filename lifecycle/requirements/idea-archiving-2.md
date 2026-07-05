---
title: Recursive Subdirectory Support for Artifact Directories
type: requirement
status: in-development
lineage: idea-archiving
created: "2026-07-05T15:30:00+10:00"
priority: high
parent: lifecycle/ideas/artefact-sub-directories.md
labels:
    - backend
    - watcher
    - index
    - artifacts
    - enhancement
assignees:
    - role: product-owner
      who: agent
---

# Recursive Subdirectory Support for Artifact Directories

## Problem

kaos-control discovers artifacts by scanning each artifact root (e.g. `lifecycle/ideas/`, `lifecycle/releases/`) as a **flat** directory. Markdown files placed in a subdirectory of an artifact root are not discovered by the startup scan, are not tracked by the fsnotify watcher, and are not returned by the artifact API. As a result users cannot organise artifacts into folders — a common and natural housekeeping need.

The most-requested use of folders is *archiving*: moving completed or parked artifacts into a `done/` or `archive/` subfolder so the active listing stays small, without deleting the file or losing its git history. Today that operation silently removes the artifact from the tool. More generally, users want the freedom to group artifacts by release, year, workstream, or any convention that suits them.

The tool should treat every markdown artifact under an artifact root as a first-class artifact regardless of how deeply it is nested, deriving its identity (type, lineage, status, parent) from frontmatter — never from directory depth or folder name.

## Goals / Non-goals

### Goals

- Discover, index, and watch markdown artifacts in **arbitrarily nested subdirectories** of every artifact root.
- Treat a nested artifact identically to a top-level one: same graph, editor, transitions, agent runs, and API behaviour.
- Preserve and surface the artifact's **relative path within its root** so the UI can optionally display or filter by folder.
- Keep moving a file between folders (including into/out of a subdirectory) a pure filesystem operation that the watcher reflects incrementally, with no change to the artifact's identity.
- Keep subdirectory names **semantically neutral** — no folder name (including `done`, `archive`) is given special meaning by the backend.

### Non-goals

- No UI redesign of the artifact list or graph in this requirement beyond exposing the folder path; folder-based grouping/filtering UI may be specified in a follow-up.
- No automatic moving of artifacts into subdirectories on status change (e.g. auto-archiving on `done`). Movement remains user- or agent-driven.
- No change to the lineage filename convention (§3.3/§4.4) — indices remain unique per lineage independent of folder location.
- No support for non-markdown files or symlink traversal.
- No per-folder access control or permissions.

## Detailed Requirements

### Functional

1. **Recursive discovery.** The startup scan MUST walk each artifact root recursively and index every `*.md` file found at any depth, not only files directly in the root.
2. **Watcher recursion.** The fsnotify watcher MUST observe subdirectories so that create, modify, delete, and move events for nested artifacts trigger incremental re-index and the existing `artifact.indexed` / `file.changed` WS broadcasts, subject to the existing 150 ms debounce.
3. **New-subdirectory detection.** When a new subdirectory is created under a watched root at runtime, its markdown contents MUST be indexed and the directory itself MUST become watched, without a restart.
4. **Relative path in index.** For every artifact the index MUST store the path relative to its artifact root (e.g. `done/login.md`, `2026/q3/release-x.md`). Files directly in the root store a relative path with no directory component. This value MUST be exposed on the artifact in API responses.
5. **Identity from frontmatter only.** Type, status, lineage, and parent MUST be derived solely from frontmatter. Directory depth and folder names MUST NOT influence any derived field.
6. **Root assignment by top-level directory.** An artifact's owning root/collection is the top-level `lifecycle/<root>/` directory it lives under; nesting below that does not reassign it to a different collection.
7. **Move preserves identity.** Moving an artifact file within its root (into, out of, or between subdirectories) MUST result in the same logical artifact with an updated relative path, not a delete-plus-create of a new artifact identity, where the artifact is keyed by a stable identifier (e.g. lineage + index) rather than by path.
8. **API path-safety on writes.** `POST /artifacts` and `PUT /artifacts/*` MUST accept and correctly resolve nested relative paths while continuing to reject path traversal outside the artifact root (via the existing sandbox resolver).
9. **Uniqueness across folders.** Lineage index uniqueness MUST be enforced across the whole root, not per folder; two files in different subfolders sharing a lineage index is an existing collision and MUST be surfaced the same way a flat collision is today.
10. **Hidden/ignored paths.** Dotfiles and dot-directories (e.g. `.git`, `.trash`) under an artifact root MUST be skipped by both scan and watcher.

### Non-functional

11. **Performance.** Recursive scan of a root with nested folders MUST complete within the same order of magnitude as the current flat scan for an equivalent file count; watching nested directories MUST NOT exhaust OS watch limits for typical repositories (hundreds of artifacts, tens of folders).
12. **Backward compatibility.** Repositories with no subdirectories MUST behave exactly as before; the relative-path field for flat files MUST be the bare filename.
13. **Safety.** Path resolution MUST remain traversal-safe; no scan or watch may escape an artifact root via `..` or symlinks.
14. **Cross-platform.** Relative paths MUST be stored and compared using forward slashes regardless of host OS.

## Acceptance Criteria

- [ ] Startup scan indexes a markdown artifact placed in `lifecycle/ideas/done/` and it appears in `GET /artifacts` results.
- [ ] The same artifact appears in the graph and is editable via the editor identically to a top-level artifact.
- [ ] Creating a new `*.md` file in a nested subdirectory at runtime triggers indexing and an `artifact.indexed` WS broadcast within the debounce window, without restart.
- [ ] Creating a brand-new subdirectory at runtime and adding an artifact to it results in that artifact being indexed and the directory being watched.
- [ ] Each artifact's API representation includes its root-relative path, using forward slashes, with the bare filename for flat files.
- [ ] Moving an artifact from the root into `archive/` (and back) updates its relative path while preserving its identity, graph edges, and history; no duplicate artifact is created.
- [ ] Type, status, lineage, and parent are unchanged by moving an artifact between folders or by folder name.
- [ ] `.git` and other dot-directories under an artifact root are not indexed or watched.
- [ ] `PUT`/`POST` to a nested relative path succeeds; a path attempting `..` traversal outside the root is rejected.
- [ ] A repository with no subdirectories produces byte-identical index results to the pre-change behaviour (relative path = filename).
- [ ] Lineage index collisions across two different subfolders are detected and surfaced the same way as flat collisions.
- [ ] Relevant integration coverage is added — see [[end-to-end-smoke-tests]] and [[editor-live-refresh-on-disk-change]] for watcher/index refresh patterns to extend.

## Resolved Questions

1. **Move detection fidelity.** fsnotify does not reliably emit atomic rename/move events across all platforms; a move may surface as delete-then-create. Is keying artifacts by lineage+index (rather than path) sufficient to preserve identity across such split events, or is an explicit move endpoint needed?

> I think type+lineage+index is unique.

2. **Relative-path surfacing in UI.** Should the folder path be shown as a column, a breadcrumb, a filter facet, or hidden by default? (Deferred to a follow-up UI requirement, but the field must be exposed now.)

> a breadcrumb

3. **Watch-limit strategy.** For very large repositories, should nested watching be capped or fall back to periodic rescans if OS inotify/kqueue limits are hit, and what limit is acceptable before degrading?

> 5000

4. **Excluded folder conventions.** Beyond dot-directories, should any well-known folders (e.g. `node_modules`, `dist`) be ignored by default, or is dot-prefix the only exclusion rule?

> only dot-prefix

5. **Index migration.** Does adding the relative-path column require an index schema version bump and rebuild on upgrade, or can it be back-filled lazily?

> back-filled lazily
