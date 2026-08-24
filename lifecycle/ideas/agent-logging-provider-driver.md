---
title: 'Agent Logging: Include Provider and Driver'
type: idea
status: draft
lineage: agent-logging-provider-driver
created: "2026-08-25T08:21:08+10:00"
priority: normal
labels:
    - agent
    - agent-runner
    - driver
    - provider
    - observability
    - runs
    - persistence
---

# Agent Logging: Include Provider and Driver

Each agent run log entry — both the on-disk job log file and the database record — should capture the provider and driver that were active at the time the run was executed. This metadata should be written at the start of the run, before any agent output is recorded.

This is important because agent configuration can change over time (e.g. switching from one LLM provider to another, or swapping drivers). Without this record, it becomes impossible to know which provider or driver produced a given piece of work, making attribution, cost analysis, and debugging unreliable.

The provider and driver values should be sourced from the resolved agent configuration at run-start and treated as immutable for that run record — they must not be updated if the agent config changes after the run completes.
