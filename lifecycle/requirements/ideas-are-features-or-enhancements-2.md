---
title: Enhancement Creation Flow and Feature Documentation Artifacts
type: requirement
status: blocked
lineage: ideas-are-features-or-enhancements
priority: normal
parent: ideas/ideas-are-features-or-enhancements.md
labels:
    - workflow
    - artefacts
    - feature
release: KC-Release1
assignees:
    - role: product-owner
      who: agent
---

## Problem

Today the UI offers two creation actions — "New Idea" and "New Defect" — but there is no first-class path for raising an **enhancement** against an existing lineage. Users who want to propose an improvement to an already-tracked feature must create a plain idea and manually set the lineage themselves, which is error-prone and loses the semantic distinction between a net-new idea and a scoped enhancement.

Separately, lineages are developer-centric groupings that do not map cleanly to user-visible **features**. There is no artifact type that captures the documented, marketable view of the product's capabilities, making it difficult for a tech-writer agent (see [[tech-writer-agent]]) to organise documentation around features rather than implementation lineages.

## Goals / Non-goals

### Goals

1. Add an "Enhancement" creation flow so users can raise scoped improvements against an existing lineage with minimal friction.
2. Introduce an `enhancement` label convention (not a new artifact type) that distinguishes enhancements from net-new ideas, while keeping both stored as `type: idea`.
3. Present three creation actions in the artifact list and board views, in order: **New Idea**, **New Enhancement**, **New Defect**.
4. Define a `feature` artifact type for documentation and marketing purposes, decoupled from lineage.

### Non-goals

- Renaming or restructuring existing lineages (noted as future work in the idea).
- Changing how defects work — the current defect flow is satisfactory.
- Automatic classification of ideas as enhancements — the user decides at creation time.

## Detailed Requirements

### Functional

#### FR-1: Enhancement creation flow

- A "New Enhancement" button SHALL appear in the artifact list view and board view, positioned between "New Idea" and "New Defect".
- Clicking "New Enhancement" SHALL open the existing idea creation modal (BrainDumpModal / IdeaChatPanel) pre-configured with:
  - `type: idea`
  - The label `enhancement` added automatically to the `labels` array.
  - A **required** lineage picker that lets the user select from existing lineages. The selected lineage SHALL be written into the artifact's `lineage` frontmatter field.
- The lineage picker SHALL support autocomplete/search over known lineages (reuse or extend the lineage-filter-autocomplete component if available — see [[lineage-filter-autocomplete]]).
- An enhancement artifact is stored under `lifecycle/ideas/` with the same filename convention as a regular idea (slug derived from title, no index suffix for the originating file).

#### FR-2: Label convention for enhancements

- Enhancements SHALL be distinguished from net-new ideas by the presence of an `enhancement` label in the `labels` frontmatter array.
- Existing filtering, board columns, and table views SHALL treat `enhancement`-labelled ideas identically to other ideas unless a view-specific filter is applied.
- The artifact list view SHOULD offer a filter chip or dropdown option to show only enhancements or only non-enhancement ideas.

#### FR-3: Updated creation button order

- The artifact list view and board view SHALL present creation actions in this order:
  1. New Idea
  2. New Enhancement
  3. New Defect

#### FR-4: Feature artifact type (documentation/marketing)

- Add `feature` to the `KnownTypes` vocabulary in `internal/artifact/artifact.go`.
- Feature artifacts SHALL be stored under `lifecycle/features/`.
- Feature frontmatter SHALL include at minimum: `title`, `type: feature`, `status`, `lineage` (optional — may reference zero, one, or many lineages), and `labels`.
- A feature artifact's body documents the user-visible capability for marketing or help documentation. It is not required to map 1:1 to a lineage; one lineage may produce several features, and one feature may span multiple lineages.
- The tech-writer agent (future) SHALL be the primary producer of feature artifacts.

### Non-functional

- **NFR-1**: The lineage picker in the enhancement flow must load and render within 200 ms for projects with up to 500 lineages.
- **NFR-2**: No new database tables are required; the existing SQLite index already stores lineage values and can serve the autocomplete query.
- **NFR-3**: The `feature` artifact type must be indexable by the existing watcher and startup scan without code changes beyond adding the type string and the `lifecycle/features/` directory.

## Acceptance Criteria

- [ ] "New Enhancement" button is visible in artifact list view and board view, between "New Idea" and "New Defect".
- [ ] Clicking "New Enhancement" opens the creation modal with the `enhancement` label pre-set and a required lineage picker.
- [ ] The lineage picker autocompletes over existing lineages from the index.
- [ ] Submitting the enhancement modal creates an idea artifact under `lifecycle/ideas/` with `type: idea`, `labels: [enhancement]`, and the selected `lineage`.
- [ ] Existing idea creation ("New Idea") is unaffected — lineage remains optional and no `enhancement` label is added.
- [ ] Artifact list view can filter to show only enhancements or exclude enhancements.
- [ ] `feature` is accepted as a valid artifact type (no validation error on parse or index).
- [ ] A manually created feature artifact under `lifecycle/features/` is indexed at startup and on filesystem change.
- [ ] Feature artifacts appear in the artifact list, table, and graph views.
- [ ] All existing tests pass; no regression in idea or defect creation flows.

## Open Questions

1. Should the "New Enhancement" flow also pre-populate the `parent` field pointing to a specific artifact within the selected lineage, or is setting the `lineage` field sufficient?
2. Should feature artifacts participate in the standard status workflow (draft → approved → done), or do they need a simpler lifecycle (e.g., draft → published)?
3. Is a dedicated `lifecycle/features/` directory acceptable, or should features live alongside requirements?
