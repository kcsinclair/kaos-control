import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as devopsApi from '@/api/devops'
import type { Pipeline } from '@/api/devops'

export type StepStatus = 'pending' | 'running' | 'passed' | 'failed' | 'cancelled'

export interface StepState {
  name: string
  status: StepStatus
  startedAt?: number
  completedAt?: number
  durationMs?: number
  output: string[]
}

export interface ActiveRun {
  runId: string
  steps: StepState[]
  overallStatus: 'running' | 'passed' | 'failed' | 'cancelled'
}

export const useDevOpsStore = defineStore('devops', () => {
  const pipelines = ref<Pipeline[]>([])
  const loading = ref(false)
  const loadError = ref<string | null>(null)

  // slug → ActiveRun
  const activeRuns = ref(new Map<string, ActiveRun>())

  const pipelinesByType = computed((): Record<string, Pipeline[]> => {
    const grouped: Record<string, Pipeline[]> = {}
    for (const p of pipelines.value) {
      if (!grouped[p.type]) grouped[p.type] = []
      grouped[p.type].push(p)
    }
    return grouped
  })

  async function fetchPipelines(project: string): Promise<void> {
    loading.value = true
    loadError.value = null
    try {
      const res = await devopsApi.listPipelines(project)
      pipelines.value = res.pipelines ?? []
    } catch (e: unknown) {
      loadError.value = e instanceof Error ? e.message : 'Failed to load pipelines'
    } finally {
      loading.value = false
    }
  }

  async function runPipeline(project: string, slug: string): Promise<string> {
    const res = await devopsApi.runPipeline(project, slug)
    const pipeline = pipelines.value.find((p) => p.slug === slug)
    activeRuns.value.set(slug, {
      runId: res.run_id,
      overallStatus: 'running',
      steps: (pipeline?.steps ?? []).map((s) => ({
        name: s.name,
        status: 'pending',
        output: [],
      })),
    })
    return res.run_id
  }

  async function cancelPipeline(project: string, slug: string): Promise<void> {
    await devopsApi.cancelPipeline(project, slug)
    const run = activeRuns.value.get(slug)
    if (run) {
      run.overallStatus = 'cancelled'
    }
  }

  // WebSocket event handlers

  function handleRunStarted(payload: Record<string, unknown>): void {
    const slug = payload['pipeline_slug'] as string
    const runId = payload['run_id'] as string
    if (!slug || !runId) return
    const pipeline = pipelines.value.find((p) => p.slug === slug)
    activeRuns.value.set(slug, {
      runId,
      overallStatus: 'running',
      steps: (pipeline?.steps ?? []).map((s) => ({
        name: s.name,
        status: 'pending',
        output: [],
      })),
    })
  }

  function handleStepStarted(payload: Record<string, unknown>): void {
    const slug = payload['pipeline_slug'] as string
    const stepIndex = payload['step_index'] as number
    if (!slug) return
    const run = activeRuns.value.get(slug)
    if (!run || stepIndex == null || stepIndex >= run.steps.length) return
    run.steps[stepIndex].status = 'running'
    run.steps[stepIndex].startedAt = Date.now()
  }

  function handleStepOutput(payload: Record<string, unknown>): void {
    const slug = payload['pipeline_slug'] as string
    const stepIndex = payload['step_index'] as number
    const line = payload['output'] as string
    if (!slug || line == null) return
    const run = activeRuns.value.get(slug)
    if (!run || stepIndex == null || stepIndex >= run.steps.length) return
    // Cap buffer at 1000 lines to avoid unbounded memory growth
    const stepOutput = run.steps[stepIndex].output
    if (stepOutput.length >= 1000) {
      stepOutput.shift()
    }
    stepOutput.push(line)
  }

  function handleStepCompleted(payload: Record<string, unknown>): void {
    const slug = payload['pipeline_slug'] as string
    const stepIndex = payload['step_index'] as number
    const status = payload['status'] as StepStatus
    const durationMs = payload['duration_ms'] as number | undefined
    if (!slug) return
    const run = activeRuns.value.get(slug)
    if (!run || stepIndex == null || stepIndex >= run.steps.length) return
    run.steps[stepIndex].status = status
    run.steps[stepIndex].completedAt = Date.now()
    if (durationMs != null) run.steps[stepIndex].durationMs = durationMs
  }

  function handleRunCompleted(payload: Record<string, unknown>): void {
    const slug = payload['pipeline_slug'] as string
    const status = payload['status'] as ActiveRun['overallStatus']
    if (!slug) return
    const run = activeRuns.value.get(slug)
    if (!run) return
    run.overallStatus = status ?? 'passed'
  }

  return {
    pipelines,
    loading,
    loadError,
    activeRuns,
    pipelinesByType,
    fetchPipelines,
    runPipeline,
    cancelPipeline,
    handleRunStarted,
    handleStepStarted,
    handleStepOutput,
    handleStepCompleted,
    handleRunCompleted,
  }
})
