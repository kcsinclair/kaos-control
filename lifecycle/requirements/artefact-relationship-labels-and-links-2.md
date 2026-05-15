---
title: Artefact Relationship Labels and Clickable Links
type: requirement
status: approved
lineage: artefact-relationship-labels-and-links
parent: ideas/artefact-relationship-labels-and-links.md
labels:
    - artefacts
    - frontend
    - enhancement
    - usability
release: KC-Release2
assignees:
    - role: product-owner
      who: agent
---

# Artefact Relationship Labels and Clickable Links

## Problem

The Artefact detail modal (`ArtifactModal.vue`) displays relationship edges but has two usability issues:

1. **Ambiguous labels** -- Both inbound and outbound parent edges are labelled with the raw `edge.kind` value (e.g. `PARENT`). This is misleading: when another artefact lists the current artefact as its parent, the inbound edge should read "PARENT OF" (meaning "this artefact is the parent of ..."), while the outbound edge should read "CHILD OF" (meaning "this artefact is a child of ..."). The same directional clarity must apply to every relationship kind that has an inherent direction (`depends_on`, `blocks`, etc.).

2. **Non-navigable paths** -- Relationship entries render the related artefact's path as plain monospace text with no click handler. Users must manually search or navigate the graph to reach the linked artefact.

## Goals / Non-goals

### Goals

- Display directionally accurate, human-readable labels for every relationship kind in both the inbound and outbound sections of the detail modal.
- Make every relationship entry a clickable link that navigates to the referenced artefact's detail view.
- Maintain consistent behaviour across all relationship kinds: `parent`, `depends_on`, `blocks`, `related_to`, `members`, `wiki`.

### Non-goals

- Editing or creating relationships from the detail modal (future scope).
- Changing the underlying data model or API response shape for `GraphEdge`.
- Adding relationship rendering to views other than the artefact detail modal (e.g. graph tooltips).

## Detailed Requirements

### Functional

#### FR-1: Directional relationship labels

For each relationship kind, the UI must use a pair of labels depending on edge direction relative to the current artefact:

| Kind          | Outbound label (current -> target) | Inbound label (source -> current) |
|---------------|-------------------------------------|-------------------------------------|
| `parent`      | CHILD OF                            | PARENT OF                           |
| `depends_on`  | DEPENDS ON                          | DEPENDED ON BY                      |
| `blocks`      | BLOCKS                              | BLOCKED BY                          |
| `related_to`  | RELATED TO                          | RELATED TO                          |
| `members`     | MEMBER OF                           | HAS MEMBER                          |
| `wiki`        | LINKS TO                            | LINKED FROM                         |

Labels must be rendered in uppercase, consistent with the existing styling.

#### FR-2: Clickable relationship links

Each relationship entry (both inbound and outbound) must be a clickable element that:

1. Navigates the application to the detail view of the referenced artefact.
2. Uses in-app (SPA) navigation -- not a full page reload.
3. Displays the artefact path (or title, if available) as the link text.
4. Shows `cursor: pointer` and a visible hover state to indicate interactivity.

#### FR-3: Label mapping extensibility

The outbound/inbound label pairs must be defined in a single lookup structure (object or map) so that adding a new relationship kind requires only one entry, not scattered conditionals.

### Non-functional

#### NFR-1: Accessibility

- Clickable relationship entries must be rendered as `<a>` elements or have `role="link"` with appropriate `aria-label` including the relationship direction and target artefact name.
- Keyboard-navigable (Tab + Enter).

#### NFR-2: Performance

- No additional API calls. The existing graph edge data already contains all information needed for both label mapping and navigation.

#### NFR-3: Visual consistency

- Link styling must follow the existing design language of the modal (monospace font for paths, muted colour palette, 11px labels).
- Hover state should use the application's standard link highlight colour.

## Acceptance Criteria

- [ ] Viewing an artefact that is the parent of another artefact shows the inbound relationship labelled "PARENT OF".
- [ ] Viewing an artefact that is the child of another artefact shows the outbound relationship labelled "CHILD OF".
- [ ] All six relationship kinds display the correct directional label per the table in FR-1.
- [ ] Clicking a relationship entry navigates to the referenced artefact's detail view without a full page reload.
- [ ] Relationship links show `cursor: pointer` and a visible hover effect.
- [ ] Relationship links are keyboard-accessible (Tab to focus, Enter to navigate).
- [ ] The label mapping is defined in a single data structure, not duplicated across inbound/outbound rendering logic.
- [ ] No new API endpoints or changes to the `GraphEdge` response shape are required.
- [ ] Existing relationship display styling (font, size, colour) is preserved apart from the added interactivity.

## Resolved Questions

- Should the link text display the artefact **title** (from frontmatter) instead of the file path, when available? The current UI shows paths; titles may be more user-friendly but require the data to be present on the edge or fetched from the index.

> file path is OK for v1

- Should `wiki`-type links (inline `[[slug]]` references) be included in this work, or deferred to a separate enhancement?

> yes, they can be included.
