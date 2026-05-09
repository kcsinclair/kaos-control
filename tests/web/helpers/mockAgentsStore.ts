// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Mock agents store helper for AppHeader run-indicator tests.
 *
 * Provides a factory that installs a Pinia store override for `useAgentsStore`
 * with a mutable `activeRuns` ref, allowing tests to simulate zero, one, or
 * many running agents without a live WebSocket connection or backend.
 */

import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import type { AgentRunRow } from '@/types/api'

/**
 * Build a minimal AgentRunRow in `running` status.
 */
export function makeRunningRun(id: string): AgentRunRow {
  return {
    run_id: id,
    agent_name: 'test-agent',
    role: 'backend-developer',
    target_path: 'lifecycle/plans/test.md',
    started_at: new Date().toISOString(),
    status: 'running',
    stderr_tail: '',
    artifacts_produced: [],
  }
}

/**
 * Factory that returns a Pinia store definition for `agents` whose
 * `activeRuns` is backed by the provided `runsRef`.
 *
 * Usage:
 *
 *   const runsRef = ref<AgentRunRow[]>([])
 *   const useMockAgentsStore = createMockAgentsStore(runsRef)
 *   // Override the real store in Pinia:
 *   pinia.use(({ store }) => {
 *     if (store.$id === 'agents') Object.assign(store, useMockAgentsStore())
 *   })
 *
 * Or simply pass `useMockAgentsStore` as the store mock via `vi.mock`.
 */
export function createMockAgentsStore(initialRuns: AgentRunRow[] = []) {
  const runsRef = ref<AgentRunRow[]>(initialRuns)

  const useMockAgentsStore = defineStore('agents', () => {
    const runs = runsRef
    const agents = ref([])
    const loading = ref(false)
    const progressLines = ref(new Map<string, string[]>())
    const artifactRuns = ref<AgentRunRow[]>([])
    const artifactRunsPath = ref('')

    const activeRuns = computed(() => runs.value.filter((r) => r.status === 'running'))

    function setRuns(newRuns: AgentRunRow[]) {
      runs.value = newRuns
    }

    return {
      runs,
      agents,
      loading,
      progressLines,
      activeRuns,
      artifactRuns,
      artifactRunsPath,
      setRuns,
      fetchRuns: async () => {},
      fetchAgents: async () => {},
      startRun: async () => '',
      killRun: async () => {},
      fetchRunsByTargetPath: async () => {},
      onWsEvent: () => {},
    }
  })

  return { useMockAgentsStore, runsRef }
}
