// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as projectsApi from '@/api/projects'
import { getConfig } from '@/api/config'
import type { ProjectSummary } from '@/types/api'

export const useProjectStore = defineStore('project', () => {
  const projects = ref<ProjectSummary[]>([])
  const current = ref<ProjectSummary | null>(null)
  const loading = ref(false)
  const initRequired = ref(false)

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

  return { projects, current, loading, initRequired, fetchProjects, setCurrent, checkInitRequired }
})
