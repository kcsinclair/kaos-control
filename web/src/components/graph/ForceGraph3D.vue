<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import ForceGraph3D from '3d-force-graph'
import * as THREE from 'three'
import type { GraphNode, GraphEdge } from '@/types/api'
import { NODE_COLORS, EDGE_COLORS, PRIORITY_COLORS, ACTIVE_STATUS_COLORS } from './graphConstants'

const props = defineProps<{
  nodes: GraphNode[]
  edges: GraphEdge[]
}>()

const emit = defineEmits<{
  nodeClick: [node: GraphNode]
}>()

const container = ref<HTMLElement>()

function nodeColor(n: GraphNode): string {
  return NODE_COLORS[n.type] ?? '#6b7280'
}

function edgeColor(e: GraphEdge): string {
  return EDGE_COLORS[e.kind] ?? '#64748b'
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
  const color = n.status === 'done' ? '#6b7280' : (PRIORITY_COLORS[n.priority!] ?? '#6b7280')
  const sphereR = Math.cbrt(nodeVal(n)) * 4
  const torusR = sphereR * 1.45
  const tubeR = sphereR * 0.18
  const geo = new THREE.TorusGeometry(torusR, tubeR, 8, 20)
  const mat = new THREE.MeshLambertMaterial({ color })
  return new THREE.Mesh(geo, mat)
}

// Build the Three.js object for a node: priority ring, active pulse ring, text sprite.
function buildNodeObject(n: GraphNode): THREE.Object3D {
  const group = new THREE.Group()
  const ring = priorityRing(n)
  if (ring) group.add(ring)
  const activeColor = ACTIVE_STATUS_COLORS[n.status]
  if (activeColor) {
    const r = Math.cbrt(nodeVal(n)) * 4
    const geo = new THREE.TorusGeometry(r * 1.85, r * 0.15, 8, 24)
    const mat = new THREE.MeshLambertMaterial({ color: activeColor, transparent: true, opacity: 0.55 })
    const mesh = new THREE.Mesh(geo, mat)
    activeRings.set(n.id, mesh)
    group.add(mesh)
  }
  if (n.type === 'label') group.add(textSprite(n.title || n.slug))
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

onMounted(() => {
  if (!container.value) return

  graph = ForceGraph3D()(container.value)
    .nodeId('id')
    .nodeLabel((n: object) => {
      const node = n as GraphNode
      return `<div style="font:12px/1.4 sans-serif;padding:4px 8px;background:#1e293b;border-radius:4px;color:#f1f5f9">${node.title || node.slug}<br/><span style="opacity:.6">${node.type} · ${node.status}</span></div>`
    })
    .nodeColor((n: object) => nodeColor(n as GraphNode))
    .nodeVal((n: object) => nodeVal(n as GraphNode))
    .nodeThreeObjectExtend(true)
    .nodeThreeObject((n: object) => buildNodeObject(n as GraphNode))
    .linkSource('source')
    .linkTarget('target')
    .linkColor((l: object) => edgeColor(l as GraphEdge))
    .linkWidth(0.5)
    .linkDirectionalArrowLength(3)
    .linkDirectionalArrowRelPos(1)
    .linkCurvature(0.1)
    .backgroundColor('#0f172a')
    .showNavInfo(false)
    .onNodeClick((n: object, _event: MouseEvent) => emit('nodeClick', n as GraphNode))
    .graphData(buildGraphData())
    .onEngineTick(() => {
      const s = 1 + 0.15 * Math.sin(Date.now() / 500)
      activeRings.forEach((mesh) => mesh.scale.setScalar(s))
    })

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
</script>

<template>
  <div ref="container" class="force-graph-container" />
</template>

<style scoped>
.force-graph-container {
  width: 100%;
  height: 100%;
  overflow: hidden;
}
</style>
