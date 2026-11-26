// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as agentsApi from '@/api/agents'
import type {
  AgentRunRow,
  AgentSummary,
  RunResult,
  PermissionDecision,
  QuotaStatusPayload,
  WarmupState,
  AgyProgressEvent,
} from '@/types/api'

function formatRawProgressLine(raw: string): string {
  const trimmed = raw.trim()
  if (trimmed.startsWith('# turn')) {
    return `▸ ${trimmed.replace(/^#\s*/, '')}`
  }
  if (trimmed.startsWith('# executing tool')) {
    return `▸ ${trimmed.replace(/^#\s*/, '')}`
  }
  if (trimmed.startsWith('# tool result')) {
    return `  ✓ ${trimmed.replace(/^#\s*/, '')}`
  }
  if (trimmed.includes('recovered') && trimmed.includes('native tool call')) {
    return `⚡ ${trimmed.replace(/^#\s*/, '')}`
  }
  if (trimmed.startsWith('# event: completed')) {
    return '▸ completed'
  }
  if (trimmed.startsWith('# error:')) {
    return `✗ ${trimmed.replace(/^# error:\s*/, '')}`
  }
  if (trimmed.startsWith('# error executing tool:')) {
    return `✗ ${trimmed.replace(/^# error executing tool:\s*/, '')}`
  }
  if (trimmed.startsWith('#')) {
    return '' // Suppress meta header comments from live progress window
  }
  return raw
}

// formatEvent renders a parsed stream event as a single line of text suitable
// for the live progress panel.
//
// Handles event shapes from:
//   - OpenAI-compatible driver: choices with deltas/tool_calls/finish_reason
//   - Claude Code stream-json:  type in {system, assistant, user, result}
//   - Ollama driver:            type in {started, output, completed, error}
//
// Falls back to raw JSON on any unknown shape so we never silently drop info.
function formatEvent(ev: Record<string, unknown>): string {
  const type = ev.type as string | undefined

  // ── OpenAI-compatible streaming & multi-turn events ─────────────────────
  if (Array.isArray(ev.choices)) {
    const choices = ev.choices as Array<{
      delta?: {
        role?: string
        content?: string
        tool_calls?: Array<{
          index?: number
          id?: string
          type?: string
          function?: { name?: string; arguments?: string }
        }>
      }
      finish_reason?: string | null
    }>
    if (choices.length > 0) {
      const choice = choices[0]
      if (choice.delta?.tool_calls?.length) {
        const calls = choice.delta.tool_calls.map((tc) => {
          const fn = tc.function?.name ?? 'tool'
          const args = tc.function?.arguments ?? ''
          return `▸ tool call: ${fn}${args ? ' ' + args : ''}`
        })
        return calls.join('  ')
      }
      if (choice.delta?.content) {
        return choice.delta.content.trimEnd()
      }
      if (choice.finish_reason) {
        return `▸ finish: ${choice.finish_reason}`
      }
    }
  }

  // ── Ollama driver events ────────────────────────────────────────────────
  if (type === 'started') return '▸ started'
  if (type === 'output') {
    const text = typeof ev.text === 'string' ? ev.text : ''
    return text.trimEnd()
  }
  if (type === 'completed') {
    const resp = typeof ev.response === 'string' ? ev.response : ''
    return resp ? `▸ completed: ${resp}` : '▸ completed'
  }
  if (type === 'error') {
    const msg = typeof ev.message === 'string' ? ev.message : JSON.stringify(ev)
    return `✗ ${msg}`
  }

  // ── Claude Code stream-json events ─────────────────────────────────────
  if (type === 'system' && ev.subtype === 'init') {
    return '▸ session started'
  }
  if (type === 'system' && ev.subtype === 'api_retry') {
    // Claude's own internal retry on transient API errors (529, 429, etc).
    // Surfaced so the run doesn't look frozen during multi-minute backoff
    // sequences — without this the user sees a silent gap until the final
    // result event.
    const attempt = ev.attempt as number | undefined
    const maxRetries = ev.max_retries as number | undefined
    const delayMs = ev.retry_delay_ms as number | undefined
    const status = ev.error_status as number | undefined
    const code = status ? `${status}` : 'API error'
    const delaySec = typeof delayMs === 'number' ? (delayMs / 1000).toFixed(1) : '?'
    const counter = attempt !== undefined && maxRetries !== undefined ? `${attempt}/${maxRetries}` : ''
    return `↻ retrying after ${code} (attempt ${counter}, ${delaySec}s backoff)`
  }
  if (type === 'assistant') {
    const msg = ev.message as Record<string, unknown> | undefined
    const content = msg?.content as Array<Record<string, unknown>> | undefined
    if (content && content.length) {
      const parts: string[] = []
      for (const block of content) {
        if (block.type === 'text' && typeof block.text === 'string') {
          parts.push(block.text.trim())
        } else if (block.type === 'tool_use') {
          const name = block.name as string
          const input = block.input as Record<string, unknown> | undefined
          const target = (input?.file_path as string) ?? (input?.path as string) ?? (input?.command as string) ?? ''
          parts.push(`▸ ${name}${target ? ' ' + target : ''}`)
        }
      }
      if (parts.length) return parts.join('  ')
    }
  }
  if (type === 'user') {
    const msg = ev.message as Record<string, unknown> | undefined
    const content = msg?.content as Array<Record<string, unknown>> | undefined
    if (content && content.length) {
      const block = content[0]
      if (block.type === 'tool_result') {
        const isErr = block.is_error === true
        return isErr ? '  ✗ tool error' : '  ✓ tool result'
      }
    }
  }
  if (type === 'result') {
    const subtype = (ev.subtype as string) ?? ''
    return `▸ result: ${subtype}`
  }

  // ── agy (gemini-cli) stream-json events ─────────────────────────────────
  // Discriminated by `event`, not `type` (agy events carry no `type` field,
  // so this branch is naturally disjoint from the ones above), with each
  // payload nested under a key matching the event name.
  if (typeof ev.event === 'string') {
    const agy = ev as unknown as AgyProgressEvent
    if (agy.event === 'init') {
      const cwd = agy.init?.cwd
      return cwd ? `▸ session started (${cwd})` : '▸ session started'
    }
    if (agy.event === 'step_update') {
      const step = agy.step_update
      const textDelta = step?.text_delta?.trim()
      if (textDelta) return textDelta
      const stepType = step?.step_type ?? 'step'
      const state = step?.state
      return `▸ ${stepType}${state ? `: ${state}` : ''}`
    }
    if (agy.event === 'result') {
      const status = agy.result?.status ?? ''
      return `▸ result: ${status.toLowerCase()}`
    }
  }

  return JSON.stringify(ev)
}

function formatPermissionEvent(ev: PermissionDecision): string {
  const icon = ev.decision === 'allow' ? '✓' : '✗'
  const target = ev.target_path ?? ev.command ?? ''
  return `[PERMISSION] ${icon} ${ev.tool_name}${target ? ' ' + target : ''} — ${ev.reason}`
}

export const useAgentsStore = defineStore('agents', () => {
  const runs = ref<AgentRunRow[]>([])
  const agents = ref<AgentSummary[]>([])
  const loading = ref(false)
  // Per-run progress lines (live stdout), capped at 200 lines each.
  const progressLines = ref(new Map<string, string[]>())

  // agent.status model_loading events broadcast before agent.started (during
  // preflight, before a run_id exists) — keyed by target_path and applied to
  // the run row once agent.started creates it (local-model-operability FR-2).
  const pendingWarmup = ref(new Map<string, { state: WarmupState; details?: string }>())

  // Per-artifact run list (for the artifact detail modal).
  const artifactRuns = ref<AgentRunRow[]>([])
  const artifactRunsPath = ref<string>('')
  // Stores the last project used with fetchRunsByTargetPath so WS events can re-fetch.
  let _lastArtifactProject = ''

  /** Ready-artifact count per agent name, populated by fetchReadyCounts(). */
  const readyCounts = ref<Record<string, number>>({})

  /** Run results received via WebSocket finish events, keyed by run_id. */
  const runResults = ref(new Map<string, RunResult>())

  /** Permission decisions received via agent.permission WS events, keyed by run_id. */
  const permissionEvents = ref(new Map<string, PermissionDecision[]>())

  /** Latest quota status received via agent.quota_status WS events, keyed by run_id. */
  const quotaByRun = ref(new Map<string, QuotaStatusPayload>())

  const activeRuns = computed(() => runs.value.filter((r) => r.status === 'running'))

  /** Running-run count per agent name, derived from activeRuns. */
  const runningCountByAgent = computed(() =>
    activeRuns.value.reduce<Record<string, number>>((acc, run) => {
      acc[run.agent_name] = (acc[run.agent_name] ?? 0) + 1
      return acc
    }, {}),
  )

  async function fetchRuns(project: string, status?: string, limit = 100): Promise<void> {
    loading.value = true
    try {
      const data = await agentsApi.listRuns(project, status, limit)
      runs.value = data.runs ?? []
    } finally {
      loading.value = false
    }
  }

  async function fetchAgents(project: string): Promise<void> {
    const data = await agentsApi.listAgents(project)
    agents.value = data.agents ?? []
  }

  async function fetchReadyCounts(project: string): Promise<void> {
    try {
      const data = await agentsApi.getReadyCounts(project)
      readyCounts.value = data.counts ?? {}
    } catch {
      // Non-fatal: counts remain stale until next successful fetch
    }
  }

  async function startRun(
    project: string,
    agentName: string,
    targetPath?: string,
    role?: string,
  ): Promise<string> {
    const data = await agentsApi.startRun(project, agentName, targetPath, role)
    return data.run_id
  }

  async function killRun(project: string, runId: string): Promise<void> {
    await agentsApi.killRun(project, runId)
  }

  async function fetchRunsByTargetPath(project: string, targetPath: string): Promise<void> {
    _lastArtifactProject = project
    artifactRunsPath.value = targetPath
    artifactRuns.value = await agentsApi.listRunsByTargetPath(project, targetPath)
  }

  function onWsEvent(type: string, payload: Record<string, unknown>): void {
    // agent.status (model_loading) is broadcast during preflight, before the
    // run has a run_id — match on target_path instead, and stash it for
    // agent.started to pick up if the run row doesn't exist yet.
    if (type === 'agent.status') {
      const targetPath = payload.target_path as string | undefined
      const state = payload.state as string | undefined
      if (!targetPath || state !== 'model_loading') return
      const details = payload.details as string | undefined
      const idx = runs.value.findIndex((r) => r.target_path === targetPath && r.status === 'running')
      if (idx >= 0) {
        runs.value[idx] = { ...runs.value[idx], warmup_state: 'model_loading', warmup_message: details ?? null }
      } else {
        pendingWarmup.value.set(targetPath, { state: 'model_loading', details })
      }
      return
    }

    const runId = payload.run_id as string | undefined
    if (!runId) return

    if (type === 'agent.started') {
      if (!runs.value.find((r) => r.run_id === runId)) {
        const targetPath = (payload.target_path as string) ?? ''
        const pending = pendingWarmup.value.get(targetPath)
        if (pending) pendingWarmup.value.delete(targetPath)
        runs.value.unshift({
          run_id: runId,
          agent_name: (payload.agent as string) ?? '',
          role: '',
          target_path: (payload.lineage as string) ?? '',
          started_at: new Date().toISOString(),
          status: 'running',
          stderr_tail: '',
          artifacts_produced: [],
          warmup_state: pending?.state ?? null,
          warmup_message: pending?.details ?? null,
        })
      }
    } else if (type === 'agent.progress') {
      const event = payload.event as Record<string, unknown> | undefined
      const raw = (payload.raw as string) ?? (payload.line as string) ?? ''
      const formatted = event ? formatEvent(event) : formatRawProgressLine(raw)
      if (formatted) {
        if (!progressLines.value.has(runId)) progressLines.value.set(runId, [])
        const lines = progressLines.value.get(runId)!
        lines.push(formatted)
        if (lines.length > 200) lines.splice(0, lines.length - 200)
      }
      // Warmup / TTFT stage transitions emitted by the openai-compatible
      // driver (local-model-operability FR-2/FR-5): "warming_up" once 5s
      // elapse without a token, "generating" on the first token received.
      const stage = event?.stage as string | undefined
      if (stage === 'warming_up' || stage === 'generating') {
        const idx = runs.value.findIndex((r) => r.run_id === runId)
        if (idx >= 0) {
          runs.value[idx] = {
            ...runs.value[idx],
            warmup_state: stage as import('@/types/api').WarmupState,
            warmup_message: (event?.message as string | undefined) ?? null,
          }
        }
      }
    } else if (type === 'agent.permission') {
      const ev = payload as unknown as PermissionDecision
      if (!permissionEvents.value.has(runId)) permissionEvents.value.set(runId, [])
      permissionEvents.value.get(runId)!.push(ev)
      // Also append a formatted line to the live progress log.
      if (!progressLines.value.has(runId)) progressLines.value.set(runId, [])
      const lines = progressLines.value.get(runId)!
      lines.push(formatPermissionEvent(ev))
      if (lines.length > 200) lines.splice(0, lines.length - 200)
    } else if (type === 'agent.quota_status') {
      quotaByRun.value.set(runId, payload as unknown as QuotaStatusPayload)
    } else if (type === 'agent.finished' || type === 'agent.failed') {
      const idx = runs.value.findIndex((r) => r.run_id === runId)
      const newStatus = (payload.status as string) ?? (type === 'agent.finished' ? 'done' : 'failed')
      if (idx >= 0) {
        runs.value[idx] = {
          ...runs.value[idx],
          status: newStatus,
          finished_at: new Date().toISOString(),
          artifacts_produced: (payload.artifacts as string[]) ?? [],
          failure_reason: (payload.failure_reason as string | null | undefined) ?? null,
          observed_permission_mode: (payload.observed_permission_mode as string | null | undefined) ?? null,
          remediation: (payload.remediation as string[] | null | undefined) ?? null,
          error_details: (payload.error_details as Record<string, unknown> | null | undefined) ?? null,
          denied_tool_calls: (payload.denied_tool_calls as import('@/types/api').DenialRecord[] | null | undefined) ?? null,
          ttft_ms: typeof payload.ttft_ms === 'number' ? payload.ttft_ms : runs.value[idx].ttft_ms,
          warmup_state: null,
          warmup_message: null,
        }
      }
      // Cache the result payload if provided by the backend.
      const wsResult = payload.result as RunResult | undefined
      if (wsResult) {
        runResults.value.set(runId, wsResult)
      }
      // The backend's per-run quota cache is cleared on run completion (FR5/AC6);
      // mirror that lifecycle here so a future badge never reads stale quota state.
      quotaByRun.value.delete(runId)
    }

    // Refresh per-artifact run list when a relevant event arrives.
    if (
      artifactRunsPath.value &&
      (type === 'agent.started' || type === 'agent.finished' || type === 'agent.failed')
    ) {
      const eventTargetPath = (payload.target_path as string) ?? (payload.lineage as string) ?? ''
      if (eventTargetPath === artifactRunsPath.value) {
        // Find the project from any existing run (we don't store it; use the path itself
        // as a cache key and rely on callers supplying project).
        // We re-use the last project supplied to fetchRunsByTargetPath via a closure variable.
        void _refreshArtifactRuns()
      }
    }
  }

  async function _refreshArtifactRuns(): Promise<void> {
    if (!_lastArtifactProject || !artifactRunsPath.value) return
    artifactRuns.value = await agentsApi.listRunsByTargetPath(_lastArtifactProject, artifactRunsPath.value)
  }

  function getRunResult(runId: string): RunResult | null {
    return runResults.value.get(runId) ?? null
  }

  function quotaForRun(runId: string): QuotaStatusPayload | null {
    return quotaByRun.value.get(runId) ?? null
  }

  return {
    runs,
    agents,
    loading,
    progressLines,
    permissionEvents,
    quotaByRun,
    activeRuns,
    runningCountByAgent,
    readyCounts,
    runResults,
    artifactRuns,
    artifactRunsPath,
    fetchRuns,
    fetchAgents,
    fetchReadyCounts,
    startRun,
    killRun,
    fetchRunsByTargetPath,
    getRunResult,
    quotaForRun,
    onWsEvent,
  }
})
