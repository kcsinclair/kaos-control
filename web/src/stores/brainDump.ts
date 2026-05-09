// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { generateIdea } from '@/api/ideaChat'
import { api } from '@/api/client'
import { ApiError } from '@/api/client'
import type { IdeaGenerateResponse } from '@/types/api'

export type BrainDumpPhase = 'input' | 'generating' | 'preview' | 'editing'

export const useBrainDumpStore = defineStore('brainDump', () => {
  const input = ref('')
  const artifactType = ref<'idea' | 'defect'>('idea')
  const phase = ref<BrainDumpPhase>('input')
  const error = ref<string | null>(null)
  const proposal = ref<IdeaGenerateResponse | null>(null)
  const editedBody = ref<string | null>(null)

  const canSubmit = computed(
    () => input.value.trim().length > 0 && phase.value === 'input',
  )

  async function generate(project: string): Promise<void> {
    error.value = null
    phase.value = 'generating'
    try {
      const res = await generateIdea(project, input.value, artifactType.value)
      proposal.value = res
      phase.value = 'preview'
    } catch (e: unknown) {
      if (e instanceof ApiError) {
        error.value = e.message
      } else {
        error.value = 'Something went wrong — please try again.'
      }
      phase.value = 'input'
    }
  }

  async function acceptProposal(project: string): Promise<string | null> {
    if (!proposal.value) return null
    const p = proposal.value
    // Derive stage from target_dir: strip leading "lifecycle/" prefix
    const stage = p.target_dir.replace(/^lifecycle\//, '')
    try {
      const res = await api.post<{ artifact: { path: string } }>(
        `/p/${encodeURIComponent(project)}/artifacts`,
        {
          stage,
          slug: p.slug,
          frontmatter: p.frontmatter,
          body: p.body,
        },
      )
      return res.artifact.path
    } catch (e: unknown) {
      if (e instanceof ApiError) {
        error.value = e.message
      } else {
        error.value = 'Something went wrong — please try again.'
      }
      return null
    }
  }

  function startEdit(): void {
    if (!proposal.value) return
    editedBody.value = proposal.value.body
    phase.value = 'editing'
  }

  function applyEdit(): boolean {
    if (!editedBody.value?.trim()) {
      error.value = 'Body cannot be empty.'
      return false
    }
    if (proposal.value) {
      proposal.value = { ...proposal.value, body: editedBody.value }
    }
    editedBody.value = null
    error.value = null
    phase.value = 'preview'
    return true
  }

  function discard(): void {
    input.value = ''
    artifactType.value = 'idea'
    phase.value = 'input'
    error.value = null
    proposal.value = null
    editedBody.value = null
  }

  function reset(): void {
    discard()
  }

  return {
    input,
    artifactType,
    phase,
    error,
    proposal,
    editedBody,
    canSubmit,
    generate,
    acceptProposal,
    startEdit,
    applyEdit,
    discard,
    reset,
  }
})
