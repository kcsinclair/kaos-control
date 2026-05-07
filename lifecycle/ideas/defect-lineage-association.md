---
title: Associate Defects with an Existing Lineage
type: idea
status: draft
lineage: defect-lineage-association
created: "2026-04-27T20:00:49+10:00"
priority: normal
labels:
    - defect
    - feature
    - workflow
    - artefacts
release: April2026
---

# Associate Defects with an Existing Lineage

When a defect is raised, it should be possible to associate it with an existing lineage so that the defect artifact becomes part of that lineage's history. The user could provide the lineage tag at raise-time (e.g. via the UI form or agent prompt), or the association could be made later by editing the artifact's frontmatter.

This would allow defects to carry a `lineage:` frontmatter field pointing back to the originating slug, and the UI/graph could then surface defects as nodes within that lineage chain rather than as disconnected artifacts.

A secondary flow should support retroactive association — either by editing the defect artifact directly or through a dedicated UI action — so that defects raised without a lineage can still be linked once the relevant lineage is identified.
