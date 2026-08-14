---
title: 'New Project Init: Existing or New Directory'
type: idea
status: approved
lineage: new-project-init-directory-options
created: "2026-08-14T11:38:41+10:00"
priority: normal
labels:
    - feature
    - frontend
    - onboarding
    - ux
    - ui
    - v1
---

# New Project Init: Existing or New Directory

When the user clicks the "New Project" button, they are presented with two initialisation options: use an existing directory on disk, or create a new directory. Both paths result in kaos-control being initialised in the chosen location.

For the existing-directory flow, the user browses to or types in a path to a pre-existing folder. kaos-control then scaffolds its `lifecycle/` structure and config inside that directory without disturbing existing files. For the new-directory flow, the user specifies a parent location and a directory name; kaos-control creates the folder and initialises it from scratch.

This change improves onboarding by accommodating users who already have a project directory they want to bring under lifecycle management, alongside users starting from a blank slate.
