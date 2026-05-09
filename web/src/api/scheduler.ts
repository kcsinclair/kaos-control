// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type { SchedulerJob, SchedulerRun } from '@/types/api'

const base = (project: string) => `/p/${encodeURIComponent(project)}/scheduler`

export function listJobs(project: string) {
  return api.get<{ jobs: SchedulerJob[] }>(`${base(project)}/jobs`)
}

export function getJob(project: string, name: string) {
  return api.get<{ job: SchedulerJob; runs: SchedulerRun[] }>(
    `${base(project)}/jobs/${encodeURIComponent(name)}`,
  )
}

export function createJob(project: string, payload: Omit<SchedulerJob, 'created_at' | 'updated_at' | 'next_run_at' | 'last_run_status' | 'last_run_at'>) {
  return api.post<{ job: SchedulerJob }>(`${base(project)}/jobs`, payload)
}

export function updateJob(project: string, name: string, payload: Partial<Omit<SchedulerJob, 'name' | 'created_at' | 'updated_at'>>) {
  return api.put<{ job: SchedulerJob }>(
    `${base(project)}/jobs/${encodeURIComponent(name)}`,
    payload,
  )
}

export function deleteJob(project: string, name: string) {
  return api.delete<void>(`${base(project)}/jobs/${encodeURIComponent(name)}`)
}

export function triggerJob(project: string, name: string) {
  return api.post<{ ok: boolean }>(`${base(project)}/jobs/${encodeURIComponent(name)}/trigger`)
}

export function pauseJob(project: string, name: string) {
  return api.post<{ job: SchedulerJob }>(
    `${base(project)}/jobs/${encodeURIComponent(name)}/pause`,
  )
}

export function resumeJob(project: string, name: string) {
  return api.post<{ job: SchedulerJob }>(
    `${base(project)}/jobs/${encodeURIComponent(name)}/resume`,
  )
}

export function listRuns(project: string, jobName: string, page = 1, perPage = 20) {
  const q = new URLSearchParams({ page: String(page), per_page: String(perPage) })
  return api.get<{ runs: SchedulerRun[]; total: number }>(
    `${base(project)}/jobs/${encodeURIComponent(jobName)}/runs?${q.toString()}`,
  )
}

export async function getRunLog(project: string, jobName: string, runId: number): Promise<string> {
  const res = await fetch(
    `/api${base(project)}/jobs/${encodeURIComponent(jobName)}/runs/${runId}/log`,
    { credentials: 'include' },
  )
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${await res.text()}`)
  }
  return res.text()
}
