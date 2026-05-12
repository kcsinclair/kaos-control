// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 4 — agents store: agent.failed WS event passes precheck payload through
 *
 * Dispatches a synthetic agent.failed WS event with the precheck payload and
 * asserts that the matching run row in the store now has the three new fields
 * populated (failure_reason, observed_permission_mode, remediation).
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAgentsStore } from '../../web/src/stores/agents'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/api/agents', () => ({
  listRuns:       vi.fn().mockResolvedValue({ runs: [] }),
  listAgents:     vi.fn().mockResolvedValue({ agents: [] }),
  startRun:       vi.fn().mockResolvedValue({ run_id: 'mock-run' }),
  killRun:        vi.fn().mockResolvedValue({}),
  getRunLog:      vi.fn().mockResolvedValue(''),
  getReadyCounts: vi.fn().mockResolvedValue({ counts: {} }),
  listRunsByTargetPath: vi.fn().mockResolvedValue([]),
}))

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
})

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('agentsStore — agent.failed WS event with precheck payload', () => {
  it('populates failure_reason, observed_permission_mode, and remediation from the payload', () => {
    const store = useAgentsStore()

    const runId = 'deadbeef-0000-0000-0000-000000000001'

    // Seed a running run
    store.$patch({
      runs: [
        {
          run_id:             runId,
          agent_name:         'backend-developer',
          role:               'developer',
          target_path:        'lifecycle/requirements/test.md',
          started_at:         '2026-01-01T10:00:00Z',
          status:             'running',
          stderr_tail:        '',
          artifacts_produced: [],
        },
      ],
    })

    // Dispatch a synthetic agent.failed event with the precheck payload
    store.onWsEvent('agent.failed', {
      run_id:                   runId,
      status:                   'failed',
      artifacts:                [],
      failure_reason:           'permission_mode_default',
      observed_permission_mode: 'default',
      remediation:              [
        'Run `claude config set permission-mode bypassPermissions`',
        'Restart the kaos-control agent process',
      ],
    })

    const row = store.runs.find((r) => r.run_id === runId)
    expect(row).toBeDefined()
    expect(row!.status).toBe('failed')
    expect(row!.failure_reason).toBe('permission_mode_default')
    expect(row!.observed_permission_mode).toBe('default')
    expect(row!.remediation).toEqual([
      'Run `claude config set permission-mode bypassPermissions`',
      'Restart the kaos-control agent process',
    ])
  })

  it('sets failure_reason to null when not present in a regular agent.failed event', () => {
    const store = useAgentsStore()

    const runId = 'deadbeef-0000-0000-0000-000000000002'

    store.$patch({
      runs: [
        {
          run_id:             runId,
          agent_name:         'backend-developer',
          role:               'developer',
          target_path:        'lifecycle/requirements/test.md',
          started_at:         '2026-01-01T10:00:00Z',
          status:             'running',
          stderr_tail:        '',
          artifacts_produced: [],
        },
      ],
    })

    // Classic failure — no precheck fields in payload
    store.onWsEvent('agent.failed', {
      run_id:    runId,
      status:    'failed',
      artifacts: [],
    })

    const row = store.runs.find((r) => r.run_id === runId)
    expect(row).toBeDefined()
    expect(row!.status).toBe('failed')
    expect(row!.failure_reason).toBeNull()
    expect(row!.observed_permission_mode).toBeNull()
    expect(row!.remediation).toBeNull()
  })

  it('does NOT mutate the run when no matching run_id exists', () => {
    const store = useAgentsStore()
    store.$patch({ runs: [] })

    // Should not throw
    expect(() => {
      store.onWsEvent('agent.failed', {
        run_id:         'nonexistent-run-id',
        failure_reason: 'permission_mode_default',
      })
    }).not.toThrow()

    expect(store.runs).toHaveLength(0)
  })
})
