// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchFeed } from '@/api/feed'
import type { FeedEvent } from '@/types/api'

const ALL_TYPES = [
  'status_transition',
  'artifact_created',
  'agent_started',
  'agent_finished',
  'agent_failed',
  'defect_raised',
  'git_committed',
]

export const useFeedStore = defineStore('feed', () => {
  const events = ref<FeedEvent[]>([])
  const nextCursor = ref<number | null>(null)
  const loading = ref(false)
  const activeTypes = ref<Set<string>>(new Set(ALL_TYPES))
  let _lastProject = ''

  async function loadPage(project: string): Promise<void> {
    if (loading.value) return
    loading.value = true
    try {
      const params: { limit?: number; before?: number; types?: string } = { limit: 50 }
      if (nextCursor.value !== null) {
        params.before = nextCursor.value
      }
      // Only send types param when filtered
      if (activeTypes.value.size < ALL_TYPES.length) {
        params.types = Array.from(activeTypes.value).join(',')
      }
      const data = await fetchFeed(project, params)
      events.value.push(...(data.events ?? []))
      nextCursor.value = data.next_cursor
    } finally {
      loading.value = false
    }
  }

  async function refresh(project: string): Promise<void> {
    _lastProject = project
    events.value = []
    nextCursor.value = null
    await loadPage(project)
  }

  function prepend(event: FeedEvent): void {
    if (activeTypes.value.has(event.event_type)) {
      events.value.unshift(event)
    }
  }

  function setFilter(type: string, enabled: boolean): void {
    const next = new Set(activeTypes.value)
    if (enabled) {
      next.add(type)
    } else {
      next.delete(type)
    }
    activeTypes.value = next
    if (_lastProject) {
      void refresh(_lastProject)
    }
  }

  return {
    events,
    nextCursor,
    loading,
    activeTypes,
    loadPage,
    refresh,
    prepend,
    setFilter,
  }
})
