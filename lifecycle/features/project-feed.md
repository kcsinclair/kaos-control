---
title: Project feed
type: feature
status: approved
lineage: feature-project-feed
created: "2026-08-21T15:12:00+10:00"
summary: Live, retained event log of transitions, agent runs, commits, and defects, streamed to a dedicated Feed view.
function: Activity & feed
labels:
    - feature
    - feed
related_to:
    - lifecycle/ideas/project-feed.md
    - lifecycle/requirements/project-feed-2.md
---

# Project feed

A single stream of everything that happened in a project.

## What it does

- **Live event log.** Every status transition, agent run, git commit,
  defect raised, artifact created — streamed to the **Feed** view via
  WebSocket. Configurable retention (`feed.retention_days`,
  `feed.max_events`).

Reachable at **Feed**.
