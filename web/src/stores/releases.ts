import { defineStore } from 'pinia'
import { ref, computed, onMounted, onUnmounted } from 'vue'
import * as releasesApi from '@/api/releases'
import type { Release, CreateReleasePayload, UpdateReleasePayload } from '@/types/release'
import { getProjectWs } from '@/api/ws'

export const useReleasesStore = defineStore('releases', () => {
  const releases = ref<Release[]>([])
  const loading = ref(false)
  let currentProject = ''
  let unsub: (() => void) | null = null

  const scheduled = computed(() =>
    releases.value
      .filter((r) => r.start_date !== null && r.end_date !== null)
      .slice()
      .sort((a, b) => {
        const ad = a.start_date ?? ''
        const bd = b.start_date ?? ''
        return ad < bd ? -1 : ad > bd ? 1 : 0
      })
  )

  const unscheduled = computed(() =>
    releases.value.filter((r) => r.start_date === null || r.end_date === null)
  )

  function byId(id: number): Release | undefined {
    return releases.value.find((r) => r.id === id)
  }

  function byName(name: string): Release | undefined {
    return releases.value.find((r) => r.name === name)
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
    releases.value = [...releases.value, release]
    return release
  }

  async function update(project: string, id: number, data: UpdateReleasePayload): Promise<Release> {
    const release = await releasesApi.updateRelease(project, id, data)
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
      if (e.type === 'release.created') {
        const release = (e.payload as { release?: Release }).release
        if (release && !releases.value.find((r) => r.id === release.id)) {
          releases.value = [...releases.value, release]
        }
      } else if (e.type === 'release.updated') {
        const release = (e.payload as { release?: Release }).release
        if (release) {
          releases.value = releases.value.map((r) => (r.id === release.id ? release : r))
        }
      } else if (e.type === 'release.deleted') {
        const id = (e.payload as { id?: number }).id
        if (id !== undefined) {
          releases.value = releases.value.filter((r) => r.id !== id)
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
    scheduled,
    unscheduled,
    byId,
    byName,
    fetch,
    create,
    update,
    remove,
    connectWs,
    disconnectWs,
  }
})
