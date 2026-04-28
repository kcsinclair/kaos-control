import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

export interface UsePaginationOptions {
  defaultSize?: number
  queryPrefix?: string
}

export function usePagination(options: UsePaginationOptions = {}) {
  const { defaultSize = 25, queryPrefix = '' } = options
  const route = useRoute()
  const router = useRouter()

  const pageKey = queryPrefix ? `${queryPrefix}_page` : 'page'
  const sizeKey = queryPrefix ? `${queryPrefix}_size` : 'size'

  function parsePositiveInt(val: unknown, fallback: number): number {
    const n = parseInt(String(val), 10)
    return isNaN(n) || n < 1 ? fallback : n
  }

  const currentPage = ref(parsePositiveInt(route.query[pageKey], 1))
  const pageSize = ref(parsePositiveInt(route.query[sizeKey], defaultSize))

  const sliceStart = computed(() => (currentPage.value - 1) * pageSize.value)
  const sliceEnd = computed(() => currentPage.value * pageSize.value)

  function syncUrl() {
    const query = { ...route.query, [pageKey]: String(currentPage.value), [sizeKey]: String(pageSize.value) }
    router.replace({ query })
  }

  function setPage(n: number) {
    if (currentPage.value === n) return
    currentPage.value = n
    syncUrl()
  }

  function setPageSize(n: number) {
    pageSize.value = n
    currentPage.value = 1
    syncUrl()
  }

  // Sync back when URL changes externally (browser back/forward)
  watch(() => route.query[pageKey], (val) => {
    const n = parsePositiveInt(val, 1)
    if (currentPage.value !== n) currentPage.value = n
  })
  watch(() => route.query[sizeKey], (val) => {
    const n = parsePositiveInt(val, defaultSize)
    if (pageSize.value !== n) pageSize.value = n
  })

  return {
    currentPage,
    pageSize,
    sliceStart,
    sliceEnd,
    setPage,
    setPageSize,
  }
}
