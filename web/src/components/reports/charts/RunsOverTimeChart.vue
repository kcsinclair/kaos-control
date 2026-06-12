<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed } from 'vue'
import { use } from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { TooltipComponent, GridComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { useECharts } from '@/composables/useECharts'
import type { AgentUsageBucketPoint } from '@/types/api'

use([BarChart, TooltipComponent, GridComponent, LegendComponent, CanvasRenderer])

const props = defineProps<{ series: AgentUsageBucketPoint[] }>()

const chartEl = ref<HTMLDivElement | null>(null)

function fmtBucket(iso: string): string {
  const d = new Date(iso)
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' }).format(d)
}

const option = computed(() => {
  if (!props.series.length) {
    return { title: { text: 'No data', left: 'center', top: 'center', textStyle: { color: '#999', fontSize: 13 } } }
  }
  const labels = props.series.map((p) => fmtBucket(p.bucket_start))
  const success = props.series.map((p) => p.success_count)
  const failure = props.series.map((p) => p.failure_count)
  return {
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { data: ['Success', 'Failure'], bottom: 0 },
    grid: { left: 50, right: 16, top: 16, bottom: 40 },
    xAxis: { type: 'category', data: labels, axisLabel: { fontSize: 11 } },
    yAxis: { type: 'value', minInterval: 1, axisLabel: { fontSize: 11 } },
    series: [
      { name: 'Success', type: 'bar', stack: 'total', data: success, itemStyle: { color: '#22c55e' } },
      { name: 'Failure', type: 'bar', stack: 'total', data: failure, itemStyle: { color: '#ef4444' } },
    ],
  }
})

useECharts(chartEl, option)
</script>

<template>
  <div class="chart-wrap">
    <h3 class="chart-title">Runs over time</h3>
    <div ref="chartEl" class="chart-canvas" role="img" aria-label="Runs over time stacked bar chart" />
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
  height: 240px;
}
</style>
