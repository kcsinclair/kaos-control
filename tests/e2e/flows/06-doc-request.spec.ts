import { test, expect } from '../fixtures.js'

// Test plan: lifecycle/test-plans/tech-writer-agent-5-test.md §Milestone 6

const DONE_ARTIFACT_REL = 'lifecycle/requirements/smoke-req-done.md'
const NOT_DONE_ARTIFACT_REL = 'lifecycle/requirements/smoke-req-01.md' // draft

test.describe('Flow 06 — "Request docs" button (FR1)', () => {
  test('TC1: "Request docs" button is visible on a done artifact', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts/${DONE_ARTIFACT_REL}`)

    // Wait for the artifact view to load
    await expect(page.locator('.status-badge, [data-status]').first()).toBeVisible({
      timeout: 10_000,
    })

    // "Request docs" button must be visible for done artifacts
    await expect(page.locator('button:has-text("Request docs")')).toBeVisible({ timeout: 5_000 })
  })

  test('TC2: "Request docs" button is NOT visible on a non-done artifact', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts/${NOT_DONE_ARTIFACT_REL}`)

    // Wait for the artifact view to load
    await expect(page.locator('.status-badge, [data-status]').first()).toBeVisible({
      timeout: 10_000,
    })

    // "Request docs" button must NOT be visible for non-done artifacts
    await expect(page.locator('button:has-text("Request docs")')).toBeHidden()
  })

  test('TC3: "Request docs" flow creates a linked doc artifact', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts/${DONE_ARTIFACT_REL}`)

    // Wait for the artifact to load
    await expect(page.locator('button:has-text("Request docs")')).toBeVisible({ timeout: 10_000 })

    // Intercept the POST /artifacts request so we can inspect the created doc
    const createResponsePromise = page.waitForResponse(
      (resp) =>
        resp.url().includes('/artifacts') &&
        resp.request().method() === 'POST' &&
        !resp.url().includes('/generate'),
      { timeout: 15_000 },
    )

    // Click "Request docs" to open the doc creation modal
    await page.click('button:has-text("Request docs")')

    // The BrainDumpModal (or dedicated doc modal) should appear
    // Fill in a brief description for the documentation
    const briefInput = page.locator('textarea, input[type="text"]').filter({ hasText: '' }).first()
    await expect(briefInput).toBeVisible({ timeout: 5_000 })
    await briefInput.fill(
      'Document the smoke requirement installation and setup steps for new users',
    )

    // Submit the form
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Submit")')

    // Wait for the API call
    const createResponse = await createResponsePromise
    expect(createResponse.status()).toBe(201)

    const responseBody = (await createResponse.json()) as {
      path?: string
      artifact?: { frontmatter?: { lineage?: string; parent?: string; type?: string } }
    }

    // The created doc must be under lifecycle/docs/
    const path = responseBody?.path ?? ''
    expect(path).toMatch(/^lifecycle\/docs\//)

    // After creation the user should navigate to the new doc artifact view
    await page.waitForURL((url) => url.pathname.includes('lifecycle/docs'), { timeout: 10_000 })
    expect(page.url()).toContain('lifecycle/docs')
  })
})
