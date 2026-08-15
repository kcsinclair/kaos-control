---
title: Project initialisation does not include architecture artefacts
type: defect
status: approved
lineage: project-init-missing-architecture-artefacts
created: "2026-08-15T13:28:13+10:00"
priority: normal
labels:
    - defect
    - architecture
    - artefacts
    - onboarding
    - lifecycle
    - backend
---

# Project initialisation does not include architecture artefacts

## Reproduction Steps

1. Create a new kaos-control project via the initialisation flow.
2. Inspect the generated project structure under `lifecycle/`.
3. Observe that `lifecycle/architecture/` and its expected contents (catalog entries, architecture summary, decisions/, standards/) are absent.
4. Also inspect existing kaos-control projects — the architecture artefacts are likewise missing.

## Expected Behaviour

Project initialisation should scaffold the `lifecycle/architecture/` directory structure, including the architecture catalog (candidate architectures and tech-stacks), placeholder files for the architecture summary, `decisions/` (ADRs), and `standards/`. Existing projects should be retrofitted with this structure via a migration or one-time setup step so they conform to the same layout.

## Actual Behaviour

The initialisation flow does not create `lifecycle/architecture/` or any of its required sub-directories and seed files. Existing kaos-control projects also lack this structure, meaning agents and the Architecture Wizard have no canonical location to read from or write to, and the architecture governance workflow cannot function correctly.
