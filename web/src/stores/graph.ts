import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as graphApi from '@/api/graph'
import type { GraphNode, GraphEdge, GraphFilter } from '@/types/api'

export const useGraphStore = defineStore('graph', () => {
  const rawNodes = ref<GraphNode[]>([])
  const rawEdges = ref<GraphEdge[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const filter = ref<GraphFilter>({ types: [], statuses: [], lineages: [] })

  const uniqueTypes = computed(() => [...new Set(rawNodes.value.map((n) => n.type))].sort())
  const uniqueStatuses = computed(() => [...new Set(rawNodes.value.map((n) => n.status))].sort())
  const uniqueLineages = computed(() => [...new Set(rawNodes.value.map((n) => n.lineage))].sort())

  const filteredNodes = computed(() => {
    const f = filter.value
    return rawNodes.value.filter((n) => {
      if (f.types?.length && !f.types.includes(n.type)) return false
      if (f.statuses?.length && !f.statuses.includes(n.status)) return false
      if (f.lineages?.length && !f.lineages.includes(n.lineage)) return false
      return true
    })
  })

  const filteredEdges = computed(() => {
    const nodeSet = new Set(filteredNodes.value.map((n) => n.id))
    return rawEdges.value.filter((e) => nodeSet.has(e.source) && nodeSet.has(e.target))
  })

  async function fetchGraph(project: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const data = await graphApi.getGraph(project)
      rawNodes.value = data.nodes ?? []
      rawEdges.value = data.edges ?? []
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load graph'
    } finally {
      loading.value = false
    }
  }

  function setFilter(f: Partial<GraphFilter>): void {
    filter.value = { ...filter.value, ...f }
  }

  function toggleFilterValue(key: keyof GraphFilter, value: string): void {
    const current = filter.value[key] ?? []
    filter.value = {
      ...filter.value,
      [key]: current.includes(value) ? current.filter((v) => v !== value) : [...current, value],
    }
  }

  return {
    rawNodes,
    rawEdges,
    loading,
    error,
    filter,
    uniqueTypes,
    uniqueStatuses,
    uniqueLineages,
    filteredNodes,
    filteredEdges,
    fetchGraph,
    setFilter,
    toggleFilterValue,
  }
})
