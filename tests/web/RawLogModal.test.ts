// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 5 — Unit tests for the RawLogModal component
 *
 * Covers:
 *   - Log content rendered inside a <pre> element with monospace font
 *   - Panel minimum height of 90vh (via CSS class)
 *   - Loading state while the API call is in-flight
 *   - Error state on fetch failure
 *   - Empty-log state when API returns an empty string
 *   - Dismiss via close button (emits "close")
 *   - Dismiss via Escape key on the overlay (emits "close")
 *
 * Component: web/src/components/agent/RawLogModal.vue
 * Props: project (string), runId (string)
 * Emits: close
 * API: agentsApi.getRunLog (mocked)
 *
 * Note: <Teleport to="body"> is stubbed so content renders inline within the
 * wrapper's DOM tree.
 *
 * Run with: pnpm --prefix tests/web test RawLogModal
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import RawLogModal from '../../web/src/components/agent/RawLogModal.vue'

// ---------------------------------------------------------------------------
// Module mock — agentsApi.getRunLog
// ---------------------------------------------------------------------------

vi.mock('@/api/agents', () => ({
  getRunLog: vi.fn().mockResolvedValue('log line one\nlog line two\nlog line three'),
  getRun: vi.fn().mockResolvedValue({ run: null }),
  listRuns: vi.fn().mockResolvedValue({ runs: [] }),
  listAgents: vi.fn().mockResolvedValue({ agents: [] }),
  listRunsByTargetPath: vi.fn().mockResolvedValue([]),
  startRun: vi.fn().mockResolvedValue({ run_id: 'mock' }),
  killRun: vi.fn().mockResolvedValue({}),
  getRunResult: vi.fn().mockResolvedValue({ result: null }),
}))

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
})

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

const teleportStub = { template: '<div><slot /></div>' }

function mountModal(props: { project?: string; runId?: string } = {}) {
  return mount(RawLogModal, {
    props: {
      project: props.project ?? 'testproject',
      runId:   props.runId   ?? 'test-run-id-log-001',
    },
    global: { stubs: { Teleport: teleportStub } },
  })
}

// ---------------------------------------------------------------------------
// Log content display
// ---------------------------------------------------------------------------

describe('RawLogModal — log content', () => {
  it('displays log content in a monospaced pre-formatted block', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const pre = wrapper.find('pre')
    expect(pre.exists()).toBe(true)
    expect(pre.text()).toContain('log line one')
    expect(pre.text()).toContain('log line two')
    expect(pre.text()).toContain('log line three')
  })

  it('pre element has the rlm-content class (monospace font applied)', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const pre = wrapper.find('pre.rlm-content')
    expect(pre.exists()).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Panel height
// ---------------------------------------------------------------------------

describe('RawLogModal — panel layout', () => {
  it('panel element has rlm-panel class (which sets min-height: 90vh)', async () => {
    const wrapper = mountModal()
    await flushPromises()

    // The min-height 90vh is set via the .rlm-panel CSS class.
    // happy-dom does not compute scoped styles, so we verify the class presence.
    const panel = wrapper.find('.rlm-panel')
    expect(panel.exists()).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Loading state
// ---------------------------------------------------------------------------

describe('RawLogModal — loading state', () => {
  it('displays loading indicator while the API call is pending', async () => {
    const agentsApi = await import('@/api/agents')
    vi.mocked(agentsApi.getRunLog).mockImplementationOnce(
      () => new Promise<string>(() => {}), // never resolves
    )

    const wrapper = mountModal()
    // Before flushPromises: loading=true
    expect(wrapper.text()).toContain('Loading')
  })
})

// ---------------------------------------------------------------------------
// Error state
// ---------------------------------------------------------------------------

describe('RawLogModal — error state', () => {
  it('displays an error message when getRunLog rejects', async () => {
    const agentsApi = await import('@/api/agents')
    vi.mocked(agentsApi.getRunLog).mockRejectedValueOnce(new Error('HTTP 500: internal error'))

    const wrapper = mountModal()
    await flushPromises()

    // No pre element (no log content)
    expect(wrapper.find('pre').exists()).toBe(false)
    // Error message is visible
    const text = wrapper.text()
    expect(text).toMatch(/HTTP 500|internal error|Failed/)
  })
})

// ---------------------------------------------------------------------------
// Empty log state
// ---------------------------------------------------------------------------

describe('RawLogModal — empty log state', () => {
  it('displays empty-state message when getRunLog returns empty string', async () => {
    const agentsApi = await import('@/api/agents')
    vi.mocked(agentsApi.getRunLog).mockResolvedValueOnce('')

    const wrapper = mountModal()
    await flushPromises()

    expect(wrapper.find('pre').exists()).toBe(false)
    expect(wrapper.text()).toContain('No log content available')
  })
})

// ---------------------------------------------------------------------------
// Dismiss behaviour
// ---------------------------------------------------------------------------

describe('RawLogModal — dismiss via close button', () => {
  it('emits "close" when the close button is clicked', async () => {
    const wrapper = mountModal()
    await flushPromises()

    await wrapper.find('button[aria-label="Close log"]').trigger('click')

    expect(wrapper.emitted('close')).toBeDefined()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})

describe('RawLogModal — dismiss via Escape key', () => {
  it('emits "close" when Escape is pressed on the overlay', async () => {
    const wrapper = mountModal()
    await flushPromises()

    await wrapper.find('.rlm-overlay').trigger('keydown', { key: 'Escape' })

    expect(wrapper.emitted('close')).toBeDefined()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('does NOT emit "close" for non-Escape keys', async () => {
    const wrapper = mountModal()
    await flushPromises()

    await wrapper.find('.rlm-overlay').trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('close')).toBeUndefined()
  })
})
