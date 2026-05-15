import { test, expect } from '../fixtures.js'

// Test plan: lifecycle/test-plans/tech-writer-agent-5-test.md §Milestone 8

// smoke-doc-approved.md is a doc fixture in `approved` status
const APPROVED_DOC_REL = 'lifecycle/docs/smoke-doc-approved.md'

test.describe('Flow 08 — Queue Work for doc artifacts (NFR3)', () => {
  test('TC1: Queue Work button is visible on an approved doc artifact', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts/${APPROVED_DOC_REL}`)

    // Wait for the artifact view to load
    await expect(page.locator('.status-badge, [data-status]').first()).toBeVisible({
      timeout: 10_000,
    })

    // Queue Work button is shown when status=approved
    await expect(page.locator('.btn-queue, button:has-text("Queue Work")')).toBeVisible({
      timeout: 5_000,
    })
  })

  test('TC2: clicking Queue Work targets the tech-writer agent', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts/${APPROVED_DOC_REL}`)

    // Wait for the Queue Work button
    const queueBtn = page.locator('.btn-queue, button:has-text("Queue Work")')
    await expect(queueBtn).toBeVisible({ timeout: 10_000 })

    // Intercept the POST /api/queue request to inspect which agent is targeted
    const enqueueResponsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/queue') && resp.request().method() === 'POST',
      { timeout: 10_000 },
    )

    await queueBtn.click()

    // Wait for enqueue response
    const enqueueResponse = await enqueueResponsePromise
    expect([200, 201]).toContain(enqueueResponse.status())

    // Inspect the request body that was sent to confirm the agent is tech-writer
    const requestBody = enqueueResponse.request().postDataJSON() as {
      agent?: string
      artifact_path?: string
      project?: string
    } | null

    expect(requestBody?.agent).toBe('tech-writer')
    expect(requestBody?.artifact_path).toBe(APPROVED_DOC_REL)
  })

  test('TC3: ready count endpoint includes approved doc for tech-writer agent', async ({
    kctest,
    loggedInPage: page,
  }) => {
    // Fetch the agents list and verify tech-writer reports ready_count >= 1
    await page.goto(`${kctest.baseURL}/p/testproject/dashboard`)
    await page.waitForURL(`${kctest.baseURL}/p/testproject/dashboard`)

    // Pull session cookies from the browser for direct API calls
    const cookies = await page.context().cookies()
    const cookieHeader = cookies.map((c) => `${c.name}=${c.value}`).join('; ')

    const res = await fetch(`${kctest.baseURL}/api/p/testproject/agents`, {
      headers: { Cookie: cookieHeader },
    })
    expect(res.status).toBe(200)

    const data = (await res.json()) as { agents?: { name: string; ready_count?: number }[] }
    const agents = data?.agents ?? []

    const techWriterAgent = agents.find((a) => a.name === 'tech-writer')
    expect(techWriterAgent).toBeTruthy()

    // The approved doc fixture means tech-writer must have at least 1 ready artifact
    expect(techWriterAgent?.ready_count ?? 0).toBeGreaterThanOrEqual(1)
  })
})
