---
title: Operations
type: feature
status: approved
lineage: feature-operations
created: "2026-08-21T15:16:00+10:00"
summary: Single-binary deployment with an embedded SPA, a project-scoped WebSocket hub, a parse-error view, and heartbeat-reaped lineage locks.
function: Operations
labels:
    - feature
    - operations
related_to:
    - lifecycle/ideas/configurable-http-port.md
---

# Operations

The operational surface: how kaos-control is deployed, configured, and kept
healthy.

## What it does

- **Single binary, embedded SPA.** Go server with `embed.FS` — one file to
  deploy.
- **WebSocket hub.** Per-project broadcast channel for indexed events, agent
  progress, pipeline output, scheduler ticks, locks.
- **Parse-error view.** Every YAML / frontmatter parse failure surfaces with
  file path + line + message; reload re-attempts.
- **App config.** Server listen address, port, TLS, auth method, projects
  directory, data directory, devops log directory all in
  `~/.kaos-control/config.yaml`.
- **Lock management.** Per-lineage editor / agent locks with heartbeat-based
  stale-reaper.

Reachable at **Parse Errors**; configured via `~/.kaos-control/config.yaml`.
