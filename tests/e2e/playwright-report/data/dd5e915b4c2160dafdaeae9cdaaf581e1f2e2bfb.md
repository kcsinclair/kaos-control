# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 02-edit-save.spec.ts >> Flow 02 — Edit and save artifact >> saves content to disk and fires file.changed WS event
- Location: flows/02-edit-save.spec.ts:9:3

# Error details

```
Error: Timed out waiting for file.changed WS event
```

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.cm-content').first()
Expected: visible
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 10000ms
  - waiting for locator('.cm-content').first()

```

```yaml
- banner:
  - link "kaos-control":
    - /url: /projects
  - navigation:
    - link "Projects":
      - /url: /projects
  - 'link "Queue: 0 pending"':
    - /url: /queue
    - text: 0 pending
  - text: admin@kaos-e2e.local
  - button "Switch to dark mode":
    - img
  - button "Sign out"
- navigation "Project navigation":
  - text: Project testproject
  - list:
    - listitem:
      - link "Dashboard":
        - /url: /p/testproject/dashboard
    - listitem:
      - link "List":
        - /url: /p/testproject/artifacts
    - listitem:
      - link "Board":
        - /url: /p/testproject/artifacts/board
    - listitem:
      - link "Testing":
        - /url: /p/testproject/testing
    - listitem:
      - link "Map":
        - /url: /p/testproject/map
    - listitem:
      - link "Roadmap":
        - /url: /p/testproject/roadmap
    - listitem:
      - link "Agents":
        - /url: /p/testproject/agents
    - listitem:
      - link "Queue":
        - /url: /queue
    - listitem:
      - link "Scheduler":
        - /url: /p/testproject/scheduler
    - listitem:
      - link "Feed":
        - /url: /p/testproject/feed
    - listitem:
      - link "Parse Errors":
        - /url: /p/testproject/parse-errors
    - listitem:
      - link "Config":
        - /url: /p/testproject/config
    - listitem:
      - link "Ollama":
        - /url: /p/testproject/settings/ollama
  - text: kaos-control 0.1.2
  - button "Collapse sidebar" [expanded]
- main:
  - button "← artifacts"
  - text: "project not found: testproject"
```

# Test source

```ts
  1  | import { test, expect } from '../fixtures.js'
  2  | import { readFile } from 'node:fs/promises'
  3  | import { join } from 'node:path'
  4  | 
  5  | const ARTIFACT_REL = 'lifecycle/requirements/smoke-req-01.md'
  6  | const SMOKE_MARKER = 'smoke-test-marker-' + Date.now()
  7  | 
  8  | test.describe('Flow 02 — Edit and save artifact', () => {
  9  |   test('saves content to disk and fires file.changed WS event', async ({
  10 |     kctest,
  11 |     loggedInPage: page,
  12 |   }) => {
  13 |     // Subscribe to WS before navigating so we catch the event
  14 |     const wsURL =
  15 |       kctest.baseURL.replace(/^http/, 'ws') + '/api/p/testproject/ws'
  16 |     const wsEvents: { type: string; payload: unknown }[] = []
  17 |     const ws = new WebSocket(wsURL)
  18 |     const fileChangedPromise = new Promise<void>((resolve, reject) => {
  19 |       const timer = setTimeout(
  20 |         () => reject(new Error('Timed out waiting for file.changed WS event')),
  21 |         8_000,
  22 |       )
  23 |       ws.addEventListener('message', (msg) => {
  24 |         let ev: { type: string; payload: unknown }
  25 |         try {
  26 |           ev = JSON.parse(msg.data as string)
  27 |         } catch {
  28 |           return
  29 |         }
  30 |         wsEvents.push(ev)
  31 |         if (ev.type === 'file.changed') {
  32 |           clearTimeout(timer)
  33 |           resolve()
  34 |         }
  35 |       })
  36 |     })
  37 | 
  38 |     // Navigate to artifact editor
  39 |     const encodedPath = ARTIFACT_REL.split('/').join('/')
  40 |     await page.goto(`${kctest.baseURL}/p/testproject/artifacts/${encodedPath}`)
  41 | 
  42 |     // Wait for CodeMirror to be ready
  43 |     const editor = page.locator('.cm-content').first()
> 44 |     await expect(editor).toBeVisible({ timeout: 10_000 })
     |                          ^ Error: expect(locator).toBeVisible() failed
  45 | 
  46 |     // Append the smoke marker via keyboard
  47 |     await editor.click()
  48 |     await page.keyboard.press('Control+End')
  49 |     await page.keyboard.type('\n' + SMOKE_MARKER)
  50 | 
  51 |     // Click Save
  52 |     await page.click('button.btn-primary:has-text("Save")')
  53 | 
  54 |     // Wait for success toast
  55 |     await expect(page.locator('.toast-message', { hasText: 'Saved' })).toBeVisible({
  56 |       timeout: 5_000,
  57 |     })
  58 | 
  59 |     // Verify marker was written to disk
  60 |     const diskContent = await readFile(join(kctest.projectRoot, ARTIFACT_REL), 'utf8')
  61 |     expect(diskContent).toContain(SMOKE_MARKER)
  62 | 
  63 |     // Wait for file.changed WS event
  64 |     await fileChangedPromise
  65 |     ws.close()
  66 |   })
  67 | })
  68 | 
```