---
title: Agent and Task Scheduler
type: idea
status: planning
lineage: agent-task-scheduler
created: "2026-04-28T08:08:14+10:00"
priority: normal
labels:
    - feature
    - agent
    - backend
    - workflow
---

# Agent and Task Scheduler

Add a scheduler subsystem that allows agent runs and arbitrary shell scripts to be queued and executed on a specific times once off or recurring hourly, daily, etc or triggered when a condition is met (e.g. rate limits have lifted, a prior job completed). This enables nightly builds, periodic QA runs, and deferred agent work that would otherwise be blocked by API quota constraints.

The scheduler should be configurable via the project config or a dedicated schedule file, supporting named jobs with expressions (cron syntax or interval-based), a target (agent role or shell script path), and optional preconditions. Job state, last-run time, and output logs should be persisted so they survive restarts and are visible in the UI.

A lightweight admin surface in the UI would let users create, update or delete jobs, view upcoming jobs, inspect recent run history, manually trigger a job, or pause/resume the schedule — keeping the feature consistent with the rest of the lifecycle management tooling.
