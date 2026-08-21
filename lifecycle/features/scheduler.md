---
title: Scheduler
type: feature
status: approved
lineage: feature-scheduler
created: "2026-08-21T15:10:00+10:00"
summary: Per-project cron-style job definitions that trigger agent runs or shell commands, with CRUD, run history, and live status.
function: Scheduler
labels:
    - feature
    - scheduler
related_to:
    - lifecycle/ideas/agent-task-scheduler.md
    - lifecycle/requirements/agent-task-scheduler-2.md
    - lifecycle/ideas/scheduled-agent-and-pipeline-runs.md
---

# Scheduler

Run agents and pipelines on a schedule instead of only on demand.

## What it does

- **Cron-style job definitions.** Per-project SQLite-backed jobs with cron
  expressions, timeouts, target type (shell / agent), and a precondition
  expression.
- **Job CRUD UI.** List / create / edit / delete / pause / resume / trigger
  now. Run history per job. Live status via WebSocket.

Reachable at **Scheduler**; API under `/scheduler/jobs`.
