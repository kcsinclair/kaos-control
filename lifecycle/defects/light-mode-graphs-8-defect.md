---
title: ForceGraph3D.vue dimColor() contains theme-dependent hex literals outside graphConstants.ts
type: defect
status: done
lineage: light-mode-graphs
parent: lifecycle/tests/light-mode-graphs-6-test.md
labels:
    - defect
release: KC-Feature-Sprint
assignees:
    - role: frontend-developer
      who: agent
---

# ForceGraph3D.vue dimColor() contains theme-dependent hex literals outside graphConstants.ts

## Reproduction Steps

1. Run the Milestone 5 grep check from the test artifact:
   ```sh
   grep -En '#[0-9a-fA-F]{3,8}' \
     web/src/components/graph/Graph2DView.vue \
     web/src/components/graph/ForceGraph3D.vue \
     web/src/components/graph/GraphLegend.vue
   ```
2. Observe the hit at `ForceGraph3D.vue:25`:
   ```
   web/src/components/graph/ForceGraph3D.vue:25:  return isDark.value ? '#1e2535' : '#d1d5db'
   ```
3. Note that the two literal values differ between themes — they cannot be classified as "identical in both themes" per the test plan criterion.

## Expected Behaviour

All hex literals remaining in graph component files after the Milestone 5 check are either:
- in comments, or
- identical in both dark and light modes (truly theme-invariant).

Theme-dependent colour choices must live exclusively in `graphConstants.ts`.

## Actual Behaviour

`ForceGraph3D.vue` contains a `dimColor()` function that explicitly branches on `isDark.value` to return two different hex literals:

```ts
// web/src/components/graph/ForceGraph3D.vue:23-26
function dimColor(): string {
  return isDark.value ? '#1e2535' : '#d1d5db'
}
```

`#1e2535` is a dark navy (used to fade unmatched nodes into the dark canvas) and `#d1d5db` is `gray-300` (the light-mode equivalent).  These are unambiguously theme-dependent — the function already has access to `isDark` from `useGraphTheme()` and the canonical location for such values is the `GraphPalette` returned by that composable.

## Logs / Output

```
web/src/components/graph/ForceGraph3D.vue:25:  return isDark.value ? '#1e2535' : '#d1d5db'
```

## Fix Guidance

1. Add a `dimBlend` field to the `GraphPalette` interface in `graphConstants.ts`:
   ```ts
   /** Dim-blend colour for unmatched nodes: matches the canvas background at reduced opacity */
   dimBlend: string
   ```
2. Set `dimBlend: '#1e2535'` in `DARK_PALETTE` and `dimBlend: '#d1d5db'` in `LIGHT_PALETTE`.
3. Replace `dimColor()` in `ForceGraph3D.vue` with `palette.value.dimBlend`.
