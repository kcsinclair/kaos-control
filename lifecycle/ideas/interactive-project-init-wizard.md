---
title: Interactive Project Initialisation Wizard
type: idea
status: draft
lineage: interactive-project-init-wizard
created: "2026-05-12T09:59:24+10:00"
priority: high
labels:
    - onboarding
    - feature
    - backend
    - usability
    - v1
release: KC-Release1
---

# Interactive Project Initialisation Wizard

When a user initialises a directory with kaos-control, the tool should interactively prompt for the inputs required to populate the project YAML (e.g. `~/.kaos-control/projects/kaos-control.yaml`) rather than requiring the user to create or edit the file manually.

The wizard should infer sensible defaults where possible — the project path defaults to the directory being initialised and the name defaults to the directory name — but prompt the user to confirm or override each value. It should also collect required fields that cannot be inferred, such as description and owner.

This removes a manual, error-prone setup step and lowers the barrier to getting a project registered and ready to use, making the first-run experience significantly smoother.
