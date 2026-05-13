// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as queueApi from '@/api/queue'
import type { QueueSnapshot, QueueJob } from '@/api/queue'
import { getAppWs } from '@/api/ws'

const MAX_RECENT = 10

const emptySnapshot = (): QueueSnapshot => ({
  running: null,
  pending: [],
  recent: [],
  paused: false,
  paused_until: null,
  pause_reason: null,
})

export const useQueueStore = defineStore('queue', () => {
  const snapshot = ref<QueueSnapshot>(emptySnapshot())
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Subscribe to the app-level WS exactly once.
  let _subscribed = false

  function _subscribe() {
    if (_subscribed) return
    _subscribed = true
    const ws = getAppWs()
    ws.on((e) => {
      const p = e.payload as Record<string, unknown>
      switch (e.type) {
        case 'queue.added': {
          // Server sends the full job record; upsert so that any optimistic
          // placeholder inserted by enqueue() is replaced with authoritative data.
          const job = p as unknown as QueueJob
          const existingIdx = snapshot.value.pending.findIndex((j) => j.id === job.id)
          if (existingIdx === -1) {
            snapshot.value.pending.push(job)
          } else {
            snapshot.value.pending[existingIdx] = job
          }
          snapshot.value.pending.sort((a, b) => a.position - b.position)
          break
        }
        case 'queue.started': {
          const id = p.id as string
          const idx = snapshot.value.pending.findIndex((j) => j.id === id)
          if (idx !== -1) {
            const [job] = snapshot.value.pending.splice(idx, 1)
            snapshot.value.running = {
              ...job,
              state: 'running',
              started_at: p.started_at as number | undefined ?? Math.floor(Date.now() / 1000),
            }
          }
          break
        }
        case 'queue.finished':
        case 'queue.skipped':
        case 'queue.cancelled': {
          const id = p.id as string
          const termState = e.type === 'queue.finished'
            ? ((p.terminal_state as QueueJob['state']) ?? 'completed')
            : e.type === 'queue.skipped'
              ? 'skipped'
              : 'cancelled'
          const reason = p.reason as string | undefined

          let finishedJob: QueueJob | null = null

          if (snapshot.value.running?.id === id) {
            finishedJob = { ...snapshot.value.running, state: termState, reason, finished_at: Math.floor(Date.now() / 1000) }
            snapshot.value.running = null
          } else {
            const idx = snapshot.value.pending.findIndex((j) => j.id === id)
            if (idx !== -1) {
              const [job] = snapshot.value.pending.splice(idx, 1)
              finishedJob = { ...job, state: termState, reason, finished_at: Math.floor(Date.now() / 1000) }
            }
          }

          if (finishedJob) {
            snapshot.value.recent.unshift(finishedJob)
            if (snapshot.value.recent.length > MAX_RECENT) {
              snapshot.value.recent = snapshot.value.recent.slice(0, MAX_RECENT)
            }
            // The WS event payload does not include the terminal reason (the
            // backend only stores it in the DB). Refresh the snapshot silently
            // so the recent list shows the correct reason from the DB.
            void _silentRefresh()
          }
          break
        }
        case 'queue.paused': {
          snapshot.value.paused = true
          snapshot.value.paused_until = (p.paused_until as string | null) ?? null
          snapshot.value.pause_reason = (p.pause_reason as 'rate_limit' | 'manual' | null) ?? null
          break
        }
        case 'queue.resumed': {
          snapshot.value.paused = false
          snapshot.value.paused_until = null
          snapshot.value.pause_reason = null
          break
        }
      }
    })
  }

  const pendingCount = computed(() => snapshot.value.pending.length)
  const isPaused = computed(() => snapshot.value.paused)
  const pausedUntilDate = computed(() =>
    snapshot.value.paused_until ? new Date(snapshot.value.paused_until) : null,
  )

  // Refreshes the snapshot from the server without touching loading/error state.
  // Used internally to pick up data (e.g. terminal reason) that WS events omit.
  async function _silentRefresh() {
    try {
      const raw = await queueApi.listQueue()
      snapshot.value = {
        ...raw,
        pending: raw.pending ?? [],
        recent: raw.recent ?? [],
      }
    } catch {
      // Ignore errors — the existing snapshot remains visible.
    }
  }

  async function fetch() {
    _subscribe()
    loading.value = true
    error.value = null
    try {
      const raw = await queueApi.listQueue()
      // Defensive normalisation: an older or buggy backend may return `null`
      // for empty slices. Consumers do `.length` / `.find(...)` directly so
      // a null array field throws `null is not an object`. Coerce to empty
      // arrays here so every reader stays safe regardless of backend version.
      snapshot.value = {
        ...raw,
        pending: raw.pending ?? [],
        recent: raw.recent ?? [],
      }
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load queue'
    } finally {
      loading.value = false
    }
  }

  async function enqueue(args: { project: string; artifact_path: string; agent: string }) {
    _subscribe()
    const result = await queueApi.enqueue(args)
    // Optimistically insert the new job so the queued badge appears immediately,
    // even before the queue.added WS event arrives.  The WS handler will upsert
    // the same entry with full server data once the event comes in.
    if (!snapshot.value.pending.find((j) => j.id === result.id)) {
      snapshot.value.pending.push({
        id: result.id,
        project: args.project,
        artifact_path: args.artifact_path,
        agent_name: args.agent,
        state: 'pending',
        position: result.position,
        attempts: 0,
        enqueued_at: new Date().toISOString(),
        enqueued_by: '',
      })
      snapshot.value.pending.sort((a, b) => a.position - b.position)
    }
  }

  async function cancel(id: string) {
    await queueApi.cancelQueue(id)
    // The WS event queue.cancelled will update the snapshot.
  }

  async function pause() {
    await queueApi.pauseQueue()
  }

  async function resume() {
    await queueApi.resumeQueue()
  }

  return {
    snapshot,
    loading,
    error,
    pendingCount,
    isPaused,
    pausedUntilDate,
    fetch,
    enqueue,
    cancel,
    pause,
    resume,
  }
})
