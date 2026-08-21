---
title: Project onboarding (create new / add existing)
type: feature
status: approved
lineage: feature-project-onboarding
created: "2026-08-21T12:15:00+10:00"
summary: Register a project by creating a new directory or adding an existing one; idempotent scaffolding lays down the lifecycle tree, config, directives, and architecture catalog.
function: Projects & onboarding
labels:
    - feature
    - onboarding
    - project
---

# Project onboarding (create new / add existing)

Bring a project under kaos-control without hand-authoring its structure.

## What it does

- **Two modes.** "New" creates the target directory; "existing" adopts a
  directory you already have. Adding an already-initialised directory registers
  it as-is rather than erroring.
- **Directory pre-check.** `POST /projects/check-directory` validates
  existence, writability, and initialised-state and echoes the resolved path
  before you commit.
- **Idempotent scaffold.** Lays down the `lifecycle/` stage tree, `tests/` and
  `devops/`, seed `config.yaml`, `.claude/settings.json`, `.gitignore`, the
  embedded architecture catalog, and the AGENTS.md-primary directive set —
  completing only what's missing, never overwriting.
- **Auto-commit on a fresh repo.** A kaos-control-created git repo gets the
  scaffold committed; an existing repo returns the commands instead.

Reachable from the project picker (**New / Add project**); API under
`/projects`.
