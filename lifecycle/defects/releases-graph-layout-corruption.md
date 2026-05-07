---
title: Show Releases Corrupts 3D/2D Graph Layout and Renders Phantom Arrow Cones
type: defect
status: in-development
lineage: releases-graph-layout-corruption
created: "2026-05-07T18:54:49+10:00"
priority: normal
labels:
    - defect
    - frontend
    - releases
    - roadmaps
---

# Show Releases Corrupts 3D/2D Graph Layout and Renders Phantom Arrow Cones

## Reproduction Steps

1. Open the application and navigate to the graph view.
2. Enable the "Show Releases" toggle.
3. Observe the 3D graph layout.
4. Switch to the 2D graph view and observe.

## Expected Behaviour

Enabling "Show Releases" should add release nodes to the graph while preserving the existing auto-layout behaviour. Both 3D and 2D graph views should render correctly with no visual artefacts.

## Actual Behaviour

When "Show Releases" is enabled:
- The 3D graph layout breaks — auto-layout appears to be disabled, causing nodes to render in incorrect or overlapping positions.
- Unexpected arrow cone artefacts appear in the 3D graph view.
- The 2D graph is also affected, suggesting the underlying graph data is being corrupted when release nodes/edges are injected into the dataset.
