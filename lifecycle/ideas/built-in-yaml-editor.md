---
title: Built-in YAML Editor for Config and DevOps Pipelines
type: idea
status: approved
lineage: built-in-yaml-editor
created: "2026-05-06T11:02:09+10:00"
priority: normal
labels:
    - feature
    - frontend
    - backend
    - enhancement
release: May2026
---

# Built-in YAML Editor for Config and DevOps Pipelines

Provide a first-class in-app YAML editor that allows users to view and modify the main application configuration (`~/.kaos-control/config.yaml`, project registrations, and `lifecycle/config.yaml`) directly within the Innovation Maker UI, without needing to leave the tool or use an external editor.

The editor should also support managing DevOps pipeline definitions (e.g. CI/CD YAML files such as GitHub Actions workflows or similar), with syntax highlighting, validation, and schema-aware autocompletion where possible. Errors should be surfaced inline before changes are saved.

This reduces friction for operators configuring the tool or managing pipeline definitions, keeping the entire workflow within a single interface and lowering the barrier to entry for new users getting the system set up correctly.
