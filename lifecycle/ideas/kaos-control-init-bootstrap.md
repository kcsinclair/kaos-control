---
title: 'kaos-control Init Bootstrap: Config, Agents, and DevOps Screen'
type: idea
status: clarifying
lineage: kaos-control-init-bootstrap
created: "2026-05-10T16:12:53+10:00"
priority: normal
labels:
    - feature
    - onboarding
    - backend
    - frontend
    - v1
release: KC-Release0
---

# kaos-control Init Bootstrap: Config, Agents, and DevOps Screen

The `kaos-control init` command should bootstrap a working installation by creating `~/kaos-control/config.yaml` with sensible defaults pre-populated, including configuration blocks for the idea-capture agent, kanban view, and dashboard. This removes the need for manual config authoring on first run and ensures new users land in a functional state immediately.

The init command should also create the `devops/` directory as part of the standard project scaffold. A DevOps screen should be added to the UI that surfaces pipeline management; the screen must include a "Create Pipeline" workflow that reuses the existing YAML editor component so users can author pipeline definitions in a consistent, familiar interface.

Together these changes make the initial setup experience self-contained and production-ready out of the box, reducing onboarding friction and ensuring the DevOps surface area is accessible from day one.
