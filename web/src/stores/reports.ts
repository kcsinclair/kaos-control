// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getAgentUsageReport } from '@/api/reports'
import type { AgentUsageFilter, AgentUsageReport } from '@/types/api'

function defaultFilter(): AgentUsageFilter {
  const now = new Date()
  const from = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
  return {
    from: from.toISOString(),
    to: now.toISOString(),
    bucket: 'day',
    tz: Intl.DateTimeFormat().resolvedOptions().timeZone,
    agent: [],
    status: [],
  }
}

export const useReportsStore = defineStore('reports', () => {
  const filter = ref<AgentUsageFilter>(defaultFilter())
  const report = ref<AgentUsageReport | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  let _debounceTimer: ReturnType<typeof setTimeout> | null = null

  async function fetch(project: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      report.value = await getAgentUsageReport(project, filter.value)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load report'
    } finally {
      loading.value = false
    }
  }

  let _lastProject = ''

  function setFilter(patch: Partial<AgentUsageFilter>, project: string): void {
    filter.value = { ...filter.value, ...patch }
    _lastProject = project
    if (_debounceTimer !== null) clearTimeout(_debounceTimer)
    _debounceTimer = setTimeout(() => {
      _debounceTimer = null
      void fetch(_lastProject)
    }, 300)
  }

  function reset(project: string): void {
    filter.value = defaultFilter()
    void fetch(project)
  }

  return { filter, report, loading, error, fetch, setFilter, reset }
})
