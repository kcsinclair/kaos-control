import { api } from './client'
import type { ArtifactRow, ArtifactDetail, ArtifactFilter, ArtifactFrontmatter, LineageSummary } from '@/types/api'

function filterParams(f: ArtifactFilter): string {
  const p = new URLSearchParams()
  if (f.stage)    p.set('stage', f.stage)
  if (f.status)   p.set('status', f.status)
  if (f.label)    p.set('label', f.label)
  if (f.lineage)  p.set('lineage', f.lineage)
  if (f.type)     p.set('type', f.type)
  if (f.priority) p.set('priority', f.priority)
  if (f.limit)    p.set('limit', String(f.limit))
  if (f.offset)   p.set('offset', String(f.offset))
  const s = p.toString()
  return s ? '?' + s : ''
}

export function listArtifacts(project: string, filter: ArtifactFilter = {}) {
  return api.get<{ items: ArtifactRow[]; total: number }>(
    `/p/${encodeURIComponent(project)}/artifacts${filterParams(filter)}`,
  )
}

export function getArtifact(project: string, path: string) {
  return api.get<{ artifact: ArtifactRow; body: string; body_html: string }>(
    `/p/${encodeURIComponent(project)}/artifacts/${path}`,
  )
}

export function listLabels(project: string) {
  return api.get<{ labels: string[] }>(`/p/${encodeURIComponent(project)}/labels`)
}

export function listPriorities(project: string) {
  return api.get<{ priorities: string[] }>(`/p/${encodeURIComponent(project)}/priorities`)
}

export function listLineages(project: string) {
  return api.get<{ lineages: LineageSummary[] }>(`/p/${encodeURIComponent(project)}/lineages`)
}

export function transitionArtifact(
  project: string,
  path: string,
  to: string,
  comment?: string,
) {
  return api.post<{ artifact: ArtifactRow; rejection_artifact?: string }>(
    `/p/${encodeURIComponent(project)}/artifacts/${path}/transition`,
    { to, comment },
  )
}

export function updateArtifact(
  project: string,
  path: string,
  payload: { frontmatter: ArtifactFrontmatter; body: string; expected_sha?: string },
) {
  return api.put<{ artifact: ArtifactRow }>(
    `/p/${encodeURIComponent(project)}/artifacts/${path}`,
    payload,
  )
}

export function patchPriority(project: string, path: string, priority: string | null) {
  return api.patch<{ artifact: ArtifactRow }>(
    `/p/${encodeURIComponent(project)}/artifacts/${path}/priority`,
    { priority },
  )
}
