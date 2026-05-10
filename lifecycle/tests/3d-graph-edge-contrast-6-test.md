---
title: "Tests: 3D Graph Edge Contrast"
type: test
status: in-qa
lineage: 3d-graph-edge-contrast
parent: lifecycle/test-plans/3d-graph-edge-contrast-5-test.md
created: "2026-05-10T00:00:00+10:00"
---

# Tests: 3D Graph Edge Contrast

Companion artifact documenting the test suite built for the
[[3d-graph-edge-contrast]] feature.

---

## Test Files

### Web component tests (Vitest / Vue Test Utils)

#### `tests/web/3d-graph-edge-contrast.graphConstants.test.ts`

Covers **Milestones 1 & 2** of the test plan.

Pure-computation tests against `web/src/components/map/graphConstants.ts`.
No DOM or canvas required — Pinia is activated, the theme is set, and assertions
are made directly on the resolved `GraphPalette` values.

**Milestone 1 — Contrast ratio verification:**
- Every `edgeColors` entry (all 7 kinds: `parent`, `depends_on`, `blocks`,
  `related_to`, `label`, `timeline`, `assigned`) is tested against the palette's
  `canvasBg` in both dark and light themes.
- Minimum threshold: ≥ 3:1 (WCAG graphical-object standard).
- Explicit named assertions for the two colours that motivated the feature:
  - Dark `assigned` (`#475569`) vs dark canvas (`#0f172a`)
  - Light `assigned` (`#64748b`) vs light canvas (`#ffffff`)
  - `timeline` in both themes

**Milestone 2 — Palette consistency:**
- Asserts `palette.timelineEdgeColor === palette.edgeColors['timeline']` in both
  dark and light palettes.
- Asserts `palette.assignedEdgeColor === palette.edgeColors['assigned']` in both
  dark and light palettes.
- Asserts both properties are defined in both palettes.

---

#### `tests/web/3d-graph-edge-contrast.linkWidth.test.ts`

Covers **Milestone 3** of the test plan.

Component-level tests for `ForceGraph3D.vue` running under happy-dom. The
`3d-force-graph` library is mocked (recording `.linkWidth()` call arguments) and
`three` is replaced with lightweight stand-ins. The captured `linkWidth` callback
is called directly with controlled edge fixtures to verify the hierarchy.

**Scenarios covered:**

| Test | Assertion |
|---|---|
| Callback registered | `linkWidth` callback is a function after mount |
| `timeline` minimum | `linkWidth({kind:'timeline'}) >= 2.0` |
| `parent` minimum | `linkWidth({kind:'parent'}) >= 1.0` |
| `depends_on` minimum | `linkWidth({kind:'depends_on'}) >= 1.0` |
| `blocks` minimum | `linkWidth({kind:'blocks'}) >= 1.0` |
| `related_to` minimum | `linkWidth({kind:'related_to'}) >= 1.0` |
| `assigned` minimum | `linkWidth({kind:'assigned'}) >= 0.6` |
| Hierarchy: timeline > parent > assigned | All three compared in one assertion |
| Hierarchy: timeline > depends_on > assigned | Same for depends_on branch |
| Semantic edges share default width | parent/depends_on/blocks/related_to/label all return equal widths |
| Unknown kind returns positive | Graceful fallback — does not throw or return ≤ 0 |

---

### Manual verification script

#### `tests/manual/3d-graph-edge-contrast.md`

Covers **Milestone 4** of the test plan — 8 manual steps for QA to verify
visual correctness on a live dev server:

1. Graph loads in dark theme without errors
2. All edges visible in dark theme (per-kind visual check)
3. All edges visible in light theme (per-kind visual check)
4. Timeline edges are visually most prominent (thickest, fully opaque)
5. Assigned edges are visually least prominent (thinnest, slightly transparent)
6. Theme toggle updates edge colours immediately, no page reload or jitter
7. Node colours, sizes, and label rendering are unaffected
8. Force simulation does not restart on theme toggle

---

## Implementation notes

### Path fix: `components/graph/` → `components/map/`

The following existing test files imported from a stale `components/graph/`
path (the directory was renamed as part of the broader map/graph rename).
They were updated as part of this ticket to restore passing status:

- `tests/web/graphConstants.test.ts`
- `tests/web/ForceGraph3D.approvedRing.test.ts`
- `tests/web/Graph2DView.approvedRing.test.ts`
- `tests/web/Graph2DView.filters.spec.ts`
- `tests/web/Graph2DView.layout.spec.ts`
- `tests/web/Graph2DView.perf.spec.ts`
- `tests/web/graph-show-tests-toggle.test.ts`
- `tests/web/LayoutSelector.spec.ts`

### Milestone 5 — Performance spot-check

No automated test is created for Milestone 5. FPS regression is verified
manually via Chrome DevTools performance panel (baseline on `main` branch vs
feature branch with `linkOpacity` applied). The manual procedure is described
in the test plan.

## Known failures (implementation gap)

The test suite correctly identifies one contrast violation in the current
implementation:

**Dark palette `assigned` edge `#475569` achieves 2.36:1 on dark canvas `#0f172a`**
(required: ≥ 3:1)

The `graphConstants.ts` comment says "~3.2:1" but the computed WCAG luminance
value is 2.36:1. A lighter shade (e.g. `#64748b`, which achieves ~4.6:1 on
`#0f172a`) is needed. This is a genuine implementation defect that the test
suite is designed to catch.

Tests affected:
- `edgeColors["assigned"] achieves >= 3:1 contrast on canvasBg` (dark palette)
- `dark palette: assigned edge (#475569) achieves >= 3:1 on dark canvas (#0f172a)`

These tests will pass once the dark `assignedEdgeColor` / `edgeColors.assigned`
is updated to a shade with sufficient contrast.

## Deferred / Out of scope

- E2E browser tests (Playwright) for visual rendering of edges — requires a
  real WebGL context.
- Performance regression automation — FPS measurement is inherently environment-
  dependent and is left as a manual DevTools check.
