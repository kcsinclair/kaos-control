// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { ref, computed } from 'vue'

// Frontend plan: lifecycle/frontend-plans/architecture-overview-view-4-fe.md
// — Milestone F3 (FR-10, NFR-5): every panel and the view as a whole degrade
// gracefully instead of erroring when parts of the architecture zone are
// absent.

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { project: 'testproject' } }),
  useRouter: () => ({ push: vi.fn() }),
}))

const overviewState = {
  loading: ref(false),
  error: ref<string | null>(null),
  hasChosenArchitecture: ref(false),
  chosenArchitecture: ref<unknown>(null),
  chosenStack: ref<unknown>(null),
  summary: ref<unknown>(null),
  standards: ref<unknown[]>([]),
  adrs: ref<unknown[]>([]),
  archive: computed(() => []),
  catalog: computed(() => []),
  reload: vi.fn(),
}

vi.mock('@/composables/useArchitectureOverview', () => ({
  useArchitectureOverview: () => overviewState,
}))

const getArtifact = vi.fn()
vi.mock('@/api/artifacts', () => ({
  getArtifact: (...args: unknown[]) => getArtifact(...args),
}))

import ArchitectureOverviewView from '../ArchitectureOverviewView.vue'

const RouterLinkStub = { template: '<a><slot /></a>' }

function mountView() {
  return mount(ArchitectureOverviewView, {
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('ArchitectureOverviewView — empty / partial states', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getArtifact.mockReset()
    overviewState.loading.value = false
    overviewState.error.value = null
    overviewState.hasChosenArchitecture.value = false
    overviewState.chosenArchitecture.value = null
    overviewState.chosenStack.value = null
    overviewState.summary.value = null
    overviewState.standards.value = []
    overviewState.adrs.value = []
  })

  it('shows one top-level empty state with wizard + map links when no architecture is chosen', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('No architecture has been chosen')
    expect(wrapper.text()).toContain('Run the Architecture Wizard')
    expect(wrapper.text()).toContain('Browse the relationship map')
    // Panels are not rendered at all in this state.
    expect(wrapper.text()).not.toContain('Chosen architecture')
    expect(wrapper.text()).not.toContain('ADRs')
  })

  it('renders the panels (not the empty state) when a summary exists but no architecture is chosen', async () => {
    // Regression: the view used to hard-gate on hasChosenArchitecture, hiding
    // an existing summary/ADRs/standards behind the "no architecture" empty
    // state. It must now render whenever any architecture content exists.
    overviewState.hasChosenArchitecture.value = false
    overviewState.summary.value = {
      path: 'lifecycle/architecture/architecture-summary.md',
      title: 'Architecture Summary',
      status: 'approved',
      type: 'doc',
      catalog_role: 'summary',
    }
    getArtifact.mockResolvedValue({
      artifact: {
        path: 'lifecycle/architecture/architecture-summary.md',
        rel_path: 'architecture/architecture-summary.md',
        slug: 'architecture-summary',
        lineage: 'architecture-summary',
        index: 0,
        stage: '',
        type: 'doc',
        status: 'approved',
        title: 'Architecture Summary',
        frontmatter: { title: 'Architecture Summary', type: 'doc', status: 'approved', lineage: 'architecture-summary' },
        mtime: '',
        created: '',
        agent_run_count: 0,
      },
      body: '# Architecture Summary\n\nWhy we chose this.',
      body_html: '',
    })

    const wrapper = mountView()
    await flushPromises()
    await flushPromises()

    // Empty state is gone; panels render (each degrading on its own).
    expect(wrapper.text()).not.toContain('Run the Architecture Wizard')
    expect(wrapper.text()).toContain('No architecture has been chosen yet.') // ChosenArchitecturePanel degrades
    expect(wrapper.text()).toContain('No tech stack has been chosen yet.')
  })

  it('degrades each panel independently when a chosen architecture exists but summary/standards/ADRs are absent', async () => {
    overviewState.hasChosenArchitecture.value = true
    overviewState.chosenArchitecture.value = {
      path: 'lifecycle/architecture/modular-monolith.md',
      title: 'Modular Monolith',
      status: 'approved',
      type: 'architecture',
      catalog_role: 'chosen-architecture',
    }
    getArtifact.mockResolvedValue({
      artifact: {
        path: 'lifecycle/architecture/modular-monolith.md',
        rel_path: 'architecture/modular-monolith.md',
        slug: 'modular-monolith',
        lineage: 'arch-modular-monolith',
        index: 0,
        stage: '',
        type: 'architecture',
        status: 'approved',
        title: 'Modular Monolith',
        frontmatter: { title: 'Modular Monolith', type: 'architecture', status: 'approved', lineage: 'arch-modular-monolith' },
        mtime: '',
        created: '',
        agent_run_count: 0,
      },
      body: '# Modular Monolith\n\nBody text.',
      body_html: '',
    })

    const wrapper = mountView()
    await flushPromises()
    await flushPromises()

    // No console error / thrown render (vue-test-utils would throw on mount errors).
    expect(wrapper.text()).toContain('Modular Monolith')
    expect(wrapper.text()).toContain('No tech stack has been chosen yet.')
    expect(wrapper.text()).toContain('No architecture summary yet.')
    expect(wrapper.text()).toContain('No standards recorded yet.')
    expect(wrapper.text()).toContain('No ADRs yet.')
  })
})
