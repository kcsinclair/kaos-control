import { api } from './client'
import type { GraphData } from '@/types/api'

export function getGraph(project: string) {
  return api.get<GraphData>(`/p/${encodeURIComponent(project)}/graph`)
}
