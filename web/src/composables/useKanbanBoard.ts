import { ref, computed, reactive } from 'vue'
import { api } from '@/api/client'
import * as artifactsApi from '@/api/artifacts'
import type { ArtifactRow, ArtifactFilter } from '@/types/api'
import { TERMINAL_STATUSES } from '@/types/api'
import { parseArtifactDate } from '@/composables/useFormatDate'

export interface KanbanColumnConfig {
  name: string
  statuses: string[]
}

export interface KanbanConfig {
  columns: KanbanColumnConfig[]
  uncategorised?: boolean
  card_fields?: string[]
}

export interface KanbanColumn {
  name: string
  statuses: string[]
  cards: ArtifactRow[]
}

export function useKanbanBoard(project: string) {
  const loading = ref(false)
  const hasConfig = ref(false)
  const config = ref<KanbanConfig | null>(null)
  const allArtifacts = ref<ArtifactRow[]>([])

  // Reactive filter state
  const filters = reactive<Partial<ArtifactFilter>>({})

  // Client-side text search
  const searchText = ref('')

  // When true, terminal-status cards (done/rejected/abandoned) are excluded
  const hideTerminal = ref(true)

  // Ordered column list — starts from config, can be reordered by drag
  const columnOrder = ref<KanbanColumnConfig[]>([])

  // Compute age string from an artifact date string (plain or RFC3339)
  function computeAge(created: string): string {
    const d = parseArtifactDate(created)
    if (!d) return '?'
    const days = Math.floor((Date.now() - d.getTime()) / 86400000)
    return `${days}d`
  }

  // Attach virtual field `age` onto artifacts — we store them in a parallel map
  // keyed by path so we don't mutate the store's objects.
  function ageOf(artifact: ArtifactRow): string {
    return artifact.created ? computeAge(artifact.created) : '?'
  }

  // Apply client-side filters to an artifact list
  function applyClientFilters(items: ArtifactRow[]): ArtifactRow[] {
    const q = searchText.value.toLowerCase()
    return items.filter(a => {
      if (hideTerminal.value && (TERMINAL_STATUSES as readonly string[]).includes(a.status)) return false
      if (filters.stage && a.stage !== filters.stage) return false
      if (filters.status && a.status !== filters.status) return false
      if (filters.type && a.type !== filters.type) return false
      if (filters.label) {
        const labels = a.frontmatter?.labels ?? []
        if (!labels.includes(filters.label)) return false
      }
      if (filters.priority) {
        if ((a.frontmatter?.priority ?? '') !== filters.priority) return false
      }
      if (filters.release !== undefined && filters.release !== '') {
        if (filters.release === '__unassigned__') {
          if (a.frontmatter?.release) return false
        } else {
          if ((a.frontmatter?.release ?? '') !== filters.release) return false
        }
      }
      if (q) {
        const haystack = [a.title, a.lineage, a.type, a.status].join(' ').toLowerCase()
        if (!haystack.includes(q)) return false
      }
      return true
    })
  }

  const columns = computed<KanbanColumn[]>(() => {
    if (!config.value) return []

    const filtered = applyClientFilters(allArtifacts.value)

    // Build a set of all statuses covered by configured columns
    const coveredStatuses = new Set<string>()
    for (const col of columnOrder.value) {
      for (const s of col.statuses) coveredStatuses.add(s)
    }

    const allTerminal = (statuses: string[]) =>
      statuses.length > 0 && statuses.every(s => (TERMINAL_STATUSES as readonly string[]).includes(s))

    const result: KanbanColumn[] = columnOrder.value
      .map(col => ({
        name: col.name,
        statuses: col.statuses,
        cards: filtered.filter(a => col.statuses.includes(a.status)),
      }))
      .filter(col => !(hideTerminal.value && allTerminal(col.statuses) && col.cards.length === 0))

    // Uncategorised column — default true when not explicitly set
    const showUncategorised = config.value.uncategorised !== false
    if (showUncategorised) {
      const uncatCards = filtered.filter(a => !coveredStatuses.has(a.status))
      if (uncatCards.length > 0 || result.length === 0) {
        result.push({ name: 'Uncategorised', statuses: [], cards: uncatCards })
      }
    }

    return result
  })

  const cardFields = computed<string[]>(() => config.value?.card_fields ?? [])

  async function fetchKanbanConfig(): Promise<KanbanConfig | null> {
    const res = await api.get<{ kanban: KanbanConfig | null }>(
      `/p/${encodeURIComponent(project)}/config/kanban`
    )
    return res.kanban
  }

  async function fetchAllArtifacts(): Promise<ArtifactRow[]> {
    // Fetch with a high limit to retrieve all artifacts in one call.
    // If the project has more than 5000 artifacts, paginate.
    const PAGE = 5000
    const first = await artifactsApi.listArtifacts(project, { limit: PAGE, offset: 0 })
    const items = first.items ?? []
    let offset = PAGE
    while (items.length < (first.total ?? 0)) {
      const page = await artifactsApi.listArtifacts(project, { limit: PAGE, offset })
      items.push(...(page.items ?? []))
      offset += PAGE
    }
    return items
  }

  async function refresh(): Promise<void> {
    loading.value = true
    try {
      const [cfg, artifacts] = await Promise.all([
        fetchKanbanConfig(),
        fetchAllArtifacts(),
      ])
      config.value = cfg
      hasConfig.value = cfg !== null
      allArtifacts.value = artifacts
      if (cfg) {
        columnOrder.value = [...cfg.columns]
      }
    } finally {
      loading.value = false
    }
  }

  function applyFilters(f: Partial<ArtifactFilter>): void {
    Object.assign(filters, f)
  }

  function reorderColumns(fromIndex: number, toIndex: number): void {
    if (fromIndex === toIndex) return
    const cols = [...columnOrder.value]
    const [moved] = cols.splice(fromIndex, 1)
    cols.splice(toIndex, 0, moved)
    columnOrder.value = cols
  }

  // Expose ageOf so KanbanCard can compute it without mutating artifact objects
  return {
    loading,
    hasConfig,
    columns,
    cardFields,
    filters,
    searchText,
    hideTerminal,
    refresh,
    applyFilters,
    reorderColumns,
    ageOf,
  }
}
