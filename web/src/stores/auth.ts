// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as authApi from '@/api/auth'
import type { MeResponse } from '@/types/api'

export const useAuthStore = defineStore('auth', () => {
  const me = ref<MeResponse | null>(null)
  const initialized = ref(false)
  const loading = ref(false)

  const isAuthenticated = computed(() => me.value !== null)

  async function fetchMe(): Promise<void> {
    try {
      me.value = await authApi.fetchMe()
    } catch {
      me.value = null
    } finally {
      initialized.value = true
    }
  }

  async function login(email: string, password: string): Promise<void> {
    loading.value = true
    try {
      await authApi.login(email, password)
      await fetchMe()
    } finally {
      loading.value = false
    }
  }

  async function logout(): Promise<void> {
    try {
      await authApi.logout()
    } finally {
      me.value = null
    }
  }

  function rolesForProject(projectName: string): string[] {
    return me.value?.roles[projectName] ?? []
  }

  return { me, initialized, loading, isAuthenticated, fetchMe, login, logout, rolesForProject }
})
