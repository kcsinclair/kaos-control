import { api } from './client'
import type { AgentSummary, AgentRunRow } from '@/types/api'

export function listAgents(project: string) {
  return api.get<{ agents: AgentSummary[] }>(`/p/${encodeURIComponent(project)}/agents`)
}

export function startRun(project: string, agentName: string, targetPath: string, role?: string) {
  return api.post<{ run_id: string }>(
    `/p/${encodeURIComponent(project)}/agents/${encodeURIComponent(agentName)}/run`,
    { target_path: targetPath, role },
  )
}

export function listRuns(project: string, status?: string, limit = 50) {
  const q = new URLSearchParams()
  if (status) q.set('status', status)
  if (limit) q.set('limit', String(limit))
  const qs = q.toString()
  return api.get<{ runs: AgentRunRow[] }>(
    `/p/${encodeURIComponent(project)}/agents/runs${qs ? '?' + qs : ''}`,
  )
}

export function getRun(project: string, runId: string) {
  return api.get<{ run: AgentRunRow }>(
    `/p/${encodeURIComponent(project)}/agents/runs/${encodeURIComponent(runId)}`,
  )
}

export function killRun(project: string, runId: string) {
  return api.post<{ ok: boolean; run_id: string }>(
    `/p/${encodeURIComponent(project)}/agents/runs/${encodeURIComponent(runId)}/kill`,
  )
}

export async function listRunsByTargetPath(project: string, targetPath: string): Promise<AgentRunRow[]> {
  const q = new URLSearchParams()
  q.set('target_path', targetPath)
  const data = await api.get<{ runs: AgentRunRow[] }>(
    `/p/${encodeURIComponent(project)}/agents/runs?${q.toString()}`,
  )
  return data.runs ?? []
}

export async function getRunLog(project: string, runId: string): Promise<string> {
  const res = await fetch(
    `/api/p/${encodeURIComponent(project)}/agents/runs/${encodeURIComponent(runId)}/log`,
    { credentials: 'include' },
  )
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${await res.text()}`)
  }
  return res.text()
}
