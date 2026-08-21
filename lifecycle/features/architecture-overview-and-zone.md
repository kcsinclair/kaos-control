---
title: Architecture Overview, ADRs & standards
type: feature
status: approved
lineage: feature-architecture-overview-and-zone
created: "2026-08-21T12:11:00+10:00"
summary: A read-mostly overview of the chosen architecture, its ADRs and standards, plus a relationship map — the architecture zone is standing reference, hidden from the flow board by default.
function: Architecture
labels:
    - feature
    - architecture
    - visualization
related_to:
    - lifecycle/ideas/architecture-overview-view.md
    - lifecycle/ideas/architecture-relationship-map.md
    - lifecycle/architecture/decisions/adr-0002-readopt-modular-monolith.md
---

# Architecture Overview, ADRs & standards

Once an architecture is chosen, the Architecture section becomes the home for
everything under `lifecycle/architecture/`.

## What it does

- **Architecture Overview.** One screen bringing together the chosen
  architecture, the tech-stack and its mapping, the wizard Q&A rationale, the
  architecture-breaking requirements, the standards, and the ADRs — with
  one-click entry points to re-run the wizard or raise a new ADR. Renders
  whenever any architecture content exists (chosen arch, summary, ADRs, or
  standards), each panel degrading independently.
- **Relationship map.** A 2D/3D graph of the architecture zone. The section
  defaults to the map before a selection exists and to the overview after.
- **ADRs.** Numbered Architecture Decision Records (`decisions/adr-NNNN-*.md`,
  `type: adr`) with supersede chaining; raise new ones from the overview.
- **Standards.** Non-functional baselines under `architecture/standards/` (e.g.
  secrets handling, filesystem sandboxing, the index-is-a-cache rule) that
  agents are directed to follow.
- **Zone-aware surfaces.** Architecture artifacts are standing reference, not
  lifecycle flow, so the list/board hide the whole architecture zone by default
  behind a "show architecture inline" toggle; explicitly filtering by the
  architecture stage/type still shows them.
- **Summary in graph tooltips.** Map node tooltips include the `summary`
  frontmatter field when present.

Reachable at **Architecture** (→ overview / map / wizard); API under
`/architecture/overview`, `/architecture/adrs`.
