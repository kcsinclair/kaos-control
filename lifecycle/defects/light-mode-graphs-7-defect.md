---
title: Stale graphConstants mock breaks ForceGraph3D and Graph2DView approved-ring tests (21 failures)
type: defect
status: done
lineage: light-mode-graphs
parent: lifecycle/tests/light-mode-graphs-6-test.md
labels:
    - defect
release: May2026
assignees:
    - role: test-developer
      who: agent
---

# Stale graphConstants mock breaks ForceGraph3D and Graph2DView approved-ring tests (21 failures)

## Reproduction Steps

1. `cd tests/web`
2. `pnpm run test -- --reporter=verbose ForceGraph3D.approvedRing.test.ts Graph2DView.approvedRing.test.ts`
3. Observe 12 failures in `ForceGraph3D.approvedRing.test.ts` and 9 failures in `Graph2DView.approvedRing.test.ts`.

## Expected Behaviour

All 21 tests pass.  The graphConstants mock provides a `useGraphTheme` function that returns a palette and `isDark` computed ref, matching the current module API.

## Actual Behaviour

Every test fails immediately with:

```
[vitest] No "useGraphTheme" export is defined on the "@/components/graph/graphConstants" mock.
Did you forget to return it from "vi.mock"?
```

Both test files mock `@/components/graph/graphConstants` using the old pre-refactor API — exporting bare constants (`NODE_COLORS`, `PRIORITY_COLORS`, `ACTIVE_STATUS_COLORS`, `EDGE_COLORS`, `APPROVED_TEST_RING_COLOR`).  The light-mode-graphs feature replaced those bare exports with a single `useGraphTheme()` composable.  `ForceGraph3D.vue` and `Graph2DView.vue` now call `useGraphTheme()` at the top of `<script setup>`, so mounting the component with the stale mock throws immediately.

## Logs / Output

```
ForceGraph3D.approvedRing.test.ts  (12 tests | 12 failed)
  ❯ ForceGraph3D — approved-test torus ring > buildNodeObject returns a group …
    → [vitest] No "useGraphTheme" export is defined on the "@/components/graph/graphConstants" mock.
  … (11 more identical errors)

Graph2DView.approvedRing.test.ts  (9 tests | 9 failed)
  ❯ Graph2DView — approved-test Cytoscape style rule > style array includes the approved-test selector
    → [vitest] No "useGraphTheme" export is defined on the "@/components/graph/graphConstants" mock.
  … (8 more identical errors)
```

## Fix Guidance

Replace the `vi.mock('@/components/graph/graphConstants', () => ({ ... }))` factory in **both** test files.  The new factory must export `useGraphTheme` as a function that returns:

```ts
{
  palette: computed(() => ({
    nodeColors: { idea: '#...', /* all 11 types */ },
    priorityColors: { high: '#ef4444', medium: '#f97316', normal: '#22c55e', low: '#3b82f6' },
    activeStatusColors: { 'in-development': '#4ade80', 'in-qa': '#fbbf24', 'in-progress': '#4ade80', clarifying: '#60a5fa', planning: '#a78bfa', /* include 'approved' for pulse-loop tests */ },
    edgeColors: { parent: '#94a3b8', depends_on: '#f97316', blocks: '#ef4444', related_to: '#64748b', label: '#a855f7' },
    approvedTestRingColor: APPROVED_TEST_RING_COLOR,
    canvasBg: '#0f172a',
    labelColor: '#f1f5f9',
    labelNodeBg: '#2e1a4a', labelNodeText: '#d8b4fe', labelNodeBorder: '#a855f7',
    releaseText: '#1e3a5f', releaseBorderColor: '#60a5fa',
    backlogText: '#d1d5db',
    edgeLabelBg: '#1e293b', edgeLabelText: '#94a3b8',
    timelineEdgeColor: '#3b82f6', timelineEdgeTextColor: '#93c5fd',
    assignedEdgeColor: '#334155',
    borderDefault: 'rgba(255,255,255,0.25)', selectedBorderColor: '#ffffff',
    searchHighlight: '#facc15',
    dimBlend: '#1e2535',
  })),
  isDark: ref(true),
}
```

The old bare-constant exports can be removed from the mock entirely.
