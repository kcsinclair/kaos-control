// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as devopsApi from '@/api/devops'
import type { Pipeline } from '@/api/devops'

export type StepStatus = 'pending' | 'running' | 'passed' | 'failed' | 'cancelled'

// ── Log line types used by PipelineLogPane ────────────────────────────────────

export type LogLineKind = 'output' | 'step-start' | 'step-end' | 'run-start' | 'run-end'

export interface LogLine {
  kind: LogLineKind
  /** The step name, if associated with a step */
  stepName?: string
  stepIndex?: number
  timestamp: number
  /** Text to display. For output lines this is the raw command output. */
  text: string
  /** For step-end / run-end: 'passed' | 'failed' | 'cancelled' */
  status?: string
  /** Duration in ms for step-end / run-end lines */
  durationMs?: number
}

/** Maximum flat log buffer size (evict oldest beyond this) */
const LOG_BUFFER_MAX = 50_000

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

export interface RunHistoryEntry {
  runId: string
  pipelineSlug: string
  pipelineName: string
  startedAt: number
  completedAt?: number
  overallStatus: ActiveRun['overallStatus']
}

export const useDevOpsStore = defineStore('devops', () => {
  const pipelines = ref<Pipeline[]>([])
  const loading = ref(false)
  const loadError = ref<string | null>(null)

  // slug → ActiveRun
  const activeRuns = ref(new Map<string, ActiveRun>())

  // Ordered list of run history (most recent last), capped at 50
  const runHistory = ref<RunHistoryEntry[]>([])

  // ── Flat log buffer for PipelineLogPane ────────────────────────────────────
  // Buffers all events for the most recently active/selected pipeline.
  const logBuffer = ref<LogLine[]>([])
  /** Slug of the pipeline whose log is currently buffered */
  const logPipelineSlug = ref<string | null>(null)
  /** Run ID currently being buffered */
  const logRunId = ref<string | null>(null)
  /** True once pipeline.run.completed has been received for the buffered run */
  const logRunCompleted = ref(false)

  const pipelinesByType = computed((): Record<string, Pipeline[]> => {
    const grouped: Record<string, Pipeline[]> = {}
    for (const p of pipelines.value) {
      if (!grouped[p.type]) grouped[p.type] = []
      grouped[p.type].push(p)
    }
    return grouped
  })

  function historyForPipeline(slug: string): RunHistoryEntry[] {
    return runHistory.value.filter((e) => e.pipelineSlug === slug)
  }

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

  async function fetchRunLog(project: string, runId: string): Promise<string> {
    return devopsApi.getRunLog(project, runId)
  }

  // WebSocket event handlers

  function appendLogLine(line: LogLine): void {
    if (logBuffer.value.length >= LOG_BUFFER_MAX) {
      logBuffer.value.shift()
    }
    logBuffer.value.push(line)
  }

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
    // Track in history
    runHistory.value.push({
      runId,
      pipelineSlug: slug,
      pipelineName: pipeline?.name ?? slug,
      startedAt: Date.now(),
      overallStatus: 'running',
    })
    if (runHistory.value.length > 50) runHistory.value.shift()

    // Reset flat log buffer for this run
    logBuffer.value = []
    logPipelineSlug.value = slug
    logRunId.value = runId
    logRunCompleted.value = false
    appendLogLine({ kind: 'run-start', timestamp: Date.now(), text: `Run ${runId} started` })
  }

  function handleStepStarted(payload: Record<string, unknown>): void {
    const slug = payload['pipeline_slug'] as string
    const stepIndex = payload['step_index'] as number
    if (!slug) return
    const run = activeRuns.value.get(slug)
    if (!run || stepIndex == null || stepIndex >= run.steps.length) return
    run.steps[stepIndex].status = 'running'
    run.steps[stepIndex].startedAt = Date.now()

    // Append step-start line to flat log buffer
    if (slug === logPipelineSlug.value) {
      const stepName = run.steps[stepIndex].name
      appendLogLine({ kind: 'step-start', stepName, stepIndex, timestamp: Date.now(), text: stepName })
    }
  }

  function handleStepOutput(payload: Record<string, unknown>): void {
    const slug = payload['pipeline_slug'] as string
    const stepIndex = payload['step_index'] as number
    const line = payload['text'] as string
    if (!slug || line == null) return
    const run = activeRuns.value.get(slug)
    if (!run || stepIndex == null || stepIndex >= run.steps.length) return
    // Cap per-step buffer at 50,000 lines
    const stepOutput = run.steps[stepIndex].output
    if (stepOutput.length >= 50_000) {
      stepOutput.shift()
    }
    stepOutput.push(line)

    // Append output line to flat log buffer
    if (slug === logPipelineSlug.value) {
      const stepName = run.steps[stepIndex].name
      appendLogLine({ kind: 'output', stepName, stepIndex, timestamp: Date.now(), text: line })
    }
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

    // Append step-end line to flat log buffer
    if (slug === logPipelineSlug.value) {
      const stepName = run.steps[stepIndex].name
      appendLogLine({ kind: 'step-end', stepName, stepIndex, timestamp: Date.now(), text: stepName, status, durationMs })
    }
  }

  function handleRunCompleted(payload: Record<string, unknown>): void {
    const slug = payload['pipeline_slug'] as string
    const status = payload['status'] as ActiveRun['overallStatus']
    const durationMs = payload['duration_ms'] as number | undefined
    if (!slug) return
    const run = activeRuns.value.get(slug)
    if (!run) return
    const finalStatus = status ?? 'passed'
    run.overallStatus = finalStatus
    // Update history entry
    const entry = runHistory.value.findLast((e) => e.runId === run.runId)
    if (entry) {
      entry.overallStatus = finalStatus
      entry.completedAt = Date.now()
    }

    // Append terminal run-end line to flat log buffer
    if (slug === logPipelineSlug.value) {
      logRunCompleted.value = true
      appendLogLine({ kind: 'run-end', timestamp: Date.now(), text: '', status: finalStatus, durationMs })
    }
  }

  /** Load a completed run log from REST and replace the flat log buffer */
  async function loadRunLog(project: string, runId: string, pipelineSlug: string): Promise<void> {
    const raw = await devopsApi.getRunLog(project, runId)
    const lines = devopsApi.parseRunLog(raw)
    logBuffer.value = lines
    logPipelineSlug.value = pipelineSlug
    logRunId.value = runId
    logRunCompleted.value = true
  }

  function clearLogBuffer(): void {
    logBuffer.value = []
    logPipelineSlug.value = null
    logRunId.value = null
    logRunCompleted.value = false
  }

  return {
    pipelines,
    loading,
    loadError,
    activeRuns,
    runHistory,
    pipelinesByType,
    historyForPipeline,
    fetchPipelines,
    runPipeline,
    cancelPipeline,
    fetchRunLog,
    handleRunStarted,
    handleStepStarted,
    handleStepOutput,
    handleStepCompleted,
    handleRunCompleted,
    // Log buffer
    logBuffer,
    logPipelineSlug,
    logRunId,
    logRunCompleted,
    loadRunLog,
    clearLogBuffer,
  }
})
