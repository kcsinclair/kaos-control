---
title: Dashboard
type: feature
status: approved
lineage: feature-dashboard
created: "2026-08-21T15:07:00+10:00"
summary: Project home screen with summary cards, status distribution, a velocity chart, and a live activity feed widget.
function: Dashboard
labels:
    - feature
    - dashboard
related_to:
    - lifecycle/ideas/dashboard-home-screen.md
    - lifecycle/requirements/dashboard-home-screen-2.md
---

# Dashboard

The project's home screen — where's the work at, right now.

## What it does

- **Summary cards.** Total work-items, in-progress, blocked, completed this
  week. Configurable `tracked_types` per project (defaults to `[ticket]`;
  set in `lifecycle/config.yaml`).
- **Status distribution pie.** Click a wedge to filter the artifacts list by
  that status.
- **Velocity bar chart.** Daily / weekly / monthly granularity, 90-day
  default lookback. Echarts. DataZoom appears for crowded ranges.
- **Activity feed widget.** Recent agent runs, transitions, defects raised,
  artifacts created — newest first, click to navigate.

Reachable at **Dashboard** (default landing view for a project).
