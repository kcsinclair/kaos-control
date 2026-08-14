import { test, expect } from '../fixtures.js'

// Test plan: lifecycle/test-plans/defect-generate-missing-template-4-test.md
// §Milestone 6 — Playwright e2e: New Defect → Generate happy path.
//
// The e2e fixture project (tests/e2e/fixtures/lifecycle/config.yaml) has no
// idea-capture agent configured at all, which is exactly the fallback path
// exercised by the defect-generate-missing-template fix: this flow must
// never surface `idea-capture agent has no template "defect-generate"`.

const RAW_TEMPLATE_ERROR = 'idea-capture agent has no template "defect-generate"'

test.describe('Flow 13 — "New Defect → Generate" happy path', () => {
  test('TC1: generating a defect shows a structured preview with no template error', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)

    const newDefectButton = page.locator('.btn-new-defect, button:has-text("New Defect")').first()
    await expect(newDefectButton).toBeVisible({ timeout: 10_000 })
    await newDefectButton.click()

    const brief = page.locator('textarea[aria-label="Brain dump input"]')
    await expect(brief).toBeVisible({ timeout: 5_000 })
    await brief.fill(
      'When I click save on the artifact editor the page refreshes and all unsaved changes are lost. Expected: changes save without a page refresh.',
    )

    const generateResponsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/ideas/generate') && resp.request().method() === 'POST',
      { timeout: 30_000 },
    )

    await page.click('button:has-text("Generate")')

    const generateResponse = await generateResponsePromise
    expect(generateResponse.status()).toBe(200)

    // No error banner, and specifically not the raw internal resolver string.
    await expect(page.locator('.bdm-error')).toBeHidden()
    await expect(page.locator('.bdm-panel')).not.toContainText(RAW_TEMPLATE_ERROR)

    // Preview phase: structured defect content is rendered.
    await expect(page.locator('.bdm-meta-title')).toBeVisible({ timeout: 10_000 })
    const preview = page.locator('.bdm-md-preview')
    await expect(preview).toContainText(/Reproduction Steps/i)
    await expect(preview).toContainText(/Expected Behaviour|Expected Behavior/i)
    await expect(preview).toContainText(/Actual Behaviour|Actual Behavior/i)

    // Accept the proposal and confirm a defect artifact is created on disk.
    const createResponsePromise = page.waitForResponse(
      (resp) =>
        resp.url().includes('/artifacts') &&
        resp.request().method() === 'POST' &&
        !resp.url().includes('/generate'),
      { timeout: 15_000 },
    )
    await page.click('button:has-text("Accept")')

    const createResponse = await createResponsePromise
    expect(createResponse.status()).toBe(201)
    const { path } = (await createResponse.json()) as { path?: string }
    expect(path ?? '').toMatch(/^lifecycle\/defects\/[a-z0-9][a-z0-9-]*\.md$/)

    await page.waitForURL((url) => url.pathname.includes('lifecycle/defects'), { timeout: 10_000 })
    expect(page.url()).toContain('lifecycle/defects')
  })
})
