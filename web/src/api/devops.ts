// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from '@/api/client'
import type { LogLine } from '@/stores/devops'

export interface PipelineStep {
  name: string
  description: string
}

export interface Pipeline {
  slug: string
  name: string
  type: string
  steps: PipelineStep[]
}

export interface PipelinesResponse {
  pipelines: Pipeline[]
}

export interface RunPipelineResponse {
  run_id: string
}

export function listPipelines(project: string): Promise<PipelinesResponse> {
  return api.get<PipelinesResponse>(`/p/${encodeURIComponent(project)}/devops/pipelines`)
}

export function runPipeline(project: string, slug: string): Promise<RunPipelineResponse> {
  return api.post<RunPipelineResponse>(`/p/${encodeURIComponent(project)}/devops/pipelines/${encodeURIComponent(slug)}/run`)
}

export function cancelPipeline(project: string, slug: string): Promise<void> {
  return api.post<void>(`/p/${encodeURIComponent(project)}/devops/pipelines/${encodeURIComponent(slug)}/cancel`)
}

export interface CreatePipelineRequest {
  slug: string
  definition: string
}

export interface CreatePipelineResponse {
  slug: string
  name: string
  type: string
  step_count: number
}

export function createPipeline(
  project: string,
  body: CreatePipelineRequest,
): Promise<CreatePipelineResponse> {
  return api.post<CreatePipelineResponse>(
    `/p/${encodeURIComponent(project)}/devops/pipelines`,
    body,
  )
}

export interface PipelineResponse {
  slug: string
  name: string
  type: string
  step_count: number
}

export function getPipelineDefinition(project: string, slug: string): Promise<string> {
  return api.getText(`/p/${encodeURIComponent(project)}/devops/pipelines/${encodeURIComponent(slug)}`)
}

export function updatePipeline(
  project: string,
  slug: string,
  definition: string,
): Promise<PipelineResponse> {
  return api.put<PipelineResponse>(
    `/p/${encodeURIComponent(project)}/devops/pipelines/${encodeURIComponent(slug)}`,
    { definition },
  )
}

export function getRunLog(project: string, runId: string): Promise<string> {
  return api.getText(`/p/${encodeURIComponent(project)}/devops/runs/${encodeURIComponent(runId)}`)
}

export interface RunHistoryRow {
  run_id: string
  status: string
  started_at: string
  ended_at: string | null
  duration_ms: number | null
  /** Email of the user who triggered the run. Present for CLI-originated runs once backend populates it. */
  triggered_by?: string
}

export interface RunsResponse {
  runs: RunHistoryRow[]
}

export function listPipelineRuns(
  project: string,
  slug: string,
  limit = 10,
): Promise<RunsResponse> {
  return api.get<RunsResponse>(
    `/p/${encodeURIComponent(project)}/devops/pipelines/${encodeURIComponent(slug)}/runs?limit=${limit}`,
  )
}

export function getPipelineRunLog(project: string, slug: string, runId: string): Promise<string> {
  return api.getText(
    `/p/${encodeURIComponent(project)}/devops/pipelines/${encodeURIComponent(slug)}/runs/${encodeURIComponent(runId)}/log`,
  )
}

/**
 * Parse a raw NDJSON run log (as returned by getRunLog) into LogLine objects
 * that PipelineLogPane can render identically to the live WebSocket stream.
 */
export function parseRunLog(raw: string | null | undefined): LogLine[] {
  if (!raw) return []
  const lines: LogLine[] = []
  for (const rawLine of raw.split('\n')) {
    const trimmed = rawLine.trim()
    if (!trimmed) continue
    try {
      const obj = JSON.parse(trimmed) as Record<string, unknown>
      const ts = (obj['timestamp'] as number | undefined) ?? Date.now()
      switch (obj['type']) {
        case 'pipeline.run.started':
          lines.push({ kind: 'run-start', timestamp: ts, text: `Run ${String(obj['run_id'] ?? '')} started` })
          break
        case 'pipeline.step.started':
          lines.push({
            kind: 'step-start',
            stepName: obj['step'] as string | undefined,
            stepIndex: obj['step_index'] as number | undefined,
            timestamp: ts,
            text: (obj['step'] as string | undefined) ?? `Step ${String(obj['step_index'] ?? '')}`,
          })
          break
        case 'pipeline.step.output':
          lines.push({
            kind: 'output',
            stepName: obj['step'] as string | undefined,
            stepIndex: obj['step_index'] as number | undefined,
            timestamp: ts,
            text: (obj['text'] as string | undefined) ?? '',
          })
          break
        case 'pipeline.step.completed':
          lines.push({
            kind: 'step-end',
            stepName: obj['step'] as string | undefined,
            stepIndex: obj['step_index'] as number | undefined,
            timestamp: ts,
            text: (obj['step'] as string | undefined) ?? `Step ${String(obj['step_index'] ?? '')}`,
            status: obj['status'] as string | undefined,
            durationMs: obj['duration_ms'] as number | undefined,
          })
          break
        case 'pipeline.run.completed':
          lines.push({
            kind: 'run-end',
            timestamp: ts,
            text: '',
            status: obj['status'] as string | undefined,
            durationMs: obj['duration_ms'] as number | undefined,
          })
          break
        default:
          // Preserve unknown event types as raw output lines so nothing is silently dropped
          lines.push({ kind: 'output', timestamp: ts, text: trimmed })
      }
    } catch {
      // Non-JSON lines treated as raw output
      lines.push({ kind: 'output', timestamp: Date.now(), text: trimmed })
    }
  }
  return lines
}
