---
title: Projects Page CRUD Operations
type: idea
status: done
lineage: projects-crud-ui
created: "2026-05-15T18:02:41+10:00"
priority: high
labels:
    - feature
    - frontend
    - backend
    - v1
release: KC-Release2
---

# Projects Page CRUD Operations

Add full create, read, update, and delete (CRUD) functionality to the projects page in the kaos-control GUI, allowing users to manage their registered projects directly from the web interface without needing to manually edit configuration files under `~/.kaos-control/projects/*.yaml`.

The feature should include a project list view showing all registered projects, a form for creating new projects and editing existing ones (covering fields such as name, path, and any relevant config), and a delete action with confirmation. All operations should be backed by corresponding REST API endpoints in the Go server.

This will significantly improve onboarding and day-to-day usability, as users currently have no GUI path to register or modify projects — they must edit YAML files directly and restart the server.

When a project is added, it should check if the kaos-control files exists, e.g. lifecycle/config.yaml and if it does not exist, initialise the directory for kaos control.
