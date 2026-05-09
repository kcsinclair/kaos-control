---
title: Roadmap Graph Shows Stale 'Unscheduled' Release Instead of KC-AgentHandling
type: defect
status: done
lineage: roadmap-graph-stale-unscheduled-release-label
created: "2026-05-09T17:40:44+10:00"
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

# Roadmap Graph Shows Stale 'Unscheduled' Release Instead of KC-AgentHandling

## Reproduction Steps

1. Open the application and navigate to the Roadmap graph view.
2. Observe the release nodes rendered in the graph.

## Expected Behaviour

The graph should display the current unscheduled release by its actual name, **KC-AgentHandling**, and should not show any release named "Unscheduled" (which no longer exists).

## Actual Behaviour

The roadmap graph displays a release node labelled **"Unscheduled"**. This release no longer exists; the unscheduled release is now called **KC-AgentHandling**. The stale label suggests the graph is either reading from a cached/outdated data source or is not correctly resolving release names from the current index.

## Resolved Questions

1. **Root cause: synthetic node vs. stale data?**
   The backend (`internal/http/releases.go` lines 487–510) deliberately creates a synthetic terminus node titled `"Unscheduled"` (id `release:unscheduled`) whenever any release has no `start_date`. The real release node for "KC-AgentHandling" is generated separately alongside it. Is the defect:
   - (a) The synthetic `"Unscheduled"` terminus node is misleading because it shares a name with a historical release that was renamed — the label "Unscheduled" is intentional backend behaviour but confusing to users?
   - (b) The real "KC-AgentHandling" release node is absent from the graph entirely (e.g., a filter bug or missing artifact `release` field), meaning only the synthetic terminus is visible?
   - (c) Something else?

> If the unscheduled node was connected to the last node in the timeline, this would work well.

2. **Frontend scope: what change is expected?**
   This defect is labelled `frontend`. The frontend (`RoadmapGraphView.vue`) passes all nodes from the `/releases/graph` API verbatim to the graph components with no filtering or renaming. If the intended fix is purely frontend, please specify exactly what the frontend should do differently — for example:
   - Filter out the synthetic `"Unscheduled"` node (`node.id === "release:unscheduled"`)
   - Relabel it (e.g., "No Release Date" or "Undated Releases")
   - Visually distinguish it further from real release nodes
   - Something else

> Connect unscheduled node to last node in the timeline.

3. **No implementation milestones provided.**
   The task requests milestone-by-milestone implementation, but this artifact contains no milestones. Please add milestones describing the intended frontend changes before implementation can proceed.

> Does this need milestones?  If so, request an analyst to review first.
