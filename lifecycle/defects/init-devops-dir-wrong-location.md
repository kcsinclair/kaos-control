---
title: Project Init Creates devops Directory in Project Root Instead of lifecycle/devops
type: defect
status: draft
lineage: init-devops-dir-wrong-location
created: "2026-08-19T09:34:52+10:00"
priority: normal
labels:
    - backend
    - defect
release: KC-Release5
assignees:
    - role: backend-developer
      who: agent
---

# Project Init Creates devops Directory in Project Root Instead of lifecycle/devops

## Reproduction Steps

1. Initialise a new project using the kaos-control init flow.
2. Observe the directory structure created in the project root.

## Expected Behaviour

The devops sample YAML file should be created at `lifecycle/devops/sample.yaml` (inside the `lifecycle/` directory tree, consistent with all other lifecycle artifact directories).

## Actual Behaviour

A `devops/` directory is created in the project root and the sample YAML is placed there, rather than under `lifecycle/devops/`.
