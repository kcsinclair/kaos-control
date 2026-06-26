// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * E2E smoke test — Run History (F4, F5, F7)
 *
 * Smoke-level check that:
 *   - Triggering a pipeline run via the UI produces a history row after
 *     completion (no manual refresh required).
 *   - Expanding the row shows the inline log pane.
 *   - The pipeline card shows the latest-run summary badge.
 *   - The column header shows a group-level badge.
 *
 * Requires: built binary at dist/kaos-control (make build).
 * Run with: make test-e2e or pnpm playwright test flows/run-history.spec.ts
 */

import { test, expect } from '@playwright/test'
import { spawnKaosControl } from '../harness/kaos-control.js'
import { bootstrapUser, loginPage } from '../harness/auth.js'
import { writeFile, mkdir } from 'node:fs/promises'
import { join } from 'node:path'

const ADMIN = {
  email: 'admin@kaos-e2e.local',
  password: 'TestPassword123!',
  name: 'Test Admin',
}

const QUICK_ECHO_YAML = `name: Quick Echo
type: build
steps:
  - name: Say Hello
    command: echo hello-from-e2e
`

test.describe('Run History smoke', () => {
  test(
    'history row appears after pipeline run completes',
    async ({ page }) => {
      const instance = await spawnKaosControl()
      try {
        await bootstrapUser(instance.baseURL, ADMIN)

        // Seed a pipeline YAML file into the project root.
        const devopsDir = join(instance.projectRoot, 'lifecycle', 'devops')
        await mkdir(devopsDir, { recursive: true })
        await writeFile(join(devopsDir, 'quick-echo.yaml'), QUICK_ECHO_YAML)

        // Login and navigate to the DevOps view.
        await loginPage(page, instance.baseURL, ADMIN)
        await page.goto(`${instance.baseURL}/p/testproject/devops`)

        // Wait for the pipeline card to appear.
        await expect(page.locator('text=Quick Echo')).toBeVisible({ timeout: 10_000 })

        // Expand the run history panel (it starts collapsed).
        await page.locator('.history-toggle').first().click()

        // The panel is currently empty — "No runs yet".
        await expect(page.locator('.history-empty')).toBeVisible({ timeout: 5_000 })

        // Click Run on the pipeline card.
        await page.locator('.btn-run').first().click()

        // Wait for the run to complete — the Cancel button disappears and a
        // history row appears (the store prepends the row on run.completed).
        await expect(page.locator('.history-row').first()).toBeVisible({ timeout: 30_000 })

        // The newest history row should show a passed-status icon.
        const firstRow = page.locator('.history-row').first()
        await expect(firstRow.locator('.history-status--passed')).toBeVisible({
          timeout: 5_000,
        })
      } finally {
        await instance.kill()
      }
    },
  )

  test(
    'expanding a history row shows the inline log pane',
    async ({ page }) => {
      const instance = await spawnKaosControl()
      try {
        await bootstrapUser(instance.baseURL, ADMIN)

        const devopsDir = join(instance.projectRoot, 'lifecycle', 'devops')
        await mkdir(devopsDir, { recursive: true })
        await writeFile(join(devopsDir, 'quick-echo.yaml'), QUICK_ECHO_YAML)

        await loginPage(page, instance.baseURL, ADMIN)
        await page.goto(`${instance.baseURL}/p/testproject/devops`)
        await expect(page.locator('text=Quick Echo')).toBeVisible({ timeout: 10_000 })

        // Trigger the run.
        await page.locator('.btn-run').first().click()

        // Wait for the history row to appear.
        await page.locator('.history-toggle').first().click()
        await expect(page.locator('.history-row').first()).toBeVisible({ timeout: 30_000 })

        // Click the expand button on the first history row.
        await page.locator('.history-expand-btn').first().click()

        // The inline log pane must appear and contain at least one log row.
        await expect(page.locator('.history-log-pane')).toBeVisible({ timeout: 10_000 })
        await expect(page.locator('.log-row').first()).toBeVisible({ timeout: 5_000 })
      } finally {
        await instance.kill()
      }
    },
  )

  test(
    'pipeline card shows the latest-run summary badge after a run',
    async ({ page }) => {
      const instance = await spawnKaosControl()
      try {
        await bootstrapUser(instance.baseURL, ADMIN)

        const devopsDir = join(instance.projectRoot, 'lifecycle', 'devops')
        await mkdir(devopsDir, { recursive: true })
        await writeFile(join(devopsDir, 'quick-echo.yaml'), QUICK_ECHO_YAML)

        await loginPage(page, instance.baseURL, ADMIN)
        await page.goto(`${instance.baseURL}/p/testproject/devops`)
        await expect(page.locator('text=Quick Echo')).toBeVisible({ timeout: 10_000 })

        // Trigger run and wait for the latest-run badge to appear.
        await page.locator('.btn-run').first().click()
        await expect(page.locator('.latest-run-badge')).toBeVisible({ timeout: 30_000 })

        // Card badge should reflect passed status.
        await expect(page.locator('.latest-run-badge--passed')).toBeVisible({
          timeout: 5_000,
        })

        // Column header badge should also appear.
        await expect(page.locator('.column-header__badge')).toBeVisible({ timeout: 5_000 })
      } finally {
        await instance.kill()
      }
    },
  )
})
