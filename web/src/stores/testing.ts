// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as artifactsApi from '@/api/artifacts'
import * as agentsApi from '@/api/agents'
import { getProjectWs } from '@/api/ws'
import type { ArtifactRow } from '@/types/api'

export const useTestingStore = defineStore('testing', () => {
  const tests = ref<ArtifactRow[]>([])
  const loading = ref(false)
  const selectedPaths = ref(new Set<string>())
  const batchQueue = ref<string[]>([])
  const batchCurrentIndex = ref(0)
  const batchRunning = ref(false)
  const filters = ref({
    status: '',
    lineage: '',
    label: '',
    priority: '',
  })

  // Cached badge count for the sidebar (fast fetch without loading all tests)
  const _approvedCount = ref(0)

  const approvedTests = computed(() => tests.value.filter((t) => t.status === 'approved'))

  // If the full test list is loaded, derive from it; otherwise use cached badge count
  const approvedCount = computed(() =>
    tests.value.length > 0 ? approvedTests.value.length : _approvedCount.value,
  )

  const selectedTests = computed(() =>
    tests.value.filter((t) => selectedPaths.value.has(t.path)),
  )

  const batchProgress = computed(() => {
    const currentPath = batchQueue.value[batchCurrentIndex.value] ?? null
    return {
      current: batchCurrentIndex.value,
      total: batchQueue.value.length,
      currentTest: currentPath ? tests.value.find((t) => t.path === currentPath) ?? null : null,
    }
  })

  async function fetchTests(project: string): Promise<void> {
    loading.value = true
    try {
      const data = await artifactsApi.listArtifacts(project, { type: 'test', limit: 500 })
      tests.value = data.items ?? []
      _approvedCount.value = approvedTests.value.length
    } finally {
      loading.value = false
    }
  }

  async function fetchApprovedCount(project: string): Promise<void> {
    try {
      // Use limit=1 with status filter — only total matters for the badge
      const data = await artifactsApi.listArtifacts(project, {
        type: 'test',
        status: 'approved',
        limit: 1,
      })
      _approvedCount.value = data.total ?? 0
    } catch {
      // Badge count is non-critical; swallow errors
    }
  }

  function toggleSelection(path: string): void {
    const test = tests.value.find((t) => t.path === path)
    if (!test || test.status !== 'approved') return
    if (selectedPaths.value.has(path)) {
      selectedPaths.value.delete(path)
    } else {
      selectedPaths.value.add(path)
    }
  }

  function selectAll(): void {
    const next = new Set<string>()
    for (const t of approvedTests.value) next.add(t.path)
    selectedPaths.value = next
  }

  function clearSelection(): void {
    selectedPaths.value = new Set()
  }

  function waitForRun(project: string, runId: string): Promise<void> {
    return new Promise((resolve) => {
      const ws = getProjectWs(project)
      const unsub = ws.on((e) => {
        if (
          (e.type === 'agent.finished' || e.type === 'agent.failed') &&
          e.payload?.run_id === runId
        ) {
          unsub()
          resolve()
        }
      })
    })
  }

  async function startBatch(project: string): Promise<void> {
    if (batchRunning.value || selectedPaths.value.size === 0) return
    batchQueue.value = [...selectedPaths.value]
    batchCurrentIndex.value = 0
    batchRunning.value = true

    try {
      while (batchCurrentIndex.value < batchQueue.value.length && batchRunning.value) {
        const path = batchQueue.value[batchCurrentIndex.value]
        try {
          const data = await agentsApi.startRun(project, 'qa', path)
          await waitForRun(project, data.run_id)
        } catch {
          // Continue to the next test even if this one fails to start
        }
        batchCurrentIndex.value++
      }
    } finally {
      batchRunning.value = false
      batchQueue.value = []
      batchCurrentIndex.value = 0
    }
  }

  function cancelBatch(): void {
    // Sets flag to false; the while loop in startBatch() will exit after current test
    batchRunning.value = false
  }

  return {
    tests,
    loading,
    selectedPaths,
    batchQueue,
    batchCurrentIndex,
    batchRunning,
    filters,
    approvedTests,
    approvedCount,
    selectedTests,
    batchProgress,
    fetchTests,
    fetchApprovedCount,
    toggleSelection,
    selectAll,
    clearSelection,
    startBatch,
    cancelBatch,
  }
})
