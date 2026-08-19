---
title: Architecture and ADR Artifacts Missing from List View and Other Surfaces
type: defect
status: approved
lineage: architecture-adr-artifacts-missing-from-list-view
created: "2026-08-19T10:53:32+10:00"
priority: normal
labels:
    - defect
    - architecture
    - artifacts
    - frontend
    - index
    - ui
---

# Architecture and ADR Artifacts Missing from List View and Other Surfaces

## Reproduction Steps

1. Open the kaos-control UI.
2. Navigate to the artifact list view (or any surface that enumerates artifacts, e.g. graph, search, lineage view).
3. Observe the listed artifact types.
4. Note that artifacts of type `architecture`, `tech-stack`, and `adr` (stored under `lifecycle/architecture/`) are absent.

## Expected Behaviour

Artifacts of type `architecture`, `tech-stack`, and `adr` should appear in the list view and any other surface (graph, search, editor navigation) that displays or links to artifacts. The canonical type vocabulary — defined in `KnownStatuses` / the type registry in `internal/artifact/artifact.go` — includes these types and they should be indexed and surfaced like any other artifact type.

## Actual Behaviour

Architecture and ADR artifacts are not shown in the list view or other artifact-browsing surfaces. They are likely either excluded by a type filter that predates the KC-Release5 architecture artifact types, or the indexer is not recognising them as known types, causing them to be omitted from query results.
