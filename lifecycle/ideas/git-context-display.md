---
title: Display Git Branch and Context in the GUI
type: idea
status: done
lineage: git-context-display
created: "2026-05-11T13:09:16+10:00"
priority: normal
labels:
    - feature
    - frontend
    - vue
    - operability
release: KC-Release1
---

# Display Git Branch and Context in the GUI

The GUI should display the current git branch and other useful repository context (such as the active project, last commit, or dirty/clean status) in a persistent and visible area of the interface, such as a status bar or header.

This helps users stay oriented — particularly when switching between branches for different features or lifecycle stages — without having to context-switch to a terminal. It also reduces mistakes caused by accidentally working on the wrong branch.

The backend already has a `git` package wrapping go-git; this can expose a lightweight endpoint or WebSocket event to surface branch name and basic repo state to the frontend.
