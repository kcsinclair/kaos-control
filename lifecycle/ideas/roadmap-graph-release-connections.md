---
title: 'Roadmap Graph: Directed Release Connections with Backlog Root and Unscheduled Leaves'
type: idea
status: approved
lineage: roadmap-graph-release-connections
created: "2026-05-07T12:19:49+10:00"
priority: normal
labels:
    - frontend
    - roadmaps
    - feature
    - vue
---

# Roadmap Graph: Directed Release Connections with Backlog Root and Unscheduled Leaves

The Roadmap graph should display all releases as connected nodes in a directed graph. Releases that have a start date should be ordered chronologically, with a directed edge from each earlier release to the next one in the timeline, forming a clear temporal chain.

The first node in the graph should be a special synthetic node called "Backlog" (currently labelled "Undefined"), which represents unplanned or unassigned work. This Backlog node should have a directed connection to the earliest scheduled release in the timeline, anchoring the graph.

Any releases without a scheduled date are treated as "Unscheduled" and appear as the final nodes in the graph. If multiple Unscheduled releases exist, they should be connected to each other in alphabetical order by release name. The last scheduled release in the timeline should have a directed connection to the first of the Unscheduled releases, completing the chain.
