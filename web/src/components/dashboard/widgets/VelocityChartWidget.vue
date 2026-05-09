<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { api } from '@/api/client'
import { use, init } from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { TooltipComponent, GridComponent, DataZoomComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { ECharts } from 'echarts/core'

use([BarChart, TooltipComponent, GridComponent, DataZoomComponent, CanvasRenderer])

const props = defineProps<{ project: string }>()

type Granularity = 'daily' | 'weekly' | 'monthly'

interface VelocityItem {
  period: string
  count: number
}

interface VelocityResponse {
  buckets: VelocityItem[]
  granularity: Granularity
}

const granularity = ref<Granularity>('daily')
const chartEl = ref<HTMLDivElement | null>(null)
let chart: ECharts | null = null
const isEmpty = ref(false)
const ariaLabel = ref('Completion velocity chart loading')

async function fetchAndRender() {
  try {
    const data = await api.get<VelocityResponse>(
      `/p/${encodeURIComponent(props.project)}/dashboard/velocity?granularity=${granularity.value}&days=90`
    )
    const items = data.buckets ?? []
    isEmpty.value = items.length === 0 || items.every((i) => i.count === 0)

    if (!chart) return
    if (isEmpty.value) {
      chart.clear()
      return
    }

    const periods = items.map((i) => i.period)
    const counts = items.map((i) => i.count)
    const total = counts.reduce((s, c) => s + c, 0)
    ariaLabel.value = `Completion velocity ${granularity.value}: ${total} completions over ${items.length} periods`

    chart.setOption({
      tooltip: {
        trigger: 'axis',
        formatter: (params: unknown) => {
          const p = (params as Array<{ name: string; value: number }>)[0]
          return `${p.name}: ${p.value} completed`
        },
      },
      grid: { left: 40, right: 16, top: 16, bottom: granularity.value === 'daily' ? 60 : 40 },
      xAxis: {
        type: 'category',
        data: periods,
        axisLabel: {
          rotate: granularity.value === 'daily' ? 45 : 0,
          fontSize: 11,
        },
      },
      yAxis: {
        type: 'value',
        minInterval: 1,
        axisLabel: { fontSize: 11 },
      },
      series: [
        {
          name: 'Completed',
          type: 'bar',
          data: counts,
          itemStyle: { color: '#6366f1', borderRadius: [3, 3, 0, 0] },
          emphasis: { itemStyle: { color: '#4f46e5' } },
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

watch(granularity, () => {
  void fetchAndRender()
})

watch(() => props.project, () => {
  void fetchAndRender()
})

const GRANULARITIES: { value: Granularity; label: string }[] = [
  { value: 'daily',   label: 'Daily' },
  { value: 'weekly',  label: 'Weekly' },
  { value: 'monthly', label: 'Monthly' },
]
</script>

<template>
  <div class="velocity-widget">
    <div class="velocity-header">
      <h3 class="widget-title">Completion Velocity</h3>
      <div
        class="granularity-toggle"
        role="group"
        aria-label="Granularity"
      >
        <button
          v-for="g in GRANULARITIES"
          :key="g.value"
          class="toggle-btn"
          :class="{ 'toggle-btn--active': granularity === g.value }"
          :aria-pressed="granularity === g.value"
          @click="granularity = g.value"
        >{{ g.label }}</button>
      </div>
    </div>
    <div
      v-if="!isEmpty"
      ref="chartEl"
      class="velocity-chart"
      role="img"
      :aria-label="ariaLabel"
    />
    <div v-else class="widget-empty">No completions in this period</div>
  </div>
</template>

<style scoped>
.velocity-widget {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.velocity-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.widget-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.granularity-toggle {
  display: flex;
  gap: 2px;
  background: var(--color-surface-hover, var(--color-sidebar-hover));
  border-radius: var(--radius-md);
  padding: 2px;
}

.toggle-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: var(--text-xs);
  padding: var(--space-1) var(--space-2);
  border-radius: calc(var(--radius-md) - 2px);
  color: var(--color-text-muted);
  transition: background 0.12s, color 0.12s;
}

.toggle-btn:hover,
.toggle-btn:focus-visible {
  background: var(--color-sidebar-hover);
  color: var(--color-text);
  outline: 2px solid var(--color-primary);
  outline-offset: 1px;
}

.toggle-btn--active {
  background: var(--color-surface);
  color: var(--color-text);
  font-weight: 600;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.velocity-chart {
  width: 100%;
  height: 240px;
}

.widget-empty {
  height: 240px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
</style>
