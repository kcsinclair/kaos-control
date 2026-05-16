import { test, expect, ADMIN_CREDS } from '../fixtures.js'
import type { Page } from '@playwright/test'

// Fixture paths for run count column tests
const RC_IDEA_A = 'lifecycle/ideas/rc-idea-a.md' // seeded with 2 runs
const RC_IDEA_B = 'lifecycle/ideas/rc-idea-b.md' // seeded with 1 run
const RC_IDEA_C = 'lifecycle/ideas/rc-idea-c.md' // seeded with 0 runs
const RC_PILL = 'lifecycle/ideas/rc-pill.md' // used for pill test
const RC_WS = 'lifecycle/ideas/rc-ws.md' // used for WS refresh test

type RunHeaders = Record<string, string>

async function getRunHeaders(page: Page, baseURL: string): Promise<RunHeaders> {
  const cookies = await page.context().cookies()
  const csrfToken = cookies.find((c) => c.name === 'kc_csrf')?.value ?? ''
  return {
    Cookie: cookies.map((c) => `${c.name}=${c.value}`).join('; '),
    'X-CSRF-Token': csrfToken,
    'Content-Type': 'application/json',
  }
}

async function triggerRun(baseURL: string, headers: RunHeaders, targetPath: string): Promise<string> {
  const res = await fetch(`${baseURL}/api/p/testproject/agents/stub-agent/run`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ target_path: targetPath }),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`triggerRun failed (${res.status}): ${text}`)
  }
  const data = (await res.json()) as { run_id?: string }
  if (!data.run_id) throw new Error('triggerRun: no run_id in response')
  return data.run_id
}

async function waitForRunStatus(
  baseURL: string,
  headers: RunHeaders,
  runId: string,
  targetStatuses: string[],
  timeoutMs = 15_000,
): Promise<string> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    const res = await fetch(`${baseURL}/api/p/testproject/agents/runs/${runId}`, { headers })
    if (res.ok) {
      const data = (await res.json()) as { run?: { status?: string } }
      const status = data.run?.status ?? ''
      if (targetStatuses.includes(status)) return status
    }
    await new Promise((r) => setTimeout(r, 200))
  }
  throw new Error(`run ${runId} did not reach ${targetStatuses.join('|')} within ${timeoutMs}ms`)
}

function wsURL(baseURL: string): string {
  return baseURL.replace(/^http/, 'ws') + '/api/p/testproject/ws'
}

function waitForWsEvent(
  url: string,
  eventType: string,
  cookieHeader: string,
  timeoutMs = 12_000,
): Promise<Record<string, unknown>> {
  return new Promise((resolve, reject) => {
    // Server closes unauthenticated WS with code 4401; pass session cookie
    // via undici's headers extension (not in the WHATWG WebSocket types).
    const wsOpts = cookieHeader
      ? ({ headers: { Cookie: cookieHeader } } as unknown as string[])
      : undefined
    const ws = new WebSocket(url, wsOpts)
    const timer = setTimeout(() => {
      ws.close()
      reject(new Error(`Timed out waiting for WS event ${eventType}`))
    }, timeoutMs)
    ws.addEventListener('message', (msg) => {
      let ev: { type: string; payload: Record<string, unknown> }
      try {
        ev = JSON.parse(msg.data as string)
      } catch {
        return
      }
      if (ev.type === eventType) {
        clearTimeout(timer)
        ws.close()
        resolve(ev.payload)
      }
    })
  })
}

// ─────────────────────────────────────────────────────────────────────────────
// Milestone 4 — Runs column rendering, counts, and sorting
// ─────────────────────────────────────────────────────────────────────────────

test.describe('Flow 10 — Artefact run count column', () => {
  test('TC1: Runs column is present, positioned correctly, shows correct counts including 0', async ({
    kctest,
    loggedInPage: page,
  }) => {
    const headers = await getRunHeaders(page, kctest.baseURL)

    // Seed 2 completed runs for rc-idea-a
    const runA1 = await triggerRun(kctest.baseURL, headers, RC_IDEA_A)
    await waitForRunStatus(kctest.baseURL, headers, runA1, ['done', 'failed'])
    const runA2 = await triggerRun(kctest.baseURL, headers, RC_IDEA_A)
    await waitForRunStatus(kctest.baseURL, headers, runA2, ['done', 'failed'])

    // Seed 1 completed run for rc-idea-b
    const runB1 = await triggerRun(kctest.baseURL, headers, RC_IDEA_B)
    await waitForRunStatus(kctest.baseURL, headers, runB1, ['done', 'failed'])

    // rc-idea-c has 0 runs

    await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)

    // Column header "Runs" is visible. Use getByRole to match the accessible
    // name — the th's textContent includes whitespace + the SortHeader icon,
    // so a strict `hasText: /^runs$/i` filter on the raw element misses it.
    const runsHeader = page.getByRole('columnheader', { name: 'Runs' })
    await expect(runsHeader).toBeVisible({ timeout: 10_000 })

    // "Runs" appears after "Type" and before "Created" in the header row
    const allHeaders = page.locator('table thead th')
    const headerTexts = await allHeaders.allTextContents()
    const normalised = headerTexts.map((h) => h.replace(/\s+/g, ' ').trim())
    const typeIdx = normalised.findIndex((h) => h.startsWith('Type'))
    const runsIdx = normalised.findIndex((h) => h.startsWith('Runs'))
    const createdIdx = normalised.findIndex((h) => h.startsWith('Created'))
    expect(typeIdx).toBeGreaterThanOrEqual(0)
    expect(runsIdx).toBeGreaterThan(typeIdx)
    expect(createdIdx).toBeGreaterThan(runsIdx)

    // rc-idea-a shows count 2
    const rowA = page.locator('tr').filter({ has: page.locator('.artifact-path', { hasText: 'rc-idea-a.md' }) })
    await expect(rowA.locator('.cell-runs')).toHaveText('2', { timeout: 10_000 })

    // rc-idea-b shows count 1
    const rowB = page.locator('tr').filter({ has: page.locator('.artifact-path', { hasText: 'rc-idea-b.md' }) })
    await expect(rowB.locator('.cell-runs')).toHaveText('1', { timeout: 5_000 })

    // rc-idea-c shows count 0 (not blank)
    const rowC = page.locator('tr').filter({ has: page.locator('.artifact-path', { hasText: 'rc-idea-c.md' }) })
    await expect(rowC.locator('.cell-runs')).toHaveText('0', { timeout: 5_000 })
  })

  test('TC2: Runs column is sortable (ascending then descending)', async ({
    kctest,
    loggedInPage: page,
  }) => {
    // Runs seeded in TC1 persist — rc-idea-a=2, rc-idea-b=1, rc-idea-c=0
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)

    // Wait for the table to load
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10_000 })

    const runsHeader = page.getByRole('columnheader', { name: 'Runs' })
    await expect(runsHeader).toBeVisible({ timeout: 5_000 })

    // Click once → ascending sort
    await runsHeader.click()
    await page.waitForTimeout(300) // allow Vue reactivity to settle

    let cells = page.locator('.cell-runs')
    let counts = (await cells.allTextContents()).map(Number)
    // Verify all counts are in non-decreasing order
    for (let i = 1; i < counts.length; i++) {
      expect(counts[i]).toBeGreaterThanOrEqual(counts[i - 1])
    }

    // Click again → descending sort
    await runsHeader.click()
    await page.waitForTimeout(300)

    cells = page.locator('.cell-runs')
    counts = (await cells.allTextContents()).map(Number)
    // Verify all counts are in non-increasing order
    for (let i = 1; i < counts.length; i++) {
      expect(counts[i]).toBeLessThanOrEqual(counts[i - 1])
    }
  })

  // ─────────────────────────────────────────────────────────────────────────
  // Milestone 5 — Active-agent status pill
  // ─────────────────────────────────────────────────────────────────────────

  test('TC3: "Agent Running" pill appears while run is active and disappears on completion', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10_000 })

    const headers = await getRunHeaders(page, kctest.baseURL)
    const cookieHeader = (await page.context().cookies())
      .map((c) => `${c.name}=${c.value}`)
      .join('; ')

    // Listen for agent.started before triggering so we don't miss it
    const agentStarted = waitForWsEvent(wsURL(kctest.baseURL), 'agent.started', cookieHeader)

    const runId = await triggerRun(kctest.baseURL, headers, RC_PILL)

    // Wait until the server has confirmed the run has started
    await agentStarted

    const pillRow = page
      .locator('tr')
      .filter({ has: page.locator('.artifact-path', { hasText: 'rc-pill.md' }) })

    // Pill with "Agent Running" text appears (data-status="running")
    const pill = pillRow.locator('.agent-status-pill[data-status="running"]')
    await expect(pill).toBeVisible({ timeout: 8_000 })
    await expect(pill).toHaveText('Agent Running')

    // Wait for the run to complete
    await waitForRunStatus(kctest.baseURL, headers, runId, ['done', 'failed'])

    // After run finishes the pill disappears (WS drives a re-fetch)
    await expect(pillRow.locator('.agent-status-pill')).not.toBeVisible({ timeout: 10_000 })
  })

  // ─────────────────────────────────────────────────────────────────────────
  // Milestone 6 — WebSocket-driven count refresh without page reload
  // ─────────────────────────────────────────────────────────────────────────

  test('TC4: run count increments without page reload on agent.finished WS event', async ({
    kctest,
    loggedInPage: page,
  }) => {
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10_000 })

    const headers = await getRunHeaders(page, kctest.baseURL)
    const cookieHeader = (await page.context().cookies())
      .map((c) => `${c.name}=${c.value}`)
      .join('; ')
    const wsRow = page
      .locator('tr')
      .filter({ has: page.locator('.artifact-path', { hasText: 'rc-ws.md' }) })

    // Note the current run count for rc-ws.md (0 at this point in the worker)
    const runsCell = wsRow.locator('.cell-runs')
    const initialCountText = await runsCell.textContent({ timeout: 5_000 })
    const initialCount = Number(initialCountText ?? '0')

    const beforeURL = page.url()

    // Listen for agent.finished so we know when to check the DOM
    const agentFinished = waitForWsEvent(wsURL(kctest.baseURL), 'agent.finished', cookieHeader, 15_000)

    await triggerRun(kctest.baseURL, headers, RC_WS)

    // Wait for the finished event — the Vue component re-fetches on this event
    await agentFinished

    // Count must have incremented by 1 without a page reload
    const expectedCount = String(initialCount + 1)
    await expect(runsCell).toHaveText(expectedCount, { timeout: 10_000 })

    // URL must not have changed (no full page navigation occurred)
    expect(page.url()).toBe(beforeURL)
  })
})
