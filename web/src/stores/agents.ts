import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as agentsApi from '@/api/agents'
import type { AgentRunRow, AgentSummary } from '@/types/api'

export const useAgentsStore = defineStore('agents', () => {
  const runs = ref<AgentRunRow[]>([])
  const agents = ref<AgentSummary[]>([])
  const loading = ref(false)
  // Per-run progress lines (live stdout), capped at 200 lines each.
  const progressLines = ref(new Map<string, string[]>())

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
      const line = payload.line as string
      if (!progressLines.value.has(runId)) progressLines.value.set(runId, [])
      const lines = progressLines.value.get(runId)!
      lines.push(line)
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
  }

  return {
    runs,
    agents,
    loading,
    progressLines,
    activeRuns,
    fetchRuns,
    fetchAgents,
    startRun,
    killRun,
    onWsEvent,
  }
})
