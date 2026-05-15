import { test, expect } from '../fixtures.js'
import { readFile } from 'node:fs/promises'
import { join } from 'node:path'

const ARTIFACT_REL = 'lifecycle/requirements/smoke-req-01.md'
const SMOKE_MARKER = 'smoke-test-marker-' + Date.now()

test.describe('Flow 02 — Edit and save artifact', () => {
  test('saves content to disk and fires file.changed WS event', async ({
    kctest,
    loggedInPage: page,
  }) => {
    // Subscribe to WS before navigating so we catch the event
    const wsURL =
      kctest.baseURL.replace(/^http/, 'ws') + '/api/p/testproject/ws'
    const wsEvents: { type: string; payload: unknown }[] = []
    const ws = new WebSocket(wsURL)
    const fileChangedPromise = new Promise<void>((resolve, reject) => {
      const timer = setTimeout(
        () => reject(new Error('Timed out waiting for file.changed WS event')),
        8_000,
      )
      ws.addEventListener('message', (msg) => {
        let ev: { type: string; payload: unknown }
        try {
          ev = JSON.parse(msg.data as string)
        } catch {
          return
        }
        wsEvents.push(ev)
        if (ev.type === 'file.changed') {
          clearTimeout(timer)
          resolve()
        }
      })
    })

    // Navigate to artifact editor
    const encodedPath = ARTIFACT_REL.split('/').join('/')
    await page.goto(`${kctest.baseURL}/p/testproject/artifacts/${encodedPath}`)

    // Wait for CodeMirror to be ready
    const editor = page.locator('.cm-content').first()
    await expect(editor).toBeVisible({ timeout: 10_000 })

    // Append the smoke marker via keyboard
    await editor.click()
    await page.keyboard.press('Control+End')
    await page.keyboard.type('\n' + SMOKE_MARKER)

    // Click Save
    await page.click('button.btn-primary:has-text("Save")')

    // Wait for success toast
    await expect(page.locator('.toast-message', { hasText: 'Saved' })).toBeVisible({
      timeout: 5_000,
    })

    // Verify marker was written to disk
    const diskContent = await readFile(join(kctest.projectRoot, ARTIFACT_REL), 'utf8')
    expect(diskContent).toContain(SMOKE_MARKER)

    // Wait for file.changed WS event
    await fileChangedPromise
    ws.close()
  })
})
