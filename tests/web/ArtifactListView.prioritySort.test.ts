// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 4 — Priority column sort tests
 *
 * Verifies that clicking the Priority column header sorts rows by logical
 * severity order (not alphabetically). The sort uses a numeric mapping:
 *   critical=4, high=3, normal=2, low=1, (none)=0
 *
 * Sort cycle per useSortableTable: asc → desc → null (reset).
 *
 * Implementation: web/src/views/project/ArtifactListView.vue
 *   priorityOrder() + useSortableTable({ priority: { type: 'number', getValue } })
 *
 * Notes on implementation behaviour:
 *  - First click  → ascending (0=empty first, then low, normal, high, critical)
 *  - Second click → descending (critical first … empty last)
 *  - Third click  → reset to original insertion order
 *  - Artifacts with no priority map to numeric 0, so they sort alongside
 *    other numeric values (not via the null-last path in compareValues).
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import ArtifactListView from '../../web/src/views/project/ArtifactListView.vue'
import { useArtifactsStore } from '../../web/src/stores/artifacts'
import type { ArtifactRow } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/api/artifacts', () => ({
  listArtifacts:  vi.fn().mockResolvedValue({ items: [], total: 0 }),
  listLabels:     vi.fn().mockResolvedValue({ labels: [] }),
  listPriorities: vi.fn().mockResolvedValue({ priorities: [] }),
  getArtifact:    vi.fn().mockResolvedValue({ artifact: {}, body: '', body_html: '' }),
}))

vi.mock('@/api/releases', () => ({
  listReleases: vi.fn().mockResolvedValue([]),
}))

vi.mock('@/api/ws', () => ({
  getProjectWs: vi.fn(() => ({
    onType: vi.fn(() => () => {}),
    on:     vi.fn(() => () => {}),
  })),
}))

vi.mock('vue-router', async (importActual) => {
  const actual = await importActual<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute:  vi.fn(() => ({ params: { project: 'testproject' }, query: {} })),
    useRouter: vi.fn(() => ({ push: vi.fn(), replace: vi.fn() })),
  }
})

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeArtifact(overrides: Partial<ArtifactRow> = {}): ArtifactRow {
  return {
    path:      'lifecycle/ideas/test.md',
    slug:      'test',
    lineage:   'test',
    index:     1,
    stage:     'ideas',
    type:      'idea',
    status:    'draft',
    title:     'Test Artifact',
    frontmatter: {
      title:   'Test Artifact',
      type:    'idea',
      status:  'draft',
      lineage: 'test',
    },
    mtime:   '2024-01-15T00:00:00Z',
    created: '2024-01-01T00:00:00Z',
    ...overrides,
  }
}

// Returns a set of artifacts covering all four priority levels plus one with
// no priority. Insertion order: high, low, critical, normal, (none).
function makePriorityFixtures(): ArtifactRow[] {
  const mkP = (prio: string | undefined, lineage: string): ArtifactRow =>
    makeArtifact({
      path:     `lifecycle/ideas/${lineage}.md`,
      lineage,
      title:    lineage,
      frontmatter: {
        title:    lineage,
        type:     'idea',
        status:   'draft',
        lineage,
        ...(prio ? { priority: prio } : {}),
      },
    })

  return [
    mkP('high',     'prio-high'),     // priorityOrder = 3
    mkP('low',      'prio-low'),      // priorityOrder = 1
    mkP('critical', 'prio-critical'), // priorityOrder = 4
    mkP('normal',   'prio-normal'),   // priorityOrder = 2
    mkP(undefined,  'prio-none'),     // priorityOrder = 0
  ]
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div/>' } }],
  })
  return mount(ArtifactListView, {
    global: { plugins: [router] },
  })
}

// ---------------------------------------------------------------------------
// Column-click helpers
// ---------------------------------------------------------------------------

async function clickSortHeader(wrapper: ReturnType<typeof mountView>, label: string) {
  const headers = wrapper.findAll('th')
  const target = headers.find(th => th.text().includes(label))
  expect(target, `Could not find column header "${label}"`).toBeDefined()
  await target!.trigger('click')
}

/** Returns the text of the .priority-pill in each row, or '' if absent. */
function getPriorityValues(wrapper: ReturnType<typeof mountView>): string[] {
  return wrapper.findAll('tbody tr').map(row => {
    const pill = row.find('.priority-pill')
    return pill.exists() ? pill.text().trim() : ''
  })
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
})

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('ArtifactListView — Priority column sort', () => {
  it('TC1: first click (ascending) orders rows: empty, low, normal, high, critical', async () => {
    const fixtures = makePriorityFixtures()
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: fixtures, total: fixtures.length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Priority')

    const values = getPriorityValues(wrapper)
    expect(values.length).toBe(5)
    // Ascending: numeric 0(empty) first → 1(low) → 2(normal) → 3(high) → 4(critical)
    expect(values[0]).toBe('')        // no priority = 0
    expect(values[1]).toBe('low')
    expect(values[2]).toBe('normal')
    expect(values[3]).toBe('high')
    expect(values[4]).toBe('critical')
  })

  it('TC2: second click (descending) orders rows: critical, high, normal, low, empty', async () => {
    const fixtures = makePriorityFixtures()
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: fixtures, total: fixtures.length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Priority') // asc
    await clickSortHeader(wrapper, 'Priority') // desc

    const values = getPriorityValues(wrapper)
    expect(values.length).toBe(5)
    // Descending: numeric 4(critical) first → 3(high) → 2(normal) → 1(low) → 0(empty)
    expect(values[0]).toBe('critical')
    expect(values[1]).toBe('high')
    expect(values[2]).toBe('normal')
    expect(values[3]).toBe('low')
    expect(values[4]).toBe('')        // no priority = 0, last in desc
  })

  it('TC3: third click resets to the original insertion order', async () => {
    const fixtures = makePriorityFixtures()
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: fixtures, total: fixtures.length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Priority') // asc
    await clickSortHeader(wrapper, 'Priority') // desc
    await clickSortHeader(wrapper, 'Priority') // reset

    // No active sort indicator after reset
    const activeHeaders = wrapper.findAll('[aria-sort="ascending"], [aria-sort="descending"]')
    expect(activeHeaders.length).toBe(0)

    // Rows should be back in original insertion order:
    // high, low, critical, normal, none
    const values = getPriorityValues(wrapper)
    expect(values[0]).toBe('high')
    expect(values[1]).toBe('low')
    expect(values[2]).toBe('critical')
    expect(values[3]).toBe('normal')
    expect(values[4]).toBe('')
  })

  it('TC4: sort does not use alphabetical comparison (critical vs high vs low vs normal)', async () => {
    // Alphabetical ascending would be: critical < high < low < normal
    // Severity ascending must be: (none) < low < normal < high < critical
    const fixtures = makePriorityFixtures()
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: fixtures, total: fixtures.length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Priority') // ascending
    const values = getPriorityValues(wrapper)

    // Alphabetical ascending would put 'critical' before 'high' before 'low' before 'normal'.
    // Severity ascending puts 'low' before 'normal' before 'high' before 'critical'.
    const lowIdx      = values.indexOf('low')
    const normalIdx   = values.indexOf('normal')
    const highIdx     = values.indexOf('high')
    const criticalIdx = values.indexOf('critical')

    expect(lowIdx).toBeLessThan(normalIdx)
    expect(normalIdx).toBeLessThan(highIdx)
    expect(highIdx).toBeLessThan(criticalIdx)
  })
})
