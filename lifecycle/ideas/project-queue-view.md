---
title: Project Queue View in Agents Panel
type: idea
status: approved
lineage: project-queue-view
created: "2026-05-16T13:04:44+10:00"
priority: high
labels:
    - queue
    - frontend
    - feature
    - agent
    - vue
release: KC-Release4
---

# Project Queue View in Agents Panel

The existing global queue remains unchanged, but each project gains a dedicated queue view embedded within the Agents panel. The queue is displayed on the right side of the Agents view, showing jobs currently queued for that specific project.

This gives users immediate visibility into pending agent work without leaving the project context. The project queue view should include a link to the global queue for users who need a system-wide perspective across all projects.

The feature is purely additive — the global queue view continues to work as before, and the project-scoped queue is a filtered subset shown contextually alongside the agents that will process those jobs.
