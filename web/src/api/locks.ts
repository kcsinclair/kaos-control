import { api } from './client'
import type { LockRow } from '@/types/api'

export function listLocks(project: string) {
  return api.get<{ locks: LockRow[] }>(`/p/${encodeURIComponent(project)}/locks`)
}

export function acquireLock(project: string, lineage: string, kind = 'editor') {
  return api.post<{ lock: LockRow }>(`/p/${encodeURIComponent(project)}/locks`, { lineage, kind })
}

export function releaseLock(project: string, lineage: string) {
  return api.delete<void>(`/p/${encodeURIComponent(project)}/locks/${encodeURIComponent(lineage)}`)
}

export function heartbeatLock(project: string, lineage: string) {
  return api.post<{ ok: boolean }>(
    `/p/${encodeURIComponent(project)}/locks/${encodeURIComponent(lineage)}/heartbeat`,
    {},
  )
}
