import { test, expect } from '../fixtures.js'

// Test plan: lifecycle/test-plans/tech-writer-agent-5-test.md §Milestone 7

test.describe('Flow 07 — "New Docs" button (FR2)', () => {
  test('TC1: "New Docs" button is present on the Dashboard', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/dashboard`)

    // Wait for the dashboard to load
    await expect(page.locator('.btn-new-docs, button:has-text("New Docs")')).toBeVisible({
      timeout: 10_000,
    })
  })

  test('TC2: "New Docs" button is present on the Artifact List', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)

    // Wait for the artifact list to load
    await expect(page.locator('.btn-new-docs, button:has-text("New Docs")')).toBeVisible({
      timeout: 10_000,
    })
  })

  test('TC3: standalone doc creation flow produces an originating artifact', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/dashboard`)

    // Wait for the "New Docs" button and click it
    const newDocsButton = page.locator('.btn-new-docs, button:has-text("New Docs")').first()
    await expect(newDocsButton).toBeVisible({ timeout: 10_000 })

    // Intercept the POST /artifacts call
    const createResponsePromise = page.waitForResponse(
      (resp) =>
        resp.url().includes('/artifacts') &&
        resp.request().method() === 'POST' &&
        !resp.url().includes('/generate'),
      { timeout: 20_000 },
    )

    await newDocsButton.click()

    // The BrainDump/doc creation modal should open
    // Fill in a brief to describe the documentation
    const briefInput = page.locator('textarea, input[placeholder*="Describe"], input[placeholder*="describe"]').first()
    await expect(briefInput).toBeVisible({ timeout: 5_000 })
    await briefInput.fill(
      'Document the initial setup and configuration steps for new users of the application',
    )

    // Submit to trigger generation + creation
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Submit")')

    // Wait for the artifact to be created
    const createResponse = await createResponsePromise
    expect(createResponse.status()).toBe(201)

    const responseBody = (await createResponse.json()) as {
      path?: string
    }

    const path = responseBody?.path ?? ''

    // Must be an originating doc: under lifecycle/docs/ with no index suffix
    // (slug.md, not slug-N-doc.md)
    expect(path).toMatch(/^lifecycle\/docs\/[a-z0-9][a-z0-9-]*\.md$/)

    // The user should be navigated to the new doc artifact view
    await page.waitForURL((url) => url.pathname.includes('lifecycle/docs'), { timeout: 10_000 })
    expect(page.url()).toContain('lifecycle/docs')

    // Verify the artifact page shows the new doc in its originating status.
    // Quick-capture artifacts (brainDump.createDoc) start in `raw`, the
    // pre-draft status from the raw-artefact-status feature — not `draft`.
    await expect(page.locator('[data-status="raw"], .status-badge:has-text("raw")')).toBeVisible({
      timeout: 5_000,
    })
  })
})
