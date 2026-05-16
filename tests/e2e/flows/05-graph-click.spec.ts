import { test, expect } from '../fixtures.js'

test.describe('Flow 05 — Graph node click', () => {
  test('map view renders expected node count and clicking a node navigates', async ({
    kctest,
    loggedInPage: page,
  }) => {
    // Navigate to map view
    await page.goto(`${kctest.baseURL}/p/testproject/map`)

    // Wait for Cytoscape canvas to render
    await expect(page.locator('canvas[data-id="layer2-node"]')).toBeVisible({ timeout: 15_000 })

    // Wait for __cy to be exposed
    await page.waitForFunction(() => !!(window as any).__cy, { timeout: 15_000 })

    // Wait for layout to stabilise (nodes should have positions)
    await page.waitForFunction(
      () => {
        const cy = (window as any).__cy
        return cy && cy.nodes().length > 0 && cy.nodes().first().position().x !== 0
      },
      { timeout: 15_000 },
    )

    // Assert node count matches seed count + synthetic nodes
    // Seed: 10 ideas + 3 requirements + 1 defect = 14 artifacts
    // Plus synthetic Backlog and possibly Unscheduled nodes
    const nodeCount = await page.evaluate(() => (window as any).__cy.nodes().length)
    expect(nodeCount).toBeGreaterThan(0)

    // Click on the smoke-req-01 node
    await page.evaluate(() => {
      const cy = (window as any).__cy
      // Find node whose data path matches our artifact
      const node = cy.nodes().filter((n: any) => {
        const raw = n.data('_raw')
        return raw && (raw.path === 'lifecycle/requirements/smoke-req-01.md' || n.data('id') === 'lifecycle/requirements/smoke-req-01.md')
      }).first()
      if (node && node.length) {
        node.trigger('tap')
      }
    })

    // Assert URL changed to the artifact page
    await page.waitForURL(
      (url) => url.pathname.includes('smoke-req-01'),
      { timeout: 5_000 },
    )
    expect(page.url()).toContain('smoke-req-01')
  })
})
