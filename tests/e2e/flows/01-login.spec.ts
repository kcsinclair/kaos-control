import { test, expect } from '../fixtures.js'

test.describe('Flow 01 — Login and project access', () => {
  test('redirects unauthenticated user to /login', async ({ kctest, page }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/dashboard`)
    await expect(page).toHaveURL(/\/login/)
  })

  test('loggedInPage lands on project dashboard with non-zero Lifecycle Total', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/dashboard`)
    await page.waitForURL(`${kctest.baseURL}/p/testproject/dashboard`)

    // Wait for the SummaryCountsWidget to render with data
    const lifecycleTotal = page.locator('.summary-card', { hasText: 'Lifecycle Total' })
    await expect(lifecycleTotal).toBeVisible({ timeout: 10_000 })

    // The value should be non-zero (14 fixture items total)
    const valueLocator = lifecycleTotal.locator('.summary-card-value').first()
    await expect(valueLocator).not.toHaveText('0', { timeout: 10_000 })
  })
})
