<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { api } from '@/api/client'
import { use, init } from 'echarts/core'
import { PieChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { ECharts } from 'echarts/core'

use([PieChart, TooltipComponent, LegendComponent, CanvasRenderer])

const props = defineProps<{ project: string }>()

interface StageDistributionItem {
  stage: string
  count: number
}

interface StageDistributionResponse {
  distribution: StageDistributionItem[]
}

const chartEl = ref<HTMLDivElement | null>(null)
let chart: ECharts | null = null
const isEmpty = ref(false)

// WCAG 2.1 AA compliant palette, visually distinct from the status palette
const STAGE_COLORS: Record<string, string> = {
  'ideas':          '#0ea5e9',
  'requirements':   '#f97316',
  'backend-plans':  '#14b8a6',
  'frontend-plans': '#a855f7',
  'test-plans':     '#eab308',
  'tests':          '#22c55e',
  'prototypes':     '#64748b',
  'releases':       '#e11d48',
  'sprints':        '#06b6d4',
  'defects':        '#dc2626',
}

function colorForStage(stage: string): string {
  return STAGE_COLORS[stage] ?? '#94a3b8'
}

async function fetchAndRender() {
  try {
    const data = await api.get<StageDistributionResponse>(
      `/p/${encodeURIComponent(props.project)}/dashboard/stage-distribution`
    )
    const items = data.distribution ?? []
    isEmpty.value = items.length === 0 || items.every((i) => i.count === 0)

    if (isEmpty.value || !chart) return

    const seriesData = items.map((i) => ({
      name: i.stage,
      value: i.count,
      itemStyle: { color: colorForStage(i.stage) },
    }))

    chart.setOption({
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)',
      },
      legend: {
        orient: 'horizontal',
        bottom: 0,
        textStyle: { fontSize: 11 },
      },
      series: [
        {
          name: 'Stage',
          type: 'pie',
          radius: ['40%', '70%'],
          avoidLabelOverlap: true,
          cursor: 'pointer',
          label: { show: false },
          emphasis: { label: { show: false } },
          data: seriesData,
        },
      ],
    })
  } catch {
    isEmpty.value = true
  }
}

function initChart() {
  if (!chartEl.value) return
  chart = init(chartEl.value)
  void fetchAndRender()
}

onMounted(() => {
  initChart()
})

onUnmounted(() => {
  chart?.dispose()
  chart = null
})

watch(() => props.project, () => {
  void fetchAndRender()
})
</script>

<template>
  <div class="stages-dist-widget">
    <h3 class="widget-title">Stages Distribution</h3>
    <div
      v-if="!isEmpty"
      ref="chartEl"
      class="stages-dist-chart"
    />
    <div v-else class="widget-empty">No artifacts yet</div>
  </div>
</template>

<style scoped>
.stages-dist-widget {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.widget-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.stages-dist-chart {
  width: 100%;
  height: 280px;
}

.widget-empty {
  height: 280px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
</style>
