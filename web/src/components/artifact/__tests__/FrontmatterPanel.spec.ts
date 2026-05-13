// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi } from 'vitest'
import { shallowMount } from '@vue/test-utils'
import FrontmatterPanel from '../FrontmatterPanel.vue'
import ReleaseDropdown from '../ReleaseDropdown.vue'
import type { ArtifactDetail } from '@/types/api'

// Stub the API so no real HTTP calls happen when child component modules load.
vi.mock('@/api/artifacts', () => ({
  patchRelease: vi.fn(),
  patchPriority: vi.fn(),
  getAllowedTargets: vi.fn(),
  transitionArtifact: vi.fn(),
}))
vi.mock('@/api/releases', () => ({
  listReleases: vi.fn(),
}))
vi.mock('@/api/devops', () => ({
  listRuns: vi.fn(),
}))
vi.mock('@/stores/agents', () => ({
  useAgentsStore: () => ({
    artifactRuns: [],
    fetchRunsForArtifact: vi.fn(),
  }),
}))
vi.mock('@/stores/theme', () => ({
  useThemeStore: () => ({
    isDark: false,
  }),
}))

/** Build a minimal ArtifactDetail for use in tests. */
function makeArtifact(overrides?: Partial<ArtifactDetail['frontmatter']>): ArtifactDetail {
  return {
    path: 'lifecycle/ideas/test.md',
    slug: 'test',
    lineage: 'test',
    index: 0,
    stage: 'ideas',
    type: 'idea',
    status: 'draft',
    title: 'Test Artifact',
    frontmatter: {
      title: 'Test Artifact',
      type: 'idea',
      status: 'draft',
      lineage: 'test',
      ...overrides,
    },
    mtime: '2026-01-01T00:00:00Z',
    created: '2026-01-01T00:00:00Z',
    body: 'Body text.',
    body_html: '<p>Body text.</p>',
    file_sha: 'abc123',
  }
}

describe('FrontmatterPanel', () => {
  // ── TC1: Field order ───────────────────────────────────────────────────────

  it('renders Status, Priority, and Release as the first three dt elements', () => {
    const wrapper = shallowMount(FrontmatterPanel, {
      props: { artifact: makeArtifact() },
    })
    const dts = wrapper.findAll('dt')
    expect(dts.length).toBeGreaterThanOrEqual(3)
    expect(dts[0].text()).toBe('Status')
    expect(dts[1].text()).toBe('Priority')
    expect(dts[2].text()).toBe('Release')
  })

  // ── TC2: Release always visible with "None" when artifact has no release ───

  it('renders the Release row with "None" when artifact has no release field', () => {
    const wrapper = shallowMount(FrontmatterPanel, {
      // No project/targetPath → renders static span, no ReleaseDropdown
      props: { artifact: makeArtifact({ release: undefined }) },
    })
    const dts = wrapper.findAll('dt')
    const releaseLabel = dts.find((dt) => dt.text() === 'Release')
    expect(releaseLabel).toBeTruthy()

    // Find the parent row element and check its text content includes 'None'
    const releaseRow = releaseLabel!.element.closest('.fm-row')
    expect(releaseRow).toBeTruthy()
    expect(releaseRow!.textContent).toContain('None')
  })

  // ── TC3: ReleaseDropdown rendered when project and targetPath are provided ─

  it('renders a ReleaseDropdown component when project and targetPath are provided', () => {
    const wrapper = shallowMount(FrontmatterPanel, {
      props: {
        artifact: makeArtifact({ release: 'v1.0' }),
        project: 'testproject',
        targetPath: 'lifecycle/ideas/test.md',
      },
    })
    // shallowMount stubs child components; findComponent resolves the stub
    expect(wrapper.findComponent(ReleaseDropdown).exists()).toBe(true)
  })

  // ── TC4: Event propagation from ReleaseDropdown ────────────────────────────

  it('emits releaseChanged when the ReleaseDropdown emits changed', async () => {
    const wrapper = shallowMount(FrontmatterPanel, {
      props: {
        artifact: makeArtifact(),
        project: 'testproject',
        targetPath: 'lifecycle/ideas/test.md',
      },
    })
    const dropdown = wrapper.findComponent(ReleaseDropdown)
    expect(dropdown.exists()).toBe(true)

    // Simulate the dropdown emitting 'changed' with a release name
    await dropdown.vm.$emit('changed', 'v2.0')

    expect(wrapper.emitted('releaseChanged')).toBeTruthy()
    expect(wrapper.emitted('releaseChanged')![0]).toEqual(['v2.0'])
  })
})
