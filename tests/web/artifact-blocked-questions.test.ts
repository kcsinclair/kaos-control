// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Tests: Auto-block on open questions — frontend behaviour
 *
 * Covers Milestone 3 of the artefact-status-blocked-on-questions test plan:
 *
 *   Test 1 — toast shown when save response carries a different (blocked) status
 *   Test 2 — no extra toast when saved status matches submitted status
 *   Test 3 — blocked-questions banner shown when artifact is blocked with OQ body
 *   Test 4 — blocked-questions banner hidden for non-blocked artifacts
 *
 * NOTE (banner tests): The frontend plan (artefact-status-blocked-on-questions-3-fe.md)
 * calls for the banner to live in `web/src/views/project/ArtifactDetailView.vue`.
 * That component does not yet exist; the existing read-mode UI is part of
 * `ArtifactEditorView.vue`.  Tests 3 and 4 are therefore written against
 * `ArtifactEditorView` in its read (non-editing) mode.  If the frontend developer
 * creates a separate `ArtifactDetailView.vue`, these tests must be migrated to
 * target that component instead.
 *
 * All four tests are intentionally TDD: tests 1–2 will fail until the save()
 * function's status-comparison toast is added; tests 3–4 will fail until the
 * blocked-questions banner is rendered.
 *
 * Component under test: web/src/views/project/ArtifactEditorView.vue
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { ref as vueRef } from 'vue'
import ArtifactEditorView from '../../web/src/views/project/ArtifactEditorView.vue'
import { useUiStore } from '../../web/src/stores/ui'
import type { ArtifactDetail } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks — must be top-level (Vitest hoists vi.mock calls)
// ---------------------------------------------------------------------------

vi.mock('@/api/artifacts', () => ({
  getArtifact:      vi.fn(),
  updateArtifact:   vi.fn().mockResolvedValue({ artifact: {} }),
  listLabels:       vi.fn().mockResolvedValue({ labels: [] }),
  listPriorities:   vi.fn().mockResolvedValue({ priorities: [] }),
  listArtifacts:    vi.fn().mockResolvedValue({ items: [], total: 0 }),
}))

// ArtifactEditorView calls agentsStore.fetchAgents() on mount → listAgents().
// Mock the agents API so that fetch resolves instead of leaking a real request
// to test.local (an unhandled rejection, which Vitest 4 treats as fatal).
vi.mock('@/api/agents', () => ({
  listAgents:            vi.fn().mockResolvedValue([]),
  listRuns:              vi.fn().mockResolvedValue([]),
  listRunsByTargetPath:  vi.fn().mockResolvedValue([]),
  getReadyCounts:        vi.fn().mockResolvedValue({}),
  startRun:              vi.fn(),
  getRun:                vi.fn(),
  killRun:               vi.fn(),
  getRunResult:          vi.fn(),
  getRunLog:             vi.fn(),
}))

vi.mock('@/composables/useLock', () => ({
  useLock: vi.fn(() => ({
    acquired:     vueRef(false),
    conflictLock: vueRef(null),
    acquire:      vi.fn().mockResolvedValue(true),
    release:      vi.fn().mockResolvedValue(undefined),
  })),
}))

vi.mock('@/composables/useExternalChange', () => ({
  useExternalChange: vi.fn(() => ({
    hasExternalChange: vueRef(false),
    markSaved:         vi.fn(),
    acknowledge:       vi.fn(),
  })),
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

vi.mock('@/stores/graph', () => ({
  useGraphStore: vi.fn(() => ({
    rawEdges:   [],
    fetchGraph: vi.fn(),
  })),
}))

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const ARTIFACT_PATH = 'lifecycle/ideas/blocked-fe-test.md'

function makeDraftArtifact(overrides: Partial<ArtifactDetail> = {}): ArtifactDetail {
  return {
    path:      ARTIFACT_PATH,
    slug:      'blocked-fe-test',
    lineage:   'blocked-fe-test',
    index:     0,
    stage:     'ideas',
    type:      'idea',
    status:    'draft',
    title:     'Blocked FE Test',
    frontmatter: {
      title:   'Blocked FE Test',
      type:    'idea',
      status:  'draft',
      lineage: 'blocked-fe-test',
    },
    mtime:    '2026-04-01T00:00:00Z',
    created:  '2026-04-01T00:00:00Z',
    body:     'Regular body.',
    body_html: '<p>Regular body.</p>',
    file_sha:  'aaaa1111',
    ...overrides,
  }
}

function makeBlockedArtifact(body = '## Open Questions\n\n- Why is X?\n'): ArtifactDetail {
  return makeDraftArtifact({
    status: 'blocked',
    body,
    body_html: '<h2>Open Questions</h2><ul><li>Why is X?</li></ul>',
    frontmatter: {
      title:     'Blocked FE Test',
      type:      'idea',
      status:    'blocked',
      lineage:   'blocked-fe-test',
      assignees: [{ role: 'product-owner', who: 'agent' }],
    },
  })
}

/** Wraps a full ArtifactDetail in the shape getArtifact returns. */
function wrapForGet(detail: ArtifactDetail) {
  return {
    artifact: {
      path:        detail.path,
      slug:        detail.slug,
      lineage:     detail.lineage,
      index:       detail.index,
      stage:       detail.stage,
      type:        detail.type,
      status:      detail.status,
      title:       detail.title,
      frontmatter: detail.frontmatter,
      mtime:       detail.mtime,
      created:     detail.created,
    },
    body:      detail.body,
    body_html: detail.body_html,
    file_sha:  detail.file_sha,
  }
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
})

afterEach(() => {
  vi.clearAllMocks()
})

// ---------------------------------------------------------------------------
// Router factory
// ---------------------------------------------------------------------------

function makeRouter() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path:      '/p/:project/artifacts/:pathMatch(.*)*',
        component: ArtifactEditorView,
      },
    ],
  })
  return router
}

// ---------------------------------------------------------------------------
// Mount helper — loads the editor with a given initial artifact state
// ---------------------------------------------------------------------------

async function mountEditor(initial: ArtifactDetail, stubs = true) {
  const { getArtifact } = await import('@/api/artifacts')
  // First call (onMounted load) returns the initial artifact.
  vi.mocked(getArtifact).mockResolvedValueOnce(wrapForGet(initial))

  const pinia = createPinia()
  setActivePinia(pinia)

  const router = makeRouter()
  await router.push(`/p/testproject/artifacts/${ARTIFACT_PATH}`)
  await router.isReady()

  const wrapper = mount(ArtifactEditorView, {
    global: {
      plugins: [pinia, router],
      // Stub heavy child components to keep tests focused.
      stubs: stubs ? {
        FrontmatterPanel:  true,
        LineageBreadcrumb: true,
        TransitionDialog:  true,
        RunAgentDialog:    true,
        LockBanner:        true,
        MarkdownEditor:    true,
        FrontmatterEditor: true,
      } : {},
    },
  })

  // Let onMounted load() complete.
  await flushPromises()

  return { wrapper, pinia }
}

// ---------------------------------------------------------------------------
// Milestone 3 — Test 1
// Save returns a different (blocked) status → info toast appears
// ---------------------------------------------------------------------------

describe('ArtifactEditorView — save triggers info toast on status override', () => {
  it('shows an info toast mentioning "blocked" and "open questions" when save response has a different status', async () => {
    const draft = makeDraftArtifact()
    const { wrapper, pinia } = await mountEditor(draft)
    setActivePinia(pinia)

    const { getArtifact, updateArtifact } = await import('@/api/artifacts')
    // updateArtifact succeeds (the response is not used directly by save()).
    vi.mocked(updateArtifact).mockResolvedValueOnce({ artifact: {} as never })
    // Post-save store.fetchOne (after invalidate) returns the blocked artifact.
    vi.mocked(getArtifact).mockResolvedValueOnce(wrapForGet(makeBlockedArtifact()))

    // Enter edit mode.
    const editBtn = wrapper.find('button.btn-primary')
    if (!editBtn.exists()) {
      // The Edit button may have a different selector; find by text.
      const buttons = wrapper.findAll('button')
      const edit = buttons.find(b => b.text() === 'Edit')
      expect(edit, 'Edit button must be present in read mode').toBeDefined()
      await edit!.trigger('click')
    } else {
      await editBtn.trigger('click')
    }
    await flushPromises()

    // Trigger save.
    const saveBtn = wrapper.findAll('button').find(b => b.text() === 'Save' || b.text() === 'Saving…')
    expect(saveBtn, 'Save button must be present in edit mode').toBeDefined()
    await saveBtn!.trigger('click')
    await flushPromises()

    // Assert: at least one info toast references blocked state and open questions.
    const ui = useUiStore()
    const infoToasts = ui.toasts.filter(t => t.type === 'info')
    const hasBlockedToast = infoToasts.some(
      t => t.message.toLowerCase().includes('blocked') && t.message.toLowerCase().includes('open questions'),
    )
    expect(
      hasBlockedToast,
      `Expected an info toast mentioning "blocked" and "open questions". Toasts: ${JSON.stringify(ui.toasts)}`,
    ).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Milestone 3 — Test 2
// Save returns the same status as submitted → no extra info toast
// ---------------------------------------------------------------------------

describe('ArtifactEditorView — no extra toast when saved status matches', () => {
  it('does not show a blocked-override toast when the returned status matches the submitted status', async () => {
    const draft = makeDraftArtifact()
    const { wrapper, pinia } = await mountEditor(draft)
    setActivePinia(pinia)

    const { getArtifact, updateArtifact } = await import('@/api/artifacts')
    vi.mocked(updateArtifact).mockResolvedValueOnce({ artifact: {} as never })
    // Post-save: server returns the same draft status (no auto-block).
    vi.mocked(getArtifact).mockResolvedValueOnce(wrapForGet(draft))

    // Enter edit mode.
    const buttons = wrapper.findAll('button')
    const edit = buttons.find(b => b.text() === 'Edit')
    if (edit) await edit.trigger('click')
    await flushPromises()

    // Trigger save.
    const saveBtn = wrapper.findAll('button').find(b => b.text() === 'Save' || b.text() === 'Saving…')
    if (saveBtn) await saveBtn.trigger('click')
    await flushPromises()

    const ui = useUiStore()
    const blockedInfoToast = ui.toasts.find(
      t => t.type === 'info' &&
        t.message.toLowerCase().includes('blocked') &&
        t.message.toLowerCase().includes('open questions'),
    )
    expect(
      blockedInfoToast,
      `Expected NO blocked-override toast, but found: ${JSON.stringify(blockedInfoToast)}`,
    ).toBeUndefined()
  })
})

// ---------------------------------------------------------------------------
// Milestone 3 — Test 3
// Blocked banner shown in detail view when status=blocked and OQ present
//
// NOTE: The frontend plan specifies this banner should live in ArtifactDetailView.vue
// (not yet created). This test targets ArtifactEditorView's read mode instead.
// Update to import ArtifactDetailView when that component is created.
// ---------------------------------------------------------------------------

describe('ArtifactEditorView — blocked-questions banner visibility', () => {
  it('renders the blocked-questions banner when status is "blocked" and body has Open Questions', async () => {
    const blocked = makeBlockedArtifact('## Open Questions\n\n- Q1\n')
    const { wrapper } = await mountEditor(blocked)

    // Banner must be present and must not be in edit mode.
    const editing = wrapper.findAll('button').find(b => b.text() === 'Save' || b.text() === 'Saving…')
    expect(editing, 'Editor should be in read mode for this test').toBeUndefined()

    const banner = wrapper.find('.blocked-questions-banner')
    expect(
      banner.exists(),
      'Expected .blocked-questions-banner to be rendered when artifact is blocked with open questions',
    ).toBe(true)
  })

  // ---------------------------------------------------------------------------
  // Milestone 3 — Test 4
  // Blocked banner hidden for non-blocked artifacts
  // ---------------------------------------------------------------------------

  it('does NOT render the blocked-questions banner when status is not "blocked"', async () => {
    const draft = makeDraftArtifact({ body: '## Open Questions\n\n- Q1\n' })
    const { wrapper } = await mountEditor(draft)

    const banner = wrapper.find('.blocked-questions-banner')
    expect(
      banner.exists(),
      'Expected .blocked-questions-banner to be absent for a non-blocked artifact',
    ).toBe(false)
  })
})
