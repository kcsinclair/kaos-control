// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Frontend plan: lifecycle/frontend-plans/agent-directives-generation-4-fe.md — Milestone 1
//
// Verifies the store surfaces directivesMigrationAvailable from the loaded
// project summary, and that refreshCurrent() re-points `current` at fresh data.

vi.mock('@/api/projects', () => ({
  listProjects: vi.fn(),
}))

import { listProjects } from '@/api/projects'
import { useProjectStore } from '@/stores/project'
import type { ProjectSummary } from '@/types/api'

function makeSummary(overrides: Partial<ProjectSummary> = {}): ProjectSummary {
  return {
    name: 'kaos-control',
    description: '',
    path: '/tmp/kaos-control',
    owner: 'keith',
    initialised: true,
    directivesMigrationAvailable: false,
    ...overrides,
  }
}

describe('project store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('exposes directivesMigrationAvailable false when unset or current is null', () => {
    const store = useProjectStore()
    expect(store.directivesMigrationAvailable).toBe(false)
  })

  it('exposes directivesMigrationAvailable from the current project summary', async () => {
    vi.mocked(listProjects).mockResolvedValue({
      projects: [makeSummary({ directivesMigrationAvailable: true })],
    })
    const store = useProjectStore()
    await store.fetchProjects()
    store.setCurrent('kaos-control')

    expect(store.directivesMigrationAvailable).toBe(true)
  })

  it('refreshCurrent() re-fetches and re-points current at the refreshed summary', async () => {
    vi.mocked(listProjects).mockResolvedValueOnce({
      projects: [makeSummary({ directivesMigrationAvailable: true })],
    })
    const store = useProjectStore()
    await store.fetchProjects()
    store.setCurrent('kaos-control')
    expect(store.directivesMigrationAvailable).toBe(true)

    vi.mocked(listProjects).mockResolvedValueOnce({
      projects: [makeSummary({ directivesMigrationAvailable: false })],
    })
    await store.refreshCurrent()

    expect(store.directivesMigrationAvailable).toBe(false)
    expect(store.current?.name).toBe('kaos-control')
  })
})
