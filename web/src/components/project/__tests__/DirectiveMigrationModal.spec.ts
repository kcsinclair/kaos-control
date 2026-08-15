// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

// Frontend plan: lifecycle/frontend-plans/agent-directives-generation-4-fe.md — Milestone 2
//
// Verifies the migration modal's confirm → result happy path, and that a
// `diff` in the response (a hand-edited AGENTS.md) routes to an explicit
// force-overwrite step rather than silently applying (FR-11/FR-16).
//
// DirectiveMigrationModal renders via <Teleport to="body">, which portals
// outside the wrapper's own element tree — query the real document body
// instead of the wrapper (see BrainDumpModal.spec.ts for precedent).
enableAutoUnmount(afterEach)

const mockMigrateDirectives = vi.fn()

vi.mock('@/api/directives', () => ({
  migrateDirectives: (...args: unknown[]) => mockMigrateDirectives(...args),
}))

import DirectiveMigrationModal from '../DirectiveMigrationModal.vue'

describe('DirectiveMigrationModal', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('migrates without a diff and shows the file report', async () => {
    mockMigrateDirectives.mockResolvedValue({
      files: [
        { path: 'AGENTS.md', created: true, changed: false, skipped: false },
        { path: 'CLAUDE.md', created: false, changed: true, skipped: false },
      ],
    })

    mount(DirectiveMigrationModal, { props: { project: 'testproject' } })
    const body = new DOMWrapper(document.body)

    const migrateBtn = body.findAll('button').find((b) => b.text().includes('Migrate now'))
    await migrateBtn?.trigger('click')
    await flushPromises()

    expect(mockMigrateDirectives).toHaveBeenCalledWith('testproject', { force: false })
    expect(body.text()).toContain('AGENTS.md')
    expect(body.text()).toContain('CLAUDE.md')
  })

  it('routes a diff response to an explicit overwrite step instead of applying silently', async () => {
    mockMigrateDirectives.mockResolvedValueOnce({
      files: [{ path: 'AGENTS.md', created: false, changed: false, skipped: false, diff: '-old\n+new' }],
    })

    mount(DirectiveMigrationModal, { props: { project: 'testproject' } })
    const body = new DOMWrapper(document.body)

    const migrateBtn = body.findAll('button').find((b) => b.text().includes('Migrate now'))
    await migrateBtn?.trigger('click')
    await flushPromises()

    // Diff phase: shown, and no second call yet — force overwrite requires confirmation.
    expect(body.text()).toContain('-old')
    expect(mockMigrateDirectives).toHaveBeenCalledTimes(1)

    mockMigrateDirectives.mockResolvedValueOnce({
      files: [{ path: 'AGENTS.md', created: false, changed: true, skipped: false }],
    })
    const overwriteBtn = body.findAll('button').find((b) => b.text().includes('Overwrite'))
    await overwriteBtn?.trigger('click')
    await flushPromises()

    expect(mockMigrateDirectives).toHaveBeenLastCalledWith('testproject', { force: true })
    expect(body.text()).toContain('changed')
  })
})
