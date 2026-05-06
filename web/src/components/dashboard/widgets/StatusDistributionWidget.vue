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

interface StatusDistributionItem {
  status: string
  count: number
}

interface StatusDistributionResponse {
  items: StatusDistributionItem[]
}

const chartEl = ref<HTMLDivElement | null>(null)
let chart: ECharts | null = null
const isEmpty = ref(false)
const ariaLabel = ref('Status distribution chart loading')

// WCAG 2.1 AA compliant palette against dark/light backgrounds
const STATUS_COLORS: Record<string, string> = {
  draft:          '#6366f1',
  clarifying:     '#f59e0b',
  planning:       '#3b82f6',
  'in-development': '#8b5cf6',
  'in-qa':        '#ec4899',
  approved:       '#10b981',
  rejected:       '#ef4444',
  abandoned:      '#6b7280',
}

function colorForStatus(status: string): string {
  return STATUS_COLORS[status] ?? '#94a3b8'
}

async function fetchAndRender() {
  try {
    const data = await api.get<StatusDistributionResponse>(
      `/p/${encodeURIComponent(props.project)}/dashboard/status-distribution`
    )
    const items = data.items ?? []
    isEmpty.value = items.length === 0 || items.every((i) => i.count === 0)

    if (isEmpty.value || !chart) return

    const seriesData = items.map((i) => ({
      name: i.status,
      value: i.count,
      itemStyle: { color: colorForStatus(i.status) },
    }))

    const total = items.reduce((s, i) => s + i.count, 0)
    ariaLabel.value = `Status distribution: ${items.map((i) => `${i.status} ${i.count} of ${total}`).join(', ')}`

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
          name: 'Status',
          type: 'pie',
          radius: ['40%', '70%'],
          avoidLabelOverlap: true,
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

const ro = typeof ResizeObserver !== 'undefined'
  ? new ResizeObserver(() => chart?.resize())
  : null

onMounted(() => {
  initChart()
  if (chartEl.value && ro) ro.observe(chartEl.value)
})

onUnmounted(() => {
  ro?.disconnect()
  chart?.dispose()
  chart = null
})

watch(() => props.project, () => {
  void fetchAndRender()
})
</script>

<template>
  <div class="status-dist-widget">
    <h3 class="widget-title">Status Distribution</h3>
    <div
      v-if="!isEmpty"
      ref="chartEl"
      class="status-dist-chart"
      role="img"
      :aria-label="ariaLabel"
    />
    <div v-else class="widget-empty">No tickets yet</div>
  </div>
</template>

<style scoped>
.status-dist-widget {
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

.status-dist-chart {
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
