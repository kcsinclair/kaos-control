---
title: Flexible Process and Process Map
type: idea
status: draft
lineage: flexible-process-and-process-map
priority: normal
labels:
    - workflow
    - process
    - enhancement
    - ux
    - feature
    - frontend
release: KC-Release3
---

# flexible process and process map

## Raw Idea

the workflow defines the steps in the process, you can reorder or skip them so long as each step gets the artifacts it needs.

the process will be displayed visually so people can adjust it, and saved as a configuration file

## Idea

The workflow should allow teams to reorder or skip lifecycle steps, provided that each step still receives the artifacts it depends on. The system enforces artifact prerequisites rather than a fixed step sequence, giving teams the freedom to adapt the process to their context without breaking downstream agent inputs.

A visual process map will let users inspect and adjust the configured workflow, dragging steps to reorder them or toggling optional stages on and off. The map makes the implicit process explicit and auditable at a glance.

The resulting configuration will be persisted to a config file (likely an extension of `lifecycle/config.yaml`), so the customised process is version-controlled alongside the artifacts it governs.
