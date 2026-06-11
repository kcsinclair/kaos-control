// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type { Release, ReleaseDetail, CreateReleasePayload, UpdateReleasePayload } from '@/types/release'
import type { ArtifactRow, GraphData } from '@/types/api'

// The releases endpoints wrap their results in an envelope ({releases: [...]}
// or {release: {...}}) — same convention as the rest of the kaos-control API.
// These functions unwrap the envelope so callers receive the bare value.

// The backend uses `omitempty` on start_date and end_date, so nil dates are
// absent from JSON rather than null. Normalise them to null here so the rest
// of the frontend can rely on the Release type contract (string | null).
function normaliseDates<T extends Release>(r: T): T {
  return {
    ...r,
    start_date: (r.start_date as string | null | undefined) ?? null,
    end_date: (r.end_date as string | null | undefined) ?? null,
    file_path: (r.file_path as string | undefined) ?? '',
    slug: (r.slug as string | undefined) ?? '',
  }
}

export function listReleases(project: string): Promise<Release[]> {
  return api
    .get<{ releases: Release[] | null }>(`/p/${encodeURIComponent(project)}/releases`)
    .then((r) => (r.releases ?? []).map(normaliseDates))
}

export function createRelease(project: string, data: CreateReleasePayload): Promise<Release> {
  return api
    .post<{ release: Release }>(`/p/${encodeURIComponent(project)}/releases`, data)
    .then((r) => normaliseDates(r.release))
}

export function getRelease(project: string, id: number): Promise<ReleaseDetail> {
  return api
    .get<{ release: ReleaseDetail }>(`/p/${encodeURIComponent(project)}/releases/${id}`)
    .then((r) => normaliseDates(r.release))
}

export function updateRelease(project: string, id: number, data: UpdateReleasePayload): Promise<Release> {
  return api
    .put<{ release: Release }>(`/p/${encodeURIComponent(project)}/releases/${id}`, data)
    .then((r) => normaliseDates(r.release))
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

export function rehydrateReleases(project: string): Promise<{ inserted: number; skipped: number; errors: string[] }> {
  return api.post<{ inserted: number; skipped: number; errors: string[] }>(
    `/p/${encodeURIComponent(project)}/releases/rehydrate`,
  )
}

export function getRoadmapGraph(project: string): Promise<GraphData> {
  return api
    .get<{ nodes: GraphData['nodes'] | null; edges: GraphData['edges'] | null }>(
      `/p/${encodeURIComponent(project)}/releases/graph`,
    )
    .then((r) => ({ nodes: r.nodes ?? [], edges: r.edges ?? [] }))
}
