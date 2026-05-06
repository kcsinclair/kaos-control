/**
 * Milestones 4 & 6 — Unit tests for the ArtifactRunHistory component
 *
 * Covers:
 *   Milestone 4 — rendering states, field display, status badges, event emission,
 *                 and store fetch on mount.
 *   Milestone 6 — reactive updates when agentsStore.artifactRuns changes without
 *                 remounting the component.
 *
 * Component: web/src/components/artifact/ArtifactRunHistory.vue
 * Props: project (string), targetPath (string)
 * Emits: select-run (runId: string)
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import ArtifactRunHistory from '../../web/src/components/artifact/ArtifactRunHistory.vue'
import { useAgentsStore } from '../../web/src/stores/agents'
import type { AgentRunRow } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

// listRunsByTargetPath is called by fetchRunsByTargetPath in the store.
// Default: resolves immediately with an empty array (no runs).
vi.mock('@/api/agents', () => ({
  listRunsByTargetPath: vi.fn().mockResolvedValue([]),
  listRuns:             vi.fn().mockResolvedValue({ runs: [] }),
  listAgents:           vi.fn().mockResolvedValue({ agents: [] }),
  startRun:             vi.fn().mockResolvedValue({ run_id: 'mock-run' }),
  killRun:              vi.fn().mockResolvedValue({}),
  getRun:               vi.fn().mockResolvedValue({ run: null }),
  getRunLog:            vi.fn().mockResolvedValue(''),
}))

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeRun(overrides: Partial<AgentRunRow> = {}): AgentRunRow {
  return {
    run_id:             'abcdefgh-1234-0000-0000-000000000000',
    agent_name:         'requirements-analyst',
    role:               'analyst',
    target_path:        'lifecycle/requirements/foo-2.md',
    started_at:         '2026-01-01T10:00:00Z',
    status:             'done',
    stderr_tail:        '',
    artifacts_produced: [],
    ...overrides,
  }
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
})

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

interface MountProps {
  project?: string
  targetPath?: string
}

function mountComponent({ project = 'testproject', targetPath = 'lifecycle/requirements/foo-2.md' }: MountProps = {}) {
  return mount(ArtifactRunHistory, { props: { project, targetPath } })
}

// ---------------------------------------------------------------------------
// Milestone 4 — Rendering states and interaction
// ---------------------------------------------------------------------------

describe('ArtifactRunHistory — loading state', () => {
  it('renders loading state while fetching: no run rows are visible before fetch resolves', async () => {
    // Arrange: make listRunsByTargetPath hang so the fetch never resolves during the test.
    const { listRunsByTargetPath } = await import('@/api/agents')
    vi.mocked(listRunsByTargetPath).mockImplementation(() => new Promise(() => {}))

    const wrapper = mountComponent()

    // While loading: store.artifactRuns is still empty → no run rows.
    expect(wrapper.findAll('.arh-row')).toHaveLength(0)
  })
})

describe('ArtifactRunHistory — empty state', () => {
  it('renders "No agent runs for this artifact." when there are no runs', async () => {
    const wrapper = mountComponent()
    await flushPromises() // allow fetchRunsByTargetPath to resolve (returns [])

    expect(wrapper.text()).toContain('No agent runs for this artifact')
  })
})

describe('ArtifactRunHistory — run list rendering', () => {
  it('renders run list rows with truncated run ID, agent name, and status', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({
      artifactRuns: [
        makeRun({
          run_id:     'aaaabbbb-0000-0000-0000-000000000000',
          agent_name: 'requirements-analyst',
          status:     'done',
        }),
        makeRun({
          run_id:     'ccccdddd-0000-0000-0000-000000000000',
          agent_name: 'backend-developer',
          status:     'failed',
        }),
      ],
    })
    await nextTick()

    const rows = wrapper.findAll('.arh-row')
    expect(rows).toHaveLength(2)

    // First row checks
    const first = rows[0]
    expect(first.find('.arh-run-id').text()).toBe('aaaabbbb')
    expect(first.find('.arh-agent').text()).toBe('requirements-analyst')
    expect(first.find('.arh-status').text()).toBe('done')

    // Second row checks
    const second = rows[1]
    expect(second.find('.arh-run-id').text()).toBe('ccccdddd')
    expect(second.find('.arh-agent').text()).toBe('backend-developer')
    expect(second.find('.arh-status').text()).toBe('failed')
  })

  it('run ID is truncated to exactly 8 characters', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({
      artifactRuns: [makeRun({ run_id: '12345678-abcd-0000-0000-000000000000' })],
    })
    await nextTick()

    const runIdEl = wrapper.find('.arh-run-id')
    expect(runIdEl.text()).toBe('12345678')
    expect(runIdEl.text()).toHaveLength(8)
  })
})

describe('ArtifactRunHistory — status badge accessibility', () => {
  it('status badges have accessible text or aria-label for each status', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    const statuses = ['running', 'done', 'failed', 'killed']
    const store = useAgentsStore()
    store.$patch({
      artifactRuns: statuses.map((status, i) =>
        makeRun({
          run_id: `0000000${i}-0000-0000-0000-000000000000`,
          status,
        }),
      ),
    })
    await nextTick()

    const badges = wrapper.findAll('.arh-status')
    expect(badges).toHaveLength(statuses.length)

    for (const badge of badges) {
      const hasAriaLabel = badge.attributes('aria-label') !== undefined && badge.attributes('aria-label') !== ''
      const hasVisibleText = badge.text().trim() !== ''
      expect(
        hasAriaLabel || hasVisibleText,
        `Badge for status "${badge.text()}" must have aria-label or visible text`,
      ).toBe(true)
    }
  })
})

describe('ArtifactRunHistory — row click', () => {
  it('emits select-run with the full run ID when a row is clicked', async () => {
    const wrapper = mountComponent()
    await flushPromises()

    const store = useAgentsStore()
    const run = makeRun({ run_id: 'click000-1234-5678-abcd-000000000000' })
    store.$patch({ artifactRuns: [run] })
    await nextTick()

    await wrapper.find('.arh-row').trigger('click')

    const emitted = wrapper.emitted('select-run')
    expect(emitted).toBeDefined()
    expect(emitted![0]).toEqual(['click000-1234-5678-abcd-000000000000'])
  })
})

describe('ArtifactRunHistory — fetch on mount', () => {
  it('calls fetchRunsByTargetPath with the correct project and targetPath on mount', async () => {
    const { listRunsByTargetPath } = await import('@/api/agents')
    const spy = vi.mocked(listRunsByTargetPath)
    spy.mockResolvedValue([])

    mountComponent({ project: 'my-project', targetPath: 'lifecycle/requirements/foo-2.md' })
    await flushPromises()

    expect(spy).toHaveBeenCalledWith('my-project', 'lifecycle/requirements/foo-2.md')
  })
})

// ---------------------------------------------------------------------------
// Milestone 6 — Reactive updates without remounting
// ---------------------------------------------------------------------------

describe('ArtifactRunHistory — reactive updates', () => {
  it('updates the list when a new run is pushed into agentsStore.artifactRuns', async () => {
    const store = useAgentsStore()
    store.$patch({
      artifactRuns: [makeRun({ run_id: 'aaaa0000-0000-0000-0000-000000000000' })],
    })

    const wrapper = mountComponent()
    await nextTick()

    expect(wrapper.findAll('.arh-row')).toHaveLength(1)

    // Push a new run into the store without remounting.
    store.$patch({
      artifactRuns: [
        ...store.artifactRuns,
        makeRun({ run_id: 'bbbb0000-0000-0000-0000-000000000000' }),
      ],
    })
    await nextTick()

    expect(wrapper.findAll('.arh-row')).toHaveLength(2)
  })

  it('updates the status badge when an existing run status changes', async () => {
    const store = useAgentsStore()
    const run = makeRun({ run_id: 'cccc0000-0000-0000-0000-000000000000', status: 'running' })
    store.$patch({ artifactRuns: [run] })

    const wrapper = mountComponent()
    await nextTick()

    expect(wrapper.find('.arh-status').text()).toBe('running')

    // Update status in the store without remounting.
    store.$patch({ artifactRuns: [{ ...run, status: 'done' }] })
    await nextTick()

    expect(wrapper.find('.arh-status').text()).toBe('done')
  })
})
