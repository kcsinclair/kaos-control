<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import ForceGraph3D from '3d-force-graph'
import * as THREE from 'three'
import type { GraphNode, GraphEdge } from '@/types/api'
import { useGraphTheme } from './graphConstants'

const props = defineProps<{
  nodes: GraphNode[]
  edges: GraphEdge[]
  matchedNodeIds?: Set<string>
  /** When set, applies a DAG layout mode (e.g. 'lr' for left-to-right chains) */
  dagMode?: string
}>()

const emit = defineEmits<{
  nodeClick: [node: GraphNode]
}>()

const container = ref<HTMLElement>()
const { palette, isDark } = useGraphTheme()

function nodeColor(n: GraphNode): string {
  const matched = props.matchedNodeIds
  if (matched && matched.size > 0 && !matched.has(n.id)) {
    return palette.value.dimBlend
  }
  const p = palette.value
  if (n.type === 'release') {
    return n.synthetic ? p.nodeColors['backlog'] : p.nodeColors['release']
  }
  return p.nodeColors[n.type] ?? '#6b7280'
}

function edgeColor(e: GraphEdge): string {
  return palette.value.edgeColors[e.kind] ?? palette.value.edgeColors['related_to'] ?? '#64748b'
}

function nodeVal(n: GraphNode): number {
  return Math.max(1, 4 - n.index * 0.3)
}

// Canvas-based text sprite — used for label nodes so their name is always visible.
function textSprite(text: string, color = '#e9d5ff'): THREE.Sprite {
  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d')!
  const fontSize = 26
  ctx.font = `${fontSize}px sans-serif`
  const textW = Math.ceil(ctx.measureText(text).width)
  canvas.width = textW + 20
  canvas.height = fontSize + 14
  ctx.font = `${fontSize}px sans-serif`
  ctx.fillStyle = color
  ctx.textBaseline = 'middle'
  ctx.fillText(text, 10, canvas.height / 2)
  const texture = new THREE.CanvasTexture(canvas)
  const mat = new THREE.SpriteMaterial({ map: texture, depthWrite: false })
  const sprite = new THREE.Sprite(mat)
  sprite.scale.set(canvas.width / 5, canvas.height / 5, 1)
  sprite.position.set(0, 9, 0)
  return sprite
}

// Build a torus ring for nodes that have a priority colour.
// 3d-force-graph uses Math.cbrt(nodeVal) * 4 as the sphere radius.
function priorityRing(n: GraphNode): THREE.Mesh | null {
  if (!n.priority && n.status !== 'done') return null
  const color = n.status === 'done'
    ? '#6b7280'
    : (palette.value.priorityColors[n.priority!] ?? '#6b7280')
  const sphereR = Math.cbrt(nodeVal(n)) * 4
  const torusR = sphereR * 1.45
  const tubeR = sphereR * 0.18
  const geo = new THREE.TorusGeometry(torusR, tubeR, 8, 20)
  const mat = new THREE.MeshLambertMaterial({ color })
  return new THREE.Mesh(geo, mat)
}

// Build a box mesh for release nodes (replaces the default sphere via nodeThreeObjectExtend=false).
function buildReleaseObject(n: GraphNode): THREE.Object3D {
  const group = new THREE.Group()
  const size = 7  // consistent with ~nodeVal 4 sphere (radius ≈ Math.cbrt(4)*4 ≈ 6.3)
  const p = palette.value
  const color = n.synthetic ? p.nodeColors['backlog'] : p.nodeColors['release']

  if (n.synthetic) {
    // Backlog: octahedron for visual distinction
    const geo = new THREE.OctahedronGeometry(size * 0.8)
    const mat = new THREE.MeshLambertMaterial({ color })
    group.add(new THREE.Mesh(geo, mat))
  } else {
    // Regular release: box geometry
    const geo = new THREE.BoxGeometry(size, size * 0.65, size * 0.65)
    const mat = new THREE.MeshLambertMaterial({ color })
    group.add(new THREE.Mesh(geo, mat))
  }

  // Always show name label for release nodes
  const labelText = n.synthetic ? p.backlogText : p.releaseText
  // Use a light color against the node background; for dark the palette releaseText
  // is dark-blue (#1e3a5f) which contrasts on the light-blue node — keep it.
  // For the synthetic backlog node use the backlogText colour.
  group.add(textSprite(n.title || n.slug, n.synthetic ? p.backlogText : '#bfdbfe'))
  return group
}

// Build the Three.js object for a node: priority ring, approved-test ring, active pulse ring, text sprite.
function buildNodeObject(n: GraphNode): THREE.Object3D {
  // Release nodes use custom geometry (this function is only called when
  // nodeThreeObjectExtend returns true, i.e. for non-release nodes).
  const group = new THREE.Group()
  const p = palette.value
  const ring = priorityRing(n)
  if (ring) group.add(ring)
  // Static ring for approved test artifacts
  if (n.type === 'test' && n.status === 'approved') {
    const sphereR = Math.cbrt(nodeVal(n)) * 4
    // Use a larger radius when a priority ring is also present so both are visible
    const hasPriority = !!(n.priority || n.status === 'done')
    const torusR = hasPriority ? sphereR * 1.75 : sphereR * 1.45
    const tubeR = sphereR * 0.18
    const geo = new THREE.TorusGeometry(torusR, tubeR, 8, 20)
    const mat = new THREE.MeshLambertMaterial({ color: p.approvedTestRingColor })
    group.add(new THREE.Mesh(geo, mat))
  }
  const activeColor = p.activeStatusColors[n.status]
  if (activeColor) {
    const r = Math.cbrt(nodeVal(n)) * 4
    const geo = new THREE.TorusGeometry(r * 1.85, r * 0.15, 8, 24)
    const mat = new THREE.MeshLambertMaterial({ color: activeColor, transparent: true, opacity: 0.55 })
    const mesh = new THREE.Mesh(geo, mat)
    activeRings.set(n.id, mesh)
    group.add(mesh)
  }
  if (n.type === 'label') group.add(textSprite(n.title || n.slug, p.labelNodeText))
  // Highlight ring for text-filter matches
  const matched = props.matchedNodeIds
  if (matched && matched.size > 0 && matched.has(n.id)) {
    const r = Math.cbrt(nodeVal(n)) * 4
    const geo = new THREE.TorusGeometry(r * 2.1, r * 0.2, 8, 24)
    const mat = new THREE.MeshLambertMaterial({ color: p.searchHighlight, transparent: true, opacity: 0.85 })
    group.add(new THREE.Mesh(geo, mat))
  }
  return group
}

// Non-reactive: the library owns these references
let graph: ReturnType<typeof ForceGraph3D> | null = null
let ro: ResizeObserver | null = null
const activeRings = new Map<string, THREE.Mesh>()

function buildGraphData() {
  // Spread to plain objects so the library can augment them (x/y/z) without
  // tripping over Vue's reactivity proxies.
  return {
    nodes: props.nodes.map((n) => ({ ...n })),
    links: props.edges.map((e) => ({ ...e })),
  }
}

function tooltipHtml(node: GraphNode): string {
  const p = palette.value
  const bg = p.edgeLabelBg
  const text = p.labelColor
  if (node.type === 'release') {
    const dateInfo = (node as any).start_date
      ? `<br/><span style="opacity:.6">${(node as any).start_date}</span>`
      : ''
    return `<div style="font:12px/1.4 sans-serif;padding:4px 8px;background:${bg};border-radius:4px;color:${text}">${node.title}<br/><span style="opacity:.6">${node.status || 'backlog'}</span>${dateInfo}</div>`
  }
  return `<div style="font:12px/1.4 sans-serif;padding:4px 8px;background:${bg};border-radius:4px;color:${text}">${node.title || node.slug}<br/><span style="opacity:.6">${node.type} · ${node.status}</span></div>`
}

function timelineLinkLabel(edge: GraphEdge): string {
  if (edge.kind === 'timeline' && edge.label) {
    const p = palette.value
    return `<div style="font:11px sans-serif;padding:2px 6px;background:${p.edgeLabelBg};border-radius:3px;color:${p.timelineEdgeTextColor}">${edge.label}</div>`
  }
  return ''
}

onMounted(() => {
  if (!container.value) return

  const p = palette.value

  graph = ForceGraph3D()(container.value)
    .nodeId('id')
    .nodeLabel((n: object) => tooltipHtml(n as GraphNode))
    .nodeColor((n: object) => nodeColor(n as GraphNode))
    .nodeVal((n: object) => nodeVal(n as GraphNode))
    .nodeThreeObjectExtend((n: object) => (n as GraphNode).type !== 'release')
    .nodeThreeObject((n: object) => {
      const node = n as GraphNode
      return node.type === 'release' ? buildReleaseObject(node) : buildNodeObject(node)
    })
    .linkSource('source')
    .linkTarget('target')
    .linkColor((l: object) => edgeColor(l as GraphEdge))
    .linkLabel((l: object) => timelineLinkLabel(l as GraphEdge))
    .linkWidth((l: object) => {
      const kind = (l as GraphEdge).kind
      if (kind === 'timeline') return 1.5
      if (kind === 'assigned') return 0.8   // membership edge — lighter than timeline
      return 0.5
    })
    .linkDirectionalArrowLength((l: object) => {
      // Assigned edges are undirected membership links — no arrow needed.
      return (l as GraphEdge).kind === 'assigned' ? 0 : 3
    })
    .linkDirectionalArrowRelPos(1)
    .linkCurvature(0.1)
    .backgroundColor(p.canvasBg)
    .showNavInfo(false)
    .onNodeClick((n: object, _event: MouseEvent) => emit('nodeClick', n as GraphNode))
    .graphData(buildGraphData())
    .onEngineTick(() => {
      const s = 1 + 0.15 * Math.sin(Date.now() / 500)
      activeRings.forEach((mesh) => mesh.scale.setScalar(s))
    })

  if (props.dagMode) {
    graph.dagMode(props.dagMode as any)
  }

  // Fit camera after initial layout settles
  setTimeout(() => graph?.zoomToFit(400, 80), 1000)

  // Keep canvas filling its container
  ro = new ResizeObserver(() => {
    if (container.value) {
      graph?.width(container.value.clientWidth).height(container.value.clientHeight)
    }
  })
  ro.observe(container.value)
})

onUnmounted(() => {
  ro?.disconnect()
  graph?._destructor()
  graph = null
  activeRings.clear()
})

// Refresh graph data when props change (filters applied upstream)
watch(
  () => [props.nodes, props.edges],
  () => graph?.graphData(buildGraphData()),
  { deep: false },
)

// Animate camera to centroid of matched nodes when text filter changes
watch(
  () => props.matchedNodeIds,
  (matched) => {
    if (!graph || !matched || matched.size === 0) return
    const data = graph.graphData() as { nodes: Array<GraphNode & { x?: number; y?: number; z?: number }> }
    const hits = data.nodes.filter((n) => matched.has(n.id))
    if (hits.length === 0) return
    const cx = hits.reduce((s, n) => s + (n.x ?? 0), 0) / hits.length
    const cy = hits.reduce((s, n) => s + (n.y ?? 0), 0) / hits.length
    const cz = hits.reduce((s, n) => s + (n.z ?? 0), 0) / hits.length
    graph.cameraPosition({ x: cx, y: cy, z: cz + 200 }, { x: cx, y: cy, z: cz }, 600)
  },
)

// Reactively update Three.js scene when theme changes — no force layout rebuild
watch(isDark, () => {
  if (!graph) return
  const p = palette.value
  // Update scene background
  graph.backgroundColor(p.canvasBg)
  // Refresh sphere colours (Three.js materials don't bind reactively)
  graph.nodeColor((n: object) => nodeColor(n as GraphNode))
  // Rebuild Three.js objects (sprites + rings) with new palette colours.
  // nodeThreeObject is O(n) with lightweight geometry and does NOT restart the force simulation.
  graph.nodeThreeObject((n: object) => {
    const node = n as GraphNode
    return node.type === 'release' ? buildReleaseObject(node) : buildNodeObject(node)
  })
  // Refresh link colours
  graph.linkColor((l: object) => edgeColor(l as GraphEdge))
  // Refresh tooltip and link label callbacks so subsequent hovers use new palette
  graph.nodeLabel((n: object) => tooltipHtml(n as GraphNode))
  graph.linkLabel((l: object) => timelineLinkLabel(l as GraphEdge))
})
</script>

<template>
  <div ref="container" class="force-graph-container" />
</template>

<style scoped>
.force-graph-container {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
}
</style>
