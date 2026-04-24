import { api } from './client'
import type { ProjectSummary } from '@/types/api'

export function listProjects() {
  return api.get<{ projects: ProjectSummary[] }>('/projects')
}
