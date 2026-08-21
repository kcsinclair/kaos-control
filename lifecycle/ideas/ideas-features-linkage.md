---
title: Link ideas and features (many-to-many) and surface it on the maps
type: idea
status: draft
lineage: ideas-features-linkage
created: "2026-08-21T12:45:00+10:00"
priority: normal
labels:
    - idea
    - features
    - visualization
    - lifecycle
related_to:
    - lifecycle/features/architecture-overview-and-zone.md
    - lifecycle/ideas/architecture-relationship-map.md
---

# Link ideas and features (many-to-many) and surface it on the maps

Now that **features** are first-class artifacts (`type: feature` under
`lifecycle/features/`), the relationship between an *idea* and the *features*
it produces is worth modelling explicitly — because it is genuinely
**many-to-many**:

- **One idea → one or more features.** A single idea can decompose into several
  distinct shippable capabilities (e.g. "architecture support" became the
  wizard+catalog feature *and* the overview/ADRs/standards feature).
- **Multiple ideas → one feature.** Several ideas can converge into, or extend,
  a single capability (e.g. separate ideas around browsing, filtering, and
  grouping all feed one "Features view" feature).

## The thought

- **Link ideas ↔ features** with an explicit, typed relationship (a frontmatter
  field such as `delivers` / `delivered_by`, or reuse `related_to` with a
  typed edge kind) so the mapping is queryable, not implied.
- **Show it on the maps.** Render idea↔feature edges on the relationship map /
  graph so you can see, for any idea, which features it produced, and for any
  feature, which ideas motivated it. Filter the graph to the ideas→features
  view.
- **Bridge the flow and the rollup.** Ideas live in the idea→release *flow*;
  features are a standing *rollup* of shipped capability. This link is the seam
  between the two — it lets "what did this idea actually ship?" and "why does
  this feature exist?" both be answered from the graph.

## Questions to explore

- Edge direction / naming: `idea --delivers--> feature`, or
  `feature --delivered_by--> idea`, or a neutral `related_to`? (A typed edge
  reads better on the map and in queries.)
- Should a feature also link to the **requirement(s)** / **release** that
  delivered it, not just the originating idea? (Features already carry
  `related_to` links to requirements/releases in some cases.)
- Does this want a dedicated map mode/filter, or just new edge kinds on the
  existing relationship map ([[architecture-relationship-map]])?
- Lifecycle implication: when an idea is `done`, can we assert every feature it
  promised exists? (A completeness check / report.)

Pondering — captured for later, not yet scoped.
