// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as devopsApi from '@/api/devops'
import type { Pipeline, RunHistoryRow } from '@/api/devops'

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
  // Run IDs that have already reached a terminal status. Guards against a
  // late writer (the optimistic set in runPipeline, or an out-of-order
  // run.started) resurrecting a completed run back to 'running'.
  const completedRunIds = ref(new Set<string>())

  // Ordered list of run history (most recent last), capped at 50
  const runHistory = ref<RunHistoryEntry[]>([])

  // Per-pipeline persisted history from API, newest-first
  const pipelineHistory = ref(new Map<string, RunHistoryRow[]>())
  const historyLoading = ref(new Map<string, boolean>())
  const historyError = ref(new Map<string, string | null>())

  // ── Flat log buffer for PipelineLogPane ────────────────────────────────────
  // Buffers all events for the most recently active/selected pipeline.
  const logBuffer = ref<LogLine[]>([])
  /** Slug of the pipeline whose log is currently buffered */
  const logPipelineSlug = ref<string | null>(null)
  /** Run ID currently being buffered */
  const logRunId = ref<string | null>(null)
  /** True once pipeline.run.completed has been received for the buffered run */
  const logRunCompleted = ref(false)

  const anyRunning = computed((): boolean => {
    for (const [, run] of activeRuns.value) {
      if (run.overallStatus === 'running') return true
    }
    return false
  })

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

  function latestRunForPipeline(slug: string): RunHistoryRow | undefined {
    return pipelineHistory.value.get(slug)?.[0]
  }

  /**
   * Merge a freshly-fetched run list with whatever is already stored, instead of
   * blind-overwriting. Guards against the REST GET (issued on mount, before any
   * run has happened) resolving after a WS-driven `handleRunCompleted` update for
   * the same slug and clobbering it with a stale/empty list.
   */
  function mergeRunHistory(fetched: RunHistoryRow[], existing: RunHistoryRow[]): RunHistoryRow[] {
    const byRunId = new Map<string, RunHistoryRow>()
    for (const row of existing) byRunId.set(row.run_id, row)
    for (const row of fetched) byRunId.set(row.run_id, row)
    return Array.from(byRunId.values())
      .sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())
      .slice(0, 50)
  }

  async function fetchPipelineHistory(project: string, slug: string, limit = 10): Promise<void> {
    historyLoading.value.set(slug, true)
    historyError.value.set(slug, null)
    try {
      const res = await devopsApi.listPipelineRuns(project, slug, limit)
      const existing = pipelineHistory.value.get(slug) ?? []
      pipelineHistory.value.set(slug, mergeRunHistory(res.runs ?? [], existing))
    } catch (e: unknown) {
      historyError.value.set(slug, e instanceof Error ? e.message : 'Failed to load run history')
    } finally {
      historyLoading.value.set(slug, false)
    }
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
    // A fast pipeline can emit its WS run.completed before this POST resolves;
    // don't resurrect an already-completed run to 'running' (the completion
    // handlers already hold the correct terminal state + history row).
    if (completedRunIds.value.has(res.run_id)) return res.run_id
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

  async function createPipeline(project: string, slug: string, definition: string): Promise<Pipeline> {
    const res = await devopsApi.createPipeline(project, { slug, definition })
    // Re-fetch pipelines to get the full pipeline object including steps
    await fetchPipelines(project)
    return pipelines.value.find((p) => p.slug === res.slug)!
  }

  async function updatePipeline(
    project: string,
    slug: string,
    definition: string,
  ): Promise<devopsApi.PipelineResponse> {
    const res = await devopsApi.updatePipeline(project, slug, definition)
    await fetchPipelines(project)
    return res
  }

  function handlePipelineUpdated(project: string): void {
    void fetchPipelines(project)
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
    if (completedRunIds.value.has(runId)) return // already completed; don't resurrect
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
    completedRunIds.value.add(run.runId) // mark terminal so no later writer resurrects it
    // Re-set the map entry with a NEW object rather than mutating the existing
    // one in place. A nested-property mutation on a value held in a ref<Map>
    // does not reliably re-trigger the `activeRuns.get(slug)` dependency, so
    // `isActive` (activeRun.overallStatus === 'running') could stay true after
    // completion — leaving the card stuck showing "Running" and the latest-run
    // badge never rendering (the residual run-history flake). A tracked
    // Map.set of a fresh object flips it deterministically.
    activeRuns.value.set(slug, { ...run, overallStatus: finalStatus })
    // Update history entry
    const entry = runHistory.value.findLast((e) => e.runId === run.runId)
    if (entry) {
      entry.overallStatus = finalStatus
      entry.completedAt = Date.now()
    }

    // Prepend to persisted pipeline history (de-duplicate by run_id, trim to 50)
    const endedAt = new Date().toISOString()
    const startedAt = entry ? new Date(entry.startedAt).toISOString() : endedAt
    const newRow: RunHistoryRow = {
      run_id: run.runId,
      status: finalStatus,
      started_at: startedAt,
      ended_at: endedAt,
      duration_ms: durationMs ?? null,
    }
    const existing = pipelineHistory.value.get(slug) ?? []
    pipelineHistory.value.set(slug, mergeRunHistory([newRow], existing))

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
    anyRunning,
    pipelinesByType,
    historyForPipeline,
    // Persisted pipeline history
    pipelineHistory,
    historyLoading,
    historyError,
    fetchPipelineHistory,
    latestRunForPipeline,
    fetchPipelines,
    createPipeline,
    updatePipeline,
    runPipeline,
    cancelPipeline,
    fetchRunLog,
    handleRunStarted,
    handleStepStarted,
    handleStepOutput,
    handleStepCompleted,
    handleRunCompleted,
    handlePipelineUpdated,
    // Log buffer
    logBuffer,
    logPipelineSlug,
    logRunId,
    logRunCompleted,
    loadRunLog,
    clearLogBuffer,
  }
})
