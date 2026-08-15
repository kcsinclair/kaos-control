// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

// Frontend plan: lifecycle/frontend-plans/agent-directives-generation-4-fe.md — Milestone 3
//
// Verifies the refresh panel calls refreshDirectives, renders the file
// report + disabledAgents, and gates a diff response behind an explicit
// overwrite (shared useDirectiveApply flow, also covered directly in
// useDirectiveApply.spec.ts).

const mockRefreshDirectives = vi.fn()

vi.mock('@/api/directives', () => ({
  refreshDirectives: (...args: unknown[]) => mockRefreshDirectives(...args),
}))

import DirectiveRefreshPanel from '../DirectiveRefreshPanel.vue'

describe('DirectiveRefreshPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('calls refreshDirectives and renders the file report + disabledAgents', async () => {
    mockRefreshDirectives.mockResolvedValue({
      files: [
        { path: 'AGENTS.md', created: false, changed: true, skipped: false },
        { path: 'GEMINI.md', created: true, changed: false, skipped: false },
      ],
      disabledAgents: ['backend-developer'],
    })

    const wrapper = mount(DirectiveRefreshPanel, { props: { project: 'testproject' } })
    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(mockRefreshDirectives).toHaveBeenCalledWith('testproject', { force: false })
    expect(wrapper.text()).toContain('AGENTS.md')
    expect(wrapper.text()).toContain('GEMINI.md')
    expect(wrapper.text()).toContain('backend-developer')
  })

  it('gates a diff response behind an explicit overwrite', async () => {
    mockRefreshDirectives.mockResolvedValueOnce({
      files: [{ path: 'AGENTS.md', created: false, changed: false, skipped: false, diff: '-old\n+new' }],
    })

    const wrapper = mount(DirectiveRefreshPanel, { props: { project: 'testproject' } })
    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('-old')
    expect(mockRefreshDirectives).toHaveBeenCalledTimes(1)

    mockRefreshDirectives.mockResolvedValueOnce({
      files: [{ path: 'AGENTS.md', created: false, changed: true, skipped: false }],
    })
    const overwriteBtn = wrapper.findAll('button').find((b) => b.text().includes('Overwrite'))
    await overwriteBtn?.trigger('click')
    await flushPromises()

    expect(mockRefreshDirectives).toHaveBeenLastCalledWith('testproject', { force: true })
    expect(wrapper.text()).toContain('changed')
  })

  it('emits done when the result phase Done button is clicked', async () => {
    mockRefreshDirectives.mockResolvedValue({ files: [] })

    const wrapper = mount(DirectiveRefreshPanel, { props: { project: 'testproject' } })
    await wrapper.find('button').trigger('click')
    await flushPromises()

    const doneBtn = wrapper.findAll('button').find((b) => b.text() === 'Done')
    await doneBtn?.trigger('click')

    expect(wrapper.emitted('done')).toBeTruthy()
  })
})
