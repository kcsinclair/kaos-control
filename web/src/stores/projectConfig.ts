import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getRoles } from '@/api/config'

export const useProjectConfigStore = defineStore('projectConfig', () => {
  const roles = ref<string[]>([])
  const users = ref<{ email: string; roles: string[] }[]>([])
  const loaded = ref(false)

  async function fetchRoles(project: string): Promise<void> {
    if (loaded.value) return
    const data = await getRoles(project)
    roles.value = data.roles ?? []
    users.value = data.users ?? []
    loaded.value = true
  }

  const availableWhoOptions = computed<string[]>(() => {
    const emails = users.value.map((u) => u.email)
    const unique = Array.from(new Set(emails))
    return ['agent', ...unique]
  })

  return { roles, users, loaded, fetchRoles, availableWhoOptions }
})
