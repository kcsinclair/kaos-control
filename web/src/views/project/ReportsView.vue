<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAgentsStore } from '@/stores/agents'
import { useReportsStore } from '@/stores/reports'
import * as agentsApi from '@/api/agents'
import ReportsFilterBar from '@/components/reports/ReportsFilterBar.vue'
import SummaryTiles from '@/components/reports/SummaryTiles.vue'
import PerModelTable from '@/components/reports/PerModelTable.vue'
import RunsOverTimeChart from '@/components/reports/charts/RunsOverTimeChart.vue'
import OutputTokensPerSecChart from '@/components/reports/charts/OutputTokensPerSecChart.vue'
import TtftChart from '@/components/reports/charts/TtftChart.vue'
import CostPerRunChart from '@/components/reports/charts/CostPerRunChart.vue'
import CostDurationScatter from '@/components/reports/charts/CostDurationScatter.vue'
import type { ScatterPoint } from '@/components/reports/charts/CostDurationScatter.vue'

const route = useRoute()
const router = useRouter()
const agentsStore = useAgentsStore()
const reportsStore = useReportsStore()

const project = computed(() => route.params.project as string)

// Universe of agent names for the filter bar
const agentNames = computed(() => agentsStore.agents.map((a) => a.name))

// Scatter chart data: built from a sample of recent completed runs
const scatterPoints = ref<ScatterPoint[]>([])

async function loadScatterPoints(proj: string) {
  try {
    const data = await agentsApi.listRuns(proj, undefined, 50)
    const finished = (data.runs ?? []).filter(
      (r) => r.status === 'done' || r.status === 'success' || r.status === 'failure' || r.status === 'failed',
    ).slice(0, 20)

    const results = await Promise.all(
      finished.map((r) => agentsApi.getRunResult(proj, r.run_id)),
    )

    const modelByAgent: Record<string, string> = {}
    for (const a of agentsStore.agents) {
      modelByAgent[a.name] = a.model ?? a.driver
    }

    const pts: ScatterPoint[] = []
    for (let i = 0; i < finished.length; i++) {
      const run = finished[i]
      const { result } = results[i]
      if (!result) continue
      const outputTps =
        result.duration_ms > 0 && result.usage.output_tokens > 0
          ? result.usage.output_tokens / (result.duration_ms / 1000)
          : null
      pts.push({
        run_id: run.run_id,
        started_at: run.started_at,
        agent_name: run.agent_name,
        model: modelByAgent[run.agent_name] ?? 'unknown',
        duration_ms: result.duration_ms,
        total_cost_usd: result.total_cost_usd,
        output_tokens_per_second: outputTps,
      })
    }
    scatterPoints.value = pts
  } catch {
    scatterPoints.value = []
  }
}

async function loadAll(proj: string) {
  if (!agentsStore.agents.length) {
    await agentsStore.fetchAgents(proj)
  }
  if (!reportsStore.report) {
    await reportsStore.fetch(proj)
  }
  void loadScatterPoints(proj)
}

onMounted(() => {
  void loadAll(project.value)
})

watch(
  () => route.params.project,
  (p) => {
    if (p) {
      reportsStore.report = null
      void loadAll(p as string)
    }
  },
)

function onFilterUpdate(patch: Partial<import('@/types/api').AgentUsageFilter>) {
  reportsStore.setFilter(patch, project.value)
}

function onScatterSelect(runId: string) {
  // AgentsRunsView does not yet consume the run= query param; navigation
  // is wired to the correct URL so it will work once that is added.
  void router.push({ path: `/p/${project.value}/agents`, query: { run: runId } })
}

const isEmpty = computed(
  () =>
    reportsStore.report != null &&
    reportsStore.report.summary.overall.run_count === 0,
)
</script>

<template>
  <div class="reports-view">
    <h1 class="reports-title">Reports</h1>

    <ReportsFilterBar
      :agents="agentNames"
      :filter="reportsStore.filter"
      @update="onFilterUpdate"
    />

    <!-- Error banner -->
    <div v-if="reportsStore.error" class="error-banner">
      <span>{{ reportsStore.error }}</span>
      <button class="btn-retry" @click="reportsStore.fetch(project)">Retry</button>
    </div>

    <!-- Loading skeleton -->
    <template v-if="reportsStore.loading">
      <div class="skeleton-tiles">
        <div v-for="i in 6" :key="i" class="skeleton-tile" />
      </div>
      <div class="skeleton-chart" />
      <div class="skeleton-chart" />
    </template>

    <!-- Empty state -->
    <div v-else-if="isEmpty" class="empty-state">
      No agent runs in this window
    </div>

    <!-- Dashboard content -->
    <template v-else-if="reportsStore.report">
      <SummaryTiles :summary="reportsStore.report.summary" />

      <RunsOverTimeChart :series="reportsStore.report.series" />

      <OutputTokensPerSecChart :series-by-model="reportsStore.report.series_by_model" />

      <TtftChart :series-by-model="reportsStore.report.series_by_model" />

      <CostPerRunChart :series-by-model="reportsStore.report.series_by_model" />

      <CostDurationScatter :points="scatterPoints" @select="onScatterSelect" />

      <PerModelTable :rows="reportsStore.report.summary.per_model" />
    </template>
  </div>
</template>

<style scoped>
.reports-view {
  padding: var(--space-6);
  max-width: 1400px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
.reports-title {
  font-size: var(--text-xl);
  font-weight: 700;
  color: var(--color-text);
  margin: 0;
}
.error-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: var(--badge-blocked-bg, #fee2e2);
  color: var(--color-error, #dc2626);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
}
.btn-retry {
  background: none;
  border: 1px solid currentColor;
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: var(--text-xs);
  padding: var(--space-1) var(--space-2);
  color: inherit;
  white-space: nowrap;
}
.btn-retry:hover {
  opacity: 0.8;
}
.skeleton-tiles {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: var(--space-4);
}
.skeleton-tile {
  height: 90px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  animation: pulse 1.4s ease-in-out infinite;
}
.skeleton-chart {
  height: 280px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  animation: pulse 1.4s ease-in-out infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 240px;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
</style>
