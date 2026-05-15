---
title: 'Tests: Artefact Relationship Labels and Clickable Links'
type: test
status: approved
lineage: artefact-relationship-labels-and-links
parent: lifecycle/test-plans/artefact-relationship-labels-and-links-5-test.md
---

# Tests: Artefact Relationship Labels and Clickable Links

## Overview

This artifact documents the integration and component tests implemented for the
directional relationship labels and clickable link features described in the
artefact-relationship-labels-and-links feature lineage.

## Test files

### Frontend unit tests

**`tests/web/edgeLabels.test.ts`** — Milestone 1

Vitest unit tests for the `edgeLabel()` helper and `EDGE_LABEL_MAP` constant in
`web/src/components/map/graphConstants.ts`. Covers:

- All 6 known relationship kinds (`parent`, `depends_on`, `blocks`, `related_to`,
  `members`, `wiki`) in both `outbound` and `inbound` directions — 12 assertions.
- Unknown kind fallback: returns the kind string uppercased.
- `EDGE_LABEL_MAP` structural completeness: all 6 kinds present, both directions
  non-empty, consistent with `edgeLabel()` return values.

### Frontend component tests

**`tests/web/ArtifactModal.relationship-labels.test.ts`** — Milestone 2

`@vue/test-utils` component tests for `ArtifactModal.vue` relationship label
rendering. Covers:

- TC1: Outbound `parent` edge renders `CHILD OF` in the Outbound section.
- TC2: Inbound `parent` edge renders `PARENT OF` in the Inbound section.
- TC3: All 6 kinds render correct labels in the correct section (outbound vs
  inbound) — 12 label assertions plus a combined mount test.
- TC4: Unknown kind `custom_rel` renders `CUSTOM_REL` (uppercase fallback) in
  both sections.
- Additional structural checks: correct section placement, empty edge list
  hides the footer.

**`tests/web/ArtifactModal.relationship-links.test.ts`** — Milestone 3

`@vue/test-utils` component tests for click navigation in `ArtifactModal.vue`.
Covers:

- TC5: Clicking an outbound edge link emits `navigate-artifact` with the target
  path.
- TC6: Clicking an inbound edge link emits `navigate-artifact` with the source
  path.
- TC7: Edge path elements are rendered as `<a>` tags bearing the
  `edge-path-link` CSS class (cursor: pointer applied via scoped CSS).
- TC8: `mouseenter`/`mouseleave` events do not remove the link element from the
  DOM; the CSS `:hover` rule applies the highlight colour.

**`tests/web/ArtifactModal.relationship-a11y.test.ts`** — Milestone 4

Accessibility tests for relationship links in `ArtifactModal.vue`. Covers:

- TC9: `<a>` elements accept programmatic focus; `document.activeElement` is
  the link after `focus()`.
- TC10: A click (the mechanism activated by Enter on `<a>`) emits
  `navigate-artifact` with the correct path for both outbound and inbound edges.
- TC11: Every `.edge-path-link` has a non-empty `aria-label` attribute that
  contains the directional label and the artefact path (e.g.,
  `"CHILD OF lifecycle/requirements/login-2.md"`). Spot-checked for
  `depends_on` inbound (`DEPENDED ON BY`) and `wiki` outbound (`LINKS TO`).

### Backend integration tests

**`tests/integration/graph_edges_test.go`** — Milestone 5

Go integration tests (build tag `integration`) for the graph API. Covers:

- `TestGraphEdges_KindValues` (TC12): Seeds one artifact of each relationship
  kind (`parent`, `depends_on`, `blocks`, `related_to`, `members`, `wiki`),
  calls `GET /api/p/testproject/graph`, and asserts every edge's `kind` field
  matches one of the canonical constants from `internal/artifact/artifact.go`.
- `TestGraphEdges_JSONShape` (TC13): Parses the raw edge JSON objects and
  asserts each has exactly the allowed fields (`source`, `target`, `kind`,
  optionally `label`) — no unexpected extra fields.
- `TestGraphEdges_ParentEdgePresent`: Focused sanity check that a seeded
  `parent:` frontmatter relationship appears as a `"parent"` kind edge in the
  graph response.

## Running the tests

```sh
# Frontend unit + component tests
cd tests/web && pnpm run test

# Backend integration tests
go test ./tests/integration/ -tags integration -run TestGraphEdges
```
