import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as graphApi from '@/api/graph'
import type { GraphNode, GraphEdge, GraphFilter } from '@/types/api'
import { TERMINAL_STATUSES } from '@/types/api'

export const useGraphStore = defineStore('graph', () => {
  const rawNodes = ref<GraphNode[]>([])
  const rawEdges = ref<GraphEdge[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const filter = ref<GraphFilter>({ types: [], statuses: [], lineages: [], labels: [], priorities: [] })

  const searchText = ref('')

  const showLabelNodes = ref(false)

  // When true, nodes with terminal statuses are excluded (unless the user has
  // explicitly filtered by status, in which case we honour their selection).
  const hideTerminal = ref(true)

  // When true, nodes with type === 'test' are excluded (unless the user has
  // explicitly filtered by type, in which case we honour their selection).
  const hideTests = ref(true)

  const uniqueTypes = computed(() => [...new Set(rawNodes.value.map((n) => n.type))].sort())
  const uniqueStatuses = computed(() => [...new Set(rawNodes.value.map((n) => n.status))].sort())
  const uniqueLineages = computed(() => [...new Set(rawNodes.value.map((n) => n.lineage))].sort())
  const uniqueLabels = computed(() =>
    [...new Set(rawNodes.value.flatMap((n) => n.labels ?? []))].sort()
  )
  const uniquePriorities = computed(() =>
    [...new Set(rawNodes.value.map((n) => n.priority ?? '').filter(Boolean))].sort()
  )

  const filteredNodes = computed(() => {
    const f = filter.value
    const noStatusFilter = !(f.statuses?.length)
    const noTypeFilter = !(f.types?.length)
    return rawNodes.value.filter((n) => {
      if (hideTerminal.value && noStatusFilter && (TERMINAL_STATUSES as readonly string[]).includes(n.status)) return false
      if (hideTests.value && noTypeFilter && n.type === 'test') return false
      if (f.types?.length && !f.types.includes(n.type)) return false
      if (f.statuses?.length && !f.statuses.includes(n.status)) return false
      if (f.lineages?.length && !f.lineages.includes(n.lineage)) return false
      if (f.labels?.length && !n.labels?.some((l) => f.labels!.includes(l))) return false
      if (f.priorities?.length && !f.priorities.includes(n.priority ?? '')) return false
      return true
    })
  })

  const filteredEdges = computed(() => {
    const nodeSet = new Set(filteredNodes.value.map((n) => n.id))
    return rawEdges.value.filter((e) => nodeSet.has(e.source) && nodeSet.has(e.target))
  })

  // Set of node IDs matching the current searchText (empty set = no active search)
  const matchedNodeIds = computed<Set<string>>(() => {
    const q = searchText.value.trim().toLowerCase()
    if (!q) return new Set()
    return new Set(
      filteredNodes.value
        .filter((n) => [n.title, n.lineage, n.type, n.status].join(' ').toLowerCase().includes(q))
        .map((n) => n.id)
    )
  })

  // Synthetic label nodes derived from filtered artifacts
  const labelNodes = computed<GraphNode[]>(() => {
    const distinctLabels = [...new Set(filteredNodes.value.flatMap((n) => n.labels ?? []))]
    return distinctLabels.map((lbl) => ({
      id: `label::${lbl}`,
      title: lbl,
      type: 'label',
      status: '',
      stage: '',
      lineage: '',
      slug: lbl,
      index: 0,
    }))
  })

  // Edges from each artifact to its label nodes
  const labelEdges = computed<GraphEdge[]>(() => {
    const edges: GraphEdge[] = []
    for (const n of filteredNodes.value) {
      for (const lbl of n.labels ?? []) {
        edges.push({ source: n.id, target: `label::${lbl}`, kind: 'label' })
      }
    }
    return edges
  })

  const augmentedNodes = computed<GraphNode[]>(() =>
    showLabelNodes.value ? [...filteredNodes.value, ...labelNodes.value] : filteredNodes.value
  )

  const augmentedEdges = computed<GraphEdge[]>(() =>
    showLabelNodes.value ? [...filteredEdges.value, ...labelEdges.value] : filteredEdges.value
  )

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

  function toggleShowLabelNodes(): void {
    showLabelNodes.value = !showLabelNodes.value
  }

  function toggleHideTerminal(): void {
    hideTerminal.value = !hideTerminal.value
  }

  function toggleHideTests(): void {
    hideTests.value = !hideTests.value
  }

  function updateNodePriority(nodeId: string, priority: string | null): void {
    const idx = rawNodes.value.findIndex((n) => n.id === nodeId)
    if (idx === -1) return
    const updated = { ...rawNodes.value[idx], priority: priority ?? undefined }
    rawNodes.value = [
      ...rawNodes.value.slice(0, idx),
      updated,
      ...rawNodes.value.slice(idx + 1),
    ]
  }

  return {
    rawNodes,
    rawEdges,
    loading,
    error,
    filter,
    searchText,
    matchedNodeIds,
    showLabelNodes,
    hideTerminal,
    hideTests,
    uniqueTypes,
    uniqueStatuses,
    uniqueLineages,
    uniqueLabels,
    uniquePriorities,
    filteredNodes,
    filteredEdges,
    labelNodes,
    labelEdges,
    augmentedNodes,
    augmentedEdges,
    fetchGraph,
    setFilter,
    toggleFilterValue,
    toggleShowLabelNodes,
    toggleHideTerminal,
    toggleHideTests,
    updateNodePriority,
  }
})
