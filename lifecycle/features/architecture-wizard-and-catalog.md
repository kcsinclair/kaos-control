---
title: Architecture Wizard & catalog
type: feature
status: approved
lineage: feature-architecture-wizard-and-catalog
created: "2026-08-21T12:10:00+10:00"
summary: Guided Q&A that recommends an architecture + tech-stack from a shipped catalog and promotes the choice into the project.
function: Architecture
labels:
    - feature
    - architecture
    - onboarding
related_to:
    - lifecycle/ideas/onboarding-architecture-selection.md
    - lifecycle/ideas/architecture-templates.md
    - lifecycle/architecture/architecture-summary.md
---

# Architecture Wizard & catalog

Choose the architecture and tech-stack a project should follow, then promote
that choice into first-class artifacts.

## What it does

- **Shipped catalog.** A curated set of candidate architectures (modular
  monolith, cloud-native microservices, serverless, standalone desktop, static
  site, …) and tech-stacks (Go + Vue, TypeScript React/Nest, Python + FastAPI,
  …) ships embedded in the binary and is seeded into each project's
  `lifecycle/architecture/` on init.
- **Guided Q&A.** A short questionnaire (offline? realtime? scale? team
  language? AI-centric? cost-sensitive?) drives a recommendation and shows the
  rejected alternatives.
- **Promote.** On confirm, the chosen architecture + tech-stack are promoted to
  the `lifecycle/architecture/` root (parent-stamped to their catalog source,
  `catalog` label stripped), an `architecture-summary.md` is written (rationale
  trail + architecture-breaking requirements), and an ADR is authored.
- **Optional scaffolding step.** An opt-in final step can scaffold project
  config, DevOps pipelines, agent directives, and a repo skeleton from the
  chosen tech-stack's `stack_profile`.
- **Launchable any time.** Product-owner-gated; not limited to project creation.

Reachable at **Architecture → Architecture Wizard**; API under
`/architecture/wizard/*` and `/architecture/promote`.
