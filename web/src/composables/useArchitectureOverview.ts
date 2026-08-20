// SPDX-License-Identifier: AGPL-3.0-or-later

import { computed, onMounted, ref } from 'vue'
import * as architectureApi from '@/api/architecture'
import { useWebSocket } from '@/composables/useWebSocket'
import type { ArchitectureOverview } from '@/types/api'

/**
 * Loads the assembled, read-only architecture-zone overview (backend M-B2)
 * and keeps it fresh on artifact.indexed / file.changed (FR-12), mirroring
 * useArchitectureMap. Panel bodies are fetched lazily by the panels
 * themselves via the artifacts store (NFR-1) — this composable only carries
 * the light classified model.
 */
export function useArchitectureOverview(project: string) {
  const overview = ref<ArchitectureOverview | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const hasChosenArchitecture = computed(() => overview.value?.has_chosen_architecture ?? false)
  const chosenArchitecture = computed(() => overview.value?.chosen_architecture ?? null)
  const chosenStack = computed(() => overview.value?.chosen_stack ?? null)
  const summary = computed(() => overview.value?.summary ?? null)
  const standards = computed(() => overview.value?.standards ?? [])
  const adrs = computed(() => overview.value?.adrs ?? [])
  const archive = computed(() => overview.value?.archive ?? [])
  const catalog = computed(() => overview.value?.catalog ?? [])

  async function reload(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      overview.value = await architectureApi.getOverview(project)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load architecture overview'
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

  return {
    overview,
    loading,
    error,
    hasChosenArchitecture,
    chosenArchitecture,
    chosenStack,
    summary,
    standards,
    adrs,
    archive,
    catalog,
    reload,
  }
}
