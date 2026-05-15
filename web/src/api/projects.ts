// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type {
  ProjectSummary,
  CreateProjectPayload,
  UpdateProjectPayload,
  CheckDirectoryResult,
  InitProjectResult,
} from '@/types/api'

export function listProjects() {
  return api.get<{ projects: ProjectSummary[] }>('/projects')
}

export function getProject(name: string) {
  return api.get<ProjectSummary>(`/projects/${encodeURIComponent(name)}`)
}

export function createProject(payload: CreateProjectPayload) {
  return api.post<ProjectSummary>('/projects', payload)
}

export function updateProject(name: string, payload: UpdateProjectPayload) {
  return api.put<ProjectSummary>(`/projects/${encodeURIComponent(name)}`, payload)
}

export function deleteProject(name: string) {
  return api.delete<{ ok: boolean }>(`/projects/${encodeURIComponent(name)}`)
}

export function initProject(name: string) {
  return api.post<InitProjectResult>(`/projects/${encodeURIComponent(name)}/init`)
}

export function checkDirectory(path: string) {
  return api.post<CheckDirectoryResult>('/projects/check-directory', { path })
}
