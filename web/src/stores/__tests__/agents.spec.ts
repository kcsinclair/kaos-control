// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAgentsStore } from '../agents'

// Frontend plan: lifecycle/frontend-plans/rate-limit-event-detection-4-fe.md
// Milestone 2 — cache latest quota status per run; Milestone 3 — graceful
// no-consumer handling for agent.quota_status.

describe('agents store — agent.quota_status caching', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('populates quotaForRun on agent.quota_status', () => {
    const store = useAgentsStore()
    store.onWsEvent('agent.quota_status', {
      run_id: 'run-1',
      bucket: 'five_hour',
      status: 'warning',
      resets_at: '2026-08-15T04:00:00Z',
      overage_available: true,
    })

    expect(store.quotaForRun('run-1')).toEqual({
      run_id: 'run-1',
      bucket: 'five_hour',
      status: 'warning',
      resets_at: '2026-08-15T04:00:00Z',
      overage_available: true,
    })
  })

  it('replaces the cached value on a subsequent event for the same run', () => {
    const store = useAgentsStore()
    store.onWsEvent('agent.quota_status', {
      run_id: 'run-1',
      bucket: 'five_hour',
      status: 'allowed',
      overage_available: false,
    })
    store.onWsEvent('agent.quota_status', {
      run_id: 'run-1',
      bucket: 'five_hour',
      status: 'rejected',
      overage_available: false,
      overage_disabled_reason: 'not_configured',
    })

    expect(store.quotaForRun('run-1')?.status).toBe('rejected')
  })

  it('clears the entry on the run reaching a terminal state', () => {
    const store = useAgentsStore()
    store.onWsEvent('agent.quota_status', {
      run_id: 'run-1',
      bucket: 'weekly',
      status: 'warning',
      overage_available: false,
    })
    expect(store.quotaForRun('run-1')).not.toBeNull()

    store.onWsEvent('agent.finished', { run_id: 'run-1' })

    expect(store.quotaForRun('run-1')).toBeNull()
  })

  it('is a no-op with no console warning and does not disrupt sibling handling', () => {
    const store = useAgentsStore()
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    store.onWsEvent('agent.quota_status', {
      run_id: 'run-1',
      bucket: 'unknown',
      status: 'unknown',
      overage_available: false,
    })
    store.onWsEvent('agent.progress', { run_id: 'run-1', raw: 'still working' })

    expect(warnSpy).not.toHaveBeenCalled()
    expect(store.progressLines.get('run-1')).toEqual(['still working'])

    warnSpy.mockRestore()
  })
})
