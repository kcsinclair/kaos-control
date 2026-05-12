// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from '@/api/client'

export interface QueueJob {
  id: string
  project: string
  artifact_path: string
  agent: string
  state: 'pending' | 'running' | 'completed' | 'failed' | 'skipped' | 'cancelled'
  reason?: string
  attempts: number
  enqueued_at: number
  started_at?: number
  finished_at?: number
  position: number
  enqueued_by: string
}

export interface QueueSnapshot {
  running: QueueJob | null
  pending: QueueJob[]
  recent: QueueJob[]
  paused: boolean
  paused_until: string | null
  pause_reason: 'rate_limit' | 'manual' | null
}

export const listQueue = () => api.get<QueueSnapshot>('/queue')

export const enqueue = (b: { project: string; artifact_path: string; agent: string }) =>
  api.post<{ id: string; position: number }>('/queue', b)

export const cancelQueue = (id: string) => api.delete<void>(`/queue/${id}`)

export const pauseQueue = () => api.post<void>('/queue/pause', null)

export const resumeQueue = () => api.post<void>('/queue/resume', null)
