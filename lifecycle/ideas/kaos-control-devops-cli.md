---
title: kaos-control CLI for DevOps
type: idea
status: draft
lineage: kaos-control-devops-cli
priority: high
labels:
    - devops
    - backend
    - go
    - security
    - feature
release: KC-Release3
---

## Raw Idea

## Raw Idea
Add cli functions to support devops, e.g kaos-control devops list, devops run test-all, etc

Needs to account for userid and permissions?  Associates Linux userid to existing user for roles, and obviously uses Linux permissions at filesystem level.

## Idea

Add a `devops` subcommand group to the `kaos-control` CLI, enabling operators to manage and trigger lifecycle operations from the terminal without using the web UI. Initial commands would include `devops list` (list runs, agents, or artifacts) and `devops run <target>` (trigger an agent run such as `test-all`), following standard Unix CLI conventions.

User identity and permissions must be handled explicitly: the CLI should associate the invoking Linux user (via `os/user`) with an existing kaos-control user account, inheriting their assigned roles and access rights. This means a user without the `devops` or `admin` role cannot trigger runs or perform destructive operations, even if they have filesystem read access.

Filesystem-level Linux permissions remain the baseline for artifact access, but role-based checks in the application layer enforce what actions the mapped user may perform. The design should consider both interactive and non-interactive (CI/scripted) use cases, potentially supporting an API token or service-account mode for automation.
