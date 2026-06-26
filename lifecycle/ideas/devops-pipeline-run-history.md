---
title: DevOps Pipeline Run History
type: idea
status: clarifying
lineage: devops-pipeline-run-history
created: "2026-05-10T16:00:54+10:00"
priority: normal
labels:
    - feature
    - frontend
    - backend
    - operability
release: KC-Release4
---

# DevOps Pipeline Run History

Each pipeline entry in the DevOps screen should display a recent run history panel beneath it, showing the last N executions with their timestamp, duration, and pass/fail status at a glance.

Users should be able to expand any run entry to access the full log output for that execution, making it easy to diagnose failures without leaving the DevOps screen.

The history should be persisted on the backend and surfaced via a REST or WebSocket endpoint, with the frontend polling or subscribing for live updates when a pipeline is actively running.
