// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchVersion as apiFetchVersion } from '@/api/version'

export const useAppStore = defineStore('app', () => {
  const version = ref('unknown')
  let fetched = false

  async function fetchVersion(): Promise<void> {
    if (fetched) return
    fetched = true
    version.value = await apiFetchVersion()
  }

  return { version, fetchVersion }
})
