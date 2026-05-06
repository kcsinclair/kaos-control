---
title: Project Feed
type: idea
status: planning
lineage: project-feed
priority: high
labels:
    - feature
    - frontend
    - workflow
    - artefacts
---

# Project Feed

Add a Project Feed view that aggregates all significant lifecycle events into a chronological activity stream: artifact status transitions, agent run completions, defects raised, and plan approvals. Each entry should display the event type, timestamp, artifact name, and the role or agent responsible.

Each feed entry should be a clickable link that navigates directly to the relevant artifact or agent run, so the user can immediately act — approve a plan, review a defect, or transition a ticket — without hunting through the graph or directory tree.

The feed should be powered by the existing WebSocket broadcast events (`artifact.indexed`, `file.changed`) so it updates in real time without polling, and should persist recent history in the SQLite index so it survives page refreshes.
