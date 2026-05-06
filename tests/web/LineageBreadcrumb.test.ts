/**
 * Integration tests for LineageBreadcrumb.vue — Remove Non-Functional Hyperlinks
 *
 * Covers:
 *   Milestone 1 — Intermediate segments render as non-interactive <span> elements
 *   Milestone 2 — Root "artifacts" link remains a clickable <button>
 *   Milestone 3 — Final segment renders as current-page indicator (not clickable)
 *   Milestone 4 — Consistent rendering across all stage directories (parameterised)
 *
 * Testing notes:
 * ─────────────────────────────────────────────────────────────────────────────
 * The component splits `path` on "/" and renders:
 *   - A <button class="crumb-link"> for the root "artifacts" link (always present)
 *   - <span class="crumb-intermediate"> for every path segment except the last
 *   - <span class="crumb-current"> for the last segment
 *
 * No network I/O or Pinia stores are involved; only vue-router is needed.
 */

import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import LineageBreadcrumb from '../../web/src/components/artifact/LineageBreadcrumb.vue'

// ---------------------------------------------------------------------------
// Router factory
// ---------------------------------------------------------------------------

function makeRouter(path = '/p/test/artifacts') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project/:sub*', component: { template: '<div/>' } },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  router.push(path)
  return router
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

interface MountOpts {
  path?: string
  project?: string
  lineage?: string
}

async function mountBreadcrumb(opts: MountOpts = {}) {
  const router = makeRouter()
  await router.isReady()

  const wrapper = mount(LineageBreadcrumb, {
    props: {
      path: opts.path ?? 'lifecycle/requirements/login-2.md',
      project: opts.project ?? 'test',
      lineage: opts.lineage ?? 'login',
    },
    global: { plugins: [router] },
  })

  await flushPromises()
  return { wrapper, router }
}

// ---------------------------------------------------------------------------
// Cleanup
// ---------------------------------------------------------------------------

afterEach(() => {
  vi.clearAllMocks()
})

// ===========================================================================
// Milestone 1 — Intermediate Segments Are Non-Interactive
// ===========================================================================

describe('LineageBreadcrumb — Milestone 1: intermediate segments are non-interactive', () => {
  it('renders "lifecycle" segment as a <span>, not a <button>', async () => {
    const { wrapper } = await mountBreadcrumb({
      path: 'lifecycle/requirements/login-2.md',
    })
    const spans = wrapper.findAll('span.crumb-intermediate')
    const labels = spans.map(s => s.text())
    expect(labels).toContain('lifecycle')
  })

  it('renders "requirements" segment as a <span>, not a <button>', async () => {
    const { wrapper } = await mountBreadcrumb({
      path: 'lifecycle/requirements/login-2.md',
    })
    const spans = wrapper.findAll('span.crumb-intermediate')
    const labels = spans.map(s => s.text())
    expect(labels).toContain('requirements')
  })

  it('intermediate segments do not have class crumb-link', async () => {
    const { wrapper } = await mountBreadcrumb({
      path: 'lifecycle/requirements/login-2.md',
    })
    const spans = wrapper.findAll('span.crumb-intermediate')
    for (const span of spans) {
      expect(span.classes()).not.toContain('crumb-link')
    }
  })

  it('intermediate segments have class crumb-intermediate', async () => {
    const { wrapper } = await mountBreadcrumb({
      path: 'lifecycle/requirements/login-2.md',
    })
    const spans = wrapper.findAll('span.crumb-intermediate')
    expect(spans.length).toBe(2) // lifecycle, requirements
    for (const span of spans) {
      expect(span.classes()).toContain('crumb-intermediate')
    }
  })

  it('clicking an intermediate segment does not call router.push', async () => {
    const router = makeRouter()
    await router.isReady()

    const pushSpy = vi.spyOn(router, 'push')

    const wrapper = mount(LineageBreadcrumb, {
      props: {
        path: 'lifecycle/requirements/login-2.md',
        project: 'test',
        lineage: 'login',
      },
      global: { plugins: [router] },
    })

    await flushPromises()

    const spans = wrapper.findAll('span.crumb-intermediate')
    for (const span of spans) {
      await span.trigger('click')
    }

    expect(pushSpy).not.toHaveBeenCalled()
  })

  it('no <button> elements are present for intermediate segments', async () => {
    const { wrapper } = await mountBreadcrumb({
      path: 'lifecycle/requirements/login-2.md',
    })
    // Only the root "artifacts" button should exist
    const buttons = wrapper.findAll('button')
    expect(buttons.length).toBe(1)
    expect(buttons[0].text()).toBe('artifacts')
  })
})

// ===========================================================================
// Milestone 2 — Root Link Remains Clickable
// ===========================================================================

describe('LineageBreadcrumb — Milestone 2: root link remains clickable', () => {
  it('renders the root "artifacts" element as a <button>', async () => {
    const { wrapper } = await mountBreadcrumb()
    const rootBtn = wrapper.find('button.crumb-link')
    expect(rootBtn.exists()).toBe(true)
    expect(rootBtn.text()).toBe('artifacts')
  })

  it('root button has class crumb-link', async () => {
    const { wrapper } = await mountBreadcrumb()
    const rootBtn = wrapper.find('button')
    expect(rootBtn.classes()).toContain('crumb-link')
  })

  it('clicking the root button calls router.push with /p/{project}/artifacts', async () => {
    const router = makeRouter()
    await router.isReady()

    const pushSpy = vi.spyOn(router, 'push')

    const wrapper = mount(LineageBreadcrumb, {
      props: {
        path: 'lifecycle/requirements/login-2.md',
        project: 'test',
        lineage: 'login',
      },
      global: { plugins: [router] },
    })

    await flushPromises()
    await wrapper.find('button.crumb-link').trigger('click')

    expect(pushSpy).toHaveBeenCalledWith('/p/test/artifacts')
  })

  it('clicking root button uses the correct project slug from props', async () => {
    const router = makeRouter()
    await router.isReady()

    const pushSpy = vi.spyOn(router, 'push')

    const wrapper = mount(LineageBreadcrumb, {
      props: {
        path: 'lifecycle/ideas/foo.md',
        project: 'my-project',
        lineage: 'foo',
      },
      global: { plugins: [router] },
    })

    await flushPromises()
    await wrapper.find('button.crumb-link').trigger('click')

    expect(pushSpy).toHaveBeenCalledWith('/p/my-project/artifacts')
  })
})

// ===========================================================================
// Milestone 3 — Final Segment Is Current-Page Indicator
// ===========================================================================

describe('LineageBreadcrumb — Milestone 3: final segment is current-page indicator', () => {
  it('final segment renders as a <span> with class crumb-current', async () => {
    const { wrapper } = await mountBreadcrumb({
      path: 'lifecycle/requirements/login-2.md',
    })
    const current = wrapper.find('span.crumb-current')
    expect(current.exists()).toBe(true)
    expect(current.text()).toBe('login-2.md')
  })

  it('final segment is not a <button>', async () => {
    const { wrapper } = await mountBreadcrumb({
      path: 'lifecycle/requirements/login-2.md',
    })
    // Only the root artifacts button should be a button
    const buttons = wrapper.findAll('button')
    const buttonTexts = buttons.map(b => b.text())
    expect(buttonTexts).not.toContain('login-2.md')
  })

  it('exactly one crumb-current span exists', async () => {
    const { wrapper } = await mountBreadcrumb({
      path: 'lifecycle/requirements/login-2.md',
    })
    const currentSpans = wrapper.findAll('span.crumb-current')
    expect(currentSpans.length).toBe(1)
  })

  it('final segment does not have class crumb-link', async () => {
    const { wrapper } = await mountBreadcrumb({
      path: 'lifecycle/requirements/login-2.md',
    })
    const current = wrapper.find('span.crumb-current')
    expect(current.classes()).not.toContain('crumb-link')
  })

  it('clicking the final segment does not call router.push', async () => {
    const router = makeRouter()
    await router.isReady()

    const pushSpy = vi.spyOn(router, 'push')

    const wrapper = mount(LineageBreadcrumb, {
      props: {
        path: 'lifecycle/requirements/login-2.md',
        project: 'test',
        lineage: 'login',
      },
      global: { plugins: [router] },
    })

    await flushPromises()
    await wrapper.find('span.crumb-current').trigger('click')

    expect(pushSpy).not.toHaveBeenCalled()
  })
})

// ===========================================================================
// Milestone 4 — Consistent Rendering Across All Stage Directories
// ===========================================================================

const stageDirectories = [
  { dir: 'ideas',          file: 'my-idea.md',               lineage: 'my-idea' },
  { dir: 'requirements',   file: 'my-idea-2.md',             lineage: 'my-idea' },
  { dir: 'backend-plans',  file: 'my-idea-3-be.md',          lineage: 'my-idea' },
  { dir: 'frontend-plans', file: 'my-idea-4-fe.md',          lineage: 'my-idea' },
  { dir: 'dev-plans',      file: 'my-idea-5-dev.md',         lineage: 'my-idea' },
  { dir: 'test-plans',     file: 'my-idea-6-test.md',        lineage: 'my-idea' },
  { dir: 'tests',          file: 'my-idea-7-test.md',        lineage: 'my-idea' },
  { dir: 'prototypes',     file: 'my-idea-8-proto.md',       lineage: 'my-idea' },
  { dir: 'releases',       file: 'my-idea-9-release.md',     lineage: 'my-idea' },
  { dir: 'sprints',        file: 'sprint-1.md',              lineage: 'sprint-1' },
  { dir: 'defects',        file: 'my-idea-10-defect.md',     lineage: 'my-idea' },
]

describe('LineageBreadcrumb — Milestone 4: all stage directories', () => {
  for (const { dir, file, lineage } of stageDirectories) {
    const path = `lifecycle/${dir}/${file}`

    it(`[${dir}] intermediate segments are <span> elements`, async () => {
      const { wrapper } = await mountBreadcrumb({ path, lineage })

      // "lifecycle" and the stage dir should be crumb-intermediate spans
      const intermediates = wrapper.findAll('span.crumb-intermediate')
      expect(intermediates.length).toBeGreaterThanOrEqual(2)

      const labels = intermediates.map(s => s.text())
      expect(labels).toContain('lifecycle')
      expect(labels).toContain(dir)
    })

    it(`[${dir}] root is a <button> with class crumb-link`, async () => {
      const { wrapper } = await mountBreadcrumb({ path, lineage })

      const rootBtn = wrapper.find('button.crumb-link')
      expect(rootBtn.exists()).toBe(true)
      expect(rootBtn.text()).toBe('artifacts')
    })

    it(`[${dir}] final segment has class crumb-current`, async () => {
      const { wrapper } = await mountBreadcrumb({ path, lineage })

      const current = wrapper.find('span.crumb-current')
      expect(current.exists()).toBe(true)
      expect(current.text()).toBe(file)
    })

    it(`[${dir}] no errors emitted during render`, async () => {
      const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
      const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      await mountBreadcrumb({ path, lineage })

      expect(warnSpy).not.toHaveBeenCalled()
      expect(errorSpy).not.toHaveBeenCalled()

      warnSpy.mockRestore()
      errorSpy.mockRestore()
    })
  }
})
