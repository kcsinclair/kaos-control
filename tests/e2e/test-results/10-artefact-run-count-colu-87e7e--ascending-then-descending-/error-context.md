# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 10-artefact-run-count-column.spec.ts >> Flow 10 — Artefact run count column >> TC2: Runs column is sortable (ascending then descending)
- Location: flows/10-artefact-run-count-column.spec.ts:142:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('th.sort-th, th[role="columnheader"]').filter({ hasText: /^Runs$/ })
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for locator('th.sort-th, th[role="columnheader"]').filter({ hasText: /^Runs$/ })

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
    - listitem:
      - link "DevOps":
        - /url: /p/testproject/devops
  - status "Git repository status": main clean 5f8b984 Initial fixture commit
  - text: kaos-control 0.1.2
  - button "Collapse sidebar" [expanded]
- main:
  - heading "Artefacts" [level=2]
  - text: 18 total
  - checkbox "Show completed"
  - text: Show completed
  - button "Check statuses"
  - button "New Idea"
  - button "New Defect"
  - button "New Docs"
  - textbox "Filter artifacts by text":
    - /placeholder: Filter by text…
  - combobox:
    - option "All stages" [selected]
    - option "ideas"
    - option "requirements"
    - option "backend-plans"
    - option "frontend-plans"
    - option "test-plans"
    - option "dev-plans"
    - option "tests"
    - option "prototypes"
    - option "defects"
    - option "releases"
  - combobox:
    - option "All statuses" [selected]
    - option "draft"
    - option "clarifying"
    - option "planning"
    - option "in-development"
    - option "in-qa"
    - option "in-progress"
    - option "done"
    - option "approved"
    - option "blocked"
    - option "rejected"
    - option "abandoned"
  - combobox:
    - option "All types" [selected]
    - option "idea"
    - option "requirement"
    - option "plan-backend"
    - option "plan-frontend"
    - option "plan-test"
    - option "test"
    - option "prototype"
    - option "defect"
  - combobox:
    - option "All labels" [selected]
    - option "defect"
  - text: Release
  - combobox "Release":
    - option "All releases" [selected]
    - option "Unassigned"
  - button "Reset"
  - table:
    - rowgroup:
      - row "Path Stage Status Priority Release Type Runs Created Modified":
        - columnheader "Path"
        - columnheader "Stage"
        - columnheader "Status"
        - columnheader "Priority"
        - columnheader "Release"
        - columnheader "Type"
        - columnheader "Runs"
        - columnheader "Created"
        - columnheader "Modified"
    - rowgroup:
      - row "RC Idea A lifecycle/ideas/rc-idea-a.md ideas draft — — idea 0 May 16, 2026 May 16, 2026":
        - cell "RC Idea A lifecycle/ideas/rc-idea-a.md"
        - cell "ideas"
        - cell "draft"
        - cell "—"
        - cell "—"
        - cell "idea"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "RC Idea B lifecycle/ideas/rc-idea-b.md ideas draft — — idea 0 May 16, 2026 May 16, 2026":
        - cell "RC Idea B lifecycle/ideas/rc-idea-b.md"
        - cell "ideas"
        - cell "draft"
        - cell "—"
        - cell "—"
        - cell "idea"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "RC Idea C lifecycle/ideas/rc-idea-c.md ideas draft — — idea 0 May 16, 2026 May 16, 2026":
        - cell "RC Idea C lifecycle/ideas/rc-idea-c.md"
        - cell "ideas"
        - cell "draft"
        - cell "—"
        - cell "—"
        - cell "idea"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "RC Pill Target lifecycle/ideas/rc-pill.md ideas draft — — idea 0 May 16, 2026 May 16, 2026":
        - cell "RC Pill Target lifecycle/ideas/rc-pill.md"
        - cell "ideas"
        - cell "draft"
        - cell "—"
        - cell "—"
        - cell "idea"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "RC WebSocket Target lifecycle/ideas/rc-ws.md ideas draft — — idea 0 May 16, 2026 May 16, 2026":
        - cell "RC WebSocket Target lifecycle/ideas/rc-ws.md"
        - cell "ideas"
        - cell "draft"
        - cell "—"
        - cell "—"
        - cell "idea"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Defect Alpha lifecycle/defects/smoke-defect-01.md defects draft — — defect 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Defect Alpha lifecycle/defects/smoke-defect-01.md"
        - cell "defects"
        - cell "draft"
        - cell "—"
        - cell "—"
        - cell "defect"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Doc Approved lifecycle/docs/smoke-doc-approved.md docs approved — — doc 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Doc Approved lifecycle/docs/smoke-doc-approved.md"
        - cell "docs"
        - cell "approved"
        - cell "—"
        - cell "—"
        - cell "doc"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Idea 01 lifecycle/ideas/smoke-idea-01.md ideas draft — — idea 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Idea 01 lifecycle/ideas/smoke-idea-01.md"
        - cell "ideas"
        - cell "draft"
        - cell "—"
        - cell "—"
        - cell "idea"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Idea 02 lifecycle/ideas/smoke-idea-02.md ideas draft — — idea 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Idea 02 lifecycle/ideas/smoke-idea-02.md"
        - cell "ideas"
        - cell "draft"
        - cell "—"
        - cell "—"
        - cell "idea"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Idea 03 lifecycle/ideas/smoke-idea-03.md ideas draft — — idea 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Idea 03 lifecycle/ideas/smoke-idea-03.md"
        - cell "ideas"
        - cell "draft"
        - cell "—"
        - cell "—"
        - cell "idea"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Idea 04 lifecycle/ideas/smoke-idea-04.md ideas draft — — idea 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Idea 04 lifecycle/ideas/smoke-idea-04.md"
        - cell "ideas"
        - cell "draft"
        - cell "—"
        - cell "—"
        - cell "idea"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Idea 05 lifecycle/ideas/smoke-idea-05.md ideas approved — — idea 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Idea 05 lifecycle/ideas/smoke-idea-05.md"
        - cell "ideas"
        - cell "approved"
        - cell "—"
        - cell "—"
        - cell "idea"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Idea 06 lifecycle/ideas/smoke-idea-06.md ideas approved — — idea 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Idea 06 lifecycle/ideas/smoke-idea-06.md"
        - cell "ideas"
        - cell "approved"
        - cell "—"
        - cell "—"
        - cell "idea"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Idea 07 lifecycle/ideas/smoke-idea-07.md ideas approved — — idea 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Idea 07 lifecycle/ideas/smoke-idea-07.md"
        - cell "ideas"
        - cell "approved"
        - cell "—"
        - cell "—"
        - cell "idea"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Requirement Alpha — Documentation lifecycle/docs/smoke-doc-linked.md docs draft — — doc 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Requirement Alpha — Documentation lifecycle/docs/smoke-doc-linked.md"
        - cell "docs"
        - cell "draft"
        - cell "—"
        - cell "—"
        - cell "doc"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Requirement Alpha lifecycle/requirements/smoke-req-01.md requirements draft — — requirement 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Requirement Alpha lifecycle/requirements/smoke-req-01.md"
        - cell "requirements"
        - cell "draft"
        - cell "—"
        - cell "—"
        - cell "requirement"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Requirement Beta lifecycle/requirements/smoke-req-02.md requirements planning — — requirement 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Requirement Beta lifecycle/requirements/smoke-req-02.md"
        - cell "requirements"
        - cell "planning"
        - cell "—"
        - cell "—"
        - cell "requirement"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
      - row "Smoke Requirement Gamma lifecycle/requirements/smoke-req-03.md requirements in-development — — requirement 0 May 16, 2026 May 16, 2026":
        - cell "Smoke Requirement Gamma lifecycle/requirements/smoke-req-03.md"
        - cell "requirements"
        - cell "in-development"
        - cell "—"
        - cell "—"
        - cell "requirement"
        - cell "0"
        - cell "May 16, 2026"
        - cell "May 16, 2026"
  - navigation "Table pagination":
    - text: Rows per page
    - combobox "Rows per page":
      - option "10"
      - option "25" [selected]
      - option "50"
      - option "100"
    - text: Showing 1–18 of 18
    - button "Previous page" [disabled]: ← Prev
    - text: Page
    - spinbutton "Jump to page" [disabled]: "1"
    - text: of 1
    - button "Next page" [disabled]: Next →
```

# Test source

```ts
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
  132 | 
  133 |     // rc-idea-b shows count 1
  134 |     const rowB = page.locator('tr').filter({ has: page.locator('.artifact-path', { hasText: 'rc-idea-b.md' }) })
  135 |     await expect(rowB.locator('.cell-runs')).toHaveText('1', { timeout: 5_000 })
  136 | 
  137 |     // rc-idea-c shows count 0 (not blank)
  138 |     const rowC = page.locator('tr').filter({ has: page.locator('.artifact-path', { hasText: 'rc-idea-c.md' }) })
  139 |     await expect(rowC.locator('.cell-runs')).toHaveText('0', { timeout: 5_000 })
  140 |   })
  141 | 
  142 |   test('TC2: Runs column is sortable (ascending then descending)', async ({
  143 |     kctest,
  144 |     loggedInPage: page,
  145 |   }) => {
  146 |     // Runs seeded in TC1 persist — rc-idea-a=2, rc-idea-b=1, rc-idea-c=0
  147 |     await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)
  148 | 
  149 |     // Wait for the table to load
  150 |     await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10_000 })
  151 | 
  152 |     const runsHeader = page.locator('th.sort-th, th[role="columnheader"]').filter({ hasText: /^Runs$/ })
> 153 |     await expect(runsHeader).toBeVisible({ timeout: 5_000 })
      |                              ^ Error: expect(locator).toBeVisible() failed
  154 | 
  155 |     // Click once → ascending sort
  156 |     await runsHeader.click()
  157 |     await page.waitForTimeout(300) // allow Vue reactivity to settle
  158 | 
  159 |     let cells = page.locator('.cell-runs')
  160 |     let counts = (await cells.allTextContents()).map(Number)
  161 |     // Verify all counts are in non-decreasing order
  162 |     for (let i = 1; i < counts.length; i++) {
  163 |       expect(counts[i]).toBeGreaterThanOrEqual(counts[i - 1])
  164 |     }
  165 | 
  166 |     // Click again → descending sort
  167 |     await runsHeader.click()
  168 |     await page.waitForTimeout(300)
  169 | 
  170 |     cells = page.locator('.cell-runs')
  171 |     counts = (await cells.allTextContents()).map(Number)
  172 |     // Verify all counts are in non-increasing order
  173 |     for (let i = 1; i < counts.length; i++) {
  174 |       expect(counts[i]).toBeLessThanOrEqual(counts[i - 1])
  175 |     }
  176 |   })
  177 | 
  178 |   // ─────────────────────────────────────────────────────────────────────────
  179 |   // Milestone 5 — Active-agent status pill
  180 |   // ─────────────────────────────────────────────────────────────────────────
  181 | 
  182 |   test('TC3: "Agent Running" pill appears while run is active and disappears on completion', async ({
  183 |     kctest,
  184 |     loggedInPage: page,
  185 |   }) => {
  186 |     await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)
  187 |     await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10_000 })
  188 | 
  189 |     const headers = await getRunHeaders(page, kctest.baseURL)
  190 | 
  191 |     // Listen for agent.started before triggering so we don't miss it
  192 |     const agentStarted = waitForWsEvent(wsURL(kctest.baseURL), 'agent.started')
  193 | 
  194 |     const runId = await triggerRun(kctest.baseURL, headers, RC_PILL)
  195 | 
  196 |     // Wait until the server has confirmed the run has started
  197 |     await agentStarted
  198 | 
  199 |     const pillRow = page
  200 |       .locator('tr')
  201 |       .filter({ has: page.locator('.artifact-path', { hasText: 'rc-pill.md' }) })
  202 | 
  203 |     // Pill with "Agent Running" text appears (data-status="running")
  204 |     const pill = pillRow.locator('.agent-status-pill[data-status="running"]')
  205 |     await expect(pill).toBeVisible({ timeout: 8_000 })
  206 |     await expect(pill).toHaveText('Agent Running')
  207 | 
  208 |     // Wait for the run to complete
  209 |     await waitForRunStatus(kctest.baseURL, headers, runId, ['done', 'failed'])
  210 | 
  211 |     // After run finishes the pill disappears (WS drives a re-fetch)
  212 |     await expect(pillRow.locator('.agent-status-pill')).not.toBeVisible({ timeout: 10_000 })
  213 |   })
  214 | 
  215 |   // ─────────────────────────────────────────────────────────────────────────
  216 |   // Milestone 6 — WebSocket-driven count refresh without page reload
  217 |   // ─────────────────────────────────────────────────────────────────────────
  218 | 
  219 |   test('TC4: run count increments without page reload on agent.finished WS event', async ({
  220 |     kctest,
  221 |     loggedInPage: page,
  222 |   }) => {
  223 |     await page.goto(`${kctest.baseURL}/p/testproject/artifacts`)
  224 |     await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10_000 })
  225 | 
  226 |     const headers = await getRunHeaders(page, kctest.baseURL)
  227 |     const wsRow = page
  228 |       .locator('tr')
  229 |       .filter({ has: page.locator('.artifact-path', { hasText: 'rc-ws.md' }) })
  230 | 
  231 |     // Note the current run count for rc-ws.md (0 at this point in the worker)
  232 |     const runsCell = wsRow.locator('.cell-runs')
  233 |     const initialCountText = await runsCell.textContent({ timeout: 5_000 })
  234 |     const initialCount = Number(initialCountText ?? '0')
  235 | 
  236 |     const beforeURL = page.url()
  237 | 
  238 |     // Listen for agent.finished so we know when to check the DOM
  239 |     const agentFinished = waitForWsEvent(wsURL(kctest.baseURL), 'agent.finished', 15_000)
  240 | 
  241 |     await triggerRun(kctest.baseURL, headers, RC_WS)
  242 | 
  243 |     // Wait for the finished event — the Vue component re-fetches on this event
  244 |     await agentFinished
  245 | 
  246 |     // Count must have incremented by 1 without a page reload
  247 |     const expectedCount = String(initialCount + 1)
  248 |     await expect(runsCell).toHaveText(expectedCount, { timeout: 10_000 })
  249 | 
  250 |     // URL must not have changed (no full page navigation occurred)
  251 |     expect(page.url()).toBe(beforeURL)
  252 |   })
  253 | })
```