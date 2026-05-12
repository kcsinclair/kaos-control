---
title: Auto-Create Projects Directory on First Run
type: idea
status: planning
lineage: auto-create-projects-dir
created: "2026-05-12T12:25:51+10:00"
priority: high
labels:
    - feature
    - onboarding
    - operability
    - go
release: KC-Release1
---

# Auto-Create Projects Directory on First Run

When `kaos-control` starts for the first time, it should automatically create the `~/.kaos-control/projects/` directory if it does not already exist. Currently, the absence of this directory can cause silent failures or confusing errors when the app attempts to load project registrations.

The directory creation should happen early in the startup sequence, before any config loading or project scanning occurs, so that all subsequent operations can safely assume the directory is present. Standard Go `os.MkdirAll` with appropriate permissions (0700 or 0750) is sufficient.

This is a small quality-of-life improvement for onboarding: a fresh install should work out of the box without requiring the user to manually create directories.
