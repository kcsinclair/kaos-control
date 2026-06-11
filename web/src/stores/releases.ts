// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed, onMounted, onUnmounted } from 'vue'
import * as releasesApi from '@/api/releases'
import type { Release, CreateReleasePayload, UpdateReleasePayload } from '@/types/release'
import { getProjectWs } from '@/api/ws'

export const useReleasesStore = defineStore('releases', () => {
  const releases = ref<Release[]>([])
  const loading = ref(false)
  const lastWsSeq = ref(0)
  let currentProject = ''
  let unsub: (() => void) | null = null

  const scheduled = computed(() =>
    releases.value
      .filter((r) => r.start_date != null && r.end_date != null)
      .slice()
      .sort((a, b) => {
        const ad = a.start_date ?? ''
        const bd = b.start_date ?? ''
        return ad < bd ? -1 : ad > bd ? 1 : 0
      })
  )

  const unscheduled = computed(() =>
    releases.value.filter((r) => r.start_date == null || r.end_date == null)
  )

  function byId(id: number): Release | undefined {
    return releases.value.find((r) => r.id === id)
  }

  function byName(name: string): Release | undefined {
    return releases.value.find((r) => r.name === name)
  }

  function bySlug(slug: string): Release | undefined {
    return releases.value.find((r) => r.slug === slug)
  }

  async function fetch(project: string): Promise<void> {
    loading.value = true
    try {
      const data = await releasesApi.listReleases(project)
      releases.value = data ?? []
    } finally {
      loading.value = false
    }
  }

  async function create(project: string, data: CreateReleasePayload): Promise<Release> {
    const release = await releasesApi.createRelease(project, data)
    // Guard: ensure releases.value is always an array before spreading.
    // A WS release.created event may have already inserted this release while
    // the HTTP response was in flight; deduplicate by id to avoid doubles.
    const current = Array.isArray(releases.value) ? releases.value : []
    if (!current.some((r) => r.id === release.id)) {
      releases.value = [...current, release]
    }
    return release
  }

  async function update(project: string, id: number, data: UpdateReleasePayload): Promise<Release> {
    const current = releases.value.find((r) => r.id === id)
    const payload: UpdateReleasePayload = current
      ? { ...data, updated_at: current.updated_at }
      : data
    const release = await releasesApi.updateRelease(project, id, payload)
    releases.value = releases.value.map((r) => (r.id === id ? release : r))
    return release
  }

  async function remove(project: string, id: number, reassignTo?: number): Promise<{ orphaned_artifact_count: number }> {
    const result = await releasesApi.deleteRelease(project, id, reassignTo)
    releases.value = releases.value.filter((r) => r.id !== id)
    return result
  }

  function connectWs(project: string): void {
    if (unsub) {
      unsub()
      unsub = null
    }
    currentProject = project
    const ws = getProjectWs(project)

    const handler = (e: { type: string; payload: Record<string, unknown> }) => {
      if (e.type === 'release.changed') {
        const action = (e.payload as { action?: string }).action
        if (action === 'deleted') {
          // Watcher-triggered delete: payload carries slug
          const slug = (e.payload as { slug?: string }).slug
          if (slug !== undefined) {
            releases.value = releases.value.filter((r) => r.slug !== slug)
            lastWsSeq.value++
          }
        } else {
          // Create or update from API/watcher: upsert by id, fallback to slug
          const raw = (e.payload as { release?: Release }).release
          if (raw) {
            const rel: Release = {
              ...raw,
              start_date: (raw.start_date as string | null | undefined) ?? null,
              end_date: (raw.end_date as string | null | undefined) ?? null,
              file_path: (raw.file_path as string | undefined) ?? '',
              slug: (raw.slug as string | undefined) ?? '',
            }
            const idx = releases.value.findIndex(
              (r) => r.id === rel.id || (rel.slug && r.slug === rel.slug),
            )
            if (idx >= 0) {
              releases.value = releases.value.map((r, i) => (i === idx ? rel : r))
            } else {
              releases.value = [...releases.value, rel]
            }
            lastWsSeq.value++
          }
        }
      } else if (e.type === 'release.created') {
        // Legacy event kept for backward compat
        const release = (e.payload as { release?: Release }).release
        if (release && !releases.value.find((r) => r.id === release.id)) {
          releases.value = [...releases.value, release]
          lastWsSeq.value++
        }
      } else if (e.type === 'release.updated') {
        // Legacy event kept for backward compat
        const release = (e.payload as { release?: Release }).release
        if (release) {
          releases.value = releases.value.map((r) => (r.id === release.id ? release : r))
          lastWsSeq.value++
        }
      } else if (e.type === 'release.deleted') {
        // API-triggered delete: payload carries id
        const id = (e.payload as { id?: number }).id
        if (id !== undefined) {
          releases.value = releases.value.filter((r) => r.id !== id)
          lastWsSeq.value++
        }
      }
    }

    unsub = ws.on(handler)
  }

  function disconnectWs(): void {
    if (unsub) {
      unsub()
      unsub = null
    }
  }

  return {
    releases,
    loading,
    lastWsSeq,
    scheduled,
    unscheduled,
    byId,
    byName,
    bySlug,
    fetch,
    create,
    update,
    remove,
    connectWs,
    disconnectWs,
  }
})
