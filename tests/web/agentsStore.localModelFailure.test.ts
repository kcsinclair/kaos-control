// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 5 — agents store: local-model failure & warmup WS event handling
 * (local-model-operability: lifecycle/test-plans/local-model-operability-5-test.md)
 *
 * Covers:
 *   - agent.failed events carrying failure_reason, remediation, and
 *     error_details (the structured local-model taxonomy) update the stored
 *     AgentRunRow, mirroring the existing precheck coverage in
 *     agentsStore.precheckFailure.test.ts for the new fields.
 *   - agent.status (model_loading) and agent.progress (warming_up/generating)
 *     events update warmup_state/warmup_message on the matching run, and are
 *     cleared once the run finishes.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAgentsStore } from '../../web/src/stores/agents'

vi.mock('@/api/agents', () => ({
  listRuns:       vi.fn().mockResolvedValue({ runs: [] }),
  listAgents:     vi.fn().mockResolvedValue({ agents: [] }),
  startRun:       vi.fn().mockResolvedValue({ run_id: 'mock-run' }),
  killRun:        vi.fn().mockResolvedValue({}),
  getRunLog:      vi.fn().mockResolvedValue(''),
  getReadyCounts: vi.fn().mockResolvedValue({ counts: {} }),
  listRunsByTargetPath: vi.fn().mockResolvedValue([]),
}))

beforeEach(() => {
  setActivePinia(createPinia())
})

function seedRunningRun(store: ReturnType<typeof useAgentsStore>, runId: string, targetPath = 'lifecycle/backend-plans/feature-3-be.md') {
  store.$patch({
    runs: [
      {
        run_id:             runId,
        agent_name:         'local-backend-developer',
        role:               'backend-developer',
        target_path:        targetPath,
        started_at:         '2026-08-25T10:00:00Z',
        status:             'running',
        stderr_tail:        '',
        artifacts_produced: [],
      },
    ],
  })
}

describe('agentsStore — local-model structured failure taxonomy', () => {
  it('populates failure_reason, remediation, and error_details from an agent.failed event', () => {
    const store = useAgentsStore()
    const runId = 'run-model-not-found'
    seedRunningRun(store, runId)

    store.onWsEvent('agent.failed', {
      run_id:         runId,
      status:         'failed',
      artifacts:      [],
      failure_reason: 'model_not_found',
      remediation: [
        'Verify the model name is spelled correctly in Agent Config.',
        'Run `ollama pull <model>` to make the model available.',
      ],
      error_details: {
        model:    'gemma-4-26b',
        provider: 'local-llama',
      },
    })

    const row = store.runs.find((r) => r.run_id === runId)
    expect(row).toBeDefined()
    expect(row!.status).toBe('failed')
    expect(row!.failure_reason).toBe('model_not_found')
    expect(row!.remediation).toEqual([
      'Verify the model name is spelled correctly in Agent Config.',
      'Run `ollama pull <model>` to make the model available.',
    ])
    expect(row!.error_details).toEqual({ model: 'gemma-4-26b', provider: 'local-llama' })
  })

  it('surfaces already-masked secret markers verbatim without altering them', () => {
    const store = useAgentsStore()
    const runId = 'run-auth-error'
    seedRunningRun(store, runId)

    store.onWsEvent('agent.failed', {
      run_id:         runId,
      status:         'failed',
      artifacts:      [],
      failure_reason: 'auth_error',
      error_details: {
        authorization: '***',
      },
    })

    const row = store.runs.find((r) => r.run_id === runId)
    expect(row!.error_details).toEqual({ authorization: '***' })
  })

  it('sets error_details to null when absent from the payload', () => {
    const store = useAgentsStore()
    const runId = 'run-no-details'
    seedRunningRun(store, runId)

    store.onWsEvent('agent.failed', {
      run_id:    runId,
      status:    'failed',
      artifacts: [],
    })

    const row = store.runs.find((r) => r.run_id === runId)
    expect(row!.error_details).toBeNull()
    expect(row!.failure_reason).toBeNull()
    expect(row!.remediation).toBeNull()
  })

  it('clears warmup_state and warmup_message when the run finishes', () => {
    const store = useAgentsStore()
    const runId = 'run-warmup-then-fail'
    seedRunningRun(store, runId)

    store.onWsEvent('agent.progress', {
      run_id: runId,
      event:  { stage: 'warming_up', message: 'Awaiting first token (model may be warming up)...' },
    })
    expect(store.runs.find((r) => r.run_id === runId)!.warmup_state).toBe('warming_up')

    store.onWsEvent('agent.failed', {
      run_id:         runId,
      status:         'failed',
      artifacts:      [],
      failure_reason: 'timeout',
    })

    const row = store.runs.find((r) => r.run_id === runId)
    expect(row!.warmup_state).toBeNull()
    expect(row!.warmup_message).toBeNull()
  })
})

describe('agentsStore — warmup progress events', () => {
  it('sets warmup_state to model_loading from an agent.status event before the run row exists', () => {
    const store = useAgentsStore()
    const targetPath = 'lifecycle/backend-plans/feature-3-be.md'

    // agent.status (model_loading) arrives during preflight, before agent.started.
    store.onWsEvent('agent.status', {
      target_path: targetPath,
      state:       'model_loading',
      details:     'Loading model weights…',
    })

    store.onWsEvent('agent.started', {
      run_id:      'run-late-start',
      agent:       'local-backend-developer',
      target_path: targetPath,
      lineage:     targetPath,
    })

    const row = store.runs.find((r) => r.run_id === 'run-late-start')
    expect(row).toBeDefined()
    expect(row!.warmup_state).toBe('model_loading')
    expect(row!.warmup_message).toBe('Loading model weights…')
  })

  it('transitions warmup_state from warming_up to generating on first token', () => {
    const store = useAgentsStore()
    const runId = 'run-ttft'
    seedRunningRun(store, runId)

    store.onWsEvent('agent.progress', {
      run_id: runId,
      event:  { stage: 'warming_up', message: 'Awaiting first token (model may be warming up)...' },
    })
    expect(store.runs.find((r) => r.run_id === runId)!.warmup_state).toBe('warming_up')

    store.onWsEvent('agent.progress', {
      run_id: runId,
      event:  { stage: 'generating' },
    })
    expect(store.runs.find((r) => r.run_id === runId)!.warmup_state).toBe('generating')
  })

  it('does NOT mutate any run when agent.status target_path matches no running run and no pending slot is claimed', () => {
    const store = useAgentsStore()
    seedRunningRun(store, 'run-unrelated', 'lifecycle/backend-plans/other-9-be.md')

    store.onWsEvent('agent.status', {
      target_path: 'lifecycle/backend-plans/feature-3-be.md',
      state:       'model_loading',
      details:     'Loading…',
    })

    const row = store.runs.find((r) => r.run_id === 'run-unrelated')
    expect(row!.warmup_state).toBeUndefined()
  })
})
