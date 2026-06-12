// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { listDocs } from '@/api/docs'
import type { DocEntry } from '@/api/docs'

export type { DocEntry }

const BINARY_SUMMARY = '(binary or non-text file — cannot preview)'

export const useDocsStore = defineStore('docs', () => {
  const docs = ref<DocEntry[]>([])
  const docsDirPresent = ref(false)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const query = ref('')

  const filteredDocs = computed((): DocEntry[] => {
    if (!query.value) return docs.value
    const q = query.value.toLowerCase()
    return docs.value.filter((doc) => {
      // Binary/non-text docs are excluded from summary matching; only title matches
      if (doc.summary === BINARY_SUMMARY) {
        return doc.title.toLowerCase().includes(q)
      }
      return doc.title.toLowerCase().includes(q) || doc.summary.toLowerCase().includes(q)
    })
  })

  const groupedDocs = computed((): { subDir: string; docs: DocEntry[] }[] => {
    const groups = new Map<string, DocEntry[]>()
    for (const doc of filteredDocs.value) {
      const key = doc.sub_dir
      if (!groups.has(key)) groups.set(key, [])
      groups.get(key)!.push(doc)
    }
    // Root group ('') always first; remaining groups sorted alphabetically
    const keys = [...groups.keys()].sort((a, b) => {
      if (a === '') return -1
      if (b === '') return 1
      return a.localeCompare(b)
    })
    return keys.map((subDir) => ({ subDir, docs: groups.get(subDir)! }))
  })

  async function fetch(project: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const data = await listDocs(project)
      docs.value = data.docs ?? []
      docsDirPresent.value = data.docs_dir_present
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load documents'
    } finally {
      loading.value = false
    }
  }

  function setQuery(q: string): void {
    query.value = q
  }

  function clearQuery(): void {
    query.value = ''
  }

  async function applyDocChanged(project: string): Promise<void> {
    await fetch(project)
  }

  return {
    docs,
    docsDirPresent,
    loading,
    error,
    query,
    filteredDocs,
    groupedDocs,
    fetch,
    setQuery,
    clearQuery,
    applyDocChanged,
  }
})
