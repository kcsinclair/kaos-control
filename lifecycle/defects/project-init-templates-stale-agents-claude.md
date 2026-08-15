---
title: Project initialisation templates do not include recent agent and CLAUDE.md changes
type: defect
status: approved
lineage: project-init-templates-stale-agents-claude
created: "2026-08-15T10:05:51+10:00"
priority: normal
labels:
    - defect
    - onboarding
    - agents
    - config
    - lifecycle
---

# Project initialisation templates do not include recent agent and CLAUDE.md changes

## Reproduction Steps

1. Initialise a new project using the project initialisation flow.
2. Inspect the generated template files for CLAUDE.md, agent definitions, and related configuration.
3. Compare the generated output against the current canonical versions in this repository.

## Expected Behaviour

All generated initialisation templates — including CLAUDE.md, agent prompts, allowed_write_paths, role definitions, and any other scaffolded config — should reflect the latest changes made to those artefacts in the kaos-control repository.

## Actual Behaviour

The initialisation templates are stale: they do not include recent updates to agent definitions, CLAUDE.md content, or associated configuration. New projects initialised from these templates are missing current guidance, agent behaviour, and config structure.
