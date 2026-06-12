<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed } from 'vue'
import { use } from 'echarts/core'
import { ScatterChart } from 'echarts/charts'
import { TooltipComponent, GridComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { useECharts } from '@/composables/useECharts'

use([ScatterChart, TooltipComponent, GridComponent, LegendComponent, CanvasRenderer])

export interface ScatterPoint {
  run_id: string
  started_at: string
  agent_name: string
  model: string
  duration_ms: number
  total_cost_usd: number
  output_tokens_per_second: number | null
}

const props = defineProps<{ points: ScatterPoint[] }>()

const emit = defineEmits<{
  select: [runId: string]
}>()

const chartEl = ref<HTMLDivElement | null>(null)

const models = computed(() => [...new Set(props.points.map((p) => p.model))])

const COLORS = ['#6366f1', '#22c55e', '#f59e0b', '#ef4444', '#06b6d4', '#a855f7', '#ec4899']

const option = computed(() => {
  if (!props.points.length) {
    return { title: { text: 'No data', left: 'center', top: 'center', textStyle: { color: '#999', fontSize: 13 } } }
  }
  const series = models.value.map((model, i) => {
    const pts = props.points.filter((p) => p.model === model)
    return {
      name: model,
      type: 'scatter',
      data: pts.map((p) => ({
        value: [p.duration_ms / 1000, p.total_cost_usd],
        run_id: p.run_id,
        agent_name: p.agent_name,
        model: p.model,
        output_tokens_per_second: p.output_tokens_per_second,
      })),
      symbolSize: 7,
      itemStyle: { color: COLORS[i % COLORS.length] },
    }
  })
  return {
    tooltip: {
      trigger: 'item',
      formatter: (params: unknown) => {
        const p = params as { data: { value: number[]; run_id: string; agent_name: string; model: string; output_tokens_per_second: number | null } }
        const d = p.data
        const toks = d.output_tokens_per_second != null ? d.output_tokens_per_second.toFixed(1) : '—'
        return [
          `Run: ${d.run_id.slice(0, 8)}…`,
          `Agent: ${d.agent_name}`,
          `Model: ${d.model}`,
          `Duration: ${d.value[0].toFixed(1)}s`,
          `Cost: $${d.value[1].toFixed(4)}`,
          `Tokens/s: ${toks}`,
        ].join('<br/>')
      },
    },
    legend: { bottom: 0 },
    grid: { left: 70, right: 16, top: 16, bottom: 40 },
    xAxis: {
      type: 'value',
      name: 'Duration (s)',
      nameLocation: 'middle',
      nameGap: 30,
      axisLabel: { fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      name: 'Cost (USD)',
      nameLocation: 'middle',
      nameGap: 55,
      axisLabel: { fontSize: 11, formatter: (v: number) => `$${v.toFixed(3)}` },
    },
    series,
  }
})

const { chart } = useECharts(chartEl, option)

function onChartClick(params: unknown) {
  const p = params as { data?: { run_id?: string } }
  const runId = p?.data?.run_id
  if (runId) emit('select', runId)
}

// Attach click after chart instance is available on next tick
import { onMounted } from 'vue'
onMounted(() => {
  // The click listener must be registered after the chart is created.
  // useECharts initialises on onMounted, so we schedule this as a
  // microtask to run after that hook completes.
  Promise.resolve().then(() => {
    chart()?.on('click', onChartClick)
  })
})
</script>

<template>
  <div class="chart-wrap">
    <h3 class="chart-title">Cost vs. duration (per run)</h3>
    <div ref="chartEl" class="chart-canvas" role="img" aria-label="Cost vs duration scatter chart" />
  </div>
</template>

<style scoped>
.chart-wrap {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
}
.chart-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 var(--space-3);
}
.chart-canvas {
  width: 100%;
  height: 280px;
  cursor: crosshair;
}
</style>
