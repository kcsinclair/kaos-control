<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed } from 'vue'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { TooltipComponent, GridComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { useECharts } from '@/composables/useECharts'
import type { AgentUsageBucketPoint } from '@/types/api'

use([LineChart, TooltipComponent, GridComponent, LegendComponent, CanvasRenderer])

const props = defineProps<{ seriesByModel: Record<string, AgentUsageBucketPoint[]> }>()

const chartEl = ref<HTMLDivElement | null>(null)

function fmtBucket(iso: string): string {
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' }).format(new Date(iso))
}

const option = computed(() => {
  const models = Object.keys(props.seriesByModel)
  if (!models.length) {
    return { title: { text: 'No data', left: 'center', top: 'center', textStyle: { color: '#999', fontSize: 13 } } }
  }
  const allBuckets = [...new Set(
    models.flatMap((m) => props.seriesByModel[m].map((p) => p.bucket_start))
  )].sort()
  const labels = allBuckets.map(fmtBucket)
  const series = models.map((model) => {
    const byBucket = Object.fromEntries(
      props.seriesByModel[model].map((p) => [p.bucket_start, p.mean_cost_usd])
    )
    return {
      name: model,
      type: 'line',
      data: allBuckets.map((b) => byBucket[b] ?? null),
      connectNulls: false,
      symbol: 'circle',
      symbolSize: 4,
    }
  })
  return {
    tooltip: {
      trigger: 'axis',
      valueFormatter: (v: unknown) => typeof v === 'number' ? `$${v.toFixed(4)}` : '—',
    },
    legend: { bottom: 0 },
    grid: { left: 70, right: 16, top: 16, bottom: 40 },
    xAxis: { type: 'category', data: labels, axisLabel: { fontSize: 11 } },
    yAxis: {
      type: 'value',
      name: 'USD',
      nameLocation: 'middle',
      nameGap: 50,
      axisLabel: { fontSize: 11, formatter: (v: number) => `$${v.toFixed(3)}` },
    },
    series,
  }
})

useECharts(chartEl, option)
</script>

<template>
  <div class="chart-wrap">
    <h3 class="chart-title">Mean cost/run by model</h3>
    <div ref="chartEl" class="chart-canvas" role="img" aria-label="Mean cost per run line chart" />
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
