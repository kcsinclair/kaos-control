import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as agentsApi from '@/api/agents'
import type { AgentRunRow, AgentSummary } from '@/types/api'

// formatEvent renders a parsed stream event as a single line of text suitable
// for the live progress panel.
//
// Handles two event shapes:
//   - Claude Code stream-json: type in {system, assistant, user, result}
//   - Ollama driver:           type in {started, output, completed, error}
//
// Falls back to raw JSON on any unknown shape so we never silently drop info.
function formatEvent(ev: Record<string, unknown>): string {
  const type = ev.type as string | undefined

  // ── Ollama driver events ────────────────────────────────────────────────
  if (type === 'started') return '▸ started'
  if (type === 'output') {
    const text = typeof ev.text === 'string' ? ev.text : ''
    return text.trimEnd()
  }
  if (type === 'completed') return '▸ completed'
  if (type === 'error') {
    const msg = typeof ev.message === 'string' ? ev.message : JSON.stringify(ev)
    return `✗ ${msg}`
  }

  // ── Claude Code stream-json events ─────────────────────────────────────
  if (type === 'system' && ev.subtype === 'init') {
    return '▸ session started'
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
  return JSON.stringify(ev)
}

export const useAgentsStore = defineStore('agents', () => {
  const runs = ref<AgentRunRow[]>([])
  const agents = ref<AgentSummary[]>([])
  const loading = ref(false)
  // Per-run progress lines (live stdout), capped at 200 lines each.
  const progressLines = ref(new Map<string, string[]>())

  // Per-artifact run list (for the artifact detail modal).
  const artifactRuns = ref<AgentRunRow[]>([])
  const artifactRunsPath = ref<string>('')
  // Stores the last project used with fetchRunsByTargetPath so WS events can re-fetch.
  let _lastArtifactProject = ''

  const activeRuns = computed(() => runs.value.filter((r) => r.status === 'running'))

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

  async function startRun(
    project: string,
    agentName: string,
    targetPath: string,
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
    const runId = payload.run_id as string | undefined
    if (!runId) return

    if (type === 'agent.started') {
      if (!runs.value.find((r) => r.run_id === runId)) {
        runs.value.unshift({
          run_id: runId,
          agent_name: (payload.agent as string) ?? '',
          role: '',
          target_path: (payload.lineage as string) ?? '',
          started_at: new Date().toISOString(),
          status: 'running',
          stderr_tail: '',
          artifacts_produced: [],
        })
      }
    } else if (type === 'agent.progress') {
      const event = payload.event as Record<string, unknown> | undefined
      const raw = (payload.raw as string) ?? (payload.line as string) ?? ''
      const formatted = event ? formatEvent(event) : raw
      if (!progressLines.value.has(runId)) progressLines.value.set(runId, [])
      const lines = progressLines.value.get(runId)!
      lines.push(formatted)
      if (lines.length > 200) lines.splice(0, lines.length - 200)
    } else if (type === 'agent.finished' || type === 'agent.failed') {
      const idx = runs.value.findIndex((r) => r.run_id === runId)
      const newStatus = (payload.status as string) ?? (type === 'agent.finished' ? 'done' : 'failed')
      if (idx >= 0) {
        runs.value[idx] = {
          ...runs.value[idx],
          status: newStatus,
          finished_at: new Date().toISOString(),
          artifacts_produced: (payload.artifacts as string[]) ?? [],
        }
      }
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

  return {
    runs,
    agents,
    loading,
    progressLines,
    activeRuns,
    artifactRuns,
    artifactRunsPath,
    fetchRuns,
    fetchAgents,
    startRun,
    killRun,
    fetchRunsByTargetPath,
    onWsEvent,
  }
})
