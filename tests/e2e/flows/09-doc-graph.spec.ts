import { test, expect } from '../fixtures.js'

// Test plan: lifecycle/test-plans/tech-writer-agent-5-test.md §Milestone 9

// smoke-doc-linked.md has lineage=smoke-req-01 and parent=lifecycle/requirements/smoke-req-01.md
const DOC_REL = 'lifecycle/docs/smoke-doc-linked.md'
const PARENT_REL = 'lifecycle/requirements/smoke-req-01.md'

// The expected colour for `doc` nodes in the dark palette (graphConstants.ts).
// Verified against the graphConstants nodeColors definition.
const DOC_NODE_COLOR_DARK = '#2dd4bf'   // teal-400
const IDEA_NODE_COLOR_DARK = '#f59e0b'  // amber-400
const REQ_NODE_COLOR_DARK = '#3b82f6'   // blue-500

test.describe('Flow 09 — Graph rendering for doc nodes (NFR1)', () => {
  test('TC1: doc node exists in the 2D map view', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/map`)

    // Wait for Cytoscape canvas and __cy to be ready
    await expect(page.locator('canvas[data-id="layer2-node"]')).toBeVisible({ timeout: 15_000 })
    await page.waitForFunction(() => !!(window as any).__cy, { timeout: 15_000 })

    // Wait for layout to stabilise with at least one positioned node
    await page.waitForFunction(
      () => {
        const cy = (window as any).__cy
        return cy && cy.nodes().length > 0 && cy.nodes().first().position().x !== 0
      },
      { timeout: 15_000 },
    )

    // Assert the doc node exists (matched by path in its data)
    const docNodeExists = await page.evaluate((docPath: string) => {
      const cy = (window as any).__cy
      const node = cy.nodes().filter((n: any) => {
        const raw = n.data('_raw')
        return (
          (raw && raw.path === docPath) ||
          n.data('id') === docPath
        )
      })
      return node.length > 0
    }, DOC_REL)

    expect(docNodeExists).toBe(true)
  })

  test('TC2: an edge connects the doc node to its parent artifact', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/map`)

    await expect(page.locator('canvas[data-id="layer2-node"]')).toBeVisible({ timeout: 15_000 })
    await page.waitForFunction(() => !!(window as any).__cy, { timeout: 15_000 })
    await page.waitForFunction(
      () => {
        const cy = (window as any).__cy
        return cy && cy.nodes().length > 0 && cy.nodes().first().position().x !== 0
      },
      { timeout: 15_000 },
    )

    // Assert an edge exists between the doc node and its parent requirement node
    const edgeExists = await page.evaluate(
      ([docPath, parentPath]: [string, string]) => {
        const cy = (window as any).__cy
        const docNode = cy.nodes().filter((n: any) => {
          const raw = n.data('_raw')
          return (raw && raw.path === docPath) || n.data('id') === docPath
        })
        const parentNode = cy.nodes().filter((n: any) => {
          const raw = n.data('_raw')
          return (raw && raw.path === parentPath) || n.data('id') === parentPath
        })
        if (!docNode.length || !parentNode.length) return false
        // Check for an edge connecting doc → parent (either direction)
        const edges = cy.edges()
        return edges.some((e: any) => {
          const src = e.data('source')
          const tgt = e.data('target')
          return (
            (src === docNode.first().id() && tgt === parentNode.first().id()) ||
            (src === parentNode.first().id() && tgt === docNode.first().id())
          )
        })
      },
      [DOC_REL, PARENT_REL] as [string, string],
    )

    expect(edgeExists).toBe(true)
  })

  test('TC3: doc node uses a distinct colour from idea and requirement nodes', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/map`)

    await expect(page.locator('canvas[data-id="layer2-node"]')).toBeVisible({ timeout: 15_000 })
    await page.waitForFunction(() => !!(window as any).__cy, { timeout: 15_000 })
    await page.waitForFunction(
      () => {
        const cy = (window as any).__cy
        return cy && cy.nodes().length > 0 && cy.nodes().first().position().x !== 0
      },
      { timeout: 15_000 },
    )

    // Retrieve the fill colour of the doc node via Cytoscape style
    const docNodeColor: string = await page.evaluate((docPath: string) => {
      const cy = (window as any).__cy
      const node = cy.nodes().filter((n: any) => {
        const raw = n.data('_raw')
        return (raw && raw.path === docPath) || n.data('id') === docPath
      }).first()
      return node ? node.style('background-color') : ''
    }, DOC_REL)

    // Retrieve colours of an idea node and a requirement node for comparison
    const ideaNodeColor: string = await page.evaluate(() => {
      const cy = (window as any).__cy
      const node = cy.nodes().filter((n: any) => {
        const raw = n.data('_raw')
        return raw && raw.type === 'idea'
      }).first()
      return node ? node.style('background-color') : ''
    })

    const reqNodeColor: string = await page.evaluate(() => {
      const cy = (window as any).__cy
      const node = cy.nodes().filter((n: any) => {
        const raw = n.data('_raw')
        return raw && raw.type === 'requirement'
      }).first()
      return node ? node.style('background-color') : ''
    })

    // Doc node must have a colour
    expect(docNodeColor).toBeTruthy()

    // The doc colour must differ from idea and requirement node colours
    expect(docNodeColor).not.toBe(ideaNodeColor)
    expect(docNodeColor).not.toBe(reqNodeColor)
  })
})
