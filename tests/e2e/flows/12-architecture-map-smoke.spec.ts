import { test, expect } from '../fixtures.js'
import { cp, readdir } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

// Test plan: lifecycle/test-plans/architecture-relationship-map-5-test.md —
// Milestone 8 (NFR-2, NFR-3): a smoke test over the shipped ~9-architecture
// catalog confirming legible, prompt rendering at real scale.
//
// The catalog fixtures at ../fixtures/lifecycle/architecture/ are a copy of
// this repo's own lifecycle/architecture/{architectures,tech-stacks}/ (see
// lifecycle/tests/architecture-relationship-map-smoke-test.md for how to
// refresh them). They are copied into the running instance's project root
// AFTER boot rather than shipped as a pre-boot fixture: lifecycle/architecture/
// is not a configured stage, so the startup scan silently skips anything
// placed there before boot (see lifecycle/tests/architectural-artefacts-6-test.md,
// gap #1) — only the live fsnotify watcher covers it, confirmed working here.

const catalogFixtureDir = join(
  dirname(fileURLToPath(import.meta.url)),
  '..',
  'fixtures',
  'lifecycle',
  'architecture',
)

test.describe('Flow 12 — Architecture map full-catalog smoke', () => {
  test('renders one node per catalog architecture and stays interactive across 2D/3D and stack-reveal', async ({
    kctest,
    loggedInPage: page,
  }) => {
    const archFiles = await readdir(join(catalogFixtureDir, 'architectures'))
    const archCount = archFiles.filter((f) => f.endsWith('.md')).length

    await cp(catalogFixtureDir, join(kctest.projectRoot, 'lifecycle', 'architecture'), { recursive: true })

    const pageErrors: string[] = []
    page.on('pageerror', (err) => pageErrors.push(err.message))

    await page.goto(`${kctest.baseURL}/p/testproject/architecture/map`)

    // Base map (2D default): wait for the live watcher to index the catalog
    // and Cytoscape to render exactly one node per shipped architecture —
    // legible at real scale (NFR-2), no non-architecture nodes leaking in.
    await expect(page.locator('canvas[data-id="layer2-node"]')).toBeVisible({ timeout: 20_000 })
    await page.waitForFunction(() => !!(window as any).__cy, { timeout: 20_000 })
    await page.waitForFunction(
      (want) => (window as any).__cy?.nodes().length === want,
      archCount,
      { timeout: 20_000 },
    )

    // Toggling to 3D and back is interactive at this scale (NFR-3): the
    // force-graph mounts its own canvas without erroring, and 2D recovers.
    await page.click('.toggle-btn:has-text("3D")')
    await expect(page.locator('.force-graph-container canvas')).toBeVisible({ timeout: 20_000 })
    await page.click('.toggle-btn:has-text("2D")')
    await expect(page.locator('canvas[data-id="layer2-node"]')).toBeVisible({ timeout: 20_000 })

    // Revealing the stack ring for one architecture is interactive too
    // (NFR-3): node count grows once the tech-stack ring loads.
    await page.click('.stack-toggle-label input[type="checkbox"]')
    await page.selectOption('.stack-select', { label: 'Local Web-based Application' })
    await page.waitForFunction(
      (base) => ((window as any).__cy?.nodes().length ?? 0) > base,
      archCount,
      { timeout: 20_000 },
    )

    expect(pageErrors, `unexpected page errors: ${pageErrors.join('; ')}`).toEqual([])
  })
})
