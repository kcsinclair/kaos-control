// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

// Frontend plan: lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md — Milestone 2
//
// Verifies the wizard shell's prior-run gate (FR-3): when the backend reports
// an already-run wizard, the shell renders PriorRunGate and blocks the normal
// stepper/step body until the user explicitly chooses Continue or Exit; Exit
// routes away without ever calling the commit endpoint.

const mockRouterPush = vi.fn()

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { project: 'testproject' } }),
  useRouter: () => ({ push: mockRouterPush }),
}))

vi.mock('@/api/architecture', () => ({
  getWizard: vi.fn(),
  recommend: vi.fn(),
  listStacks: vi.fn(),
  saveWizardState: vi.fn(),
  discardWizardState: vi.fn(),
  commitWizard: vi.fn(),
}))

import { getWizard, commitWizard } from '@/api/architecture'
import ArchitectureWizardView from '../ArchitectureWizardView.vue'

describe('ArchitectureWizardView — prior-run gate', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders PriorRunGate and hides the stepper when a prior run is detected', async () => {
    vi.mocked(getWizard).mockResolvedValue({
      questions: [],
      default_architecture: 'modular-monolith',
      prior_run: {
        detected: true,
        architecture: 'lifecycle/architecture/modular-monolith.md',
        tech_stack: 'lifecycle/architecture/go-vue.md',
        adr_path: 'lifecycle/architecture/decisions/adr-0001-modular-monolith.md',
      },
      resumable_state: null,
    })

    const wrapper = mount(ArchitectureWizardView)
    await flushPromises()

    expect(wrapper.find('[role="alertdialog"]').exists()).toBe(true)
    expect(wrapper.find('[role="list"][aria-label="Architecture wizard progress"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('modular-monolith.md')
  })

  it('Continue dismisses the gate and reveals the stepper', async () => {
    vi.mocked(getWizard).mockResolvedValue({
      questions: [],
      default_architecture: 'modular-monolith',
      prior_run: { detected: true, architecture: 'lifecycle/architecture/modular-monolith.md' },
      resumable_state: null,
    })

    const wrapper = mount(ArchitectureWizardView)
    await flushPromises()

    await wrapper.find('button.btn-primary').trigger('click') // "Continue (re-run)"
    await flushPromises()

    expect(wrapper.find('[role="alertdialog"]').exists()).toBe(false)
    expect(wrapper.find('[role="list"][aria-label="Architecture wizard progress"]').exists()).toBe(true)
  })

  it('Exit routes away without calling the commit endpoint', async () => {
    vi.mocked(getWizard).mockResolvedValue({
      questions: [],
      default_architecture: 'modular-monolith',
      prior_run: { detected: true, architecture: 'lifecycle/architecture/modular-monolith.md' },
      resumable_state: null,
    })

    const wrapper = mount(ArchitectureWizardView)
    await flushPromises()

    await wrapper.find('button.btn-secondary').trigger('click') // "Exit"
    await flushPromises()

    expect(mockRouterPush).toHaveBeenCalledWith('/p/testproject/architecture/map')
    expect(vi.mocked(commitWizard)).not.toHaveBeenCalled()
  })

  it('skips the gate and shows the stepper directly when no prior run is detected', async () => {
    vi.mocked(getWizard).mockResolvedValue({
      questions: [],
      default_architecture: 'modular-monolith',
      prior_run: { detected: false },
      resumable_state: null,
    })

    const wrapper = mount(ArchitectureWizardView)
    await flushPromises()

    expect(wrapper.find('[role="alertdialog"]').exists()).toBe(false)
    expect(wrapper.find('[role="list"][aria-label="Architecture wizard progress"]').exists()).toBe(true)
  })
})
