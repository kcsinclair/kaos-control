---
title: kaos-control CLI Init Command
type: idea
status: draft
lineage: cli-init-scaffold
created: "2026-05-08T17:55:01+10:00"
priority: normal
labels:
    - feature
    - onboarding
    - backend
    - go
    - v1
---

# kaos-control CLI Init Command

Add a `kaos-control init` (or `kaos-control init <path>`) CLI subcommand that bootstraps a new project directory tree in place. Running the command in (or targeting) a directory should create all the standard `lifecycle/` subdirectories (`ideas/`, `requirements/`, `backend-plans/`, `frontend-plans/`, `test-plans/`, `tests/`, `defects/`, `releases/`, `sprints/`, `prototypes/`), a skeleton `lifecycle/config.yaml` pre-populated with the standard role and agent definitions, and any other files required for the tool to operate out of the box.

The scaffold must also emit a `CLAUDE.md` if one does not exist, at the project root containing the standard guidance Claude Code needs to operate correctly within the kaos-control lifecycle — covering repo layout, artifact conventions, frontmatter requirements, commit conventions, and agent roles. Additional Claude Code integration files (e.g. `.claude/` settings stubs) should be included where they add value to the onboarding experience.

The goal is that a developer can point `kaos-control init` at any existing or empty repository and immediately have a fully wired lifecycle environment — no manual directory creation, no copy-pasting config templates, and no need to read the full spec before making a first commit.
