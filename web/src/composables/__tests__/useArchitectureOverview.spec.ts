// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import type { WsEvent } from '@/types/api'

// Frontend plan: lifecycle/frontend-plans/architecture-overview-view-4-fe.md
// — Milestone F1 (FR-12).

const handlers = new Map<string, (e: WsEvent) => void>()
const unsub = vi.fn()

vi.mock('@/api/ws', () => ({
  getProjectWs: () => ({
    onType: (type: string, handler: (e: WsEvent) => void) => {
      handlers.set(type, handler)
      return unsub
    },
  }),
}))

const overviewFixture = {
  has_chosen_architecture: true,
  chosen_architecture: null,
  chosen_stack: null,
  summary: null,
  standards: [],
  adrs: [],
  archive: [],
  catalog: [],
}

const getOverview = vi.fn()
vi.mock('@/api/architecture', () => ({
  getOverview: (...args: unknown[]) => getOverview(...args),
}))

import { useArchitectureOverview } from '../useArchitectureOverview'

function mountComposable(project: string) {
  let result!: ReturnType<typeof useArchitectureOverview>
  mount(defineComponent({
    setup() {
      result = useArchitectureOverview(project)
      return () => h('div')
    },
  }))
  return result
}

describe('useArchitectureOverview', () => {
  beforeEach(() => {
    handlers.clear()
    unsub.mockClear()
    getOverview.mockReset()
    getOverview.mockResolvedValue(overviewFixture)
  })

  it('fetches the overview on mount and populates the model', async () => {
    const result = mountComposable('proj')
    await vi.waitFor(() => expect(getOverview).toHaveBeenCalledWith('proj'))
    await vi.waitFor(() => expect(result.overview.value).toEqual(overviewFixture))
    expect(result.hasChosenArchitecture.value).toBe(true)
    expect(result.error.value).toBeNull()
  })

  it('sets error and does not throw on an API failure', async () => {
    getOverview.mockReset()
    getOverview.mockRejectedValue(new Error('boom'))
    const result = mountComposable('proj')
    await vi.waitFor(() => expect(result.error.value).toBe('boom'))
    expect(result.overview.value).toBeNull()
  })

  it('reloads exactly once per artifact.indexed event', async () => {
    const result = mountComposable('proj')
    await vi.waitFor(() => expect(getOverview).toHaveBeenCalledTimes(1))

    handlers.get('artifact.indexed')!({ type: 'artifact.indexed', payload: {} })
    await vi.waitFor(() => expect(getOverview).toHaveBeenCalledTimes(2))
    expect(result.overview.value).toEqual(overviewFixture)
  })

  it('reloads exactly once per file.changed event', async () => {
    mountComposable('proj')
    await vi.waitFor(() => expect(getOverview).toHaveBeenCalledTimes(1))

    handlers.get('file.changed')!({ type: 'file.changed', payload: {} })
    await vi.waitFor(() => expect(getOverview).toHaveBeenCalledTimes(2))
  })
})
