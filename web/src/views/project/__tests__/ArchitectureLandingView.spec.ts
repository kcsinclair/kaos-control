// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

// Frontend plan: lifecycle/frontend-plans/architecture-overview-view-4-fe.md
// — Milestone F4 (FR-1): the Architecture section landing resolves to the
// overview when a chosen architecture exists, else the relationship map.

const push = vi.fn()
const replace = vi.fn()

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { project: 'testproject' } }),
  useRouter: () => ({ push, replace }),
}))

const getOverview = vi.fn()
vi.mock('@/api/architecture', () => ({
  getOverview: (...args: unknown[]) => getOverview(...args),
}))

import ArchitectureLandingView from '../ArchitectureLandingView.vue'

describe('ArchitectureLandingView', () => {
  beforeEach(() => {
    replace.mockClear()
    getOverview.mockReset()
  })

  it('redirects to the overview when a chosen architecture exists', async () => {
    getOverview.mockResolvedValue({ has_chosen_architecture: true })
    mount(ArchitectureLandingView)
    await flushPromises()
    expect(replace).toHaveBeenCalledWith('/p/testproject/architecture/overview')
  })

  it('redirects to the relationship map when the zone is empty (catalog only)', async () => {
    getOverview.mockResolvedValue({
      has_chosen_architecture: false,
      chosen_stack: null,
      summary: null,
      standards: [],
      adrs: [],
      archive: [],
      catalog: [{ path: 'lifecycle/architecture/architectures/x.md' }],
    })
    mount(ArchitectureLandingView)
    await flushPromises()
    expect(replace).toHaveBeenCalledWith('/p/testproject/architecture/map')
  })

  it('redirects to the overview when a summary exists but no architecture is chosen', async () => {
    getOverview.mockResolvedValue({
      has_chosen_architecture: false,
      chosen_stack: null,
      summary: { path: 'lifecycle/architecture/architecture-summary.md' },
      standards: [],
      adrs: [],
      archive: [],
    })
    mount(ArchitectureLandingView)
    await flushPromises()
    expect(replace).toHaveBeenCalledWith('/p/testproject/architecture/overview')
  })

  it('redirects to the overview when ADRs exist but no architecture is chosen', async () => {
    getOverview.mockResolvedValue({
      has_chosen_architecture: false,
      chosen_stack: null,
      summary: null,
      standards: [],
      adrs: [{ path: 'lifecycle/architecture/decisions/adr-0001-x.md' }],
      archive: [],
    })
    mount(ArchitectureLandingView)
    await flushPromises()
    expect(replace).toHaveBeenCalledWith('/p/testproject/architecture/overview')
  })

  it('degrades to the relationship map on an API error (NFR-5)', async () => {
    getOverview.mockRejectedValue(new Error('boom'))
    mount(ArchitectureLandingView)
    await flushPromises()
    expect(replace).toHaveBeenCalledWith('/p/testproject/architecture/map')
  })
})
