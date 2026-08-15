// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type { ArtifactRow, ArtifactDetail, ArtifactFilter, ArtifactFrontmatter, LineageSummary, OpenQuestionsResponse } from '@/types/api'
import type { RiceComponents } from '@/lib/rice'

function filterParams(f: ArtifactFilter): string {
  const p = new URLSearchParams()
  if (f.stage)    p.set('stage', f.stage)
  if (f.status)   p.set('status', f.status)
  if (f.label)    p.set('label', f.label)
  if (f.lineage)  p.set('lineage', f.lineage)
  if (f.type)     p.set('type', f.type)
  if (f.priority) p.set('priority', f.priority)
  if (f.release)  p.set('release', f.release)
  if (f.q)        p.set('q', f.q)
  if (f.sort)      p.set('sort', f.sort)
  if (f.limit !== undefined) p.set('limit', String(f.limit))
  if (f.offset)   p.set('offset', String(f.offset))
  if (f.awaiting_answers) p.set('awaiting_answers', 'true')
  const s = p.toString()
  return s ? '?' + s : ''
}

export function listArtifacts(project: string, filter: ArtifactFilter = {}) {
  return api.get<{ items: ArtifactRow[]; total: number }>(
    `/p/${encodeURIComponent(project)}/artifacts${filterParams(filter)}`,
  )
}

export function fetchAwaitingAnswersCount(project: string) {
  return api.get<{ count: number }>(
    `/p/${encodeURIComponent(project)}/artifacts?awaiting_answers=true&count_only=true`,
  )
}

export function getOpenQuestions(project: string, path: string) {
  return api.get<OpenQuestionsResponse>(
    `/p/${encodeURIComponent(project)}/artifacts/${path}/open-questions`,
  )
}

export function previewOpenQuestions(
  project: string,
  path: string,
  answers: Record<number, string>,
  complete: boolean,
) {
  return api.post<{ body: string }>(
    `/p/${encodeURIComponent(project)}/artifacts/${path}/open-questions/preview`,
    { answers, complete },
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

export function patchRice(project: string, path: string, components: RiceComponents) {
  return api.patch<{ artifact: ArtifactRow }>(
    `/p/${encodeURIComponent(project)}/artifacts/${path}/rice`,
    components,
  )
}

export function patchRelease(project: string, path: string, release: string | null) {
  return api.patch<{ artifact: ArtifactRow }>(
    `/p/${encodeURIComponent(project)}/artifacts/${path}/release`,
    { release },
  )
}

export function getAllowedTargets(project: string, path: string) {
  return api.get<{ targets: string[] }>(
    `/p/${encodeURIComponent(project)}/artifacts/${path}/allowed-targets`,
  )
}
