---
title: 'Artefacts List: Release & Priority Columns with Releases Filter'
type: idea
status: clarifying
lineage: artefacts-list-release-priority-columns
created: "2026-05-07T10:04:43+10:00"
priority: normal
labels:
    - artefacts
    - frontend
    - enhancement
    - releases
release: KC-Feature-Sprint
---

# Artefacts List: Release & Priority Columns with Releases Filter

The artefacts list view currently lacks visibility into which release an artefact is associated with and what its priority is. Adding dedicated columns for both `release` and `priority` would allow users to quickly assess the state of work without needing to open individual artefacts or switch to Board view.

The Board view already supports filtering by release, but this filter is not available in the List view. The releases filter should be ported across so that users working in List view have parity with Board view when narrowing down artefacts to a specific release.

These changes improve the utility of the List view as a primary working surface, reducing the need to context-switch between views to get a complete picture of release scope and priorities.
