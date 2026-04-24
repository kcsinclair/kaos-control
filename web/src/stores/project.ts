import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as projectsApi from '@/api/projects'
import type { ProjectSummary } from '@/types/api'

export const useProjectStore = defineStore('project', () => {
  const projects = ref<ProjectSummary[]>([])
  const current = ref<ProjectSummary | null>(null)
  const loading = ref(false)

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
    current.value = projects.value.find((p) => p.name === name) ?? null
  }

  return { projects, current, loading, fetchProjects, setCurrent }
})
