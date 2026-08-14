// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { shallowMount, flushPromises } from '@vue/test-utils'

// Frontend plan: lifecycle/frontend-plans/architectural-artefacts-4-fe.md — Milestone 3
//
// Verifies the minimal "propose ADR" affordance: previewing the next ADR number on open,
// and submitting with status: 'draft' by default, then emitting the created path.

const mockNextAdrNumber = vi.fn()
const mockCreateAdr = vi.fn()

vi.mock('@/api/architecture', () => ({
  nextAdrNumber: (...args: unknown[]) => mockNextAdrNumber(...args),
  createAdr: (...args: unknown[]) => mockCreateAdr(...args),
}))

import NewAdrModal from '../NewAdrModal.vue'

describe('NewAdrModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockNextAdrNumber.mockResolvedValue(4)
    mockCreateAdr.mockResolvedValue({ path: 'lifecycle/architecture/decisions/adr-0004-use-grpc.md', number: 4 })
  })

  it('fetches and shows the previewed next ADR number on mount', async () => {
    const wrapper = shallowMount(NewAdrModal, { props: { project: 'testproject' } })
    await flushPromises()

    expect(mockNextAdrNumber).toHaveBeenCalledWith('testproject')
    expect(wrapper.text()).toContain('ADR-0004')
  })

  it('submits with status: draft by default and emits created with the returned path', async () => {
    const wrapper = shallowMount(NewAdrModal, { props: { project: 'testproject' } })
    await flushPromises()

    await wrapper.find('#adr-title').setValue('Use gRPC')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(mockCreateAdr).toHaveBeenCalledWith('testproject', expect.objectContaining({
      title: 'Use gRPC',
      slug: 'use-grpc',
      status: 'draft',
    }))
    expect(wrapper.emitted('created')).toBeTruthy()
    expect(wrapper.emitted('created')![0]).toEqual(['lifecycle/architecture/decisions/adr-0004-use-grpc.md'])
  })

  it('shows a validation error and does not submit when title is empty', async () => {
    const wrapper = shallowMount(NewAdrModal, { props: { project: 'testproject' } })
    await flushPromises()

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(mockCreateAdr).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Title is required.')
  })
})
