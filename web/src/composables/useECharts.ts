// SPDX-License-Identifier: AGPL-3.0-or-later

import { onMounted, onUnmounted, watch } from 'vue'
import type { Ref } from 'vue'
import { init } from 'echarts/core'
import type { ECharts, EChartsOption } from 'echarts/core'
import { useThemeStore } from '@/stores/theme'

export function useECharts(
  container: Ref<HTMLElement | null>,
  option: Ref<EChartsOption>,
) {
  let chart: ECharts | null = null
  const themeStore = useThemeStore()

  function initChart() {
    if (!container.value) return
    const theme = themeStore.isDark ? 'dark' : undefined
    chart = init(container.value, theme)
    chart.setOption(option.value, true)
  }

  function applyOption() {
    if (!chart) return
    chart.setOption(option.value, true)
  }

  let ro: ResizeObserver | null = null

  onMounted(() => {
    initChart()
    if (container.value && typeof ResizeObserver !== 'undefined') {
      ro = new ResizeObserver(() => {
        chart?.resize()
      })
      ro.observe(container.value)
    }
  })

  onUnmounted(() => {
    ro?.disconnect()
    chart?.dispose()
    chart = null
  })

  watch(option, applyOption, { deep: true })

  watch(
    () => themeStore.isDark,
    () => {
      chart?.dispose()
      chart = null
      initChart()
      if (container.value && ro) {
        ro.disconnect()
        ro.observe(container.value)
      }
    },
  )

  return { chart: () => chart }
}
