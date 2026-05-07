import { api } from './client'
import type { GraphData } from '@/types/api'

export function getGraph(project: string, includeReleases?: boolean) {
  const url = includeReleases
    ? `/p/${encodeURIComponent(project)}/graph?include_releases=true`
    : `/p/${encodeURIComponent(project)}/graph`
  return api.get<GraphData>(url)
}
