---
title: 'Test Plan: Artefact Relationship Labels and Clickable Links'
type: plan-test
status: in-development
lineage: artefact-relationship-labels-and-links
parent: lifecycle/requirements/artefact-relationship-labels-and-links-2.md
---

# Test Plan: Artefact Relationship Labels and Clickable Links

## Overview

This plan covers integration and component tests for the directional relationship labels and clickable link features described in [[artefact-relationship-labels-and-links]]. Since the backend changes are limited to constant extraction (no API changes), testing focuses heavily on the frontend: verifying label correctness, click navigation, and accessibility.

## Milestone 1 — Unit tests for the edge label map

### Description

Test the `edgeLabel()` helper and `EDGE_LABEL_MAP` to verify that all six relationship kinds return the correct directional labels, and that unknown kinds fall back gracefully.

### Files to change

- `tests/web/edgeLabels.test.ts` (new file) — Vitest unit tests:

  ```typescript
  import { describe, it, expect } from 'vitest'
  import { edgeLabel, EDGE_LABEL_MAP } from '@/components/map/graphConstants'

  describe('edgeLabel()', () => {
    it('returns CHILD OF for parent outbound', () => { ... })
    it('returns PARENT OF for parent inbound', () => { ... })
    it('returns DEPENDS ON for depends_on outbound', () => { ... })
    it('returns DEPENDED ON BY for depends_on inbound', () => { ... })
    it('returns BLOCKS for blocks outbound', () => { ... })
    it('returns BLOCKED BY for blocks inbound', () => { ... })
    it('returns RELATED TO for related_to both directions', () => { ... })
    it('returns MEMBER OF for members outbound', () => { ... })
    it('returns HAS MEMBER for members inbound', () => { ... })
    it('returns LINKS TO for wiki outbound', () => { ... })
    it('returns LINKED FROM for wiki inbound', () => { ... })
    it('falls back to uppercased kind for unknown kinds', () => { ... })
  })
  ```

### Acceptance criteria

- [ ] All 12 label assertions pass (6 kinds × 2 directions).
- [ ] Unknown kind fallback returns the kind string in uppercase.
- [ ] `pnpm run test` passes.

## Milestone 2 — Component tests for ArtifactModal label rendering

### Description

Mount `ArtifactModal.vue` with mock edge data and verify that directional labels are rendered correctly in the inbound and outbound sections.

### Files to change

- `tests/web/ArtifactModal.relationship-labels.test.ts` (new file) — Vitest + @vue/test-utils:

  **Test cases:**

  1. **TC1: Outbound parent edge displays "CHILD OF"** — Provide an edge `{ source: 'current.md', target: 'parent.md', kind: 'parent' }`. Assert the outbound section renders "CHILD OF".

  2. **TC2: Inbound parent edge displays "PARENT OF"** — Provide an edge `{ source: 'child.md', target: 'current.md', kind: 'parent' }`. Assert the inbound section renders "PARENT OF".

  3. **TC3: All six kinds render correct labels** — Mount with one outbound and one inbound edge for each of the six kinds. Assert each label matches the FR-1 table.

  4. **TC4: Unknown kind falls back to uppercase** — Provide an edge with `kind: 'custom_rel'`. Assert it renders "CUSTOM_REL".

### Acceptance criteria

- [ ] TC1–TC4 pass.
- [ ] Labels are rendered in the correct section (inbound vs outbound).
- [ ] No label shows raw edge kind values for known relationship types.

## Milestone 3 — Component tests for clickable navigation

### Description

Verify that clicking a relationship entry in `ArtifactModal.vue` emits the correct navigation event or updates the selected node, enabling SPA navigation to the linked artefact.

### Files to change

- `tests/web/ArtifactModal.relationship-links.test.ts` (new file) — Vitest + @vue/test-utils:

  **Test cases:**

  1. **TC5: Clicking an outbound edge emits navigate event with target path** — Mount modal, click the outbound edge link for `target.md`. Assert the component emits `navigate-artifact` with `'target.md'` (or the equivalent navigation mechanism chosen in the frontend plan).

  2. **TC6: Clicking an inbound edge emits navigate event with source path** — Mount modal, click the inbound edge link for `source.md`. Assert the component emits `navigate-artifact` with `'source.md'`.

  3. **TC7: Link renders as `<a>` element with cursor pointer** — Assert the edge path element is an `<a>` tag. Assert computed style or class includes `cursor: pointer`.

  4. **TC8: Hover state applies highlight class** — Trigger mouseenter on the link element. Assert the hover class or computed colour changes.

### Acceptance criteria

- [ ] TC5–TC8 pass.
- [ ] Clicks trigger SPA navigation — no `window.location` changes or full reloads.
- [ ] Links render as semantic `<a>` elements.

## Milestone 4 — Accessibility tests

### Description

Verify keyboard navigation and ARIA attributes on relationship links.

### Files to change

- `tests/web/ArtifactModal.relationship-a11y.test.ts` (new file) — Vitest + @vue/test-utils:

  **Test cases:**

  1. **TC9: Links are focusable via Tab** — Mount modal with edges. Call `wrapper.find('.edge-path-link').element.focus()`. Assert `document.activeElement` is the link.

  2. **TC10: Enter key triggers navigation** — Focus a link, dispatch `keydown` Enter event. Assert the navigation event is emitted.

  3. **TC11: aria-label includes direction and target** — Assert each link's `aria-label` attribute contains the directional label and the target artefact path (e.g. `"CHILD OF lifecycle/requirements/login-2.md"`).

### Acceptance criteria

- [ ] TC9–TC11 pass.
- [ ] Every relationship link has an `aria-label`.
- [ ] Keyboard navigation (Tab + Enter) works end-to-end.

## Milestone 5 — Backend constant consistency check

### Description

Verify that the edge kind constants introduced in the backend plan ([[artefact-relationship-labels-and-links]] Milestone 1) are used consistently and that the graph API response shape is unchanged.

### Files to change

- `tests/integration/graph_edges_test.go` (new test functions in existing file, or new file if none exists):

  **Test cases:**

  1. **TC12: Graph API returns edges with expected kind values** — Seed a project with artifacts that have `parent`, `depends_on`, `blocks`, `related_to`, `members`, and `wiki` (`[[slug]]`) relationships. Call `GET /api/p/{project}/graph`. Assert each edge's `kind` field matches one of the canonical constants.

  2. **TC13: GraphEdge JSON shape unchanged** — Assert each edge in the response has exactly `source`, `target`, `kind`, and optionally `label` — no new fields.

  ```go
  func TestGraphEdges_KindValues(t *testing.T) {
      env := newTestEnv(t, []seedArtifact{
          makeArtifact("Parent", "idea", "draft", "test-lineage", "", "body"),
          makeArtifact("Child", "requirement", "draft", "test-lineage",
              "lifecycle/ideas/test-lineage.md",
              "depends on X. blocks Y. [[other-slug]]"),
      })
      env.login("admin@test.local", "admin-pass-123")
      resp := env.doRequest("GET", "/api/p/testproject/graph", nil)
      requireStatus(t, resp, 200)
      // Assert edge kinds
  }
  ```

### Acceptance criteria

- [ ] TC12–TC13 pass.
- [ ] `go test ./tests/integration/ -tags integration -run TestGraphEdges` passes.
- [ ] No new fields in the GraphEdge JSON response.
- [ ] All edge kinds in the response match canonical constants.
