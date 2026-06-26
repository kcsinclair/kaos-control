---
title: kaos-control CLI for DevOps
type: idea
status: clarifying
lineage: kaos-control-devops-cli
priority: high
labels:
    - devops
    - backend
    - security
    - feature
    - tooling
    - go
release: KC-Release4
---

## Raw Idea

## Raw Idea
Add cli functions to support devops, e.g kaos-control devops list, devops run test-all, etc

Needs to account for userid and permissions?  Associates Linux userid to existing user for roles, and obviously uses Linux permissions at filesystem level.

## Idea

Add a `devops` subcommand group to the `kaos-control` CLI to support operational tasks from the terminal without requiring the web UI, e.g. `kaos-control devops list`, `kaos-control devops run test-all`, `kaos-control devops status`. This gives operators and CI pipelines a scriptable interface to the lifecycle tool.

The CLI should associate the invoking Linux user identity (via `os/user`) with an existing kaos-control user account, enabling role-based access control without requiring a separate login. Permissions at the filesystem level are respected naturally via the OS, but the mapping to kaos-control roles (analyst, developer, qa, etc.) must be resolved so that agent runs and artifact writes are correctly attributed and gated. A user without the appropriate role cannot trigger runs or perform destructive operations, even if they have filesystem read access.

Authz rules should mirror those enforced by the HTTP API — the same workflow gates and allowed-write-path restrictions apply regardless of whether the action is triggered via CLI or web. This ensures the CLI is a first-class interface rather than a privileged back door. The design should also consider non-interactive (CI/scripted) use cases, potentially supporting an API token or service-account mode for automation.
