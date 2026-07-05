// SPDX-License-Identifier: AGPL-3.0-or-later

import { ref, reactive } from 'vue'
import * as artifactsApi from '@/api/artifacts'
import type { ArtifactFrontmatter, OpenQuestion } from '@/types/api'

// Holds the guided-resolution state for one artefact's `## Open Questions`
// section: the parsed questions, the in-progress answers keyed by question
// index, and the save (partial) / finish (complete) actions. Persistence
// always goes through the compute-only preview endpoint followed by the
// existing artefact PUT — body-only, never `status` (NFR1).
export function useOpenQuestions(project: string, path: string) {
  const questions = ref<OpenQuestion[]>([])
  const answers = reactive<Record<number, string>>({})
  const loading = ref(false)
  const saving = ref(false)
  const error = ref<string | null>(null)

  async function load(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const data = await artifactsApi.getOpenQuestions(project, path)
      questions.value = data.questions ?? []
      for (const key of Object.keys(answers)) delete answers[Number(key)]
      for (const q of questions.value) {
        answers[q.index] = q.answer ?? ''
      }
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load questions'
    } finally {
      loading.value = false
    }
  }

  async function persist(frontmatter: ArtifactFrontmatter, complete: boolean): Promise<void> {
    saving.value = true
    error.value = null
    try {
      const { body } = await artifactsApi.previewOpenQuestions(project, path, { ...answers }, complete)
      await artifactsApi.updateArtifact(project, path, { frontmatter, body })
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to save answers'
      throw e
    } finally {
      saving.value = false
    }
  }

  function save(frontmatter: ArtifactFrontmatter): Promise<void> {
    return persist(frontmatter, false)
  }

  function finish(frontmatter: ArtifactFrontmatter): Promise<void> {
    return persist(frontmatter, true)
  }

  return { questions, answers, loading, saving, error, load, save, finish }
}
