// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { LockRow } from '@/types/api'

export const useLocksStore = defineStore('locks', () => {
  const locks = ref(new Map<string, LockRow>())

  function setLock(lock: LockRow): void {
    locks.value.set(lock.lineage, lock)
  }

  function removeLock(lineage: string): void {
    locks.value.delete(lineage)
  }

  function getLock(lineage: string): LockRow | undefined {
    return locks.value.get(lineage)
  }

  function isLockedByOther(lineage: string, myEmail: string): boolean {
    const l = locks.value.get(lineage)
    return l !== undefined && l.holder !== myEmail
  }

  // Called from WS event handlers to keep lock state current.
  function applyEvent(type: string, payload: Record<string, unknown>): void {
    if (type === 'lock.acquired') {
      const row = payload as unknown as LockRow
      if (row.lineage) locks.value.set(row.lineage, row)
    } else if (type === 'lock.released') {
      const lineage = payload.lineage as string | undefined
      if (lineage) locks.value.delete(lineage)
    }
  }

  return { locks, setLock, removeLock, getLock, isLockedByOther, applyEvent }
})
