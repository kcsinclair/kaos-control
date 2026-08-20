---
title: 3D Architecture Map Missing Artifact Name Labels
type: defect
status: done
lineage: 3d-map-missing-artifact-labels
created: "2026-08-19T09:22:35+10:00"
priority: normal
labels:
    - defect
    - 3d-graph
    - architecture
    - map
    - frontend
    - visualization
release: KC-Release5
---

# 3D Architecture Map Missing Artifact Name Labels

## Reproduction Steps

1. Navigate to the Architecture Map view in the application.
2. Switch to or open the 3D graph/map view.
3. Observe the nodes representing architecture artifacts.

## Expected Behaviour

Each node on the 3D architecture map should display a visible label showing the name of the artifact it represents, making it easy to identify artifacts at a glance without needing to hover or click.

## Actual Behaviour

Nodes on the 3D architecture map are rendered without labels — the artifact names are not shown, making it impossible to identify artifacts directly on the graph.
