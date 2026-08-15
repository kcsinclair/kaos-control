// SPDX-License-Identifier: AGPL-3.0-or-later

import { ref } from 'vue'
import { useProjectStore } from '@/stores/project'
import { ApiError } from '@/api/client'
import type { GenerateResult, DirectiveFileWrite } from '@/types/api'

export type DirectiveApiCall = (project: string, opts: { force: boolean }) => Promise<GenerateResult>

/**
 * Shared call → inspect diff → optional force re-call → summary refresh flow
 * for the two directive-generation entry points (migrate, refresh). A `diff`
 * on any file in a non-forced response means the file was hand-edited since
 * it was last generated (FR-11) — the caller must show it and re-invoke with
 * `force: true` to confirm the overwrite; nothing is ever applied silently.
 */
export function useDirectiveApply(project: string, call: DirectiveApiCall) {
  const projectStore = useProjectStore()

  type Phase = 'idle' | 'diff' | 'result'

  const phase = ref<Phase>('idle')
  const loading = ref(false)
  const error = ref('')
  const result = ref<GenerateResult | null>(null)
  const pendingDiff = ref<DirectiveFileWrite | null>(null)

  async function apply(force = false): Promise<void> {
    loading.value = true
    error.value = ''
    try {
      const res = await call(project, { force })
      const diffFile = !force ? res.files.find((f) => f.diff) : undefined
      if (diffFile) {
        pendingDiff.value = diffFile
        phase.value = 'diff'
        return
      }
      pendingDiff.value = null
      result.value = res
      phase.value = 'result'
      await projectStore.refreshCurrent()
    } catch (err) {
      error.value =
        err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Operation failed.'
    } finally {
      loading.value = false
    }
  }

  function reset(): void {
    phase.value = 'idle'
    error.value = ''
    result.value = null
    pendingDiff.value = null
  }

  return { phase, loading, error, result, pendingDiff, apply, reset }
}
