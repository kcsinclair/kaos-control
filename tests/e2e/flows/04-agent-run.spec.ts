import { test, expect } from '../fixtures.js'

test.describe('Flow 04 — Agent run', () => {
  test('stub agent runs and reaches done status without Claude Code', async ({
    kctest,
    loggedInPage: page,
  }) => {
    // Subscribe to WS for agent.started event. Server requires auth on WS;
    // pass the browser-context session cookie via undici's headers extension.
    const cookies = await page.context().cookies()
    const cookieHeader = cookies.map((c) => `${c.name}=${c.value}`).join('; ')
    const wsURL = kctest.baseURL.replace(/^http/, 'ws') + '/api/p/testproject/ws'
    const wsOpts = { headers: { Cookie: cookieHeader } } as unknown as string[]
    const agentStartedPromise = new Promise<{ run_id: string }>((resolve, reject) => {
      const ws = new WebSocket(wsURL, wsOpts)
      const timer = setTimeout(
        () => reject(new Error('Timed out waiting for agent.started WS event')),
        8_000,
      )
      ws.addEventListener('message', (msg) => {
        let ev: { type: string; payload: { run_id?: string } }
        try {
          ev = JSON.parse(msg.data as string)
        } catch {
          return
        }
        if (ev.type === 'agent.started') {
          clearTimeout(timer)
          ws.close()
          resolve({ run_id: ev.payload.run_id ?? '' })
        }
      })
    })

    // Navigate to agents page
    await page.goto(`${kctest.baseURL}/p/testproject/agents`)
    await expect(page.locator('text=stub-agent')).toBeVisible({ timeout: 10_000 })

    // Click "Run Agent" to open the run dialog
    await page.click('button:has-text("Run Agent")')

    // Select the stub-agent chip
    await expect(page.locator('.agent-chip', { hasText: 'stub-agent' })).toBeVisible({
      timeout: 5_000,
    })
    await page.click('.agent-chip:has-text("stub-agent")')

    // Fill in the target path input (placeholder: "lifecycle/requirements/…")
    await page.fill('input.rad-input[placeholder*="lifecycle"]', 'lifecycle/requirements/smoke-req-01.md')

    // Click "Run" button
    await page.click('button.btn-primary:has-text("Run")')

    // Wait for agent.started WS event
    const { run_id: runId } = await agentStartedPromise
    expect(runId).toBeTruthy()

    // Assert the run row appears with status "running"
    // (May briefly show running then quickly switch to done since stub is fast)

    // Poll for done status via API (up to 10s)
    const deadline = Date.now() + 10_000
    let finalStatus = ''
    while (Date.now() < deadline) {
      const res = await fetch(
        `${kctest.baseURL}/api/p/testproject/agents/runs/${runId}`,
        {
          headers: {
            // Pass session cookie from browser context
            Cookie: (await page.context().cookies()).map((c) => `${c.name}=${c.value}`).join('; '),
          },
        },
      )
      if (res.ok) {
        // API returns { run: { ...row..., status } } — extract from nested run.
        const data = (await res.json()) as { run?: { status?: string } }
        finalStatus = data.run?.status ?? ''
        if (finalStatus === 'done' || finalStatus === 'failed') break
      }
      await new Promise((r) => setTimeout(r, 500))
    }
    expect(finalStatus).toBe('done')
  })
})
