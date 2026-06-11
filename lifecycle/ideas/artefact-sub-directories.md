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
---

## Raw Idea

## Raw Idea
The artefact directories should support subdirectories, kaos-control flattens the structure and treats them all as if they were in the main directory, but the subdirectories mean that file management becomes easier, for example there could be a done or archive subdirectory where completed items are moved to, keeping the active directory with less content.

Subdirectories could be releases or anything else people want to use, but kaos-control handles them all as first-class ideas.

## Idea

Currently kaos-control flattens all artifact directories, treating every markdown file as if it lives directly in the top-level stage directory regardless of its actual path. This means users cannot organise files into subdirectories without breaking indexing or lineage resolution.

Support for subdirectories would allow teams to create conventions like `done/` or `archive/` within any artifact directory, moving completed or superseded items out of the active working set while keeping them indexed and fully functional as first-class artifacts. The subdirectory name carries no semantic meaning to kaos-control — all files beneath a stage directory are indexed and resolved identically regardless of nesting depth.

This is a quality-of-life improvement for larger projects where active artifact directories grow unwieldy over time. Users gain the freedom to impose any folder structure that suits their workflow without losing lineage tracking, graph edges, or agent visibility into those artifacts.
