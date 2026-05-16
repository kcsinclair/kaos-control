// SPDX-License-Identifier: AGPL-3.0-or-later

import { ref, onUnmounted } from 'vue'
import { ApiError } from '@/api/client'
import * as locksApi from '@/api/locks'
import { useLocksStore } from '@/stores/locks'
import type { LockRow } from '@/types/api'

const HEARTBEAT_MS = 30_000

export function useLock(project: string, lineage: string | (() => string)) {
  const getLineage = typeof lineage === 'function' ? lineage : () => lineage
  const locksStore = useLocksStore()
  const acquired = ref(false)
  const conflictLock = ref<LockRow | null>(null)
  let timer: ReturnType<typeof setInterval> | null = null

  async function acquire(): Promise<boolean> {
    const lin = getLineage()
    try {
      const data = await locksApi.acquireLock(project, lin)
      locksStore.setLock(data.lock)
      acquired.value = true
      conflictLock.value = null
      timer = setInterval(async () => {
        try {
          await locksApi.heartbeatLock(project, getLineage())
        } catch {
          acquired.value = false
        }
      }, HEARTBEAT_MS)
      return true
    } catch (e: unknown) {
      if (e instanceof ApiError && e.status === 409) {
        // Locked by someone else — record who for the banner.
        const body = (e as unknown as { lock?: LockRow }).lock
        if (body) conflictLock.value = body
      }
      // 503 = lock manager not configured; treat as lock-free.
      if (e instanceof ApiError && e.status === 503) return true
      return false
    }
  }

  async function release(): Promise<void> {
    if (!acquired.value) return
    if (timer) { clearInterval(timer); timer = null }
    const lin = getLineage()
    try {
      await locksApi.releaseLock(project, lin)
      locksStore.removeLock(lin)
    } catch {
      // best-effort
    }
    acquired.value = false
  }

  onUnmounted(release)

  return { acquired, conflictLock, acquire, release }
}
