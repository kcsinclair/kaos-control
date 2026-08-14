---
title: Architecture Overview — Visualise the Chosen Architecture
type: idea
status: draft
lineage: architecture-overview-view
created: "2026-08-14T12:30:00+10:00"
priority: normal
labels:
    - architecture
    - visualization
    - frontend
    - ux
    - feature
release: KC-Release5
parent: lifecycle/ideas/architecture-templates.md
---

# Architecture Overview — Visualise the Chosen Architecture

Once a project has run the Architecture Wizard
([[onboarding-architecture-selection]]) and its choices have been promoted
into `lifecycle/architecture/` ([[architectural-artefacts]]), the
**Architecture** left-menu section gains an **overview view of the chosen
architecture** — the place to visualise, review, and understand the current
architecture and what options exist for architectural, design, and technology
change.

The view brings together, on one screen:

- the **chosen architecture** and its components / interactions (rendered
  from the promoted architecture artifact);
- the **tech stack** information (the promoted stack artifact) and how the
  tech maps onto the architecture's components;
- the **questions and answers** from the wizard — the rationale trail that
  led to this selection, sourced from `architecture-summary.md`;
- the **critical / architecture-breaking requirements** and how each relates
  to the architecture and stack (also from the summary);
- **design elements and non-functional requirements** (the
  `architecture/standards/` set);
- the **Architecture Decision Records** for the project
  (`architecture/decisions/`), newest first, each click-through to the
  artifact.

Navigation behaviour (converged with [[architecture-relationship-map]] FR-1):
before a selection exists, the Architecture section defaults to the
relationship map for browsing the catalog; once a selection exists this
overview becomes the section's default view, with the map and the wizard one
click away. The view is read-mostly — editing happens in the artifacts, which
remain the source of truth — but it offers the entry points to re-run the
wizard and to raise a new ADR.

A later enhancement can add an auto-generated diagram of the *actual* current
system derived from the codebase ([[architecture-auto-diagram]]), displayed
beside the intended architecture for drift comparison.
