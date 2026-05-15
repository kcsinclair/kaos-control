import { test, expect } from '../fixtures.js'
import { readFile } from 'node:fs/promises'
import { join } from 'node:path'
import { execSync } from 'node:child_process'

const ARTIFACT_REL = 'lifecycle/requirements/smoke-req-01.md'

test.describe('Flow 03 — Status transition', () => {
  test('transitions artifact status, writes frontmatter, commits git', async ({
    kctest,
    loggedInPage: page,
  }) => {
    // Navigate to the draft requirement (read mode)
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts/${ARTIFACT_REL}`)

    // Wait for artifact to load in read mode (frontmatter panel visible)
    await expect(page.locator('.status-badge, [data-status]').first()).toBeVisible({
      timeout: 10_000,
    })

    // Intercept the transition API response before clicking
    const transitionResponsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/transition') && resp.request().method() === 'POST',
      { timeout: 10_000 },
    )

    // Click the interactive status badge to open the dropdown
    await page.click('.status-badge--interactive, .status-badge[role="button"]')

    // Wait for the status dropdown menu to appear
    await expect(page.locator('[role="listbox"], .status-menu')).toBeVisible({ timeout: 5_000 })

    // Click the "clarifying" option
    await page.click('[role="option"]:has-text("clarifying"), .status-option:has-text("clarifying")')

    // Assert HTTP 200
    const transitionResponse = await transitionResponsePromise
    expect(transitionResponse.status()).toBe(200)

    // Verify frontmatter on disk
    const diskContent = await readFile(join(kctest.projectRoot, ARTIFACT_REL), 'utf8')
    expect(diskContent).toMatch(/status:\s*clarifying/)

    // Verify git commit message contains transition info
    const gitLog = execSync('git log --oneline -1', {
      cwd: kctest.projectRoot,
    })
      .toString()
      .trim()
    expect(gitLog).toMatch(/transition\(smoke-req-01\)/)
    expect(gitLog).toMatch(/clarifying/)
  })
})
