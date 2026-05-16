# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 09-doc-graph.spec.ts >> Flow 09 — Graph rendering for doc nodes (NFR1) >> TC1: doc node exists in the 2D map view
- Location: flows/09-doc-graph.spec.ts:16:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('canvas')
Expected: visible
Error: strict mode violation: locator('canvas') resolved to 3 elements:
    1) <canvas width="860" height="668" data-id="layer0-selectbox"></canvas> aka locator('canvas').first()
    2) <canvas width="860" height="668" data-id="layer1-drag"></canvas> aka locator('canvas').nth(1)
    3) <canvas width="860" height="668" data-id="layer2-node"></canvas> aka locator('canvas').nth(2)

Call log:
  - Expect "toBeVisible" with timeout 15000ms
  - waiting for locator('canvas')

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
        - listitem [ref=e126]:
          - link "DevOps" [ref=e128] [cursor=pointer]:
            - /url: /p/testproject/devops
            - img [ref=e130]
            - generic [ref=e134]: DevOps
      - status "Git repository status" [ref=e135]:
        - generic [ref=e136]:
          - img [ref=e137]
          - generic "main" [ref=e141]
          - generic "Working tree is clean" [ref=e142]: clean
        - generic [ref=e143]:
          - generic [ref=e144]: 5f8b984
          - generic "Initial fixture commit" [ref=e145]
      - generic "Application version" [ref=e146]:
        - generic [ref=e147]: kaos-control 0.1.2
      - button "Collapse sidebar" [expanded] [ref=e149] [cursor=pointer]:
        - img [ref=e150]
    - main [ref=e152]:
      - generic [ref=e153]:
        - complementary [ref=e154]:
          - generic [ref=e156]: Filters
          - generic [ref=e157]: 18 / 22 nodes
          - generic [ref=e159]:
            - img [ref=e160]
            - textbox "Filter artifacts by text" [ref=e163]:
              - /placeholder: Search nodes…
          - generic [ref=e164]:
            - generic [ref=e165] [cursor=pointer]:
              - checkbox "Show label nodes" [ref=e166]
              - generic [ref=e167]: Show label nodes
            - generic [ref=e168] [cursor=pointer]:
              - checkbox "Show completed" [ref=e169]
              - generic [ref=e170]: Show completed
            - generic [ref=e171] [cursor=pointer]:
              - checkbox "Show tests" [ref=e172]
              - generic [ref=e173]: Show tests
            - generic "Show release overlay nodes and timeline" [ref=e174] [cursor=pointer]:
              - checkbox "Show release overlay nodes and timeline" [ref=e175]
              - generic [ref=e176]: Show Releases
            - generic [ref=e177] [cursor=pointer]:
              - checkbox "Show node titles" [ref=e178]
              - generic [ref=e179]: Show node titles
            - generic [ref=e180] [cursor=pointer]:
              - checkbox "Show node lineage" [ref=e181]
              - generic [ref=e182]: Show node lineage
          - generic [ref=e183]:
            - generic [ref=e184]: Type
            - generic [ref=e185]:
              - button "defect" [ref=e186] [cursor=pointer]
              - button "doc" [ref=e187] [cursor=pointer]
              - button "idea" [ref=e188] [cursor=pointer]
              - button "requirement" [ref=e189] [cursor=pointer]
          - generic [ref=e190]:
            - generic [ref=e191]: Status
            - generic [ref=e192]:
              - button "abandoned" [ref=e193] [cursor=pointer]
              - button "approved" [ref=e194] [cursor=pointer]
              - button "done" [ref=e195] [cursor=pointer]
              - button "draft" [ref=e196] [cursor=pointer]
              - button "in-development" [ref=e197] [cursor=pointer]
              - button "planning" [ref=e198] [cursor=pointer]
          - generic [ref=e199]:
            - generic [ref=e200]: Lineage
            - generic [ref=e201]:
              - button "rc-idea-a" [ref=e202] [cursor=pointer]
              - button "rc-idea-b" [ref=e203] [cursor=pointer]
              - button "rc-idea-c" [ref=e204] [cursor=pointer]
              - button "rc-pill" [ref=e205] [cursor=pointer]
              - button "rc-ws" [ref=e206] [cursor=pointer]
              - button "smoke-defect-01" [ref=e207] [cursor=pointer]
              - button "smoke-doc-approved" [ref=e208] [cursor=pointer]
              - button "smoke-idea-01" [ref=e209] [cursor=pointer]
              - button "smoke-idea-02" [ref=e210] [cursor=pointer]
              - button "smoke-idea-03" [ref=e211] [cursor=pointer]
              - button "smoke-idea-04" [ref=e212] [cursor=pointer]
              - button "smoke-idea-05" [ref=e213] [cursor=pointer]
              - button "smoke-idea-06" [ref=e214] [cursor=pointer]
              - button "smoke-idea-07" [ref=e215] [cursor=pointer]
              - button "smoke-idea-08" [ref=e216] [cursor=pointer]
              - button "smoke-idea-09" [ref=e217] [cursor=pointer]
              - button "smoke-idea-10" [ref=e218] [cursor=pointer]
              - button "smoke-req-01" [ref=e219] [cursor=pointer]
              - button "smoke-req-02" [ref=e220] [cursor=pointer]
              - button "smoke-req-03" [ref=e221] [cursor=pointer]
              - button "smoke-req-done" [ref=e222] [cursor=pointer]
          - generic [ref=e223]:
            - generic [ref=e224]: Label
            - button "defect" [ref=e226] [cursor=pointer]
        - generic [ref=e227]:
          - generic [ref=e228]:
            - group "Map view mode" [ref=e229]:
              - button "3D" [ref=e230] [cursor=pointer]
              - button "2D" [ref=e231] [cursor=pointer]
            - generic "2D map layout controls" [ref=e232]:
              - generic [ref=e233]: Layout
              - combobox "Select map layout algorithm" [ref=e234] [cursor=pointer]:
                - option "fCoSE (Force-Directed)" [selected]
                - option "Breadth-First"
                - option "Concentric"
                - option "Circle"
                - option "Dagre (DAG)"
              - button "Toggle directed map mode" [ref=e235] [cursor=pointer]: Directed
            - button "Check all statuses" [ref=e236] [cursor=pointer]
          - img "2D artifact map" [ref=e237]
          - generic:
            - generic:
              - generic:
                - generic: Nodes
                - generic:
                  - generic: Idea
                - generic:
                  - generic: Requirement
                - generic:
                  - generic: Plan Backend
                - generic:
                  - generic: Plan Frontend
                - generic:
                  - generic: Plan Test
                - generic:
                  - generic: Test
                - generic:
                  - generic: Prototype
                - generic:
                  - generic: Defect
                - generic:
                  - generic: Doc
              - generic:
                - generic: Priority
                - generic:
                  - generic: High
                - generic:
                  - generic: Medium
                - generic:
                  - generic: Normal
                - generic:
                  - generic: Low
              - generic:
                - generic: Edges
                - generic:
                  - generic: Parent
                - generic:
                  - generic: Depends On
                - generic:
                  - generic: Blocks
                - generic:
                  - generic: Related To
          - generic: Scroll to zoom · Drag to pan · Click node to inspect
```

# Test source

```ts
  1   | import { test, expect } from '../fixtures.js'
  2   | 
  3   | // Test plan: lifecycle/test-plans/tech-writer-agent-5-test.md §Milestone 9
  4   | 
  5   | // smoke-doc-linked.md has lineage=smoke-req-01 and parent=lifecycle/requirements/smoke-req-01.md
  6   | const DOC_REL = 'lifecycle/docs/smoke-doc-linked.md'
  7   | const PARENT_REL = 'lifecycle/requirements/smoke-req-01.md'
  8   | 
  9   | // The expected colour for `doc` nodes in the dark palette (graphConstants.ts).
  10  | // Verified against the graphConstants nodeColors definition.
  11  | const DOC_NODE_COLOR_DARK = '#2dd4bf'   // teal-400
  12  | const IDEA_NODE_COLOR_DARK = '#f59e0b'  // amber-400
  13  | const REQ_NODE_COLOR_DARK = '#3b82f6'   // blue-500
  14  | 
  15  | test.describe('Flow 09 — Graph rendering for doc nodes (NFR1)', () => {
  16  |   test('TC1: doc node exists in the 2D map view', async ({
  17  |     kctest,
  18  |     loggedInPage: page,
  19  |   }) => {
  20  |     await page.goto(`${kctest.baseURL}/p/testproject/map`)
  21  | 
  22  |     // Wait for Cytoscape canvas and __cy to be ready
> 23  |     await expect(page.locator('canvas')).toBeVisible({ timeout: 15_000 })
      |                                          ^ Error: expect(locator).toBeVisible() failed
  24  |     await page.waitForFunction(() => !!(window as any).__cy, { timeout: 15_000 })
  25  | 
  26  |     // Wait for layout to stabilise with at least one positioned node
  27  |     await page.waitForFunction(
  28  |       () => {
  29  |         const cy = (window as any).__cy
  30  |         return cy && cy.nodes().length > 0 && cy.nodes().first().position().x !== 0
  31  |       },
  32  |       { timeout: 15_000 },
  33  |     )
  34  | 
  35  |     // Assert the doc node exists (matched by path in its data)
  36  |     const docNodeExists = await page.evaluate((docPath: string) => {
  37  |       const cy = (window as any).__cy
  38  |       const node = cy.nodes().filter((n: any) => {
  39  |         const raw = n.data('_raw')
  40  |         return (
  41  |           (raw && raw.path === docPath) ||
  42  |           n.data('id') === docPath
  43  |         )
  44  |       })
  45  |       return node.length > 0
  46  |     }, DOC_REL)
  47  | 
  48  |     expect(docNodeExists).toBe(true)
  49  |   })
  50  | 
  51  |   test('TC2: an edge connects the doc node to its parent artifact', async ({
  52  |     kctest,
  53  |     loggedInPage: page,
  54  |   }) => {
  55  |     await page.goto(`${kctest.baseURL}/p/testproject/map`)
  56  | 
  57  |     await expect(page.locator('canvas')).toBeVisible({ timeout: 15_000 })
  58  |     await page.waitForFunction(() => !!(window as any).__cy, { timeout: 15_000 })
  59  |     await page.waitForFunction(
  60  |       () => {
  61  |         const cy = (window as any).__cy
  62  |         return cy && cy.nodes().length > 0 && cy.nodes().first().position().x !== 0
  63  |       },
  64  |       { timeout: 15_000 },
  65  |     )
  66  | 
  67  |     // Assert an edge exists between the doc node and its parent requirement node
  68  |     const edgeExists = await page.evaluate(
  69  |       ([docPath, parentPath]: [string, string]) => {
  70  |         const cy = (window as any).__cy
  71  |         const docNode = cy.nodes().filter((n: any) => {
  72  |           const raw = n.data('_raw')
  73  |           return (raw && raw.path === docPath) || n.data('id') === docPath
  74  |         })
  75  |         const parentNode = cy.nodes().filter((n: any) => {
  76  |           const raw = n.data('_raw')
  77  |           return (raw && raw.path === parentPath) || n.data('id') === parentPath
  78  |         })
  79  |         if (!docNode.length || !parentNode.length) return false
  80  |         // Check for an edge connecting doc → parent (either direction)
  81  |         const edges = cy.edges()
  82  |         return edges.some((e: any) => {
  83  |           const src = e.data('source')
  84  |           const tgt = e.data('target')
  85  |           return (
  86  |             (src === docNode.first().id() && tgt === parentNode.first().id()) ||
  87  |             (src === parentNode.first().id() && tgt === docNode.first().id())
  88  |           )
  89  |         })
  90  |       },
  91  |       [DOC_REL, PARENT_REL] as [string, string],
  92  |     )
  93  | 
  94  |     expect(edgeExists).toBe(true)
  95  |   })
  96  | 
  97  |   test('TC3: doc node uses a distinct colour from idea and requirement nodes', async ({
  98  |     kctest,
  99  |     loggedInPage: page,
  100 |   }) => {
  101 |     await page.goto(`${kctest.baseURL}/p/testproject/map`)
  102 | 
  103 |     await expect(page.locator('canvas')).toBeVisible({ timeout: 15_000 })
  104 |     await page.waitForFunction(() => !!(window as any).__cy, { timeout: 15_000 })
  105 |     await page.waitForFunction(
  106 |       () => {
  107 |         const cy = (window as any).__cy
  108 |         return cy && cy.nodes().length > 0 && cy.nodes().first().position().x !== 0
  109 |       },
  110 |       { timeout: 15_000 },
  111 |     )
  112 | 
  113 |     // Retrieve the fill colour of the doc node via Cytoscape style
  114 |     const docNodeColor: string = await page.evaluate((docPath: string) => {
  115 |       const cy = (window as any).__cy
  116 |       const node = cy.nodes().filter((n: any) => {
  117 |         const raw = n.data('_raw')
  118 |         return (raw && raw.path === docPath) || n.data('id') === docPath
  119 |       }).first()
  120 |       return node ? node.style('background-color') : ''
  121 |     }, DOC_REL)
  122 | 
  123 |     // Retrieve colours of an idea node and a requirement node for comparison
```