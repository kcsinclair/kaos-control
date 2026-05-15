# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 08-doc-queue.spec.ts >> Flow 08 — Queue Work for doc artifacts (NFR3) >> TC3: ready count endpoint includes approved doc for tech-writer agent
- Location: flows/08-doc-queue.spec.ts:59:3

# Error details

```
Error: expect(received).toBe(expected) // Object.is equality

Expected: 200
Received: 404
```

# Page snapshot

```yaml
- generic [ref=e3]:
  - banner [ref=e4]:
    - link "kaos-control" [ref=e6] [cursor=pointer]:
      - /url: /projects
    - navigation [ref=e7]:
      - link "Projects" [ref=e8] [cursor=pointer]:
        - /url: /projects
    - generic [ref=e9]:
      - 'link "Queue: 0 pending" [ref=e10] [cursor=pointer]':
        - /url: /queue
        - generic [ref=e11]: "0"
        - generic [ref=e12]: pending
      - generic [ref=e13]: admin@kaos-e2e.local
      - button "Switch to dark mode" [ref=e14] [cursor=pointer]:
        - img [ref=e15]
      - button "Sign out" [ref=e17] [cursor=pointer]
  - generic [ref=e18]:
    - navigation "Project navigation" [ref=e19]:
      - generic [ref=e20]:
        - generic [ref=e21]: Project
        - generic [ref=e22]: testproject
      - list [ref=e23]:
        - listitem [ref=e24]:
          - link "Dashboard" [ref=e26] [cursor=pointer]:
            - /url: /p/testproject/dashboard
            - img [ref=e28]
            - generic [ref=e33]: Dashboard
        - listitem [ref=e34]:
          - link "List" [ref=e36] [cursor=pointer]:
            - /url: /p/testproject/artifacts
            - img [ref=e38]
            - generic [ref=e39]: List
        - listitem [ref=e40]:
          - link "Board" [ref=e42] [cursor=pointer]:
            - /url: /p/testproject/artifacts/board
            - img [ref=e44]
            - generic [ref=e46]: Board
        - listitem [ref=e47]:
          - link "Testing" [ref=e49] [cursor=pointer]:
            - /url: /p/testproject/testing
            - img [ref=e51]
            - generic [ref=e53]: Testing
        - listitem [ref=e54]:
          - link "Map" [ref=e56] [cursor=pointer]:
            - /url: /p/testproject/map
            - img [ref=e58]
            - generic [ref=e63]: Map
        - listitem [ref=e64]:
          - link "Roadmap" [ref=e66] [cursor=pointer]:
            - /url: /p/testproject/roadmap
            - img [ref=e68]
            - generic [ref=e70]: Roadmap
        - listitem [ref=e71]:
          - link "Agents" [ref=e73] [cursor=pointer]:
            - /url: /p/testproject/agents
            - img [ref=e75]
            - generic [ref=e78]: Agents
        - listitem [ref=e79]:
          - link "Queue" [ref=e81] [cursor=pointer]:
            - /url: /queue
            - img [ref=e83]
            - generic [ref=e86]: Queue
        - listitem [ref=e87]:
          - link "Scheduler" [ref=e89] [cursor=pointer]:
            - /url: /p/testproject/scheduler
            - img [ref=e91]
            - generic [ref=e95]: Scheduler
        - listitem [ref=e96]:
          - link "Feed" [ref=e98] [cursor=pointer]:
            - /url: /p/testproject/feed
            - img [ref=e100]
            - generic [ref=e102]: Feed
        - listitem [ref=e103]:
          - link "Parse Errors" [ref=e105] [cursor=pointer]:
            - /url: /p/testproject/parse-errors
            - img [ref=e107]
            - generic [ref=e109]: Parse Errors
        - listitem [ref=e110]:
          - link "Config" [ref=e112] [cursor=pointer]:
            - /url: /p/testproject/config
            - img [ref=e114]
            - generic [ref=e117]: Config
        - listitem [ref=e118]:
          - link "Ollama" [ref=e120] [cursor=pointer]:
            - /url: /p/testproject/settings/ollama
            - img [ref=e122]
            - generic [ref=e125]: Ollama
      - generic "Application version" [ref=e126]:
        - generic [ref=e127]: kaos-control 0.1.2
      - button "Collapse sidebar" [expanded] [ref=e129] [cursor=pointer]:
        - img [ref=e130]
    - main [ref=e132]:
      - generic [ref=e133]:
        - generic [ref=e134]:
          - heading "Dashboard" [level=2] [ref=e135]
          - generic [ref=e136]:
            - button "New Idea" [ref=e137] [cursor=pointer]:
              - img [ref=e138]
              - text: New Idea
            - button "New Defect" [ref=e140] [cursor=pointer]:
              - img [ref=e141]
              - text: New Defect
            - button "New Docs" [ref=e150] [cursor=pointer]:
              - img [ref=e151]
              - text: New Docs
        - generic [ref=e153]:
          - region "Summary statistics" [ref=e154]:
            - link "View 0 lifecycle total artifacts" [ref=e155] [cursor=pointer]:
              - img [ref=e157]
              - generic [ref=e159]:
                - generic [ref=e160]: "0"
                - generic [ref=e161]: Lifecycle Total
            - 'figure "In Progress: 0" [ref=e162]':
              - img [ref=e164]
              - generic [ref=e166]:
                - generic [ref=e167]: "0"
                - generic [ref=e168]: In Progress
            - link "View 0 blocked artifacts" [ref=e169] [cursor=pointer]:
              - img [ref=e171]
              - generic [ref=e173]:
                - generic [ref=e174]: "0"
                - generic [ref=e175]: Blocked
            - 'figure "Completed This Week: 0" [ref=e176]':
              - img [ref=e178]
              - generic [ref=e181]:
                - generic [ref=e182]: "0"
                - generic [ref=e183]: Completed This Week
          - region "Charts" [ref=e184]:
            - generic [ref=e189]:
              - heading "Recent Ideas & Defects" [level=3] [ref=e191]
              - generic [ref=e192]: Loading…
          - region "Velocity and activity" [ref=e193]:
            - generic [ref=e194]:
              - generic [ref=e195]:
                - heading "Recent Activity" [level=3] [ref=e196]
                - button "View all" [ref=e197] [cursor=pointer]
              - generic [ref=e199]: Loading…
```

# Test source

```ts
  1  | import { test, expect } from '../fixtures.js'
  2  | 
  3  | // Test plan: lifecycle/test-plans/tech-writer-agent-5-test.md §Milestone 8
  4  | 
  5  | // smoke-doc-approved.md is a doc fixture in `approved` status
  6  | const APPROVED_DOC_REL = 'lifecycle/docs/smoke-doc-approved.md'
  7  | 
  8  | test.describe('Flow 08 — Queue Work for doc artifacts (NFR3)', () => {
  9  |   test('TC1: Queue Work button is visible on an approved doc artifact', async ({
  10 |     kctest,
  11 |     loggedInPage: page,
  12 |   }) => {
  13 |     await page.goto(`${kctest.baseURL}/p/testproject/artifacts/${APPROVED_DOC_REL}`)
  14 | 
  15 |     // Wait for the artifact view to load
  16 |     await expect(page.locator('.status-badge, [data-status]').first()).toBeVisible({
  17 |       timeout: 10_000,
  18 |     })
  19 | 
  20 |     // Queue Work button is shown when status=approved
  21 |     await expect(page.locator('.btn-queue, button:has-text("Queue Work")')).toBeVisible({
  22 |       timeout: 5_000,
  23 |     })
  24 |   })
  25 | 
  26 |   test('TC2: clicking Queue Work targets the tech-writer agent', async ({
  27 |     kctest,
  28 |     loggedInPage: page,
  29 |   }) => {
  30 |     await page.goto(`${kctest.baseURL}/p/testproject/artifacts/${APPROVED_DOC_REL}`)
  31 | 
  32 |     // Wait for the Queue Work button
  33 |     const queueBtn = page.locator('.btn-queue, button:has-text("Queue Work")')
  34 |     await expect(queueBtn).toBeVisible({ timeout: 10_000 })
  35 | 
  36 |     // Intercept the POST /api/queue request to inspect which agent is targeted
  37 |     const enqueueResponsePromise = page.waitForResponse(
  38 |       (resp) => resp.url().includes('/queue') && resp.request().method() === 'POST',
  39 |       { timeout: 10_000 },
  40 |     )
  41 | 
  42 |     await queueBtn.click()
  43 | 
  44 |     // Wait for enqueue response
  45 |     const enqueueResponse = await enqueueResponsePromise
  46 |     expect([200, 201]).toContain(enqueueResponse.status())
  47 | 
  48 |     // Inspect the request body that was sent to confirm the agent is tech-writer
  49 |     const requestBody = enqueueResponse.request().postDataJSON() as {
  50 |       agent?: string
  51 |       artifact_path?: string
  52 |       project?: string
  53 |     } | null
  54 | 
  55 |     expect(requestBody?.agent).toBe('tech-writer')
  56 |     expect(requestBody?.artifact_path).toBe(APPROVED_DOC_REL)
  57 |   })
  58 | 
  59 |   test('TC3: ready count endpoint includes approved doc for tech-writer agent', async ({
  60 |     kctest,
  61 |     loggedInPage: page,
  62 |   }) => {
  63 |     // Fetch the agents list and verify tech-writer reports ready_count >= 1
  64 |     await page.goto(`${kctest.baseURL}/p/testproject/dashboard`)
  65 |     await page.waitForURL(`${kctest.baseURL}/p/testproject/dashboard`)
  66 | 
  67 |     // Pull session cookies from the browser for direct API calls
  68 |     const cookies = await page.context().cookies()
  69 |     const cookieHeader = cookies.map((c) => `${c.name}=${c.value}`).join('; ')
  70 | 
  71 |     const res = await fetch(`${kctest.baseURL}/api/p/testproject/agents`, {
  72 |       headers: { Cookie: cookieHeader },
  73 |     })
> 74 |     expect(res.status).toBe(200)
     |                        ^ Error: expect(received).toBe(expected) // Object.is equality
  75 | 
  76 |     const data = (await res.json()) as { agents?: { name: string; ready_count?: number }[] }
  77 |     const agents = data?.agents ?? []
  78 | 
  79 |     const techWriterAgent = agents.find((a) => a.name === 'tech-writer')
  80 |     expect(techWriterAgent).toBeTruthy()
  81 | 
  82 |     // The approved doc fixture means tech-writer must have at least 1 ready artifact
  83 |     expect(techWriterAgent?.ready_count ?? 0).toBeGreaterThanOrEqual(1)
  84 |   })
  85 | })
  86 | 
```