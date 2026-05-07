---
title: Tests — Light Mode Colour Scheme for Graphs
type: test
status: draft
lineage: light-mode-graphs
parent: lifecycle/test-plans/light-mode-graphs-5-test.md
---

# Tests — Light Mode Colour Scheme for Graphs

## Overview

This artifact documents the automated test suite implementing Milestone 1 of the
light-mode-graphs test plan.  Milestones 2–5 are covered by manual procedures
documented inline below.

---

## Milestone 1 — Automated unit tests for `useGraphTheme()`

**Test file:** `tests/web/graphConstants.test.ts`

78 tests, all passing.  Runs in < 300 ms.

### Scenarios covered

#### 1. Palette selection (6 tests)

- `useGraphTheme()` returns a palette with a dark canvas background (`luminance < 0.1`) when the theme store is set to `'dark'`.
- Returns a light canvas background (`luminance > 0.8`) when set to `'light'`.
- Dark palette `labelColor` is lighter than its `canvasBg` (white text on dark canvas).
- Light palette `labelColor` is darker than its `canvasBg` (dark text on light canvas).
- Dark and light palettes return different `canvasBg` values.
- Dark and light palettes return different `nodeColors.idea` values.

#### 2. Completeness (14 tests — 7 per palette)

For both `dark` and `light` palettes:

- All 21 top-level `GraphPalette` keys are defined and non-`undefined`.
- `nodeColors` defines all 11 artifact types: `idea`, `requirement`, `plan-backend`, `plan-frontend`, `plan-test`, `test`, `prototype`, `defect`, `label`, `release`, `backlog`.
- `priorityColors` defines all 4 levels: `high`, `medium`, `normal`, `low`.
- `activeStatusColors` defines all 5 statuses: `in-development`, `in-qa`, `in-progress`, `clarifying`, `planning`.
- `edgeColors` defines all 5 edge kinds: `parent`, `depends_on`, `blocks`, `related_to`, `label`.
- `approvedTestRingColor` is a 6-digit hex string.
- `canvasBg` is a 6-digit hex string.

#### 3. WCAG AA contrast (58 tests — 29 per palette)

**Text pairs — ≥ 4.5:1 (normal text, WCAG AA):**

| Foreground | Background | Rationale |
|---|---|---|
| `labelColor` | `canvasBg` | Node label text below nodes |
| `edgeLabelText` | `edgeLabelBg` | Edge label pill text |
| `labelNodeText` | `labelNodeBg` | Purple pill node text |
| `backlogText` | `canvasBg` | Backlog node label |
| `timelineEdgeTextColor` | `canvasBg` | Timeline duration label |

**Graphical object pairs — ≥ 3:1 (WCAG AA, graphical objects):**

All 11 `nodeColors` values, all 5 `edgeColors` values, all 4 `priorityColors`
values, and `approvedTestRingColor` are tested against `canvasBg`.

`searchHighlight` is **excluded** from canvas-background contrast testing because
the search ring is rendered on top of a node fill rather than directly on the
canvas.  It would need a separate test against each node fill colour.

#### 4. No stale exports (6 tests)

- `NODE_COLORS` is not exported.
- `PRIORITY_COLORS` is not exported.
- `ACTIVE_STATUS_COLORS` is not exported.
- `EDGE_COLORS` is not exported.
- `APPROVED_TEST_RING_COLOR` is not exported.
- `useGraphTheme` is exported as a function.

---

## Milestone 2 — Visual regression: 2D graph (manual)

No automated Playwright suite — the project does not have browser-mode tests
configured.  Manual test procedure is defined in the test plan at
`lifecycle/test-plans/light-mode-graphs-5-test.md` § Milestone 2.

Key checks: canvas background matches app surface colour in light mode; all node
types visually distinguishable; node labels, edge lines, label pills, and priority
rings legible; theme toggle switches instantly with no layout shift; no flash of
wrong theme on first paint.

---

## Milestone 3 — Visual regression: 3D graph (manual)

No automated tests.  Manual procedure in the test plan § Milestone 3.

Key checks: light scene background; visible node spheres, labels, and edge lines;
tooltip styling; priority/approved-test/search rings visible; instant theme switch;
no first-paint flash.

---

## Milestone 4 — Performance: theme switch on 200-node graph (manual)

No automated tests.  Manual DevTools procedure in the test plan § Milestone 4.

Target: < 16 ms per frame on main thread for both 2D and 3D graph theme switch
on a ≥ 200-node graph.

---

## Milestone 5 — Code hygiene: no raw hex literals in graph components (grep)

Grep-based check, no test file.  Run:

```sh
grep -E '#[0-9a-fA-F]{3,8}' \
  web/src/components/graph/Graph2DView.vue \
  web/src/components/graph/ForceGraph3D.vue \
  web/src/components/graph/GraphLegend.vue
```

Any hits must be non-theme-dependent values (comments, or identical in both
themes).  `graphConstants.ts` is the single source of truth for all
theme-dependent colours.
