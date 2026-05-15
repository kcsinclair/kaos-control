// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as projectsApi from '@/api/projects'
import { getConfig } from '@/api/config'
import type {
  ProjectSummary,
  CreateProjectPayload,
  UpdateProjectPayload,
  CheckDirectoryResult,
} from '@/types/api'

export const useProjectStore = defineStore('project', () => {
  const projects = ref<ProjectSummary[]>([])
  const current = ref<ProjectSummary | null>(null)
  const loading = ref(false)
  const initRequired = ref(false)

  // Mutation state (create / update / delete / init)
  const mutating = ref(false)
  const error = ref<string | null>(null)

  async function fetchProjects(): Promise<void> {
    loading.value = true
    try {
      const data = await projectsApi.listProjects()
      projects.value = data.projects ?? []
    } finally {
      loading.value = false
    }
  }

  function setCurrent(name: string): void {
    initRequired.value = false
    current.value = projects.value.find((p) => p.name === name) ?? null
  }

  async function checkInitRequired(project: string): Promise<void> {
    try {
      const data = await getConfig(project)
      initRequired.value = data.raw === ''
    } catch {
      initRequired.value = false
    }
  }

  async function create(payload: CreateProjectPayload): Promise<ProjectSummary> {
    mutating.value = true
    error.value = null
    try {
      const result = await projectsApi.createProject(payload)
      await fetchProjects()
      return result
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to create project'
      throw err
    } finally {
      mutating.value = false
    }
  }

  async function update(name: string, payload: UpdateProjectPayload): Promise<ProjectSummary> {
    mutating.value = true
    error.value = null
    try {
      const result = await projectsApi.updateProject(name, payload)
      await fetchProjects()
      return result
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to update project'
      throw err
    } finally {
      mutating.value = false
    }
  }

  async function remove(name: string): Promise<void> {
    mutating.value = true
    error.value = null
    try {
      await projectsApi.deleteProject(name)
      if (current.value?.name === name) {
        current.value = null
      }
      await fetchProjects()
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to delete project'
      throw err
    } finally {
      mutating.value = false
    }
  }

  async function init(name: string) {
    mutating.value = true
    error.value = null
    try {
      const result = await projectsApi.initProject(name)
      await fetchProjects()
      return result
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to initialise project'
      throw err
    } finally {
      mutating.value = false
    }
  }

  async function checkDirectory(path: string): Promise<CheckDirectoryResult> {
    return projectsApi.checkDirectory(path)
  }

  return {
    projects,
    current,
    loading,
    initRequired,
    mutating,
    error,
    fetchProjects,
    setCurrent,
    checkInitRequired,
    create,
    update,
    remove,
    init,
    checkDirectory,
  }
})
