---
title: DevOps Pipeline Management
type: idea
status: approved
lineage: devops-pipelines
created: "2026-05-05T19:42:31+10:00"
priority: normal
labels:
    - feature
    - backend
    - frontend
    - workflow
    - v1
---

# DevOps Pipeline Management

Introduce a `devops/` directory under `lifecycle/` to hold YAML pipeline definitions. Each YAML file declares a pipeline type (`build`, `deploy`, or `release`), a human-readable name, and an ordered list of steps — each step carrying a name, description, and shell command to execute. The pipeline type vocabulary is open for extension so future pipeline categories (e.g. `migrate`, `rollback`) can be added without schema changes.

Add a new **DevOps** section to the left navigation menu. The DevOps page discovers all YAML files in `lifecycle/devops/` and renders them grouped by type into three kanban-style columns — Build, Deploy, Release. Each pipeline appears as a card with a launch button; clicking it executes the steps in order, streaming output back to the UI in real time via WebSocket so the Product Owner can monitor progress.

The backend exposes new API endpoints to list discovered pipeline definitions and to trigger a pipeline run, using the existing agent-runner and hub infrastructure for execution and broadcast. Access is gated to the Product Owner role. Step state (pending, running, passed, failed) is tracked per-run so the UI can highlight the active step and surface errors inline.
