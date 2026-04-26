# Active Node Visualization in 3D Graph

## Context

When an agent starts work on an artifact its status changes to `in-development` or `in-qa`. The 3D (and 2D) graph has no visual distinction for these "in-flight" states today. Nodes look the same whether they're `draft`, `in-development`, or `done`. The goal is to make it immediately obvious which artifacts are actively being worked on.

## Library capabilities (3d-force-graph v1.80.0 + THREE.js 0.184.0)

The current node rendering pipeline:
- **Sphere** — color from `nodeColor()` (encodes artifact type), size from `nodeVal()` (encodes lineage index)
- **Priority torus ring** — static `THREE.TorusGeometry` + `THREE.MeshLambertMaterial` added via `nodeThreeObjectExtend`
- **Text sprite** — canvas `THREE.Sprite` for synthetic label nodes only
- **`onEngineTick`** — per-frame callback, available but **not yet used**; this is the key primitive for animation

## Options considered

### A — Pulsing scale ring (recommended)
A second torus around active nodes that breathes in and out using `onEngineTick`.

```
scale = 1 + 0.15 * Math.sin(Date.now() / 500)
mesh.scale.setScalar(scale)
```

- Looks like a "heartbeat" — clearly alive
- Color by status: `in-development` → green `#4ade80`, `in-qa` → amber `#fbbf24`
- Semi-transparent (`opacity: 0.55, transparent: true`) so it doesn't overwhelm
- Slightly larger radius than the priority ring so the two don't overlap

### B — Spinning orbit ring
A torus on a tilted axis that continuously rotates (`mesh.rotation.z += 0.025` per tick).

- Looks like a planetary orbit
- More visually noisy; clashes with the static priority ring on the same axis

### C — Node sphere colour override
Change `nodeColor` to return a status-specific colour for active nodes, overriding type colour.

- Simplest to implement
- Loses the type colour — you can no longer tell what kind of artifact is being worked on

---

**Recommendation: Option A.** It adds a new visual layer without replacing any existing information, and the pulsing motion makes active nodes immediately stand out in a busy graph.

## What needs to change

### 1. `web/src/components/graph/graphConstants.ts`
Add the active-status colour map:
```ts
export const ACTIVE_STATUS_COLORS: Record<string, string> = {
  'in-development': '#4ade80',  // green
  'in-qa':          '#fbbf24',  // amber
  'in-progress':    '#4ade80',  // green (future-proofing)
}
```

### 2. `web/src/components/graph/ForceGraph3D.vue`

**Add active ring mesh tracking** (module-level, cleared on unmount):
```ts
const activeRings = new Map<string, THREE.Mesh>()
```

**Extend `buildNodeObject(n)`**: if `ACTIVE_STATUS_COLORS[n.status]` exists, create an active ring and register it:
```ts
if (ACTIVE_STATUS_COLORS[n.status]) {
  const color = ACTIVE_STATUS_COLORS[n.status]
  const r = Math.cbrt(nodeVal(n)) * 4
  const geo = new THREE.TorusGeometry(r * 1.85, r * 0.15, 8, 24)
  const mat = new THREE.MeshLambertMaterial({ color, transparent: true, opacity: 0.55 })
  const ring = new THREE.Mesh(geo, mat)
  activeRings.set(n.id, ring)
  group.add(ring)
}
```

**Wire `onEngineTick`** immediately after `graph` is constructed in `onMounted`:
```ts
graph.onEngineTick(() => {
  const s = 1 + 0.15 * Math.sin(Date.now() / 500)
  activeRings.forEach(mesh => mesh.scale.setScalar(s))
})
```

**Clean up** in `onUnmounted`:
```ts
activeRings.clear()
```

### 3. `web/src/components/graph/Graph2DView.vue`
Cytoscape doesn't have a native animation loop exposed the same way, but supports CSS `transition` and periodic `style()` updates. Two approaches:
- **Simple (recommended)**: add a pulsing CSS `border-width` animation via a `setInterval` that cycles between two Cytoscape `style()` calls (thick → thin → thick, ~800 ms period) for nodes with active statuses.
- **Alternative**: use a `box-shadow`-style approach via `outline-opacity` (Cytoscape supports animating style properties via `node.animate()`).

The recommended approach for 2D:
```ts
// In onMounted, after cy is built:
const ACTIVE = new Set(['in-development', 'in-qa', 'in-progress'])
let tick = false
setInterval(() => {
  tick = !tick
  cy?.nodes().forEach(n => {
    if (ACTIVE.has(n.data('status'))) {
      n.style('border-width', tick ? 5 : 2)
    }
  })
}, 700)
```
Clean up the interval in `onUnmounted`.

## Files to modify

| File | Change |
|---|---|
| `web/src/components/graph/graphConstants.ts` | Add `ACTIVE_STATUS_COLORS` |
| `web/src/components/graph/ForceGraph3D.vue` | `activeRings` map; active ring in `buildNodeObject`; `onEngineTick` pulse |
| `web/src/components/graph/Graph2DView.vue` | Pulsing border-width interval for active Cytoscape nodes |

## Verification

1. `pnpm exec vue-tsc --noEmit` — clean.
2. Open a project with at least one artifact whose status is `in-development` or `in-qa`.
3. **3D view**: confirm the active node has a second (larger) pulsing ring distinct from the priority ring.
4. **2D view**: confirm the active node's border pulses visibly.
5. Confirm non-active nodes show no pulsing ring.
6. Confirm priority ring and type colour are unaffected on active nodes.
