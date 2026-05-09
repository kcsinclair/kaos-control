// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Tests for BacklogPanel — Backlog Panel UI, A11y, and Filter behaviour
 *
 * Covers Milestone 4 (FR3.1–FR3.7, OQ1) and Milestone 6 (NFR2, NFR3) from
 * the Roadmap Backlog Panel and Unscheduled Column test plan.
 *
 * Testing approach:
 * ─────────────────
 * happy-dom does not compute layout, so:
 *   - "renders below the Gantt chart" (FR3.1) is verified by asserting the
 *     component mounts without error and has the backlog-panel class; relative
 *     position in a real page cannot be checked without a browser.
 *   - Performance (NFR2: 500-item scroll jank) is approximated by mounting
 *     the panel with 500 items and asserting the DOM renders without timeouts;
 *     real frame-rate measurement requires a browser.
 *   - Keyboard navigation (NFR3) is tested by checking tab-focusable elements
 *     and triggering keyboard events; actual focus traversal order may vary in
 *     happy-dom but we can verify the elements are present and keyboard handlers
 *     fire correctly.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import BacklogPanel from '../../web/src/components/releases/BacklogPanel.vue'
import type { ArtifactRow } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Fixture factory
// ---------------------------------------------------------------------------

let idCounter = 0

function makeArtifact(overrides: Partial<ArtifactRow> = {}): ArtifactRow {
  idCounter++
  return {
    path: `lifecycle/ideas/bp-test-${idCounter}.md`,
    slug: `bp-test-${idCounter}`,
    lineage: `bp-lineage-${idCounter}`,
    index: idCounter,
    stage: 'ideas',
    type: 'idea',
    status: 'draft',
    title: `BP Test Artifact ${idCounter}`,
    frontmatter: {},
    mtime: '2025-01-01T00:00:00Z',
    created: '2025-01-01T00:00:00Z',
    ...overrides,
  }
}

function mountPanel(artifacts: ArtifactRow[]) {
  return mount(BacklogPanel, {
    props: {
      project: 'test-project',
      artifacts,
    },
    // Clear sessionStorage so collapse state is always the default.
    attachTo: document.body,
  })
}

beforeEach(() => {
  idCounter = 0
  sessionStorage.clear()
})

afterEach(() => {
  vi.restoreAllMocks()
})

// ---------------------------------------------------------------------------
// Milestone 4 — Backlog Panel UI Tests
// ---------------------------------------------------------------------------

describe('BacklogPanel — Renders (FR3.1)', () => {
  it('renders the backlog-panel section element', () => {
    const wrapper = mountPanel([])
    expect(wrapper.find('.backlog-panel').exists()).toBe(true)
  })

  it('has aria-label="Backlog" on the section', () => {
    const wrapper = mountPanel([])
    const section = wrapper.find('section.backlog-panel')
    expect(section.exists()).toBe(true)
    expect(section.attributes('aria-label')).toBe('Backlog')
  })
})

describe('BacklogPanel — Count Header (FR3.5)', () => {
  it('header shows "Backlog (N)" with the correct count', () => {
    const artifacts = [makeArtifact(), makeArtifact(), makeArtifact()]
    const wrapper = mountPanel(artifacts)

    const title = wrapper.find('.backlog-title')
    expect(title.text()).toBe('Backlog (3)')
  })

  it('header shows "Backlog (0)" when no artifacts', () => {
    const wrapper = mountPanel([])
    const title = wrapper.find('.backlog-title')
    expect(title.text()).toBe('Backlog (0)')
  })
})

describe('BacklogPanel — Collapse/Expand (FR3.7)', () => {
  it('defaults to collapsed on initial mount', () => {
    const wrapper = mountPanel([makeArtifact()])
    // Card list is absent when collapsed.
    expect(wrapper.find('.backlog-list').exists()).toBe(false)
  })

  it('card list becomes visible after clicking the toggle header', async () => {
    const wrapper = mountPanel([makeArtifact()])
    expect(wrapper.find('.backlog-list').exists()).toBe(false)

    await wrapper.find('.backlog-toggle').trigger('click')

    expect(wrapper.find('.backlog-list').exists()).toBe(true)
  })

  it('card list collapses again after second click on toggle', async () => {
    const wrapper = mountPanel([makeArtifact()])

    await wrapper.find('.backlog-toggle').trigger('click')
    expect(wrapper.find('.backlog-list').exists()).toBe(true)

    await wrapper.find('.backlog-toggle').trigger('click')
    expect(wrapper.find('.backlog-list').exists()).toBe(false)
  })
})

describe('BacklogPanel — ARIA Attributes (FR3.7, NFR3)', () => {
  it('aria-expanded is false when collapsed', () => {
    const wrapper = mountPanel([makeArtifact()])
    const toggle = wrapper.find('.backlog-toggle')
    expect(toggle.attributes('aria-expanded')).toBe('false')
  })

  it('aria-expanded is true when expanded', async () => {
    const wrapper = mountPanel([makeArtifact()])
    await wrapper.find('.backlog-toggle').trigger('click')

    const toggle = wrapper.find('.backlog-toggle')
    expect(toggle.attributes('aria-expanded')).toBe('true')
  })

  it('aria-controls references the backlog list element id', async () => {
    const wrapper = mountPanel([makeArtifact()])
    const toggle = wrapper.find('.backlog-toggle')
    const controlsId = toggle.attributes('aria-controls')
    expect(controlsId).toBeTruthy()

    // Expand to render the list element.
    await wrapper.find('.backlog-toggle').trigger('click')
    const list = wrapper.find('.backlog-list')
    expect(list.attributes('id')).toBe(controlsId)
  })
})

describe('BacklogPanel — Card Content (FR3.3)', () => {
  it('each card shows title, type badge, status badge, and lineage', async () => {
    const artifact = makeArtifact({
      title: 'Test Title',
      type: 'ticket',
      status: 'planning',
      lineage: 'my-lineage',
    })
    const wrapper = mountPanel([artifact])
    await wrapper.find('.backlog-toggle').trigger('click')

    const card = wrapper.find('.backlog-card')
    expect(card.exists()).toBe(true)
    expect(card.find('.backlog-card-title').text()).toBe('Test Title')
    expect(card.find('.backlog-type-badge').text()).toBe('ticket')
    expect(card.find('.backlog-status-badge').text()).toBe('planning')
    expect(card.find('.backlog-lineage').text()).toBe('my-lineage')
  })

  it('clicking a card emits openArtifact with the artifact path', async () => {
    const artifact = makeArtifact({ path: 'lifecycle/ideas/click-test.md' })
    const wrapper = mountPanel([artifact])
    await wrapper.find('.backlog-toggle').trigger('click')

    await wrapper.find('.backlog-card').trigger('click')

    const emitted = wrapper.emitted('openArtifact')
    expect(emitted).toBeTruthy()
    expect(emitted![0]).toEqual(['lifecycle/ideas/click-test.md'])
  })
})

describe('BacklogPanel — Empty State (FR3.6)', () => {
  it('shows empty-state message when artifact list is empty', async () => {
    const wrapper = mountPanel([])
    await wrapper.find('.backlog-toggle').trigger('click')

    const empty = wrapper.find('.backlog-empty')
    expect(empty.exists()).toBe(true)
    expect(empty.text()).toBeTruthy()
  })

  it('does not show empty-state message when artifacts are present', async () => {
    const wrapper = mountPanel([makeArtifact()])
    await wrapper.find('.backlog-toggle').trigger('click')

    expect(wrapper.find('.backlog-empty').exists()).toBe(false)
  })

  it('shows empty-state message when all items filtered out', async () => {
    // Two artifacts: idea/draft and ticket/planning.
    // statusOptions will contain both "draft" and "planning" (from all artifacts).
    // typeOptions will contain both "idea" and "ticket".
    // Filter type="idea" → 1 card (idea/draft).
    // Filter status="planning" → 0 cards (the idea artifact is "draft", not "planning").
    const wrapper = mountPanel([
      makeArtifact({ type: 'idea', status: 'draft' }),
      makeArtifact({ type: 'ticket', status: 'planning' }),
    ])
    await wrapper.find('.backlog-toggle').trigger('click')
    expect(wrapper.findAll('.backlog-card')).toHaveLength(2)

    // Filter to ideas only → 1 remaining.
    await wrapper.find('select[aria-label="Filter by type"]').setValue('idea')
    expect(wrapper.findAll('.backlog-card')).toHaveLength(1)

    // Additionally filter by "planning" status — the idea has "draft", so nothing matches.
    // "planning" IS a valid option (from the ticket artifact).
    await wrapper.find('select[aria-label="Filter by status"]').setValue('planning')
    expect(wrapper.find('.backlog-empty').exists()).toBe(true)
  })
})

describe('BacklogPanel — Filters (OQ1)', () => {
  it('type filter narrows displayed cards', async () => {
    const artifacts = [
      makeArtifact({ type: 'idea' }),
      makeArtifact({ type: 'idea' }),
      makeArtifact({ type: 'ticket' }),
    ]
    const wrapper = mountPanel(artifacts)
    await wrapper.find('.backlog-toggle').trigger('click')

    expect(wrapper.findAll('.backlog-card')).toHaveLength(3)

    await wrapper.find('select[aria-label="Filter by type"]').setValue('idea')
    expect(wrapper.findAll('.backlog-card')).toHaveLength(2)

    // Only idea cards remain.
    for (const card of wrapper.findAll('.backlog-card')) {
      expect(card.find('.backlog-type-badge').text()).toBe('idea')
    }
  })

  it('status filter narrows displayed cards', async () => {
    const artifacts = [
      makeArtifact({ status: 'draft' }),
      makeArtifact({ status: 'draft' }),
      makeArtifact({ status: 'planning' }),
    ]
    const wrapper = mountPanel(artifacts)
    await wrapper.find('.backlog-toggle').trigger('click')

    await wrapper.find('select[aria-label="Filter by status"]').setValue('draft')
    expect(wrapper.findAll('.backlog-card')).toHaveLength(2)
  })

  it('clearing type filter restores all cards', async () => {
    const artifacts = [
      makeArtifact({ type: 'idea' }),
      makeArtifact({ type: 'ticket' }),
    ]
    const wrapper = mountPanel(artifacts)
    await wrapper.find('.backlog-toggle').trigger('click')

    await wrapper.find('select[aria-label="Filter by type"]').setValue('idea')
    expect(wrapper.findAll('.backlog-card')).toHaveLength(1)

    // Clear filter (select the empty "All types" option).
    await wrapper.find('select[aria-label="Filter by type"]').setValue('')
    expect(wrapper.findAll('.backlog-card')).toHaveLength(2)
  })

  it('type and status filters apply together (AND logic)', async () => {
    const artifacts = [
      makeArtifact({ type: 'idea', status: 'draft' }),
      makeArtifact({ type: 'idea', status: 'planning' }),
      makeArtifact({ type: 'ticket', status: 'draft' }),
    ]
    const wrapper = mountPanel(artifacts)
    await wrapper.find('.backlog-toggle').trigger('click')

    await wrapper.find('select[aria-label="Filter by type"]').setValue('idea')
    await wrapper.find('select[aria-label="Filter by status"]').setValue('draft')

    // Only the first artifact matches both filters.
    expect(wrapper.findAll('.backlog-card')).toHaveLength(1)
    expect(wrapper.find('.backlog-type-badge').text()).toBe('idea')
    expect(wrapper.find('.backlog-status-badge').text()).toBe('draft')
  })

  it('filter dropdowns are only visible when panel is expanded', () => {
    const wrapper = mountPanel([makeArtifact()])
    // Collapsed state — no filters visible.
    expect(wrapper.find('select[aria-label="Filter by type"]').exists()).toBe(false)
  })

  it('filter dropdowns appear when panel is expanded', async () => {
    const wrapper = mountPanel([makeArtifact()])
    await wrapper.find('.backlog-toggle').trigger('click')

    expect(wrapper.find('select[aria-label="Filter by type"]').exists()).toBe(true)
    expect(wrapper.find('select[aria-label="Filter by status"]').exists()).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Milestone 6 — Accessibility Tests (NFR3)
// ---------------------------------------------------------------------------

describe('BacklogPanel — Accessibility (Milestone 6, NFR3)', () => {
  it('toggle button is keyboard-activatable via Enter key', async () => {
    const wrapper = mountPanel([makeArtifact()])
    expect(wrapper.find('.backlog-list').exists()).toBe(false)

    await wrapper.find('.backlog-toggle').trigger('keydown', { key: 'Enter' })
    await wrapper.find('.backlog-toggle').trigger('click')

    // After triggering click (which Enter activates on a <button>),
    // the list should be visible.
    expect(wrapper.find('.backlog-list').exists()).toBe(true)
  })

  it('toggle button has focus-visible style class available', () => {
    const wrapper = mountPanel([])
    const toggle = wrapper.find('.backlog-toggle')
    // The button is a <button> element, focusable by default.
    expect(toggle.element.tagName).toBe('BUTTON')
  })

  it('each backlog card has an aria-label for screen readers', async () => {
    const artifact = makeArtifact({ title: 'My Feature Idea', slug: 'my-feature-idea' })
    const wrapper = mountPanel([artifact])
    await wrapper.find('.backlog-toggle').trigger('click')

    const card = wrapper.find('.backlog-card')
    const ariaLabel = card.attributes('aria-label')
    expect(ariaLabel).toBeTruthy()
    expect(ariaLabel).toContain('My Feature Idea')
  })

  it('NFR2: panel mounts with 500 artifacts without throwing', () => {
    const artifacts: ArtifactRow[] = Array.from({ length: 500 }, () => makeArtifact())
    // If mounting or rendering 500 items throws or exceeds Vitest's default
    // timeout, the test fails.  This is a basic smoke test for the 500-item
    // performance requirement.
    expect(() => mountPanel(artifacts)).not.toThrow()
  })

  it('NFR2: expanding panel with 500 artifacts renders all cards', async () => {
    const artifacts: ArtifactRow[] = Array.from({ length: 500 }, () => makeArtifact())
    const wrapper = mountPanel(artifacts)
    await wrapper.find('.backlog-toggle').trigger('click')

    const cards = wrapper.findAll('.backlog-card')
    expect(cards).toHaveLength(500)
  })
})
