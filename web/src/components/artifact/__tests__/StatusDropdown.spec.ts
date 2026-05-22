// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import StatusDropdown from '../StatusDropdown.vue'

// Test plan: lifecycle/test-plans/raw-artefact-status-5-test.md §Milestone 3, Scenario 2
//
// Verifies that StatusDropdown.vue renders the correct transition options for
// a 'raw' artefact: the mocked allowed-targets response is shown and 'raw'
// itself is not listed (no self-transition).

vi.mock('@/api/artifacts', () => ({
  getAllowedTargets: vi.fn(),
  transitionArtifact: vi.fn(),
}))

import { getAllowedTargets, transitionArtifact } from '@/api/artifacts'

const DEFAULT_PROPS = {
  project: 'testproject',
  path: 'lifecycle/ideas/raw-test.md',
  status: 'raw',
}

describe('StatusDropdown — raw artefact', () => {
  beforeEach(() => {
    vi.mocked(getAllowedTargets).mockResolvedValue({
      targets: ['draft', 'rejected', 'abandoned', 'blocked'],
    })
    vi.mocked(transitionArtifact).mockResolvedValue({ artifact: {} as never, rejection_artifact: '' })
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  // TC1: Opening the dropdown fetches and renders the allowed targets.
  it('renders allowed targets when the dropdown is opened', async () => {
    const wrapper = mount(StatusDropdown, {
      props: DEFAULT_PROPS,
      attachTo: document.body,
    })

    // Click to open the dropdown.
    await wrapper.find('[role="button"]').trigger('click')
    await flushPromises()

    expect(vi.mocked(getAllowedTargets)).toHaveBeenCalledWith('testproject', 'lifecycle/ideas/raw-test.md')

    const options = wrapper.findAll('[role="option"]')
    const optionTexts = options.map((o) => o.text())

    expect(optionTexts).toContain('draft')
    expect(optionTexts).toContain('blocked')
    expect(optionTexts).toContain('rejected')
    expect(optionTexts).toContain('abandoned')

    wrapper.unmount()
  })

  // TC2: 'raw' is not listed in the dropdown (no self-transition).
  it('does not list "raw" as a transition target (no self-transition)', async () => {
    const wrapper = mount(StatusDropdown, {
      props: DEFAULT_PROPS,
      attachTo: document.body,
    })

    await wrapper.find('[role="button"]').trigger('click')
    await flushPromises()

    const options = wrapper.findAll('[role="option"]')
    const optionTexts = options.map((o) => o.text())

    expect(optionTexts).not.toContain('raw')

    wrapper.unmount()
  })

  // TC3: The trigger badge shows the current 'raw' status before opening.
  it('shows the current status badge as "raw" before the dropdown is opened', () => {
    const wrapper = mount(StatusDropdown, {
      props: DEFAULT_PROPS,
    })

    const badge = wrapper.find('[data-status="raw"]')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('raw')
  })

  // TC4: Selecting 'draft' calls transitionArtifact with the correct args.
  it('calls transitionArtifact with "draft" when the draft option is selected', async () => {
    const wrapper = mount(StatusDropdown, {
      props: DEFAULT_PROPS,
      attachTo: document.body,
    })

    await wrapper.find('[role="button"]').trigger('click')
    await flushPromises()

    const options = wrapper.findAll('[role="option"]')
    const draftOption = options.find((o) => o.text() === 'draft')
    expect(draftOption).toBeTruthy()

    await draftOption!.trigger('click')
    await flushPromises()

    expect(vi.mocked(transitionArtifact)).toHaveBeenCalledWith(
      'testproject',
      'lifecycle/ideas/raw-test.md',
      'draft',
    )

    wrapper.unmount()
  })
})
