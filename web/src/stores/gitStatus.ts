// SPDX-License-Identifier: AGPL-3.0-or-later

import { ref } from 'vue'
import { defineStore } from 'pinia'
import { fetchGitStatus } from '@/api/git'
import type { GitStatusResponse } from '@/types/api'

export const useGitStatusStore = defineStore('gitStatus', () => {
  const available = ref(false)
  const branch = ref('')
  const dirty = ref(false)
  const headSha = ref('')
  const headMessage = ref('')
  const headAuthor = ref('')
  const headWhen = ref('')

  function reset() {
    available.value = false
    branch.value = ''
    dirty.value = false
    headSha.value = ''
    headMessage.value = ''
    headAuthor.value = ''
    headWhen.value = ''
  }

  async function fetch(project: string) {
    reset()
    try {
      const res = await fetchGitStatus(project)
      applyWsEvent(res)
    } catch {
      available.value = false
    }
  }

  function applyWsEvent(data: GitStatusResponse) {
    available.value = data.available
    branch.value = data.branch ?? ''
    dirty.value = data.dirty ?? false
    headSha.value = data.head_sha ?? ''
    headMessage.value = data.head_message ?? ''
    headAuthor.value = data.head_author ?? ''
    headWhen.value = data.head_when ?? ''
  }

  return { available, branch, dirty, headSha, headMessage, headAuthor, headWhen, fetch, applyWsEvent }
})
