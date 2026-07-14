// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as artifactsApi from '@/api/artifacts'

// Tracks the live count of artefacts `blocked` with a non-empty
// `## Open Questions` section for the current project (FR1, NFR2).
export const useOpenQuestionsStore = defineStore('openQuestions', () => {
  const awaitingAnswersCount = ref(0)

  async function fetchAwaitingAnswersCount(project: string): Promise<void> {
    try {
      const data = await artifactsApi.fetchAwaitingAnswersCount(project)
      awaitingAnswersCount.value = data.count ?? 0
    } catch {
      // Leave the last known count — a subsequent WS-triggered refresh will retry.
    }
  }

  return { awaitingAnswersCount, fetchAwaitingAnswersCount }
})
