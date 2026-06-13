---
title: Graph Maps Display Release Artifacts That Are Already Shown via Release Database
type: defect
status: done
lineage: graph-maps-show-release-artifacts
created: "2026-06-13T11:42:26+10:00"
priority: normal
labels:
    - defect
    - map
    - 3d-graph
    - cytoscape
    - releases
    - filter
    - frontend
assignees:
    - role: frontend-developer
      who: agent
parent: lifecycle/tests/release-artefacts-6-test.md
---

# Graph Maps Display Release Artifacts That Are Already Shown via Release Database

## Reproduction Steps

1. Open the application and navigate to the 2D or 3D graph map view.
2. Observe the nodes rendered in the map.
3. Note that release artifact documents (e.g. files under `lifecycle/releases/`) appear as nodes in the graph.

## Expected Behaviour

Release artifact markdown documents should be excluded from the 2D and 3D graph maps. Releases are already represented via entries sourced from the release database, so displaying the underlying artifact documents in the map creates duplicate representation.

## Actual Behaviour

Release artifact documents are included as nodes in both the 3D force-graph and the 2D Cytoscape graph maps, resulting in duplicate or redundant entries alongside the release database-sourced nodes.
