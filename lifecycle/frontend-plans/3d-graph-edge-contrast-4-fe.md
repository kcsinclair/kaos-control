---
title: "Frontend Plan: Improve Edge Line Contrast in 3D Graph"
type: plan-frontend
status: approved
lineage: 3d-graph-edge-contrast
parent: lifecycle/requirements/3d-graph-edge-contrast-2.md
---

# Frontend Plan: Improve Edge Line Contrast in 3D Graph

## Summary

Update edge colours in both dark and light palettes to meet WCAG 3:1 contrast, increase link widths to establish visual hierarchy, and add `linkOpacity` for non-timeline edges. All changes are confined to `web/src/components/graph/graphConstants.ts` and `web/src/components/graph/ForceGraph3D.vue`.

---

## Milestone 1: Update Edge Colours in `graphConstants.ts`

### Description

Adjust `edgeColors` entries and corresponding named properties (`assignedEdgeColor`, `timelineEdgeColor`) in both `DARK_PALETTE` and `LIGHT_PALETTE` to meet the minimum 3:1 contrast ratio against their respective canvas backgrounds.

### Files to Change

- `web/src/components/graph/graphConstants.ts`

### Changes — Dark Palette (canvas `#0f172a`)

| Edge kind    | Current       | New           | Contrast vs canvas |
|-------------|---------------|---------------|--------------------|
| `related_to`| `#64748b`     | `#94a3b8`     | ~4.5:1             |
| `assigned`  | `#334155`     | `#475569`     | ~3.2:1             |
| `parent`    | `#94a3b8`     | no change     | ~4.5:1 (pass)      |

Update `assignedEdgeColor` to match `edgeColors.assigned` (`#475569`).

### Changes — Light Palette (canvas `#ffffff`)

| Edge kind    | Current       | New           | Contrast vs canvas |
|-------------|---------------|---------------|--------------------|
| `assigned`  | `#94a3b8`     | `#64748b`     | ~4.6:1             |
| `related_to`| `#475569`     | no change     | ~5.5:1 (pass)      |

Update `assignedEdgeColor` to match `edgeColors.assigned` (`#64748b`).

### Acceptance Criteria

- [ ] Every edge kind in dark theme has >= 3:1 contrast against `#0f172a`.
- [ ] Every edge kind in light theme has >= 3:1 contrast against `#ffffff`.
- [ ] `edgeColors.assigned` === `assignedEdgeColor` within each palette.
- [ ] `edgeColors.timeline` === `timelineEdgeColor` within each palette.

---

## Milestone 2: Increase Link Widths in `ForceGraph3D.vue`

### Description

Replace the current `linkWidth` callback with increased values that establish the hierarchy: `timeline` > semantic edges > `assigned`.

### Files to Change

- `web/src/components/graph/ForceGraph3D.vue`

### Implementation

Replace the `.linkWidth(...)` callback (currently lines 203-208) with:

```typescript
.linkWidth((l: object) => {
  const kind = (l as GraphEdge).kind
  if (kind === 'timeline') return 2.0
  if (kind === 'assigned') return 0.8
  return 1.2  // parent, depends_on, blocks, related_to, label
})
```

### Acceptance Criteria

- [ ] `timeline` edges render at width >= 2.0.
- [ ] `parent`, `depends_on`, `blocks`, `related_to`, `label` edges render at width >= 1.0.
- [ ] `assigned` edges render at width >= 0.6.
- [ ] Visual hierarchy: timeline > semantic > assigned.

---

## Milestone 3: Add Link Opacity

### Description

Apply `linkOpacity` to reduce visual clutter from dense edge bundles while keeping timeline edges fully opaque.

### Files to Change

- `web/src/components/graph/ForceGraph3D.vue`

### Implementation

Add a `.linkOpacity(...)` call on the graph builder:

```typescript
.linkOpacity((l: object) => {
  return (l as GraphEdge).kind === 'timeline' ? 1.0 : 0.75
})
```

If 3d-force-graph does not support per-link opacity via callback, use a global `.linkOpacity(0.75)` and keep timeline edges distinguishable via their greater width and colour. Alternatively, encode opacity in the link colour via RGBA hex (append alpha channel to the colour string returned by `edgeColor()`).

### Acceptance Criteria

- [ ] Non-timeline edges rendered at ~0.75 opacity (visually softer than timeline).
- [ ] Timeline edges remain fully opaque (or near-opaque at >= 0.95).
- [ ] No measurable FPS regression on graphs with <= 500 edges.

---

## Milestone 4: Theme Reactivity for New Properties

### Description

Ensure the `watch(isDark, ...)` handler in `ForceGraph3D.vue` refreshes `linkWidth` and `linkOpacity` alongside `linkColor` when the theme toggles.

### Files to Change

- `web/src/components/graph/ForceGraph3D.vue`

### Implementation

Inside the existing `watch(isDark, () => { ... })` block (line 270), add after the `graph.linkColor(...)` call:

```typescript
graph.linkWidth((l: object) => {
  const kind = (l as GraphEdge).kind
  if (kind === 'timeline') return 2.0
  if (kind === 'assigned') return 0.8
  return 1.2
})
graph.linkOpacity((l: object) => {
  return (l as GraphEdge).kind === 'timeline' ? 1.0 : 0.75
})
```

Note: `linkWidth` values are theme-independent but re-applying the callback ensures the library picks up any future theme-dependent widths.

### Acceptance Criteria

- [ ] Toggling dark/light theme updates edge colours immediately without page reload.
- [ ] Edge widths and opacity remain correct after theme toggle.
- [ ] No visible layout jitter or force restart on toggle.

---

## Cross-references

- [[3d-graph-edge-contrast]] — test plan covers visual verification.
- Backend plan confirms no API changes needed.
