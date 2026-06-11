---
title: Artefact Sub Directories to Assist with Artefact Management
type: idea
status: draft
lineage: idea-archiving
priority: high
labels:
    - feature
    - artifacts
    - backend
    - watcher
    - enhancement
---

## Raw Idea

## Raw Idea
The artefact directories should support subdirectories, kaos-control flattens the structure and treats them all as if they were in the main directory, but the subdirectories mean that file management becomes easier, for example there could be a done or archive subdirectory where completed items are moved to, keeping the active directory with less content.

Subdirectories could be releases or anything else people want to use, but kaos-control handles them all as first-class ideas.

## Idea

Currently kaos-control scans artifact directories (e.g. `lifecycle/ideas/`, `lifecycle/releases/`) as flat directories, ignoring any subdirectory structure. This means users cannot organise artifacts into subdirectories without those files being missed or mishandled by the indexer, watcher, and API.

The proposal is to extend the artifact discovery, indexing, and fsnotify watcher to recurse into subdirectories within each artifact root. All markdown files found in any subdirectory should be indexed and treated as first-class artifacts — their type, lineage, and status derived from frontmatter as usual, not from their directory depth. This unlocks natural housekeeping patterns such as moving completed items into a `done/` or `archive/` subdirectory to reduce noise in the active listing without losing history.

Subdirectory names should carry no special semantic meaning to kaos-control; users are free to create whatever folder structure suits their workflow (e.g. `releases/2026/`, `ideas/parked/`). The relative path within the artifact root should be stored and surfaced in the index so the UI can optionally display or filter by folder, but the core graph and editor features should work identically regardless of nesting depth.
