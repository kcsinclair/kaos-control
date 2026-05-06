import { reactive } from 'vue'
import type { AsyncComponentLoader, Component } from 'vue'

export type WidgetSlot = 'summary' | 'chart' | 'panel'

export interface WidgetEntry {
  id: string
  component: Component | AsyncComponentLoader
  slot: WidgetSlot
  order: number
}

export const widgetList = reactive<WidgetEntry[]>([])

export function registerWidget(
  id: string,
  component: Component | AsyncComponentLoader,
  options: { slot: WidgetSlot; order: number },
): void {
  // Avoid duplicate registration (e.g. hot module reload)
  if (widgetList.some((w) => w.id === id)) return
  widgetList.push({ id, component, slot: options.slot, order: options.order })
  widgetList.sort((a, b) => {
    if (a.slot !== b.slot) return a.slot.localeCompare(b.slot)
    return a.order - b.order
  })
}
