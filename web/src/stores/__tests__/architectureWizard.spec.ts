// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Frontend plan: lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md — Milestone 1
//
// Verifies the wizard store's core state-machine actions: start() populates
// questions + prior-run, setAnswer/skip mutate answers, commit() calls the
// commit endpoint with architecture + stack + answers + Q&A, and a server
// error surfaces via `error` without throwing.

vi.mock('@/api/architecture', () => ({
  getWizard: vi.fn(),
  recommend: vi.fn(),
  listStacks: vi.fn(),
  saveWizardState: vi.fn().mockResolvedValue(undefined),
  discardWizardState: vi.fn(),
  commitWizard: vi.fn(),
}))

import {
  getWizard,
  recommend,
  commitWizard,
  saveWizardState,
} from '@/api/architecture'
import { ApiError } from '@/api/client'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import type { CatalogItem } from '@/types/api'

function makeArchitecture(overrides: Partial<CatalogItem> = {}): CatalogItem {
  return {
    path: 'lifecycle/architecture/architectures/modular-monolith.md',
    slug: 'modular-monolith',
    title: 'Modular Monolith',
    summary: 'A single deployable, internally modular.',
    type: 'architecture',
    labels: ['offline-capable'],
    related_to: ['lifecycle/architecture/tech-stacks/go-vue.md'],
    ...overrides,
  }
}

function makeStack(overrides: Partial<CatalogItem> = {}): CatalogItem {
  return {
    path: 'lifecycle/architecture/tech-stacks/go-vue.md',
    slug: 'go-vue',
    title: 'Go + Vue',
    summary: 'Go backend, Vue frontend.',
    type: 'tech-stack',
    labels: ['go'],
    related_to: [],
    ...overrides,
  }
}

describe('architectureWizard store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('start() populates questions, default architecture, and prior-run', async () => {
    vi.mocked(getWizard).mockResolvedValue({
      questions: [{ id: 'offline', prompt: 'Offline-capable?', kind: 'hard', options: [] }],
      default_architecture: 'modular-monolith',
      prior_run: { detected: false },
      resumable_state: null,
    })

    const store = useArchitectureWizardStore()
    await store.start('testproject')

    expect(store.questions).toHaveLength(1)
    expect(store.defaultArchitecture).toBe('modular-monolith')
    expect(store.priorRun).toEqual({ detected: false })
    expect(store.priorRunResolved).toBe(true)
    expect(store.error).toBeNull()
  })

  it('start() adopts a resumable saved state when present', async () => {
    vi.mocked(getWizard).mockResolvedValue({
      questions: [],
      default_architecture: 'modular-monolith',
      prior_run: { detected: false },
      resumable_state: {
        path: 'guided',
        answers: [{ question_id: 'offline', value: 'yes' }],
        step: 'questions',
        updated_unix: 100,
      },
    })

    const store = useArchitectureWizardStore()
    await store.start('testproject')

    expect(store.path).toBe('guided')
    expect(store.answers).toEqual([{ question_id: 'offline', value: 'yes' }])
    expect(store.step).toBe('questions')
  })

  it('setAnswer adds an answer and skip removes it', () => {
    const store = useArchitectureWizardStore()

    store.setAnswer('offline', 'yes')
    expect(store.answerFor('offline')).toBe('yes')

    store.setAnswer('offline', 'no')
    expect(store.answers).toHaveLength(1)
    expect(store.answerFor('offline')).toBe('no')

    store.skip('offline')
    expect(store.answerFor('offline')).toBeUndefined()
    expect(store.skippedQuestionIds).toContain('offline')
  })

  it('fetchRecommendations populates recommendations and dropped constraints', async () => {
    const arch = makeArchitecture()
    vi.mocked(recommend).mockResolvedValue({
      recommendations: [{ item: arch, score: 2, why: ['Offline-capable? → yes'] }],
      dropped_constraints: ['some-label'],
    })

    const store = useArchitectureWizardStore()
    store.setAnswer('offline', 'yes')
    await store.fetchRecommendations('testproject')

    expect(store.recommendations).toHaveLength(1)
    expect(store.recommendations[0].item.slug).toBe('modular-monolith')
    expect(store.droppedConstraints).toEqual(['some-label'])
    expect(vi.mocked(recommend)).toHaveBeenCalledWith('testproject', [
      { question_id: 'offline', value: 'yes' },
    ])
  })

  it('commit() calls the commit endpoint with architecture + stack + answers + Q&A', async () => {
    const arch = makeArchitecture()
    const stack = makeStack()
    vi.mocked(commitWizard).mockResolvedValue({
      promoted_architecture: arch.path,
      promoted_tech_stack: stack.path,
      archived: [],
      adr_path: 'lifecycle/architecture/decisions/adr-0001-modular-monolith.md',
      superseded_adr_path: '',
      summary_path: 'lifecycle/architecture/architecture-summary.md',
    })

    const store = useArchitectureWizardStore()
    store.setAnswer('offline', 'yes')
    store.chooseArchitecture(arch)
    store.chooseStack(stack)

    const result = await store.commit('testproject', {
      breakingRequirements: [{ Label: 'offline', Requirement: 'Must work offline', Mapping: 'modular-monolith' }],
      qa: [{ Question: 'Offline-capable?', Answer: 'yes' }],
    })

    expect(result?.adr_path).toBe('lifecycle/architecture/decisions/adr-0001-modular-monolith.md')
    expect(vi.mocked(commitWizard)).toHaveBeenCalledWith('testproject', {
      architecture_path: 'architectures/modular-monolith.md',
      tech_stack_path: 'tech-stacks/go-vue.md',
      answers: [{ question_id: 'offline', value: 'yes' }],
      breaking_requirements: [{ Label: 'offline', Requirement: 'Must work offline', Mapping: 'modular-monolith' }],
      qa: [{ Question: 'Offline-capable?', Answer: 'yes' }],
    })
    expect(store.commitResult?.adr_path).toBe('lifecycle/architecture/decisions/adr-0001-modular-monolith.md')
  })

  it('commit() refuses without a chosen architecture/stack and does not call the API', async () => {
    const store = useArchitectureWizardStore()
    const result = await store.commit('testproject', { breakingRequirements: [], qa: [] })

    expect(result).toBeNull()
    expect(store.error).toBeTruthy()
    expect(vi.mocked(commitWizard)).not.toHaveBeenCalled()
  })

  it('currentQuestion advances through the set as answers/skips land, and answerCurrentQuestion/skipCurrentQuestion report completion', () => {
    const store = useArchitectureWizardStore()
    store.setPath('guided')
    store.questions = [
      { id: 'offline', prompt: 'Offline-capable?', kind: 'hard', options: [] },
      { id: 'realtime', prompt: 'Realtime?', kind: 'soft', options: [] },
    ]

    expect(store.currentQuestion?.id).toBe('offline')

    expect(store.answerCurrentQuestion('testproject', 'yes')).toBe(false)
    expect(store.answerFor('offline')).toBe('yes')
    expect(store.currentQuestion?.id).toBe('realtime')

    expect(store.skipCurrentQuestion('testproject')).toBe(true)
    expect(store.skippedQuestionIds).toContain('realtime')
    expect(store.currentQuestion).toBeNull()
  })

  it('answering a question triggers a debounced saveWizardState', () => {
    vi.useFakeTimers()
    try {
      const store = useArchitectureWizardStore()
      store.setPath('guided')
      store.questions = [{ id: 'offline', prompt: 'Offline-capable?', kind: 'hard', options: [] }]

      store.answerCurrentQuestion('testproject', 'yes')
      expect(vi.mocked(saveWizardState)).not.toHaveBeenCalled()

      vi.advanceTimersByTime(500)
      expect(vi.mocked(saveWizardState)).toHaveBeenCalledTimes(1)
      expect(vi.mocked(saveWizardState)).toHaveBeenCalledWith(
        'testproject',
        expect.objectContaining({ path: 'guided', answers: [{ question_id: 'offline', value: 'yes' }] }),
      )
    } finally {
      vi.useRealTimers()
    }
  })

  it('a server error surfaces via `error` without throwing', async () => {
    vi.mocked(getWizard).mockRejectedValue(new ApiError('fs_error', 'disk on fire', 500))

    const store = useArchitectureWizardStore()
    await expect(store.start('testproject')).resolves.toBeUndefined()

    expect(store.error).toBe('disk on fire')
  })
})
