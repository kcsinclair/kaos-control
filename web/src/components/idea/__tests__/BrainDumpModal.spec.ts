// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

// BrainDumpModal renders via <Teleport to="body">, which portals outside the
// wrapper's own element tree — query the real document body instead of the
// wrapper. enableAutoUnmount removes teleported nodes between tests.
enableAutoUnmount(afterEach)

// Frontend plan: lifecycle/frontend-plans/defect-generate-missing-template-3-fe.md
// §Milestone 2 — modal presentation; §Milestone 4 — these tests.
//
// Verifies the modal renders the mapped error as an actionable alert, only
// offers the manual-entry escape hatch for the config/template error class on
// a defect, and never leaks the raw "has no template" string.

vi.mock('@/api/ideaChat', () => ({
  generateIdea: vi.fn(),
}))
vi.mock('@/api/client', () => {
  const postMock = vi.fn()
  return {
    api: { post: postMock },
    ApiError: class ApiError extends Error {
      constructor(public code: string, message: string, public status: number) {
        super(message)
        this.name = 'ApiError'
      }
    },
  }
})

import { api, ApiError } from '@/api/client'
import { generateIdea } from '@/api/ideaChat'
import BrainDumpModal from '../BrainDumpModal.vue'

describe('BrainDumpModal — actionable error + manual escape hatch', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('shows the actionable alert and a manual-entry action for a config/template error', async () => {
    vi.mocked(generateIdea).mockRejectedValue(
      new ApiError('template_unavailable', 'idea-capture agent has no template "defect-generate"', 422),
    )

    mount(BrainDumpModal, {
      props: { project: 'testproject', artifactType: 'defect' },
    })
    const body = new DOMWrapper(document.body)
    await body.find('textarea').setValue('The submit button does nothing when clicked.')
    await body.find('.btn-primary').trigger('click')
    await flushPromises()

    expect(body.text()).not.toContain('has no template')
    expect(body.find('.bdm-error').text()).toContain("Defect generation isn't configured")
    expect(body.find('.bdm-error-manual-btn').exists()).toBe(true)
  })

  it('omits the manual-entry action for a generic (non-config) error', async () => {
    vi.mocked(generateIdea).mockRejectedValue(new ApiError('rate_limited', 'Too many requests', 429))

    mount(BrainDumpModal, {
      props: { project: 'testproject', artifactType: 'defect' },
    })
    const body = new DOMWrapper(document.body)
    await body.find('textarea').setValue('The submit button does nothing when clicked.')
    await body.find('.btn-primary').trigger('click')
    await flushPromises()

    expect(body.find('.bdm-error').text()).toContain('Too many requests')
    expect(body.find('.bdm-error-manual-btn').exists()).toBe(false)
  })

  it('creates a defect artifact via the manual action and emits created', async () => {
    vi.mocked(generateIdea).mockRejectedValue(
      new ApiError('template_unavailable', 'idea-capture agent has no template "defect-generate"', 422),
    )
    vi.mocked(api.post).mockResolvedValue({ artifact: { path: 'lifecycle/defects/submit-button-broken.md' } })

    const wrapper = mount(BrainDumpModal, {
      props: { project: 'testproject', artifactType: 'defect' },
    })
    const body = new DOMWrapper(document.body)
    await body.find('textarea').setValue('The submit button does nothing when clicked.')
    await body.find('.btn-primary').trigger('click')
    await flushPromises()

    await body.find('.bdm-error-manual-btn').trigger('click')
    await flushPromises()

    expect(vi.mocked(api.post)).toHaveBeenCalledWith(
      '/p/testproject/artifacts',
      expect.objectContaining({ stage: 'defects' }),
    )
    expect(wrapper.emitted('created')).toBeTruthy()
    expect(wrapper.emitted('created')![0]).toEqual(['lifecycle/defects/submit-button-broken.md'])
  })
})
