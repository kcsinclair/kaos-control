import { computed, ref, type Ref } from 'vue'

export type SortType = 'string' | 'date' | 'number' | 'text'
export type SortDirection = 'asc' | 'desc' | null

export interface SortColumnDef {
  type: SortType
  // Optional custom extractor for computed/derived sort values
  getValue?: (row: Record<string, unknown>) => unknown
}

export type SortColumnMap = Record<string, SortColumnDef | SortType>

function normalizeDef(def: SortColumnDef | SortType): SortColumnDef {
  return typeof def === 'string' ? { type: def } : def
}

export function useSortableTable<T extends Record<string, unknown>>(
  rows: Ref<T[]>,
  columns: SortColumnMap,
) {
  const sortColumn = ref<string | null>(null)
  const sortDirection = ref<SortDirection>(null)

  function toggleSort(column: string): void {
    if (!(column in columns)) return

    if (sortColumn.value !== column) {
      // Switching to a new column: reset to ascending
      sortColumn.value = column
      sortDirection.value = 'asc'
    } else {
      // Cycle: null -> asc -> desc -> null
      if (sortDirection.value === 'asc') {
        sortDirection.value = 'desc'
      } else if (sortDirection.value === 'desc') {
        sortColumn.value = null
        sortDirection.value = null
      } else {
        sortDirection.value = 'asc'
      }
    }
  }

  function resetSort(): void {
    sortColumn.value = null
    sortDirection.value = null
  }

  function extractValue(row: T, column: string): unknown {
    const raw = columns[column]
    if (!raw) return undefined
    const def = normalizeDef(raw)
    if (def.getValue) return def.getValue(row as Record<string, unknown>)
    return (row as Record<string, unknown>)[column]
  }

  function compareValues(a: unknown, b: unknown, type: SortType): number {
    // Nulls always sort last regardless of direction
    if (a == null && b == null) return 0
    if (a == null) return 1
    if (b == null) return -1

    switch (type) {
      case 'string':
      case 'text':
        return String(a).localeCompare(String(b), undefined, { sensitivity: 'base' })
      case 'date': {
        const aTime = new Date(String(a)).getTime()
        const bTime = new Date(String(b)).getTime()
        if (isNaN(aTime) && isNaN(bTime)) return 0
        if (isNaN(aTime)) return 1
        if (isNaN(bTime)) return -1
        return aTime - bTime
      }
      case 'number': {
        const aNum = Number(a)
        const bNum = Number(b)
        if (isNaN(aNum) && isNaN(bNum)) return 0
        if (isNaN(aNum)) return 1
        if (isNaN(bNum)) return -1
        return aNum - bNum
      }
    }
  }

  const sortedRows = computed((): T[] => {
    const col = sortColumn.value
    const dir = sortDirection.value

    if (!col || !dir) return rows.value

    const raw = columns[col]
    if (!raw) return rows.value
    const def = normalizeDef(raw)

    return [...rows.value].sort((a, b) => {
      const aVal = extractValue(a, col)
      const bVal = extractValue(b, col)

      // Pin nulls to end regardless of sort direction.
      // Empty strings are NOT pinned — they sort naturally ('' < any non-empty
      // string), so they appear first in ascending and last in descending.
      const aIsNull = aVal == null
      const bIsNull = bVal == null
      if (aIsNull && bIsNull) return 0
      if (aIsNull) return 1
      if (bIsNull) return -1

      const cmp = compareValues(aVal, bVal, def.type)
      return dir === 'asc' ? cmp : -cmp
    })
  })

  return {
    sortColumn,
    sortDirection,
    sortedRows,
    toggleSort,
    resetSort,
  }
}
