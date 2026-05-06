import { api } from './client'
import type { Release, ReleaseDetail, CreateReleasePayload, UpdateReleasePayload } from '@/types/release'
import type { ArtifactRow, GraphData } from '@/types/api'

export function listReleases(project: string): Promise<Release[]> {
  return api.get<Release[]>(`/p/${encodeURIComponent(project)}/releases`)
}

export function createRelease(project: string, data: CreateReleasePayload): Promise<Release> {
  return api.post<Release>(`/p/${encodeURIComponent(project)}/releases`, data)
}

export function getRelease(project: string, id: number): Promise<ReleaseDetail> {
  return api.get<ReleaseDetail>(`/p/${encodeURIComponent(project)}/releases/${id}`)
}

export function updateRelease(project: string, id: number, data: UpdateReleasePayload): Promise<Release> {
  return api.put<Release>(`/p/${encodeURIComponent(project)}/releases/${id}`, data)
}

export function deleteRelease(project: string, id: number, reassignTo?: number): Promise<{ orphaned_artifact_count: number }> {
  const qs = reassignTo !== undefined ? `?reassign_to=${reassignTo}` : ''
  return api.delete<{ orphaned_artifact_count: number }>(`/p/${encodeURIComponent(project)}/releases/${id}${qs}`)
}

export function listReleaseArtifacts(project: string, id: number): Promise<ArtifactRow[]> {
  return api.get<ArtifactRow[]>(`/p/${encodeURIComponent(project)}/releases/${id}/artifacts`)
}

export function getRoadmapGraph(project: string): Promise<GraphData> {
  return api.get<GraphData>(`/p/${encodeURIComponent(project)}/roadmap/graph`)
}
