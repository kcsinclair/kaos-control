// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'
import { converseIdea } from '@/api/ideaChat'
import { useUiStore } from '@/stores/ui'
import { ApiError } from '@/api/client'

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export type ConversationStatus = 'idle' | 'conversing' | 'proposed' | 'created'

export const useIdeaChatStore = defineStore('ideaChat', () => {
  const sessionId = ref<string | null>(null)
  const messages = ref<ChatMessage[]>([])
  const status = ref<ConversationStatus>('idle')
  const loading = ref(false)
  const preview = ref<{ frontmatter: Record<string, unknown>; body: string } | null>(null)
  const createdPath = ref<string | null>(null)

  function reset(): void {
    sessionId.value = null
    messages.value = []
    status.value = 'idle'
    loading.value = false
    preview.value = null
    createdPath.value = null
  }

  async function sendMessage(project: string, text: string): Promise<void> {
    messages.value.push({ role: 'user', content: text })
    loading.value = true
    try {
      const res = await converseIdea(project, sessionId.value, text)
      sessionId.value = res.session_id
      messages.value.push({ role: 'assistant', content: res.reply })
      status.value = res.status
      preview.value = res.preview
      if (res.artifact_path) {
        createdPath.value = res.artifact_path
      }
    } catch (e: unknown) {
      const msg = e instanceof ApiError ? e.message : 'Failed to send message'
      useUiStore().error(msg)
    } finally {
      loading.value = false
    }
  }

  async function acceptProposal(project: string): Promise<void> {
    loading.value = true
    try {
      const res = await converseIdea(project, sessionId.value, '__accept__')
      sessionId.value = res.session_id
      messages.value.push({ role: 'assistant', content: res.reply })
      status.value = 'created'
      if (res.artifact_path) {
        createdPath.value = res.artifact_path
      }
    } catch (e: unknown) {
      const msg = e instanceof ApiError ? e.message : 'Failed to accept proposal'
      useUiStore().error(msg)
    } finally {
      loading.value = false
    }
  }

  async function rejectProposal(project: string): Promise<void> {
    loading.value = true
    try {
      await converseIdea(project, sessionId.value, '__reject__')
      reset()
    } catch (e: unknown) {
      const msg = e instanceof ApiError ? e.message : 'Failed to discard proposal'
      useUiStore().error(msg)
      loading.value = false
    }
  }

  return {
    sessionId,
    messages,
    status,
    loading,
    preview,
    createdPath,
    reset,
    sendMessage,
    acceptProposal,
    rejectProposal,
  }
})
