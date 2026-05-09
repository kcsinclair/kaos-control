---
title: Unscheduled Release Should be Connected in 2D and 3D Graph
type: defect
status: in-development
lineage: unscheduled-release-connected-in-graph
created: "2026-05-10T08:20:44+10:00"
priority: high
labels:
    - defect
    - frontend
    - roadmaps
    - releases
release: KC-Release0
assignees:
    - role: frontend-developer
      who: agent
---

# Unscheduled Release Should be Connected in 2D and 3D Graph

## Reproduction Steps

1. Open the application and navigate to the 3D graph view.
2. Select Show Releases
3. Observe the Unscheduled Release is not connected to the last last scheduled release.

## Expected Behaviour

The graph should show the Unscheduled Release being connected to the last scheduled release in this case KC-Release2.

## Actual Behaviour

The roadmap graph displays a release node labelled **"Unscheduled"**. Unscheduled Release is not connected to the last last scheduled release.
