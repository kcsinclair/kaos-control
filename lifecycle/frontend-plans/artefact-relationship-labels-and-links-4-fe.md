---
title: 'Frontend Plan: Artefact Relationship Labels and Clickable Links'
type: plan-frontend
status: done
lineage: artefact-relationship-labels-and-links
parent: lifecycle/requirements/artefact-relationship-labels-and-links-2.md
---

# Frontend Plan: Artefact Relationship Labels and Clickable Links

## Overview

The artefact detail modal (`ArtifactModal.vue`) and the frontmatter panel (`FrontmatterPanel.vue`) both render relationship edges with the raw `kind` value and non-clickable paths. This plan adds a centralised directional label map and converts edge entries into clickable SPA links. No new API calls are required — the existing `GraphEdge` data (`source`, `target`, `kind`) provides everything needed.

Depends on [[artefact-relationship-labels-and-links]] backend plan (Milestone 1) for canonical edge kind values.

## Milestone 1 — Create the directional label map

### Description

Define a single lookup structure that maps each edge kind to its outbound and inbound human-readable labels, per FR-1 and FR-3 of the requirement. This map will be imported by both `ArtifactModal.vue` and `FrontmatterPanel.vue`.

### Files to change

- `web/src/components/map/graphConstants.ts` — Add a new exported constant alongside the existing edge color palettes:

  ```typescript
  export interface EdgeLabelPair {
    outbound: string  // current artifact is source
    inbound: string   // current artifact is target
  }

  export const EDGE_LABEL_MAP: Record<string, EdgeLabelPair> = {
    parent:     { outbound: 'CHILD OF',        inbound: 'PARENT OF' },
    depends_on: { outbound: 'DEPENDS ON',      inbound: 'DEPENDED ON BY' },
    blocks:     { outbound: 'BLOCKS',          inbound: 'BLOCKED BY' },
    related_to: { outbound: 'RELATED TO',      inbound: 'RELATED TO' },
    members:    { outbound: 'MEMBER OF',       inbound: 'HAS MEMBER' },
    wiki:       { outbound: 'LINKS TO',        inbound: 'LINKED FROM' },
  }
  ```

- Add a helper function in the same file:

  ```typescript
  export function edgeLabel(kind: string, direction: 'inbound' | 'outbound'): string {
    return EDGE_LABEL_MAP[kind]?.[direction] ?? kind.toUpperCase()
  }
  ```

### Acceptance criteria

- [ ] `EDGE_LABEL_MAP` contains all six relationship kinds from FR-1 with correct outbound/inbound labels.
- [ ] `edgeLabel()` returns the mapped label for known kinds and falls back to uppercased `kind` for unknown kinds.
- [ ] The map is a single data structure — no duplicated label logic elsewhere (FR-3).
- [ ] TypeScript compilation passes (`pnpm run type-check`).

## Milestone 2 — Apply directional labels in ArtifactModal

### Description

Replace the raw `e.kind` rendering in `ArtifactModal.vue` with calls to `edgeLabel()`. The outbound section (lines ~251-257) should use `edgeLabel(e.kind, 'outbound')` and the inbound section (lines ~258-264) should use `edgeLabel(e.kind, 'inbound')`.

### Files to change

- `web/src/components/artifact/ArtifactModal.vue`
  - Import `edgeLabel` from `graphConstants.ts`.
  - In the outbound edge loop, replace `{{ e.kind }}` with `{{ edgeLabel(e.kind, 'outbound') }}`.
  - In the inbound edge loop, replace `{{ e.kind }}` with `{{ edgeLabel(e.kind, 'inbound') }}`.

### Acceptance criteria

- [ ] Viewing an artefact that is the parent of another shows "PARENT OF" in the inbound section.
- [ ] Viewing an artefact that is a child shows "CHILD OF" in the outbound section.
- [ ] All six relationship kinds display the correct directional label.
- [ ] Labels are rendered in uppercase, consistent with existing styling.

## Milestone 3 — Apply directional labels in FrontmatterPanel

### Description

`FrontmatterPanel.vue` (lines ~154-179) has identical edge-rendering logic. Apply the same `edgeLabel()` calls to maintain consistency.

### Files to change

- `web/src/components/artifact/FrontmatterPanel.vue`
  - Import `edgeLabel` from `graphConstants.ts`.
  - Replace raw `e.kind` text with `edgeLabel(e.kind, 'outbound')` and `edgeLabel(e.kind, 'inbound')` in the respective sections.

### Acceptance criteria

- [ ] FrontmatterPanel displays the same directional labels as ArtifactModal.
- [ ] No duplicated label logic — both components use the shared `edgeLabel()` function.

## Milestone 4 — Make relationship entries clickable links

### Description

Convert each relationship entry in `ArtifactModal.vue` from plain `<span>` text into a clickable element that navigates to the referenced artefact's detail view using Vue Router (SPA navigation, no full page reload). The link text remains the artefact file path (per resolved question: "file path is OK for v1").

### Files to change

- `web/src/components/artifact/ArtifactModal.vue`
  - Wrap each edge path (`e.target` for outbound, `e.source` for inbound) in a `<a>` element or `<router-link>`.
  - On click, navigate to the artefact detail view. The modal currently opens from `MapView` and `RoadmapView` — clicking a relationship link should:
    1. Close the current modal (or update `selectedNode`).
    2. Open the detail view for the linked artefact.
  - The simplest approach: emit an event (e.g. `navigate-artifact`) with the target path, and let the parent view handle the navigation by updating `selectedNode` / `selectedArtifactNode`. Alternatively, if the modal can directly update the graph store's selected node, do that.
  - Add CSS for interactivity:
    ```css
    .edge-path-link {
      cursor: pointer;
      text-decoration: none;
      color: inherit;
    }
    .edge-path-link:hover {
      color: var(--link-highlight, #60a5fa);
      text-decoration: underline;
    }
    .edge-path-link:focus-visible {
      outline: 2px solid var(--link-highlight, #60a5fa);
      outline-offset: 2px;
    }
    ```

- `web/src/views/project/MapView.vue` — Handle the `navigate-artifact` event from `ArtifactModal`: look up the target node in `store.rawNodes`, update `selectedNode`, and let the modal re-render with the new artefact's edges.

- `web/src/views/project/RoadmapView.vue` — Same handler: update `selectedArtifactNode` to the target node.

### Acceptance criteria

- [ ] Clicking a relationship entry navigates to the referenced artefact's detail view without a full page reload (FR-2).
- [ ] Link text displays the artefact file path.
- [ ] `cursor: pointer` is shown on hover (FR-2.4).
- [ ] A visible hover state (colour change or underline) indicates interactivity (FR-2.4).
- [ ] SPA navigation — no full page reload.

## Milestone 5 — Accessibility

### Description

Ensure relationship links meet NFR-1 accessibility requirements: semantic `<a>` elements (or `role="link"`), keyboard navigation (Tab + Enter), and descriptive `aria-label` attributes.

### Files to change

- `web/src/components/artifact/ArtifactModal.vue`
  - Render links as `<a>` elements with `href="#"` and `@click.prevent` (or `role="link"` + `tabindex="0"` if using `<span>`).
  - Add `aria-label` to each link, e.g.: `aria-label="CHILD OF lifecycle/requirements/login-2.md"` — combining the directional label with the target path.
  - Ensure Tab order flows naturally through the relationship list.

- `web/src/components/artifact/FrontmatterPanel.vue` — Apply the same accessibility attributes if edges are made clickable here too. If FrontmatterPanel is read-only context, links may be omitted, but labels must still use `edgeLabel()`.

### Acceptance criteria

- [ ] Relationship links are keyboard-navigable: Tab focuses each link, Enter activates navigation (NFR-1).
- [ ] Each link has an `aria-label` that includes the relationship direction and target artefact path.
- [ ] Links use semantic `<a>` elements or appropriate ARIA roles.

## Milestone 6 — Visual consistency check

### Description

Verify that the updated relationship rendering preserves the existing design language (NFR-3): monospace font for paths, muted colour palette, 11px labels. The only additions are the hover state and cursor change.

### Files to change

- `web/src/components/artifact/ArtifactModal.vue` — Verify CSS classes `.edge-kind` and the new `.edge-path-link` use consistent font sizes, families, and colours. Adjust if needed to match the existing `.edge-path` styling.

### Acceptance criteria

- [ ] Relationship labels remain 11px, uppercase, monospace-adjacent (matching existing `.edge-kind` style).
- [ ] Path text remains monospace.
- [ ] Hover colour uses the application's standard link highlight colour.
- [ ] No visual regressions to the rest of the modal.
