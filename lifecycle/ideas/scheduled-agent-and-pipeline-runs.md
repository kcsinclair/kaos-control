---
title: Scheduled Agent and Pipeline Runs
type: idea
status: draft
lineage: scheduled-agent-and-pipeline-runs
created: "2026-05-09T10:57:38+10:00"
priority: normal
labels:
    - feature
    - agent-runner
    - workflow
    - operability
    - backend
---

# Scheduled Agent and Pipeline Runs

The current scheduler lacks the ability to trigger agent work or DevOps pipelines on a time-based schedule. Users need to be able to define recurring or one-off scheduled runs, such as starting a markdown-processing agent run at 6:00pm or running a test pipeline daily at a fixed time.

This feature would introduce a scheduling layer that supports both agent runs (e.g. 'run requirements-analyst agent at 18:00') and pipeline triggers (e.g. 'run test pipeline every day at midnight'). Schedules should be configurable via the project config or UI, with cron-style or natural-language time expressions.

The scheduler should persist schedule definitions, track last-run state, and surface schedule status in the UI so users can monitor upcoming and past runs without needing external tooling.
