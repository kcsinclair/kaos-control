---
title: DevOps pipelines
type: feature
status: approved
lineage: feature-devops-pipelines
created: "2026-08-21T15:09:00+10:00"
summary: Declarative YAML pipelines with live step output streaming, run-log persistence, and role-gated triggering.
function: DevOps
labels:
    - feature
    - devops
related_to:
    - lifecycle/ideas/devops-pipelines.md
    - lifecycle/requirements/devops-pipelines-2.md
---

# DevOps pipelines

Build/deploy/release steps defined as artifacts, run and watched from the
browser.

## What it does

- **Declarative YAML.** One file per pipeline in
  `<project>/lifecycle/devops/`, with `name`, `type`
  (`build` / `deploy` / `release` / arbitrary), ordered `steps`. Each step
  has `name`, `command`, optional `description` and `timeout`.
- **Trigger from the UI.** Cards grouped by type; Run / Cancel / re-run with
  one click.
- **Live output streaming.** Per-step stdout / stderr to the browser via
  WebSocket: `pipeline.run.started → step.started → step.output* →
  step.completed → run.completed`.
- **Run log persistence.** NDJSON logs at
  `~/.kaos-control/devops/<project>/<run_id>.log`. Browseable post-mortem.
- **Role-gated.** Only `product-owner` / `devops` can trigger.
- **Cancellation + timeout.** SIGTERM running step; per-step timeouts
  honoured; failed step skips the rest.

Reachable at **DevOps**; API under `/devops/pipelines`.
