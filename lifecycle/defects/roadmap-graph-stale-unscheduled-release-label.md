---
title: Roadmap Graph Shows Stale 'Unscheduled' Release Instead of KC-AgentHandling
type: defect
status: approved
lineage: roadmap-graph-stale-unscheduled-release-label
created: "2026-05-09T17:40:44+10:00"
priority: normal
labels:
    - defect
    - frontend
    - roadmaps
    - releases
---

# Roadmap Graph Shows Stale 'Unscheduled' Release Instead of KC-AgentHandling

## Reproduction Steps

1. Open the application and navigate to the Roadmap graph view.
2. Observe the release nodes rendered in the graph.

## Expected Behaviour

The graph should display the current unscheduled release by its actual name, **KC-AgentHandling**, and should not show any release named "Unscheduled" (which no longer exists).

## Actual Behaviour

The roadmap graph displays a release node labelled **"Unscheduled"**. This release no longer exists; the unscheduled release is now called **KC-AgentHandling**. The stale label suggests the graph is either reading from a cached/outdated data source or is not correctly resolving release names from the current index.
