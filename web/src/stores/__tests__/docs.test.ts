// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDocsStore } from '../docs'
import type { DocEntry } from '../docs'

function makeDoc(overrides: Partial<DocEntry> = {}): DocEntry {
  return {
    path: 'readme.md',
    title: 'Readme',
    summary: 'A readme file',
    is_markdown: true,
    sub_dir: '',
    ...overrides,
  }
}

describe('useDocsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('filteredDocs', () => {
    it('returns all docs when query is empty', () => {
      const store = useDocsStore()
      store.docs = [makeDoc({ title: 'Alpha' }), makeDoc({ title: 'Beta', path: 'beta.md' })]
      expect(store.filteredDocs).toHaveLength(2)
    })

    it('matches on title case-insensitively', () => {
      const store = useDocsStore()
      store.docs = [
        makeDoc({ title: 'Architecture Overview', path: 'arch.md' }),
        makeDoc({ title: 'Readme', path: 'readme.md' }),
      ]
      store.setQuery('arch')
      expect(store.filteredDocs).toHaveLength(1)
      expect(store.filteredDocs[0].title).toBe('Architecture Overview')
    })

    it('matches on summary case-insensitively', () => {
      const store = useDocsStore()
      store.docs = [
        makeDoc({ title: 'Alpha', summary: 'Contains Architecture details', path: 'alpha.md' }),
        makeDoc({ title: 'Beta', summary: 'Something else', path: 'beta.md' }),
      ]
      store.setQuery('architecture')
      expect(store.filteredDocs).toHaveLength(1)
      expect(store.filteredDocs[0].title).toBe('Alpha')
    })

    it('excludes binary docs from results when query is non-empty and title does not match', () => {
      const store = useDocsStore()
      store.docs = [
        makeDoc({
          title: 'diagram.png',
          summary: '(binary or non-text file — cannot preview)',
          path: 'diagram.png',
          is_markdown: false,
        }),
        makeDoc({ title: 'Readme', path: 'readme.md' }),
      ]
      store.setQuery('read')
      expect(store.filteredDocs).toHaveLength(1)
      expect(store.filteredDocs[0].title).toBe('Readme')
    })

    it('includes binary docs when their title matches the query', () => {
      const store = useDocsStore()
      store.docs = [
        makeDoc({
          title: 'diagram.png',
          summary: '(binary or non-text file — cannot preview)',
          path: 'diagram.png',
          is_markdown: false,
        }),
      ]
      store.setQuery('diagram')
      expect(store.filteredDocs).toHaveLength(1)
    })

    it('shows binary docs when query is empty', () => {
      const store = useDocsStore()
      store.docs = [
        makeDoc({
          title: 'diagram.png',
          summary: '(binary or non-text file — cannot preview)',
          path: 'diagram.png',
          is_markdown: false,
        }),
      ]
      expect(store.filteredDocs).toHaveLength(1)
    })

    it('clearQuery resets to empty and shows all docs', () => {
      const store = useDocsStore()
      store.docs = [
        makeDoc({ title: 'Alpha', path: 'alpha.md' }),
        makeDoc({ title: 'Beta', path: 'beta.md' }),
      ]
      store.setQuery('alpha')
      expect(store.filteredDocs).toHaveLength(1)
      store.clearQuery()
      expect(store.filteredDocs).toHaveLength(2)
    })
  })

  describe('groupedDocs', () => {
    it('places root-level docs (sub_dir === "") first', () => {
      const store = useDocsStore()
      store.docs = [
        makeDoc({ title: 'Alpha', sub_dir: 'alpha', path: 'alpha/a.md' }),
        makeDoc({ title: 'Root', sub_dir: '', path: 'root.md' }),
      ]
      const groups = store.groupedDocs
      expect(groups[0].subDir).toBe('')
    })

    it('orders non-root groups alphabetically', () => {
      const store = useDocsStore()
      store.docs = [
        makeDoc({ title: 'Zeta', sub_dir: 'zeta', path: 'zeta/a.md' }),
        makeDoc({ title: 'Alpha', sub_dir: 'alpha', path: 'alpha/a.md' }),
        makeDoc({ title: 'Beta', sub_dir: 'beta', path: 'beta/a.md' }),
      ]
      const groups = store.groupedDocs
      expect(groups.map((g) => g.subDir)).toEqual(['alpha', 'beta', 'zeta'])
    })

    it('groups docs by sub_dir correctly', () => {
      const store = useDocsStore()
      store.docs = [
        makeDoc({ title: 'A', sub_dir: 'sub', path: 'sub/a.md' }),
        makeDoc({ title: 'B', sub_dir: 'sub', path: 'sub/b.md' }),
        makeDoc({ title: 'Root', sub_dir: '', path: 'root.md' }),
      ]
      const groups = store.groupedDocs
      expect(groups).toHaveLength(2)
      const subGroup = groups.find((g) => g.subDir === 'sub')
      expect(subGroup?.docs).toHaveLength(2)
    })

    it('returns empty array when filteredDocs is empty', () => {
      const store = useDocsStore()
      store.docs = []
      expect(store.groupedDocs).toHaveLength(0)
    })
  })
})
