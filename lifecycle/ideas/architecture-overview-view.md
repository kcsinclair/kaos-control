---
title: Architecture Overview — Visualise the Chosen Architecture
type: idea
status: clarifying
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

## Owning the architecture zone (catalog visibility)

The generic lifecycle surfaces — list and board — are about **flow**: artifacts
moving idea → requirement → plan → dev → QA → release. Everything under
`lifecycle/architecture/` is **standing reference**, not flow, and so does not
belong on those surfaces by default. This view is its proper home, and the
end-state should be that it *owns the whole architecture zone*:

- the **catalog** (candidate `architectures/` + `tech-stacks/`, carrying the
  `catalog` label) — browsable here as the menu of options for change, so users
  never need it cluttering the board;
- the **chosen** architecture + tech-stack (promoted to the architecture root);
- the **summary**, **ADRs** (`decisions/`), and **standards**;
- the **archive** (`architecture/archive/` — superseded promoted choices), shown
  as a history/provenance strip, not as live work items.

**Interim vs. destination.** As a first step (shipped ahead of this view) the
list and board hide *catalog material* — candidates (`catalog` label) and
archived choices (`architecture/archive/`) — behind a single **"Show catalog"**
toggle, keyed on the label rather than on `type` so the project's *chosen*
architecture, ADRs, and standards stay visible. See `isCatalogMaterial()` in
`web/src/types/api.ts` and the toggle in `ArtifactListView.vue` /
`KanbanBoardView.vue`. This is deliberately a stopgap.

Once this overview view exists, the cleaner model is: the list/board **exclude
the architecture zone entirely by default** (it has its own home here), demoting
the per-view "Show catalog" toggle to an optional "show architecture inline"
escape hatch — or removing it, since the zone is fully represented in this view.
The discriminator stays **catalog-role**, not artifact `type`: candidates and
archive are reference/history; chosen + ADRs + standards are the live
architecture. Nothing under `lifecycle/architecture/` is ever **deleted** on
selection — superseded choices are archived, and (in the kaos-control repo
itself) the catalog files are the byte-identical source of the shipped embedded
catalog (`catalogfs`), so removal would break the product.
