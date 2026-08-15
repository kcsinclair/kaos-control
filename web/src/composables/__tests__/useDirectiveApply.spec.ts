// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

// Frontend plan: lifecycle/frontend-plans/agent-directives-generation-4-fe.md — Milestone 3
//
// Verifies the shared call → inspect diff → optional force re-call →
// summary refresh flow consumed by both DirectiveMigrationModal (M2) and
// DirectiveRefreshPanel (M3): a non-forced response carrying a `diff` routes
// to the 'diff' phase without applying; re-calling with force:true applies
// and refreshes the project summary (FR-11).

const mockRefreshCurrent = vi.fn()

vi.mock('@/stores/project', () => ({
  useProjectStore: () => ({ refreshCurrent: mockRefreshCurrent }),
}))

import { useDirectiveApply } from '../useDirectiveApply'
import type { DirectiveApiCall } from '../useDirectiveApply'

function mountComposable(project: string, call: DirectiveApiCall) {
  let result!: ReturnType<typeof useDirectiveApply>
  mount(defineComponent({
    setup() {
      result = useDirectiveApply(project, call)
      return () => h('div')
    },
  }))
  return result
}

describe('useDirectiveApply', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('applies directly and refreshes the summary when there is no diff', async () => {
    const call = vi.fn().mockResolvedValue({
      files: [{ path: 'AGENTS.md', created: true, changed: false, skipped: false }],
    })
    const { apply, phase, result } = mountComposable('testproject', call)

    await apply(false)

    expect(call).toHaveBeenCalledWith('testproject', { force: false })
    expect(phase.value).toBe('result')
    expect(result.value?.files).toHaveLength(1)
    expect(mockRefreshCurrent).toHaveBeenCalledOnce()
  })

  it('routes a diff response to the diff phase without applying or refreshing', async () => {
    const call = vi.fn().mockResolvedValue({
      files: [{ path: 'AGENTS.md', created: false, changed: false, skipped: false, diff: '-old\n+new' }],
    })
    const { apply, phase, pendingDiff } = mountComposable('testproject', call)

    await apply(false)

    expect(phase.value).toBe('diff')
    expect(pendingDiff.value?.diff).toBe('-old\n+new')
    expect(mockRefreshCurrent).not.toHaveBeenCalled()
  })

  it('a forced re-call applies and refreshes even if a diff is present', async () => {
    const call = vi.fn().mockResolvedValue({
      files: [{ path: 'AGENTS.md', created: false, changed: true, skipped: false, diff: '-old\n+new' }],
    })
    const { apply, phase } = mountComposable('testproject', call)

    await apply(true)

    expect(call).toHaveBeenCalledWith('testproject', { force: true })
    expect(phase.value).toBe('result')
    expect(mockRefreshCurrent).toHaveBeenCalledOnce()
  })

  it('surfaces a failed call as an error without changing phase', async () => {
    const call = vi.fn().mockRejectedValue(new Error('boom'))
    const { apply, phase, error } = mountComposable('testproject', call)

    await apply(false)

    expect(phase.value).toBe('idle')
    expect(error.value).toBe('boom')
    expect(mockRefreshCurrent).not.toHaveBeenCalled()
  })
})
