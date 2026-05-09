// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import * as artifactsApi from '@/api/artifacts'
import type { ArtifactRow, ArtifactDetail, ArtifactFilter } from '@/types/api'

export const useArtifactsStore = defineStore('artifacts', () => {
  const items = ref<ArtifactRow[]>([])
  const total = ref(0)
  const loading = ref(false)
  const filter = reactive<ArtifactFilter>({ limit: 50, offset: 0 })

  // Per-path cache for the editor view.
  const detailCache = new Map<string, ArtifactDetail>()

  const labels = ref<string[]>([])
  const priorities = ref<string[]>([])

  async function fetchList(project: string, f?: Partial<ArtifactFilter>): Promise<void> {
    if (f) Object.assign(filter, f)
    loading.value = true
    try {
      const data = await artifactsApi.listArtifacts(project, filter)
      items.value = data.items ?? []
      total.value = data.total ?? 0
    } finally {
      loading.value = false
    }
  }

  async function fetchOne(project: string, path: string): Promise<ArtifactDetail> {
    const cached = detailCache.get(path)
    if (cached) return cached

    const data = await artifactsApi.getArtifact(project, path)
    const detail: ArtifactDetail = { ...data.artifact, body: data.body, body_html: data.body_html }
    detailCache.set(path, detail)
    return detail
  }

  async function fetchLabels(project: string): Promise<void> {
    const data = await artifactsApi.listLabels(project)
    labels.value = data.labels ?? []
  }

  async function fetchPriorities(project: string): Promise<void> {
    const data = await artifactsApi.listPriorities(project)
    priorities.value = data.priorities ?? []
  }

  function invalidate(path?: string): void {
    if (path) {
      detailCache.delete(path)
    } else {
      detailCache.clear()
    }
    // Mark list as stale — next fetchList call will refetch.
    items.value = []
    total.value = 0
  }

  return {
    items,
    total,
    loading,
    filter,
    labels,
    priorities,
    fetchList,
    fetchOne,
    fetchLabels,
    fetchPriorities,
    invalidate,
  }
})
