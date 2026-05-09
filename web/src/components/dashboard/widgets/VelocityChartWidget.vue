<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
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

const MIN_PERIODS: Record<Granularity, number> = {
  daily: 7,
  weekly: 4,
  monthly: 3,
}

function subtractPeriod(period: string, gran: Granularity, n: number): string {
  if (gran === 'daily') {
    // period: "YYYY-MM-DD"
    const d = new Date(`${period}T00:00:00Z`)
    d.setUTCDate(d.getUTCDate() - n)
    return d.toISOString().slice(0, 10)
  }
  if (gran === 'weekly') {
    // period: "YYYY-WNN"
    const [yearStr, wStr] = period.split('-W')
    let year = parseInt(yearStr, 10)
    let week = parseInt(wStr, 10) - n
    while (week <= 0) {
      year -= 1
      week += isoWeeksInYear(year)
    }
    return `${String(year).padStart(4, '0')}-W${String(week).padStart(2, '0')}`
  }
  // monthly: "YYYY-MM"
  const [yearStr, monStr] = period.split('-')
  let year = parseInt(yearStr, 10)
  let month = parseInt(monStr, 10) - n
  while (month <= 0) {
    year -= 1
    month += 12
  }
  return `${String(year).padStart(4, '0')}-${String(month).padStart(2, '0')}`
}

function isoWeeksInYear(year: number): number {
  // A year has 53 ISO weeks if Jan 1 or Dec 31 is Thursday (for non-leap: Jan 1 Thu; leap: Jan 1 Wed or Thu)
  const jan1 = new Date(Date.UTC(year, 0, 1)).getUTCDay() // 0=Sun
  const dec31 = new Date(Date.UTC(year, 11, 31)).getUTCDay()
  return jan1 === 4 || dec31 === 4 ? 53 : 52
}

function todayPeriodKey(gran: Granularity): string {
  const now = new Date()
  if (gran === 'daily') return now.toISOString().slice(0, 10)
  if (gran === 'monthly') {
    return `${now.getUTCFullYear()}-${String(now.getUTCMonth() + 1).padStart(2, '0')}`
  }
  // weekly
  const d = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()))
  const day = d.getUTCDay() || 7 // Mon=1..Sun=7
  d.setUTCDate(d.getUTCDate() - day + 1) // move to Monday
  // ISO week
  const jan4 = new Date(Date.UTC(d.getUTCFullYear(), 0, 4))
  const week = Math.ceil(((d.getTime() - jan4.getTime()) / 86400000 + (jan4.getUTCDay() || 7) - 1) / 7) + 1
  return `${d.getUTCFullYear()}-W${String(week).padStart(2, '0')}`
}

function padBuckets(items: VelocityItem[], gran: Granularity): VelocityItem[] {
  const min = MIN_PERIODS[gran]
  if (items.length >= min) return items
  const needed = min - items.length
  const anchor = items.length > 0 ? items[0].period : todayPeriodKey(gran)
  const pads: VelocityItem[] = []
  for (let i = needed; i >= 1; i--) {
    pads.push({ period: subtractPeriod(anchor, gran, i), count: 0 })
  }
  return [...pads, ...items]
}

const MIN_BAR_WIDTH = 20  // px — threshold below which scrolling is needed
const MAX_BAR_WIDTH = 60  // px — cap to avoid excessively wide bars

const granularity = ref<Granularity>('daily')
const chartEl = ref<HTMLDivElement | null>(null)
const containerWidth = ref(0)
const chartHeight = ref(240)  // increases by 30px when DataZoom slider is shown
let chart: ECharts | null = null
const isEmpty = ref(false)
const ariaLabel = ref('Completion velocity chart loading')

// Cached data for re-render without re-fetch
let cachedPeriods: string[] = []
let cachedCounts: number[] = []
let cachedGranularity: Granularity = 'daily'

async function renderChart() {
  if (!chart || isEmpty.value) {
    if (isEmpty.value) chart?.clear()
    return
  }

  const periods = cachedPeriods
  const counts = cachedCounts
  const gran = cachedGranularity

  const maxVisibleBars = containerWidth.value > 0
    ? Math.floor(containerWidth.value / MIN_BAR_WIDTH)
    : periods.length
  const needsScroll = periods.length > maxVisibleBars

  chartHeight.value = needsScroll ? 270 : 240
  await nextTick()

  const dzStart = needsScroll
    ? Math.max(0, (1 - maxVisibleBars / periods.length) * 100)
    : 0
  const baseBottom = gran === 'daily' ? 60 : 40

  chart.setOption({
    tooltip: {
      trigger: 'axis',
      formatter: (params: unknown) => {
        const p = (params as Array<{ name: string; value: number }>)[0]
        return `${p.name}: ${p.value} completed`
      },
    },
    grid: { left: 40, right: 16, top: 16, bottom: needsScroll ? baseBottom + 40 : baseBottom },
    xAxis: {
      type: 'category',
      data: periods,
      axisLabel: {
        rotate: gran === 'daily' ? 45 : 0,
        fontSize: 11,
      },
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      axisLabel: { fontSize: 11 },
    },
    dataZoom: needsScroll
      ? [
          { type: 'inside', xAxisIndex: 0, start: dzStart, end: 100 },
          { type: 'slider', xAxisIndex: 0, start: dzStart, end: 100, height: 20 },
        ]
      : [],
    series: [
      {
        name: 'Completed',
        type: 'bar',
        data: counts,
        barMaxWidth: MAX_BAR_WIDTH,
        itemStyle: { color: '#6366f1', borderRadius: [3, 3, 0, 0] },
        emphasis: { itemStyle: { color: '#4f46e5' } },
      },
    ],
  })
  chart.resize()
}

async function fetchAndRender() {
  try {
    const data = await api.get<VelocityResponse>(
      `/p/${encodeURIComponent(props.project)}/dashboard/velocity?granularity=${granularity.value}&days=90`
    )
    const rawItems = data.buckets ?? []
    isEmpty.value = rawItems.length === 0 || rawItems.every((i) => i.count === 0)
    const items = padBuckets(rawItems, granularity.value)

    cachedGranularity = granularity.value
    cachedPeriods = items.map((i) => i.period)
    cachedCounts = items.map((i) => i.count)

    const realTotal = rawItems.reduce((s, i) => s + i.count, 0)
    ariaLabel.value = `Completion velocity ${granularity.value}: ${realTotal} completions over ${items.length} periods`

    await renderChart()
  } catch {
    isEmpty.value = true
  }
}

function initChart() {
  if (!chartEl.value) return
  containerWidth.value = chartEl.value.clientWidth
  chart = init(chartEl.value)
  void fetchAndRender()
}

// Inline 150ms debounce for resize callbacks
let resizeTimer: ReturnType<typeof setTimeout> | null = null
function debouncedResize(width: number) {
  if (resizeTimer !== null) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(() => {
    resizeTimer = null
    containerWidth.value = width
    void renderChart()
  }, 150)
}

const ro = typeof ResizeObserver !== 'undefined'
  ? new ResizeObserver((entries) => {
      const width = entries[0]?.contentRect.width ?? containerWidth.value
      debouncedResize(width)
    })
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
      :style="{ height: chartHeight + 'px' }"
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
