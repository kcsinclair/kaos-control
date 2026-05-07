<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import type { GraphNode, GraphEdge } from '@/types/api'
import { NODE_COLORS, PRIORITY_COLORS, ACTIVE_STATUS_COLORS, APPROVED_TEST_RING_COLOR, EDGE_COLORS } from './graphConstants'

const props = defineProps<{
  nodes: GraphNode[]
  edges: GraphEdge[]
  onNodeClick: (node: GraphNode) => void
  matchedNodeIds?: Set<string>
  /** Use left-to-right breadthfirst layout instead of fcose (for directed chains) */
  directed?: boolean
}>()

const container = ref<HTMLDivElement | null>(null)
let cy: any = null
let pulseInterval: ReturnType<typeof setInterval> | null = null
let pulseTick = false

function nodeColor(type: string, synthetic?: boolean): string {
  if (type === 'release') {
    return synthetic ? NODE_COLORS['backlog'] : NODE_COLORS['release']
  }
  return NODE_COLORS[type] ?? '#6b7280'
}

function buildElements() {
  const nodes = props.nodes.map((n) => ({
    data: {
      id: n.id,
      label: n.title || n.slug,
      type: n.type,
      status: n.status,
      // 'synthetic' stored as string so Cytoscape attribute selectors work
      synthetic: n.synthetic ? 'true' : 'false',
      color: nodeColor(n.type, n.synthetic),
      priorityColor: (n.priority || n.status === 'done')
        ? (n.status === 'done' ? '#6b7280' : (PRIORITY_COLORS[n.priority!] ?? '#6b7280'))
        : null,
      _raw: n,
    },
  }))
  const edges = props.edges.map((e, i) => ({
    data: {
      id: `e${i}`,
      source: e.source,
      target: e.target,
      kind: e.kind,
      // Show duration for timeline edges; hide kind label for other edges
      label: e.kind === 'timeline' && e.label ? e.label : '',
    },
  }))
  return [...nodes, ...edges]
}

async function init() {
  if (!container.value) return
  const [cytoscape, fcose] = await Promise.all([
    import('cytoscape'),
    import('cytoscape-fcose'),
  ])
  const Cy = cytoscape.default
  Cy.use(fcose.default)

  cy = Cy({
    container: container.value,
    elements: buildElements(),
    style: [
      {
        selector: 'node',
        style: {
          'background-color': 'data(color)',
          label: 'data(label)',
          color: '#f1f5f9',
          'font-size': 11,
          'text-valign': 'bottom',
          'text-halign': 'center',
          'text-margin-y': 4,
          width: 28,
          height: 28,
          'text-wrap': 'ellipsis',
          'text-max-width': 100,
          'border-width': 1.5,
          'border-color': 'rgba(255,255,255,0.25)',
          'overlay-padding': 4,
        },
      },
      {
        selector: 'node:selected',
        style: {
          'border-width': 3,
          'border-color': '#ffffff',
        },
      },
      {
        // Nodes with a priority colour get a thicker coloured border (ring effect)
        selector: 'node[priorityColor]',
        style: {
          'border-width': 4,
          'border-color': 'data(priorityColor)',
        },
      },
      {
        // Approved test artifacts get a static blue ring (overrides priority ring)
        selector: 'node[type="test"][status="approved"]',
        style: {
          'border-width': 4,
          'border-color': APPROVED_TEST_RING_COLOR,
        },
      },
      {
        // Label nodes: pill-shaped tag with centred text, auto-width to fit label
        selector: 'node[type="label"]',
        style: {
          shape: 'round-rectangle',
          width: 'label',
          height: 20,
          padding: '8px',
          'background-color': '#2e1a4a',
          'border-color': '#a855f7',
          'border-width': 1.5,
          'text-valign': 'center',
          'text-halign': 'center',
          color: '#d8b4fe',
          'font-size': 10,
          'font-weight': 'bold',
          'text-max-width': 200,
        },
      },
      {
        // Release nodes: rounded rectangle, light blue
        selector: 'node[type="release"]',
        style: {
          shape: 'round-rectangle',
          width: 'label',
          height: 24,
          padding: '10px',
          'background-color': NODE_COLORS['release'],
          'border-color': '#60a5fa',
          'border-width': 1.5,
          'text-valign': 'center',
          'text-halign': 'center',
          color: '#1e3a5f',
          'font-size': 11,
          'font-weight': '600',
          'text-max-width': 160,
        },
      },
      {
        // Backlog synthetic node: diamond shape in gray
        selector: 'node[type="release"][synthetic="true"]',
        style: {
          shape: 'diamond',
          width: 36,
          height: 36,
          padding: '0px',
          'background-color': NODE_COLORS['backlog'],
          'border-color': '#9ca3af',
          'border-width': 1.5,
          'text-valign': 'bottom',
          'text-halign': 'center',
          'text-margin-y': 6,
          color: '#d1d5db',
          'font-size': 10,
          'font-weight': '600',
        },
      },
      {
        selector: 'edge',
        style: {
          width: 1.5,
          'line-color': '#475569',
          'target-arrow-color': '#475569',
          'target-arrow-shape': 'triangle',
          'curve-style': 'bezier',
          label: 'data(label)',
          'font-size': 9,
          color: '#94a3b8',
          'text-background-color': '#1e293b',
          'text-background-opacity': 0.85,
          'text-background-padding': '2px',
        },
      },
      {
        // Timeline edges: blue directional arrows with duration label
        selector: 'edge[kind="timeline"]',
        style: {
          'line-color': '#3b82f6',
          'target-arrow-color': '#3b82f6',
          'target-arrow-shape': 'triangle',
          width: 2,
          label: 'data(label)',
          'font-size': 9,
          color: '#93c5fd',
          'text-background-color': '#1e293b',
          'text-background-opacity': 0.9,
          'text-background-padding': '2px',
        },
      },
      {
        // Assigned edges (artifact → release): lighter, no arrow, no label
        selector: 'edge[kind="assigned"]',
        style: {
          'line-color': '#334155',
          'target-arrow-color': '#334155',
          'target-arrow-shape': 'none',
          width: 1,
          label: '',
        },
      },
    ],
    layout: props.directed
      ? {
          name: 'breadthfirst',
          directed: true,
          padding: 40,
          spacingFactor: 1.6,
          avoidOverlap: true,
          animate: false,
        } as any
      : {
          name: 'fcose',
          quality: 'proof',
          randomize: true,
          animate: false,
          nodeSeparation: 120,
          idealEdgeLength: () => 80,
        } as any,
    userZoomingEnabled: true,
    userPanningEnabled: true,
    boxSelectionEnabled: false,
  })

  cy.on('tap', 'node', (evt: any) => {
    const raw = evt.target.data('_raw') as GraphNode
    props.onNodeClick(raw)
  })

  pulseInterval = setInterval(() => {
    pulseTick = !pulseTick
    cy?.nodes().forEach((n: any) => {
      // Approved test nodes keep their static blue ring — skip pulse override
      if (n.data('type') === 'test' && n.data('status') === 'approved') return
      const color = ACTIVE_STATUS_COLORS[n.data('status')]
      if (color) {
        n.style({ 'border-color': color, 'border-width': pulseTick ? 6 : 2 })
      }
    })
  }, 700)
}

function applySearchHighlight() {
  if (!cy) return
  const matched = props.matchedNodeIds
  if (!matched || matched.size === 0) {
    // Restore full opacity for all elements
    cy.nodes().style({ opacity: 1, 'border-width': undefined, 'border-color': undefined })
    cy.edges().style({ opacity: 1 })
    return
  }
  cy.nodes().forEach((n: any) => {
    const id: string = n.data('id')
    if (matched.has(id)) {
      n.style({ opacity: 1, 'border-width': 4, 'border-color': '#facc15' })
    } else {
      n.style({ opacity: 0.15 })
    }
  })
  cy.edges().forEach((e: any) => {
    const srcMatched = matched.has(e.data('source'))
    const tgtMatched = matched.has(e.data('target'))
    e.style({ opacity: srcMatched || tgtMatched ? 1 : 0.1 })
  })
  // Fit viewport to matched nodes
  const matchedEles = cy.nodes().filter((n: any) => matched.has(n.data('id')))
  if (matchedEles.length > 0) {
    cy.fit(matchedEles, 80)
  }
}

function update() {
  if (!cy) return
  cy.elements().remove()
  cy.add(buildElements())
  const layoutOpts = props.directed
    ? { name: 'breadthfirst', directed: true, padding: 40, spacingFactor: 1.6, avoidOverlap: true, animate: false } as any
    : { name: 'fcose', quality: 'proof', randomize: false, animate: false } as any
  cy.layout(layoutOpts).run()
  nextTick(applySearchHighlight)
}

watch(() => [props.nodes, props.edges], update, { deep: false })

watch(() => props.matchedNodeIds, applySearchHighlight)

onMounted(init)
onUnmounted(() => {
  if (pulseInterval !== null) clearInterval(pulseInterval)
  cy?.destroy()
})
</script>

<template>
  <div ref="container" class="graph-2d" aria-label="2D artifact graph" role="img" />
</template>

<style scoped>
.graph-2d {
  position: relative;
  width: 100%;
  height: 100%;
  background: #0f172a;
}
</style>
