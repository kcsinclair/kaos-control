---
title: Universal Text Filter Across All Views
type: requirement
status: approved
lineage: universal-text-filter
created: "2026-05-06"
priority: high
parent: lifecycle/ideas/universal-text-filter.md
labels:
    - feature
    - frontend
    - usability
    - enhancement
assignees:
    - role: product-owner
      who: agent
---

# Universal Text Filter Across All Views

## Problem

The application has four primary views — Artifact List (table), Kanban Board, Graph, and Project Feed — each displaying potentially large sets of artifacts. Users can narrow results using dropdown filters (stage, status, type, label, priority), but there is no way to search by text. When a project contains dozens or hundreds of artifacts, locating a specific item by name, slug, or keyword requires visually scanning the entire view. This slows down everyday workflows such as finding a known artifact, checking the status of a lineage, or triaging a set of related items.

## Goals / Non-goals

### Goals

- **G1** — Provide a single, consistent free-text filter input on every data view (Artifact List, Kanban Board, Graph, Project Feed).
- **G2** — Filter results in real time as the user types (client-side, no round-trip required for views that already hold the data; server-side param for paginated views).
- **G3** — Compose with existing dropdown filters using AND logic so users can combine text and categorical filters without conflict.
- **G4** — Match against the most relevant text fields per view (at minimum: title, lineage slug, type, status).
- **G5** — Case-insensitive matching with visible highlighting of matched text where practical.
- **G6** — Consistent placement, appearance, and keyboard behaviour so users build one mental model.

### Non-goals

- **NG1** — Full-text search of artifact markdown body content (may be added later; v1 filters on frontmatter fields only).
- **NG2** — Persisting filter state in the URL or across navigation (filter resets on route change; shareability is deferred).
- **NG3** — Saved / named filter presets.
- **NG4** — Fuzzy matching or typo tolerance (exact substring match is sufficient for v1).
- **NG5** — Server-side search index (e.g. FTS5); the backend only needs a simple `q` query parameter for the paginated list endpoint.

## Detailed Requirements

### Functional

#### FR-1: Filter input component

A reusable `TextFilter` (or similarly named) component must be created and placed in a consistent location on each view. It must:

- Render as a single-line text input with a search/magnifying-glass icon and a clear (×) button.
- Emit filter-change events debounced at 150–300 ms to avoid excessive re-renders.
- Be visually aligned with the existing dropdown filter bar on each view.

#### FR-2: Artifact List view (table) integration

- Add a `q` (or `search`) query parameter to the `GET /artifacts` list endpoint. When present, the backend filters rows where `title`, `slug`, `lineage`, `type`, or `status` contain the search string (case-insensitive substring match).
- The frontend must send this parameter alongside existing filter params and reset pagination to page 1 when the text changes.
- Matched substrings in the title column should be highlighted (e.g. `<mark>` tag or equivalent CSS).

#### FR-3: Kanban Board view integration

- Apply client-side filtering to the already-loaded artifact cards. A card is visible if any of its filterable text fields (title, lineage slug, type, status) contain the search string as a case-insensitive substring.
- Cards that do not match must be hidden (not merely dimmed) to keep columns compact.
- Empty columns after filtering should remain visible with a "No matching items" indicator.

#### FR-4: Graph view integration

- Nodes that do not match the filter text must be visually dimmed (reduced opacity) rather than removed, so that the graph topology remains stable and comprehensible.
- Matched nodes should retain full opacity and may optionally receive a highlight outline.
- Edge visibility follows node visibility: edges connecting two dimmed nodes are dimmed; edges connecting at least one matched node remain visible.

#### FR-5: Project Feed view integration

- Apply client-side filtering to feed entries. A feed entry is visible if its displayed text (artifact title, event description) contains the search string as a case-insensitive substring.
- Entries that do not match are hidden.

#### FR-6: Composition with existing filters

- The text filter is ANDed with all active dropdown filters. An artifact must satisfy every active dropdown filter AND contain the search text to be displayed.
- Clearing the text filter must restore the view to the state defined by the remaining dropdown filters alone.

#### FR-7: Keyboard interaction

- The filter input should be focusable via a keyboard shortcut (suggested: `/` when no other input is focused, following common convention).
- Pressing `Escape` while the filter input is focused should clear its value and blur the input.

### Non-functional

#### NFR-1: Performance

- Client-side filtering must complete within one animation frame (< 16 ms) for datasets of up to 500 artifacts.
- The debounce interval must prevent perceptible UI jank during rapid typing.

#### NFR-2: Accessibility

- The input must have an accessible label (`aria-label` or associated `<label>`).
- The clear button must be keyboard-accessible and have an `aria-label`.
- Highlighted matches must not rely solely on colour (use `<mark>` or a visible background change).

#### NFR-3: Responsiveness

- The filter input must remain usable on viewports as narrow as 360 px; it may collapse behind a search icon toggle on small screens.

## Acceptance Criteria

- [ ] A text filter input is present on the Artifact List view and filters the table in real time.
- [ ] A text filter input is present on the Kanban Board view and hides non-matching cards in real time.
- [ ] A text filter input is present on the Graph view and dims non-matching nodes in real time.
- [ ] A text filter input is present on the Project Feed view and hides non-matching entries in real time.
- [ ] Typing a substring of an artifact's title shows only artifacts whose title contains that substring (case-insensitive).
- [ ] Filtering also matches on lineage slug, type, and status fields.
- [ ] Text filter composes with dropdown filters using AND logic — enabling both simultaneously narrows results correctly.
- [ ] Clearing the text input restores the full (dropdown-filtered) result set.
- [ ] Matched text is highlighted in the Artifact List title column.
- [ ] Pressing `/` focuses the filter input; pressing `Escape` clears and blurs it.
- [ ] On the Artifact List view, changing the filter text resets pagination to page 1.
- [ ] The backend `GET /artifacts` endpoint accepts a `q` parameter and returns only matching rows.
- [ ] Client-side filter applies within one animation frame for 500 artifacts.
- [ ] Filter input has appropriate `aria-label` and the clear button is keyboard-accessible.
- [ ] Filter input placement and behaviour are consistent across all four views.

## Questions

- **OQ-1**: Should the Graph view offer a "focus" mode that, in addition to dimming, re-centres the camera on matched nodes? This could improve usability for large graphs but adds complexity.

> Yes, I think this code from https://github.com/kcsinclair/tekadm/blob/main/link-tag-visualisation/generate_tag_graph_3d.py should be helpful

```
  function focusNode(node) {
    const distance = 80;
    const distRatio = 1 + distance / Math.hypot(node.x || 1, node.y || 1, node.z || 1);
    Graph.cameraPosition(
      { x: (node.x || 0) * distRatio, y: (node.y || 0) * distRatio, z: (node.z || 0) * distRatio },
      node,
      1000
    );
  }

  function handleNodeClick(node) {
    nodes.forEach(n => { n.__highlighted = false; });
    node.__highlighted = true;
    const connectedIds = new Set();
    links.forEach(l => {
      const src = typeof l.source === "object" ? l.source.id : l.source;
      const tgt = typeof l.target === "object" ? l.target.id : l.target;
      if (src === node.id) connectedIds.add(tgt);
      if (tgt === node.id) connectedIds.add(src);
    });
    connectedIds.forEach(id => { if (nodeMap[id]) nodeMap[id].__highlighted = true; });
    Graph.nodeColor(Graph.nodeColor());
    focusNode(node);
    showModal(node);
  }
```

- **OQ-2**: Should the Project Feed text filter also match on actor/agent names, or only on artifact titles and event descriptions?

> rtifact titles and event descriptions works for now.
  
- **OQ-3**: Is the `/` keyboard shortcut acceptable, or does it conflict with any planned shortcut (e.g. command palette)?

> Yes / is perfect, as I am a VI guy!
