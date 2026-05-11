---
title: Activity Feed Graph with Stacked Bars
type: idea
status: draft
lineage: activity-feed-graph
created: "2026-05-12T09:55:44+10:00"
priority: normal
labels:
    - feature
    - frontend
    - vue
release: KC-Release2
---

# Activity Feed Graph with Stacked Bars

Add a graph view to the activity feed that displays artifact and agent activity as stacked bar charts, with each bar representing a discrete time period. The chart should allow the user to switch between day, week, and month granularity to zoom in or out on project cadence.

Each bar should be broken down by activity type (e.g. artifact created, artifact status changed, agent run completed) using distinct colour segments, giving a quick visual summary of where effort is being spent over time.

The graph should sit alongside or above the existing activity feed list, sharing the same data source, and update in real time via the existing WebSocket event stream.
