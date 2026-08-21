---
title: Multi-project
type: feature
status: approved
lineage: feature-multi-project
created: "2026-08-21T15:13:00+10:00"
summary: A project registry with per-project config and an isolated SQLite cache lets one kaos-control instance serve several projects.
function: Projects & onboarding
labels:
    - feature
    - projects
related_to:
    - lifecycle/ideas/projects-crud-ui.md
    - lifecycle/requirements/projects-crud-ui-2.md
---

# Multi-project

One kaos-control instance, many independently-configured projects.

## What it does

- **Project registry.** YAML files in `~/.kaos-control/projects/*.yaml`
  register each project (name, path, owner, description). The picker on
  first load lets you choose.
- **Per-project config.** Roles, users, agents, plan-gates, kanban layout,
  dashboard tracked types, scheduler defaults, and DevOps pipelines all live
  in `<project>/lifecycle/config.yaml`.
- **Per-project SQLite cache.** Each project gets its own
  `~/.kaos-control/data/<project>/index.db`; rebuilt from disk if the
  schema version changes.

Reachable via the project picker; API under `/projects`. See also
[[project-onboarding]] for registering a new or existing project.
