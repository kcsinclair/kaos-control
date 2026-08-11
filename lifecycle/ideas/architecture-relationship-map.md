---
title: Architecture Relationship Map
type: idea
status: draft
lineage: architecture-relationship-map
priority: normal
parent: lifecycle/ideas/architecture-templates.md
labels:
    - architecture
    - onboarding
    - visualization
    - graph
    - feature
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

# Architecture Relationship Map

A visual map of the [architecture catalog](../architecture/README.md) that shows
how the architectures relate to one another — so a person can *see* the
landscape and navigate to the right choice, instead of reading nine pages. It is
the visual "browse" surface for [[onboarding-architecture-selection]].

## Why

The architectures aren't independent — they form a landscape with real
relationships:

- **Evolves-into** — Modular Monolith → Cloud-Native Microservices; Single-Service
  SaaS → Microservices.
- **Alternative-to / simpler-than** — Modular Monolith ↔ Single-Service SaaS;
  Standalone Desktop ↔ Local Web.
- **Composed-with / layers-on** — Event-Driven / Streaming on top of
  Microservices; Serverless as the compute tier of a SaaS; Edge/Hybrid pairing
  a device tier with a cloud tier.

Seeing those relationships helps someone place their instinct ("I think I want a
web app for my team") next to neighbours ("…which is a Local Web app; hosted for
many customers it's a Single-Service SaaS") and pick with confidence. A decision
is easier as a *move on a map* than as a cold list.

## What it is

A focused graph view — architectures as nodes, typed relationships as edges —
rendered with the existing graph stack (Cytoscape 2D / 3d-force-graph), reusing
what the roadmap/map already do. Distinct from the main artifact graph in that
it is **scoped to `type: architecture`** and uses **relationship semantics**
rather than lineage.

Node encodes decision signals at a glance (e.g. colour by scale, icon for
offline/mobile); clicking a node opens the architecture artifact and reveals its
compatible stacks (the `related_to` edges). Optionally, extend to a second ring
showing stacks around the selected architecture.

## Data it needs

The relationships must be captured as data, not drawn by hand. Two options:

1. **Reuse `related_to`** among architecture artifacts (quick; loses the edge
   *kind* — evolves-into vs. alternative-to look the same). The current catalog
   already encodes arch↔arch links as body wiki-links, which is a start.
2. **Typed relationship fields** (`evolves_into`, `alternative_to`,
   `composed_with`) — cleaner, and drives edge styling/legend. Depends on
   [[artefact-relationship-labels-and-links]] and the frontmatter-schema work in
   [[architecture-templates]] §6.

Recommendation: ship with (1) using the wiki-links already in the catalog, and
move to (2) when typed relationships land.

## Scope

- A read-only, scoped graph view (its own route, or a filter/preset on the
  existing graph).
- Legend for relationship kinds + a decision-signal colour key.
- Click-through to the artifact and its compatible stacks.
- Entry point from the project-create / onboarding flow and from the catalog
  README.

## Resolved Questions

- New dedicated view, or a saved preset/filter on the existing graph engine?

> New dedicated view, new left menu for Architecture, this will be the primary view for now.

- 2D (Cytoscape, better for a small curated set + labelled edges) or reuse the
  3D force graph? A small, typed graph probably reads better in 2D.

> 2D and 3D options, we have the engines already.

- Do we show stacks in the same map (bipartite) or only on node-select to keep
  it legible?

> Option to show related tech stack

- Should the map be editable by maintainers in-app, or is it purely derived from
  the catalog artifacts? (Leaning: derived — the artifacts are the source of
  truth.)

> Derived, artifacts are the source of truth.
