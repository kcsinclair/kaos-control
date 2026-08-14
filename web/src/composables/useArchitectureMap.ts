// SPDX-License-Identifier: AGPL-3.0-or-later

import { onMounted, ref } from 'vue'
import * as graphApi from '@/api/graph'
import { useWebSocket } from '@/composables/useWebSocket'
import type { GraphNode, GraphEdge } from '@/types/api'

/**
 * Loads the read-only architecture relationship map (FR-2/FR-3) and keeps it
 * fresh as the catalog changes on disk (FR-12). Mirrors useGraphData/api/graph
 * so ArchitectureMapView stays agnostic of the 2D/3D rendering engine.
 */
export function useArchitectureMap(project: string) {
  const nodes = ref<GraphNode[]>([])
  const edges = ref<GraphEdge[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Off by default (FR-8): the base map is architecture-only until a stack
  // reveal is explicitly requested for a selected architecture.
  const selectedArchId = ref<string | null>(null)
  const showStacks = ref(false)

  async function reload(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const stackFor = showStacks.value && selectedArchId.value ? selectedArchId.value : undefined
      const data = await graphApi.getArchitectureMap(project, stackFor)
      nodes.value = data.nodes ?? []
      edges.value = data.edges ?? []
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load architecture map'
    } finally {
      loading.value = false
    }
  }

  useWebSocket(project, 'artifact.indexed', () => {
    reload()
  })
  useWebSocket(project, 'file.changed', () => {
    reload()
  })

  onMounted(() => {
    reload()
  })

  return { nodes, edges, loading, error, selectedArchId, showStacks, reload }
}
