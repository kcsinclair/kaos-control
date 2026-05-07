import { api } from './client'
import type { Release, ReleaseDetail, CreateReleasePayload, UpdateReleasePayload } from '@/types/release'
import type { ArtifactRow, GraphData } from '@/types/api'

// The releases endpoints wrap their results in an envelope ({releases: [...]}
// or {release: {...}}) — same convention as the rest of the kaos-control API.
// These functions unwrap the envelope so callers receive the bare value.

export function listReleases(project: string): Promise<Release[]> {
  return api
    .get<{ releases: Release[] | null }>(`/p/${encodeURIComponent(project)}/releases`)
    .then((r) => r.releases ?? [])
}

export function createRelease(project: string, data: CreateReleasePayload): Promise<Release> {
  return api
    .post<{ release: Release }>(`/p/${encodeURIComponent(project)}/releases`, data)
    .then((r) => r.release)
}

export function getRelease(project: string, id: number): Promise<ReleaseDetail> {
  return api
    .get<{ release: ReleaseDetail }>(`/p/${encodeURIComponent(project)}/releases/${id}`)
    .then((r) => r.release)
}

export function updateRelease(project: string, id: number, data: UpdateReleasePayload): Promise<Release> {
  return api
    .put<{ release: Release }>(`/p/${encodeURIComponent(project)}/releases/${id}`, data)
    .then((r) => r.release)
}

export function deleteRelease(project: string, id: number, reassignTo?: number): Promise<{ orphaned_artifact_count: number }> {
  const qs = reassignTo !== undefined ? `?reassign_to=${reassignTo}` : ''
  return api.delete<{ orphaned_artifact_count: number }>(`/p/${encodeURIComponent(project)}/releases/${id}${qs}`)
}

export function listReleaseArtifacts(project: string, id: number): Promise<ArtifactRow[]> {
  return api
    .get<{ items: ArtifactRow[] | null }>(`/p/${encodeURIComponent(project)}/releases/${id}/artifacts`)
    .then((r) => r.items ?? [])
}

export function getRoadmapGraph(project: string): Promise<GraphData> {
  return api
    .get<{ nodes: GraphData['nodes'] | null; edges: GraphData['edges'] | null }>(
      `/p/${encodeURIComponent(project)}/releases/graph`,
    )
    .then((r) => ({ nodes: r.nodes ?? [], edges: r.edges ?? [] }))
}
