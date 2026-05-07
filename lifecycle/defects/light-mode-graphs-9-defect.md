---
title: ForceGraph3D.vue hardcodes #bfdbfe for release-node label — invisible in light mode
type: defect
status: in-development
lineage: light-mode-graphs
parent: lifecycle/tests/light-mode-graphs-6-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
---

# ForceGraph3D.vue hardcodes #bfdbfe for release-node label — invisible in light mode

## Reproduction Steps

1. Run the Milestone 5 grep check:
   ```sh
   grep -En '#[0-9a-fA-F]{3,8}' \
     web/src/components/graph/Graph2DView.vue \
     web/src/components/graph/ForceGraph3D.vue \
     web/src/components/graph/GraphLegend.vue
   ```
2. Observe the hit at `ForceGraph3D.vue:108`:
   ```
   web/src/components/graph/ForceGraph3D.vue:108:  group.add(textSprite(n.title || n.slug, n.synthetic ? p.backlogText : '#bfdbfe'))
   ```
3. Switch the app to light mode and navigate to the 3D graph view.
4. Observe that release node title labels are rendered in `#bfdbfe` (blue-200) against a `#ffffff` canvas — contrast ratio ≈ 1.24:1, making labels effectively invisible.

## Expected Behaviour

Release node text labels in the 3D graph are legible in both dark and light modes.  The colour used for the label sprite must be drawn from the `GraphPalette` so it adapts to the active theme.

## Actual Behaviour

`buildReleaseObject()` passes the hardcoded hex literal `'#bfdbfe'` as the text colour to `textSprite()` for non-synthetic release nodes.  `#bfdbfe` is `blue-200` — a very pale blue that is clearly readable against the dark canvas (`#0f172a`) but has a contrast ratio of approximately **1.24:1** against the light canvas (`#ffffff`), failing WCAG AA (3:1 minimum for graphical text) by a wide margin.

The `GraphPalette` already defines `releaseText: '#1e3a5f'` for both dark and light palettes (same dark-blue value), which was intended for this exact use-case.  The code bypasses it.

## Logs / Output

```
web/src/components/graph/ForceGraph3D.vue:108:  group.add(textSprite(n.title || n.slug, n.synthetic ? p.backlogText : '#bfdbfe'))
```

Contrast check — `#bfdbfe` on `#ffffff`:
- Luminance(`#bfdbfe`) ≈ 0.693
- Luminance(`#ffffff`) = 1.0
- Contrast ratio = (1.0 + 0.05) / (0.693 + 0.05) ≈ **1.41:1**  (fails 3:1 and 4.5:1)

## Fix Guidance

Replace the hardcoded `'#bfdbfe'` with `p.releaseText`:

```ts
// before
group.add(textSprite(n.title || n.slug, n.synthetic ? p.backlogText : '#bfdbfe'))

// after
group.add(textSprite(n.title || n.slug, n.synthetic ? p.backlogText : p.releaseText))
```

`p.releaseText` is already `'#1e3a5f'` in both palettes, providing adequate contrast against both the light-blue release node fill and (via the sprite's transparent background) the canvas.
