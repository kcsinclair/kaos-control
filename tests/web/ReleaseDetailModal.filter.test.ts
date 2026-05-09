// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Tests for ReleaseDetailModal — idea/defect filter behaviour
 *
 * Covers Milestones 1 and 2 from the Release Drill-Down Filter to Ideas and
 * Defects test plan:
 *
 *   Milestone 1 — Only idea and defect artifacts are rendered in the modal list
 *   Milestone 2 — The heading count reflects the filtered set, not the API total
 *
 * Component: web/src/components/releases/ReleaseDetailModal.vue
 * Props: releaseId (number), project (string)
 * Emits: close, edit, delete
 *
 * Mocking strategy:
 *   - releasesApi.getRelease            → resolves per-test via vi.mocked()
 *   - releasesApi.listReleaseArtifacts  → resolves per-test via vi.mocked()
 *   - vue-router (useRouter)            → stubbed so router.push never throws
 *
 * NOTE: vi.mock() is hoisted to the top of the compiled module by Vitest.
 * The factory must therefore be self-contained (no references to variables
 * declared below it).  Per-test return values are set with
 * vi.mocked(...).mockResolvedValue() inside each test.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ReleaseDetailModal from '../../web/src/components/releases/ReleaseDetailModal.vue'
import * as releasesApi from '@/api/releases'
import type { ArtifactRow } from '../../web/src/types/api'
import type { ReleaseDetail } from '../../web/src/types/release'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

// Use the same specifier the component uses so Vitest intercepts the same
// module instance (after @-alias resolution).
vi.mock('@/api/releases', () => ({
  getRelease: vi.fn(),
  listReleaseArtifacts: vi.fn(),
  listReleases: vi.fn().mockResolvedValue([]),
  createRelease: vi.fn(),
  updateRelease: vi.fn(),
  deleteRelease: vi.fn(),
  getRoadmapGraph: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

// ---------------------------------------------------------------------------
// Fixture helpers
// ---------------------------------------------------------------------------

const baseRelease: ReleaseDetail = {
  id: 1,
  name: 'v-test',
  status: 'planned',
  start_date: '2026-01-01',
  end_date: '2026-03-31',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  idea_count: 0,
  defect_count: 0,
}

function makeArtifact(type: string, index: number): ArtifactRow {
  return {
    path: `lifecycle/${type}s/art-${index}.md`,
    slug: `art-${index}`,
    lineage: `art-${index}`,
    index,
    stage: type,
    type,
    status: 'draft',
    title: `Artifact ${index} (${type})`,
    frontmatter: {},
    mtime: '2026-01-01T00:00:00Z',
    created: '2026-01-01T00:00:00Z',
  } as ArtifactRow
}

async function mountModal(
  releaseDetail: ReleaseDetail,
  artifacts: ArtifactRow[],
) {
  vi.mocked(releasesApi.getRelease).mockResolvedValue(releaseDetail)
  vi.mocked(releasesApi.listReleaseArtifacts).mockResolvedValue(artifacts)

  const wrapper = mount(ReleaseDetailModal, {
    props: {
      releaseId: releaseDetail.id,
      project: 'test-project',
    },
  })
  await flushPromises()
  return wrapper
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

// ---------------------------------------------------------------------------
// Milestone 1 — Filtered artifact list
// ---------------------------------------------------------------------------

describe('ReleaseDetailModal — artifact filter (Milestone 1)', () => {
  it('renders only idea and defect cards from a mixed set of 8 types', async () => {
    // One artifact of each type listed in the test plan.
    const allTypes = [
      'idea',
      'defect',
      'requirement',
      'plan-backend',
      'plan-frontend',
      'plan-test',
      'test',
      'prototype',
    ]
    const artifacts = allTypes.map((t, i) => makeArtifact(t, i + 1))

    const wrapper = await mountModal(baseRelease, artifacts)

    // Exactly 2 artifact cards should be rendered (idea + defect).
    const cards = wrapper.findAll('.artifact-card')
    expect(cards).toHaveLength(2)

    // The two visible cards must be the idea and the defect.
    const visibleTypes = cards.map((c) => c.find('.type-badge').text())
    expect(visibleTypes).toContain('idea')
    expect(visibleTypes).toContain('defect')

    // Every non-idea/non-defect type badge must be absent.
    const excluded = ['requirement', 'plan-backend', 'plan-frontend', 'plan-test', 'prototype']
    for (const t of excluded) {
      const match = cards.find((c) => c.find('.type-badge').text() === t)
      expect(match, `artifact type "${t}" should not be rendered`).toBeUndefined()
    }
  })

  it('shows empty state message when no ideas or defects exist in the artifact list', async () => {
    const artifacts = [
      makeArtifact('requirement', 1),
      makeArtifact('plan-backend', 2),
      makeArtifact('test', 3),
    ]

    const wrapper = await mountModal(baseRelease, artifacts)

    expect(wrapper.findAll('.artifact-card')).toHaveLength(0)
    // Component renders "No artifacts assigned." when filteredArtifacts is empty.
    expect(wrapper.text()).toContain('No artifacts assigned')
  })

  it('renders all 5 cards when every artifact is an idea or defect', async () => {
    const artifacts = [
      makeArtifact('idea', 1),
      makeArtifact('idea', 2),
      makeArtifact('idea', 3),
      makeArtifact('defect', 4),
      makeArtifact('defect', 5),
    ]

    const wrapper = await mountModal(baseRelease, artifacts)

    expect(wrapper.findAll('.artifact-card')).toHaveLength(5)
  })
})

// ---------------------------------------------------------------------------
// Milestone 2 — Filtered count in heading
// ---------------------------------------------------------------------------

describe('ReleaseDetailModal — artifact count heading (Milestone 2)', () => {
  it('heading shows filtered count (3) when 5 total artifacts contain 2 ideas + 1 defect + 2 plans', async () => {
    const artifacts = [
      makeArtifact('idea', 1),
      makeArtifact('idea', 2),
      makeArtifact('defect', 3),
      makeArtifact('plan-backend', 4),
      makeArtifact('plan-frontend', 5),
    ]

    const wrapper = await mountModal(baseRelease, artifacts)

    const heading = wrapper.find('.artifacts-heading')
    expect(heading.exists()).toBe(true)
    // Must reflect the filtered count (3 ideas+defects), not the API total (5).
    expect(heading.text()).toContain('(3)')
    expect(heading.text()).not.toContain('(5)')
  })

  it('heading shows (0) when only non-idea/non-defect artifacts are assigned', async () => {
    const artifacts = [
      makeArtifact('requirement', 1),
      makeArtifact('plan-test', 2),
      makeArtifact('prototype', 3),
    ]

    const wrapper = await mountModal(baseRelease, artifacts)

    const heading = wrapper.find('.artifacts-heading')
    expect(heading.exists()).toBe(true)
    expect(heading.text()).toContain('(0)')
  })
})
