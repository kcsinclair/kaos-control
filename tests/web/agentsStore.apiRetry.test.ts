// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Surfaces Claude's `{type:"system", subtype:"api_retry"}` events in the run
 * progress log so the UI doesn't look frozen during multi-minute backoff
 * sequences. Reported 2026-06-02 against an Overloaded (529) run where the
 * Claude binary retried 10 times over ~3.5 minutes silently.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAgentsStore } from '../../web/src/stores/agents'

vi.mock('@/api/agents', () => ({
  listRuns:             vi.fn().mockResolvedValue({ runs: [] }),
  listAgents:           vi.fn().mockResolvedValue({ agents: [] }),
  startRun:             vi.fn().mockResolvedValue({ run_id: 'mock-run' }),
  killRun:              vi.fn().mockResolvedValue({}),
  getRunLog:            vi.fn().mockResolvedValue(''),
  getReadyCounts:       vi.fn().mockResolvedValue({ counts: {} }),
  listRunsByTargetPath: vi.fn().mockResolvedValue([]),
}))

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('agentsStore — api_retry events render a friendly progress line', () => {
  const runId = 'aaaaaaaa-0000-0000-0000-000000000001'

  function dispatchProgress(store: ReturnType<typeof useAgentsStore>, event: Record<string, unknown>) {
    store.onWsEvent('agent.progress', { run_id: runId, event })
  }

  it('formats a 529 retry with attempt counter and backoff seconds', () => {
    const store = useAgentsStore()
    dispatchProgress(store, {
      type: 'system',
      subtype: 'api_retry',
      attempt: 3,
      max_retries: 10,
      retry_delay_ms: 2287.96,
      error_status: 529,
      error: 'rate_limit',
    })
    const lines = store.progressLines.get(runId) ?? []
    expect(lines).toHaveLength(1)
    expect(lines[0]).toBe('↻ retrying after 529 (attempt 3/10, 2.3s backoff)')
  })

  it('formats a 429 retry without max_retries gracefully', () => {
    const store = useAgentsStore()
    dispatchProgress(store, {
      type: 'system',
      subtype: 'api_retry',
      attempt: 1,
      retry_delay_ms: 500,
      error_status: 429,
    })
    const lines = store.progressLines.get(runId) ?? []
    expect(lines[0]).toContain('429')
    expect(lines[0]).toContain('0.5s backoff')
  })

  it('does NOT classify api_retry as the existing session-started line', () => {
    const store = useAgentsStore()
    dispatchProgress(store, {
      type: 'system',
      subtype: 'api_retry',
      attempt: 1,
      max_retries: 10,
      retry_delay_ms: 100,
      error_status: 529,
    })
    const lines = store.progressLines.get(runId) ?? []
    expect(lines[0]).not.toBe('▸ session started')
  })
})
