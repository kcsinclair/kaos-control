import { api } from './client'

export interface StaleChild {
  path: string
  status: string
}

export interface StaleArtifact {
  path: string
  lineage: string
  title?: string
  current_status: string
  suggested_status: string
  reason: string
  children: StaleChild[]
  can_advance: boolean
  blocked_reason?: string
}

export interface StatusCheckResponse {
  stale: StaleArtifact[]
}

export interface AdvanceResult {
  path: string
  outcome: string
  ok: boolean
  advanced_to?: string
  reason?: string
}

export interface AdvanceResponse {
  results: AdvanceResult[]
}

export function checkStatus(project: string, lineage?: string): Promise<StatusCheckResponse> {
  const params = lineage ? `?lineage=${encodeURIComponent(lineage)}` : ''
  return api.get<StatusCheckResponse>(`/p/${encodeURIComponent(project)}/status-check${params}`)
}

export function advanceStatuses(project: string, paths: string[]): Promise<AdvanceResponse> {
  return api.post<AdvanceResponse>(`/p/${encodeURIComponent(project)}/status-check/advance`, { paths })
}
