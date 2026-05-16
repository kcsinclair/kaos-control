# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 10-artefact-run-count-column.spec.ts >> Flow 10 — Artefact run count column >> TC1: Runs column is present, positioned correctly, shows correct counts including 0
- Location: flows/10-artefact-run-count-column.spec.ts:94:3

# Error details

```
Error: triggerRun failed (409): {"error":{"code":"run_error","message":"agent \"stub-agent\" has no prompt template for role \"product-owner\""}}

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
  - main [ref=e18]:
    - generic [ref=e19]:
      - generic [ref=e20]:
        - heading "Projects" [level=2] [ref=e21]
        - button "New Project" [ref=e22] [cursor=pointer]
      - table [ref=e24]:
        - rowgroup [ref=e25]:
          - row "Name Description Owner Path Status Actions" [ref=e26]:
            - columnheader "Name" [ref=e27]
            - columnheader "Description" [ref=e28]
            - columnheader "Owner" [ref=e29]
            - columnheader "Path" [ref=e30]
            - columnheader "Status" [ref=e31]
            - columnheader "Actions" [ref=e32]
        - rowgroup [ref=e33]:
          - row "testproject E2E smoke test project admin@kaos-e2e.local /var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/kc-proj-KOANlI Initialised Edit Delete" [ref=e34]:
            - cell "testproject" [ref=e35]:
              - link "testproject" [ref=e36] [cursor=pointer]:
                - /url: "#"
            - cell "E2E smoke test project" [ref=e37]
            - cell "admin@kaos-e2e.local" [ref=e38]
            - cell "/var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/kc-proj-KOANlI" [ref=e39]:
              - generic "/var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/kc-proj-KOANlI" [ref=e40]
            - cell "Initialised" [ref=e41]:
              - generic [ref=e42]: Initialised
            - cell "Edit Delete" [ref=e43]:
              - generic [ref=e44]:
                - button "Edit" [ref=e45] [cursor=pointer]
                - button "Delete" [ref=e46] [cursor=pointer]
```

# Test source

```ts
  1   | import { test, expect, ADMIN_CREDS } from '../fixtures.js'
  2   | import type { Page } from '@playwright/test'
  3   | 
  4   | // Fixture paths for run count column tests
  5   | const RC_IDEA_A = 'lifecycle/ideas/rc-idea-a.md' // seeded with 2 runs
  6   | const RC_IDEA_B = 'lifecycle/ideas/rc-idea-b.md' // seeded with 1 run
  7   | const RC_IDEA_C = 'lifecycle/ideas/rc-idea-c.md' // seeded with 0 runs
  8   | const RC_PILL = 'lifecycle/ideas/rc-pill.md' // used for pill test
  9   | const RC_WS = 'lifecycle/ideas/rc-ws.md' // used for WS refresh test
  10  | 
  11  | type RunHeaders = Record<string, string>
  12  | 
  13  | async function getRunHeaders(page: Page, baseURL: string): Promise<RunHeaders> {
  14  |   const cookies = await page.context().cookies()
  15  |   const csrfToken = cookies.find((c) => c.name === 'kc_csrf')?.value ?? ''
  16  |   return {
  17  |     Cookie: cookies.map((c) => `${c.name}=${c.value}`).join('; '),
  18  |     'X-CSRF-Token': csrfToken,
  19  |     'Content-Type': 'application/json',
  20  |   }
  21  | }
  22  | 
  23  | async function triggerRun(baseURL: string, headers: RunHeaders, targetPath: string): Promise<string> {
  24  |   const res = await fetch(`${baseURL}/api/p/testproject/agents/stub-agent/run`, {
  25  |     method: 'POST',
  26  |     headers,
  27  |     body: JSON.stringify({ target_path: targetPath }),
  28  |   })
  29  |   if (!res.ok) {
  30  |     const text = await res.text()
> 31  |     throw new Error(`triggerRun failed (${res.status}): ${text}`)
      |           ^ Error: triggerRun failed (409): {"error":{"code":"run_error","message":"agent \"stub-agent\" has no prompt template for role \"product-owner\""}}
  32  |   }
  33  |   const data = (await res.json()) as { run_id?: string }
  34  |   if (!data.run_id) throw new Error('triggerRun: no run_id in response')
  35  |   return data.run_id
  36  | }
  37  | 
  38  | async function waitForRunStatus(
  39  |   baseURL: string,
  40  |   headers: RunHeaders,
  41  |   runId: string,
  42  |   targetStatuses: string[],
  43  |   timeoutMs = 15_000,
  44  | ): Promise<string> {
  45  |   const deadline = Date.now() + timeoutMs
  46  |   while (Date.now() < deadline) {
  47  |     const res = await fetch(`${baseURL}/api/p/testproject/agents/runs/${runId}`, { headers })
  48  |     if (res.ok) {
  49  |       const data = (await res.json()) as { run?: { status?: string } }
  50  |       const status = data.run?.status ?? ''
  51  |       if (targetStatuses.includes(status)) return status
  52  |     }
  53  |     await new Promise((r) => setTimeout(r, 200))
  54  |   }
  55  |   throw new Error(`run ${runId} did not reach ${targetStatuses.join('|')} within ${timeoutMs}ms`)
  56  | }
  57  | 
  58  | function wsURL(baseURL: string): string {
  59  |   return baseURL.replace(/^http/, 'ws') + '/api/p/testproject/ws'
  60  | }
  61  | 
  62  | function waitForWsEvent(
  63  |   url: string,
  64  |   eventType: string,
  65  |   timeoutMs = 12_000,
  66  | ): Promise<Record<string, unknown>> {
  67  |   return new Promise((resolve, reject) => {
  68  |     const ws = new WebSocket(url)
  69  |     const timer = setTimeout(() => {
  70  |       ws.close()
  71  |       reject(new Error(`Timed out waiting for WS event ${eventType}`))
  72  |     }, timeoutMs)
  73  |     ws.addEventListener('message', (msg) => {
  74  |       let ev: { type: string; payload: Record<string, unknown> }
  75  |       try {
  76  |         ev = JSON.parse(msg.data as string)
  77  |       } catch {
  78  |         return
  79  |       }
  80  |       if (ev.type === eventType) {
  81  |         clearTimeout(timer)
  82  |         ws.close()
  83  |         resolve(ev.payload)
  84  |       }
  85  |     })
  86  |   })
  87  | }
  88  | 
  89  | // ─────────────────────────────────────────────────────────────────────────────
  90  | // Milestone 4 — Runs column rendering, counts, and sorting
  91  | // ─────────────────────────────────────────────────────────────────────────────
  92  | 
  93  | test.describe('Flow 10 — Artefact run count column', () => {
  94  |   test('TC1: Runs column is present, positioned correctly, shows correct counts including 0', async ({
  95  |     kctest,
  96  |     loggedInPage: page,
  97  |   }) => {
  98  |     const headers = await getRunHeaders(page, kctest.baseURL)
  99  | 
  100 |     // Seed 2 completed runs for rc-idea-a
  101 |     const runA1 = await triggerRun(kctest.baseURL, headers, RC_IDEA_A)
  102 |     await waitForRunStatus(kctest.baseURL, headers, runA1, ['done', 'failed'])
  103 |     const runA2 = await triggerRun(kctest.baseURL, headers, RC_IDEA_A)
  104 |     await waitForRunStatus(kctest.baseURL, headers, runA2, ['done', 'failed'])
  105 | 
  106 |     // Seed 1 completed run for rc-idea-b
  107 |     const runB1 = await triggerRun(kctest.baseURL, headers, RC_IDEA_B)
  108 |     await waitForRunStatus(kctest.baseURL, headers, runB1, ['done', 'failed'])
  109 | 
  110 |     // rc-idea-c has 0 runs
  111 | 
  112 |     await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)
  113 | 
  114 |     // Column header "Runs" is visible
  115 |     const runsHeader = page.locator('th.sort-th, th[role="columnheader"]').filter({ hasText: /^Runs$/ })
  116 |     await expect(runsHeader).toBeVisible({ timeout: 10_000 })
  117 | 
  118 |     // "Runs" appears after "Type" and before "Created" in the header row
  119 |     const allHeaders = page.locator('table thead th')
  120 |     const headerTexts = await allHeaders.allTextContents()
  121 |     const normalised = headerTexts.map((h) => h.replace(/\s+/g, ' ').trim())
  122 |     const typeIdx = normalised.findIndex((h) => h.startsWith('Type'))
  123 |     const runsIdx = normalised.findIndex((h) => h.startsWith('Runs'))
  124 |     const createdIdx = normalised.findIndex((h) => h.startsWith('Created'))
  125 |     expect(typeIdx).toBeGreaterThanOrEqual(0)
  126 |     expect(runsIdx).toBeGreaterThan(typeIdx)
  127 |     expect(createdIdx).toBeGreaterThan(runsIdx)
  128 | 
  129 |     // rc-idea-a shows count 2
  130 |     const rowA = page.locator('tr').filter({ has: page.locator('.artifact-path', { hasText: 'rc-idea-a.md' }) })
  131 |     await expect(rowA.locator('.cell-runs')).toHaveText('2', { timeout: 10_000 })
```