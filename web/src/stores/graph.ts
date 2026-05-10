// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as graphApi from '@/api/graph'
import type { GraphNode, GraphEdge, GraphFilter } from '@/types/api'
import { TERMINAL_STATUSES } from '@/types/api'
import { LAYOUT_CONFIGS } from '@/components/map/layoutConfigs'

export const useGraphStore = defineStore('graph', () => {
  const rawNodes = ref<GraphNode[]>([])
  const rawEdges = ref<GraphEdge[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const filter = ref<GraphFilter>({ types: [], statuses: [], lineages: [], labels: [], priorities: [] })

  const searchText = ref('')

  // Layout state — session-scoped (no localStorage persistence)
  const activeLayout = ref<string>('fcose')
  const directed = ref<boolean>(false)
  // True while a layout animation is running (used to disable the selector)
  const layoutAnimating = ref<boolean>(false)

  const showLabelNodes = ref(false)

  // Release overlay — off by default; toggling fetches release-augmented data.
  const showReleases = ref(false)
  // True once rawNodes has been fetched with include_releases=true.
  const releaseDataFetched = ref(false)
  // Tracks the most-recently-fetched project so toggleShowReleases can call fetchGraph.
  const currentProject = ref('')

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
      // Release nodes are managed by the overlay (releaseNodes computed) — always excluded here.
      if (n.type === 'release') return false
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

  // Release overlay: release nodes and their connecting edges (timeline spine + assigned).
  const releaseNodes = computed<GraphNode[]>(() =>
    rawNodes.value.filter((n) => n.type === 'release')
  )

  // Only include release edges whose non-release endpoints are currently visible.
  // Without this guard, assigned edges that reference filtered-out artifact nodes cause
  // 3d-force-graph to auto-create phantom nodes at (0,0,0), corrupting the force
  // simulation layout and generating phantom arrow-cone artefacts.
  const releaseEdges = computed<GraphEdge[]>(() => {
    const artifactIds = new Set(filteredNodes.value.map((n) => n.id))
    const releaseIds = new Set(releaseNodes.value.map((n) => n.id))
    const filtered = rawEdges.value.filter((e) => {
      if (e.kind === 'timeline') {
        // Timeline edges connect two release nodes — both must exist.
        return releaseIds.has(e.source) && releaseIds.has(e.target)
      }
      if (e.kind === 'assigned') {
        // One end is a release node; the other is an artifact.
        // Only include the edge when the artifact is currently visible.
        const srcIsRelease = releaseIds.has(e.source)
        const tgtIsRelease = releaseIds.has(e.target)
        if (srcIsRelease && !tgtIsRelease) return artifactIds.has(e.target)
        if (!srcIsRelease && tgtIsRelease) return artifactIds.has(e.source)
        return true // both endpoints are release nodes
      }
      return false
    })

    // Augment: if a synthetic `release:unscheduled` terminus exists, connect the
    // last node in the main scheduled timeline chain to it.  The backend emits
    // edges from each unscheduled artifact *to* the terminus but does not connect
    // the last scheduled release to it; we fill that gap here so the node is
    // visually attached to the end of the timeline in both 2D and 3D views.
    const UNSCHEDULED_ID = 'release:unscheduled'
    const BACKLOG_ID = 'release:backlog'
    if (releaseIds.has(UNSCHEDULED_ID) && releaseNodes.value.some((n) => n.id === UNSCHEDULED_ID && n.synthetic)) {
      const chainSources = new Set<string>()
      const chainTargets = new Set<string>()
      for (const e of filtered) {
        if (e.kind === 'timeline' && e.target !== UNSCHEDULED_ID) {
          chainSources.add(e.source)
          chainTargets.add(e.target)
        }
      }
      const tails = [...chainTargets].filter((id) => !chainSources.has(id))
      const lastChainNode = tails.length > 0 ? tails[0] : BACKLOG_ID
      const alreadyExists = filtered.some(
        (e) => e.source === lastChainNode && e.target === UNSCHEDULED_ID && e.kind === 'timeline'
      )
      if (!alreadyExists) {
        return [...filtered, { source: lastChainNode, target: UNSCHEDULED_ID, kind: 'timeline' }]
      }
    }

    return filtered
  })

  const augmentedNodes = computed<GraphNode[]>(() => {
    const base = showLabelNodes.value ? [...filteredNodes.value, ...labelNodes.value] : filteredNodes.value
    return showReleases.value ? [...base, ...releaseNodes.value] : base
  })

  const augmentedEdges = computed<GraphEdge[]>(() => {
    const base = showLabelNodes.value ? [...filteredEdges.value, ...labelEdges.value] : filteredEdges.value
    return showReleases.value ? [...base, ...releaseEdges.value] : base
  })

  async function fetchGraph(project: string, includeReleases?: boolean): Promise<void> {
    loading.value = true
    error.value = null
    currentProject.value = project
    // If the overlay is already on and caller didn't specify, always include releases
    // so live WebSocket refreshes don't silently drop the release data.
    const withReleases = includeReleases ?? showReleases.value
    try {
      const data = await graphApi.getGraph(project, withReleases)
      rawNodes.value = data.nodes ?? []
      rawEdges.value = data.edges ?? []
      if (withReleases) releaseDataFetched.value = true
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

  async function toggleShowReleases(project?: string): Promise<void> {
    showReleases.value = !showReleases.value
    if (showReleases.value && !releaseDataFetched.value) {
      const p = project ?? currentProject.value
      if (p) await fetchGraph(p, true)
    }
  }

  function toggleHideTerminal(): void {
    hideTerminal.value = !hideTerminal.value
  }

  function toggleHideTests(): void {
    hideTests.value = !hideTests.value
  }

  function setLayout(key: string): void {
    if (key in LAYOUT_CONFIGS) {
      activeLayout.value = key
    }
  }

  function toggleDirected(): void {
    directed.value = !directed.value
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
    showReleases,
    releaseDataFetched,
    hideTerminal,
    hideTests,
    activeLayout,
    directed,
    layoutAnimating,
    uniqueTypes,
    uniqueStatuses,
    uniqueLineages,
    uniqueLabels,
    uniquePriorities,
    filteredNodes,
    filteredEdges,
    labelNodes,
    labelEdges,
    releaseNodes,
    releaseEdges,
    augmentedNodes,
    augmentedEdges,
    fetchGraph,
    setFilter,
    toggleFilterValue,
    toggleShowLabelNodes,
    toggleShowReleases,
    toggleHideTerminal,
    toggleHideTests,
    setLayout,
    toggleDirected,
    updateNodePriority,
  }
})
