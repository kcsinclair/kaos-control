// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * E2E smoke test — Recursive Subdirectory Support: archive round-trip
 * (Milestone 7, idea-archiving-5-test.md)
 *
 * Seeds a flat artifact, moves it into lifecycle/ideas/archive/ on disk
 * (simulating an "archive" action), and confirms:
 *   - the list shows it at its new archive/ path with no duplicate row
 *   - opening it in the editor shows a folder breadcrumb segment
 * Then moves it back to the root and confirms parity (AC 1, 2, 6).
 *
 * These moves happen directly on disk (not via the API), so the frontend's
 * `artifact.indexed` WS-triggered auto-refresh never fires for them (that
 * event is only broadcast on API writes — see internal/http/write.go). Each
 * assertion therefore polls the API directly until the watcher has caught up
 * before navigating, rather than relying on a live in-page refresh.
 */

import { test, expect } from '../fixtures.js'
import type { Page } from '@playwright/test'
import type { KcTestInstance } from '../harness/kaos-control.js'
import { mkdir, rename, writeFile } from 'node:fs/promises'
import { join } from 'node:path'

const SLUG = 'archive-flow-idea'
const FLAT_REL = `lifecycle/ideas/${SLUG}.md`
const ARCHIVED_REL = `lifecycle/ideas/archive/${SLUG}.md`

const CONTENT = `---
title: "Archive Flow Idea"
type: idea
status: draft
lineage: ${SLUG}
---

# Archive Flow Idea

Seeded by the archive round-trip E2E smoke test.
`

// Polls GET /artifacts until exactly one row for the lineage sits at
// expectedPath (and no row remains at any other path for it), i.e. until the
// watcher has caught up with the on-disk move.
async function waitForIndexedAt(page: Page, kctest: KcTestInstance, expectedPath: string): Promise<void> {
  const cookies = await page.context().cookies()
  const cookieHeader = cookies.map((c) => `${c.name}=${c.value}`).join('; ')

  const deadline = Date.now() + 10_000
  while (Date.now() < deadline) {
    const res = await fetch(`${kctest.baseURL}/api/p/testproject/artifacts?limit=0`, {
      headers: { Cookie: cookieHeader },
    })
    const data = (await res.json()) as { items?: Array<{ path: string; lineage: string }> }
    const items = data.items ?? []
    const matchingLineage = items.filter((it) => it.lineage === SLUG)
    if (matchingLineage.length === 1 && matchingLineage[0].path === expectedPath) {
      return
    }
    await new Promise((r) => setTimeout(r, 200))
  }
  throw new Error(`timed out waiting for exactly one indexed row for lineage ${SLUG} at ${expectedPath}`)
}

test.describe('Flow 11 — Archive round-trip for nested artifacts', () => {
  test('moving an artifact into archive/ and back preserves listing, breadcrumb, and identity', async ({
    kctest,
    loggedInPage: page,
  }) => {
    // Seed the flat artifact directly on disk.
    await writeFile(join(kctest.projectRoot, FLAT_REL), CONTENT)
    await waitForIndexedAt(page, kctest, FLAT_REL)

    // It should appear in the list at its flat path.
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)
    await expect(page.locator('.artifact-path', { hasText: new RegExp(`^${SLUG}\\.md$`) })).toBeVisible({
      timeout: 10_000,
    })

    // Move it into archive/ on disk — this is the "archive" use case.
    await mkdir(join(kctest.projectRoot, 'lifecycle', 'ideas', 'archive'), { recursive: true })
    await rename(join(kctest.projectRoot, FLAT_REL), join(kctest.projectRoot, ARCHIVED_REL))
    await waitForIndexedAt(page, kctest, ARCHIVED_REL)

    // The list should show it at its new archive/ path, with no duplicate.
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)
    await expect(page.locator('.artifact-path', { hasText: `archive/${SLUG}.md` })).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.locator('.artifact-path', { hasText: new RegExp(`^${SLUG}\\.md$`) })).toHaveCount(0)
    await expect(page.locator('.artifact-row', { hasText: SLUG })).toHaveCount(1)

    // Opening it in the editor shows a folder breadcrumb segment for "archive".
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts/${ARCHIVED_REL}`)
    await expect(page.locator('.crumb-current', { hasText: `${SLUG}.md` })).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.locator('.crumb-intermediate', { hasText: 'archive' })).toBeVisible()

    // Move it back to the root and confirm parity.
    await rename(join(kctest.projectRoot, ARCHIVED_REL), join(kctest.projectRoot, FLAT_REL))
    await waitForIndexedAt(page, kctest, FLAT_REL)

    await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)
    await expect(page.locator('.artifact-path', { hasText: new RegExp(`^${SLUG}\\.md$`) })).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.locator('.artifact-path', { hasText: `archive/${SLUG}.md` })).toHaveCount(0)
    await expect(page.locator('.artifact-row', { hasText: SLUG })).toHaveCount(1)

    await page.goto(`${kctest.baseURL}/p/testproject/artifacts/${FLAT_REL}`)
    await expect(page.locator('.crumb-current', { hasText: `${SLUG}.md` })).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.locator('.crumb-intermediate', { hasText: 'archive' })).toHaveCount(0)
  })
})
