import { ref, computed, onMounted, onUnmounted, type Ref } from 'vue'

export const VIRTUAL_SCROLL_ROW_HEIGHT = 20 // px — monospace line height
const OVERSCAN = 25 // extra rows rendered above and below visible window

export interface VirtualItem<T> {
  item: T
  index: number
  offsetTop: number
}

export function useVirtualScroll<T>(items: Ref<T[]>, containerRef: Ref<HTMLElement | null>) {
  const scrollTop = ref(0)
  const containerHeight = ref(300)

  const totalHeight = computed(() => items.value.length * VIRTUAL_SCROLL_ROW_HEIGHT)

  const visibleRange = computed(() => {
    const first = Math.max(0, Math.floor(scrollTop.value / VIRTUAL_SCROLL_ROW_HEIGHT) - OVERSCAN)
    const last = Math.min(
      items.value.length,
      Math.ceil((scrollTop.value + containerHeight.value) / VIRTUAL_SCROLL_ROW_HEIGHT) + OVERSCAN,
    )
    return { first, last }
  })

  const visibleItems = computed((): VirtualItem<T>[] => {
    const { first, last } = visibleRange.value
    return items.value.slice(first, last).map((item, i) => ({
      item,
      index: first + i,
      offsetTop: (first + i) * VIRTUAL_SCROLL_ROW_HEIGHT,
    }))
  })

  function handleScroll(e: Event) {
    scrollTop.value = (e.currentTarget as HTMLElement).scrollTop
  }

  let ro: ResizeObserver | null = null

  onMounted(() => {
    if (containerRef.value) {
      containerHeight.value = containerRef.value.clientHeight
      ro = new ResizeObserver((entries) => {
        if (entries[0]) containerHeight.value = entries[0].contentRect.height
      })
      ro.observe(containerRef.value)
    }
  })

  onUnmounted(() => {
    ro?.disconnect()
  })

  return {
    scrollTop,
    totalHeight,
    visibleItems,
    handleScroll,
    ROW_HEIGHT: VIRTUAL_SCROLL_ROW_HEIGHT,
  }
}
