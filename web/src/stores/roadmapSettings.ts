// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getConfig, parseConfigYaml } from '@/api/config'

export type PeriodMode = 'autoscale' | 'fixed'
export type FixedPeriod = 'month' | 'quarter' | 'half-year' | 'year'

export const useRoadmapSettingsStore = defineStore('roadmapSettings', () => {
  const periodMode = ref<PeriodMode>('autoscale')
  const fixedPeriod = ref<FixedPeriod>('month')
  /** True once the config-API default has been applied; prevents overwriting a user selection on re-mount. */
  const defaultPeriodModeLoaded = ref(false)

  async function loadDefaultPeriodMode(project: string): Promise<void> {
    if (defaultPeriodModeLoaded.value) return
    try {
      const { raw } = await getConfig(project)
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const config = parseConfigYaml(raw) as any
      const defaultMode = config?.roadmap?.default_period_mode as string | undefined
      const fixedPeriods: FixedPeriod[] = ['month', 'quarter', 'half-year', 'year']
      if (defaultMode && fixedPeriods.includes(defaultMode as FixedPeriod)) {
        periodMode.value = 'fixed'
        fixedPeriod.value = defaultMode as FixedPeriod
      } else {
        periodMode.value = 'autoscale'
      }
    } catch {
      // non-fatal; default to autoscale
    } finally {
      defaultPeriodModeLoaded.value = true
    }
  }

  return { periodMode, fixedPeriod, defaultPeriodModeLoaded, loadDefaultPeriodMode }
})
