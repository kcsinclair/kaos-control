---
title: Init Does Not Create lifecycle/devops Folder
type: defect
status: in-development
lineage: init-missing-lifecycle-devops-folder
created: "2026-05-12T14:11:38+10:00"
priority: normal
labels:
    - defect
    - onboarding
    - backend
    - go
---

# Init Does Not Create lifecycle/devops Folder

## Reproduction Steps

1. Run `kaos-control init` (or the equivalent initialisation command) on a new or empty project directory.
2. Inspect the generated `lifecycle/` directory structure.

## Expected Behaviour

The `lifecycle/devops` folder should be created as part of the standard directory scaffold produced by `kaos-control init`, alongside other lifecycle subdirectories (e.g. `ideas/`, `requirements/`, `releases/`, etc.).

## Actual Behaviour

The `lifecycle/devops` folder is not created. The directory is absent from the scaffolded structure, requiring users to create it manually before devops-related artifacts can be stored.
