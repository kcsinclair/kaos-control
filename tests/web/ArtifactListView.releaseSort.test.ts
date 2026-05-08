/**
 * Milestone 5 — Release column sort tests
 *
 * Verifies that sorting by the Release column uses case-insensitive
 * alphabetical order (localeCompare with sensitivity: 'base').
 *
 * Implementation: web/src/views/project/ArtifactListView.vue
 *   release: { type: 'string', getValue: row => row.frontmatter?.release ?? '' }
 *
 * Notes on implementation behaviour:
 *  - Missing release maps to '' (empty string, not null), but useSortableTable
 *    pins empty strings to the END of the sorted list in both directions.
 *  - localeCompare with sensitivity:'base' treats 'Alpha' === 'alpha'.
 *  - Sort cycle: asc → desc → null (reset).
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

/** Returns a set of artifacts with releases: v1.0, v2.0, alpha, and (none). */
function makeReleaseFixtures(): ArtifactRow[] {
  const mkR = (release: string | undefined, lineage: string): ArtifactRow =>
    makeArtifact({
      path:    `lifecycle/ideas/${lineage}.md`,
      lineage,
      title:   lineage,
      frontmatter: {
        title:   lineage,
        type:    'idea',
        status:  'draft',
        lineage,
        ...(release !== undefined ? { release } : {}),
      },
    })

  // Insertion order: v1.0, v2.0, alpha, (none)
  return [
    mkR('v1.0',     'rel-v1'),
    mkR('v2.0',     'rel-v2'),
    mkR('alpha',    'rel-alpha'),
    mkR(undefined,  'rel-none'),
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

async function clickSortHeader(wrapper: ReturnType<typeof mountView>, label: string) {
  const headers = wrapper.findAll('th')
  const target = headers.find(th => th.text().includes(label))
  expect(target, `Could not find column header "${label}"`).toBeDefined()
  await target!.trigger('click')
}

/** Returns the text of .cell-release in each row. */
function getReleaseValues(wrapper: ReturnType<typeof mountView>): string[] {
  return wrapper.findAll('tbody tr').map(row => {
    const cell = row.find('.cell-release')
    return cell.exists() ? cell.text().trim() : ''
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

describe('ArtifactListView — Release column sort', () => {
  it('TC1: ascending sort orders rows alphabetically: alpha, v1.0, v2.0, (empty)', async () => {
    const fixtures = makeReleaseFixtures()
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: fixtures, total: fixtures.length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Release') // ascending

    const values = getReleaseValues(wrapper)
    expect(values.length).toBe(4)
    // 'alpha' < 'v1.0' < 'v2.0', missing release (empty string) pinned to end
    expect(values[0]).toBe('alpha')
    expect(values[1]).toBe('v1.0')
    expect(values[2]).toBe('v2.0')
    expect(values[3]).toBe('—')      // missing release renders as '—' in cell, sorted last
  })

  it('TC2: descending sort orders rows: v2.0, v1.0, alpha, (empty)', async () => {
    const fixtures = makeReleaseFixtures()
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: fixtures, total: fixtures.length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Release') // asc
    await clickSortHeader(wrapper, 'Release') // desc

    const values = getReleaseValues(wrapper)
    expect(values.length).toBe(4)
    // Reversed: 'v2.0' > 'v1.0' > 'alpha' > ''
    expect(values[0]).toBe('v2.0')
    expect(values[1]).toBe('v1.0')
    expect(values[2]).toBe('alpha')
    expect(values[3]).toBe('—')      // empty string is last in descending
  })

  it('TC3: artifact with no release appears last in both ascending and descending', async () => {
    const fixtures = makeReleaseFixtures()
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: fixtures, total: fixtures.length })
    await flushPromises()

    // Ascending: no-release last (pinned to end by useSortableTable)
    await clickSortHeader(wrapper, 'Release')
    let values = getReleaseValues(wrapper)
    expect(values[values.length - 1]).toBe('—')

    // Descending: no-release last
    await clickSortHeader(wrapper, 'Release')
    values = getReleaseValues(wrapper)
    expect(values[values.length - 1]).toBe('—')
  })

  it('TC4: sort is case-insensitive (Alpha and alpha sort adjacently)', async () => {
    // Add both 'Alpha' and 'alpha' to verify they sort together (not separated by case)
    const fixtures = [
      makeArtifact({
        path: 'lifecycle/ideas/alpha-lower.md',
        lineage: 'alpha-lower',
        frontmatter: { title: 'alpha-lower', type: 'idea', status: 'draft', lineage: 'alpha-lower', release: 'alpha' },
      }),
      makeArtifact({
        path: 'lifecycle/ideas/alpha-upper.md',
        lineage: 'alpha-upper',
        frontmatter: { title: 'alpha-upper', type: 'idea', status: 'draft', lineage: 'alpha-upper', release: 'Alpha' },
      }),
      makeArtifact({
        path: 'lifecycle/ideas/beta.md',
        lineage: 'beta',
        frontmatter: { title: 'beta', type: 'idea', status: 'draft', lineage: 'beta', release: 'beta' },
      }),
    ]

    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: fixtures, total: fixtures.length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Release') // ascending

    const values = getReleaseValues(wrapper)
    // 'alpha' and 'Alpha' are equivalent under case-insensitive sort so both
    // must appear before 'beta'. Neither 'beta' should appear between them.
    const alphaIdx  = values.indexOf('alpha')
    const AlphaIdx  = values.indexOf('Alpha')
    const betaIdx   = values.indexOf('beta')

    // Both must precede beta
    expect(alphaIdx).toBeLessThan(betaIdx)
    expect(AlphaIdx).toBeLessThan(betaIdx)
    // They must be adjacent (differ by at most 1 position)
    expect(Math.abs(alphaIdx - AlphaIdx)).toBeLessThanOrEqual(1)
  })
})
