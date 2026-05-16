# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 10-artefact-run-count-column.spec.ts >> Flow 10 — Artefact run count column >> TC4: run count increments without page reload on agent.finished WS event
- Location: flows/10-artefact-run-count-column.spec.ts:219:3

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
          - generic [ref=e144]: 4a483c1
          - generic "Initial fixture commit" [ref=e145]
      - generic "Application version" [ref=e146]:
        - generic [ref=e147]: kaos-control 0.1.2
      - button "Collapse sidebar" [expanded] [ref=e149] [cursor=pointer]:
        - img [ref=e150]
    - main [ref=e152]:
      - generic [ref=e153]:
        - generic [ref=e154]:
          - heading "Artefacts" [level=2] [ref=e155]
          - generic [ref=e156]: 18 total
          - generic [ref=e157] [cursor=pointer]:
            - checkbox "Show completed" [ref=e158]
            - generic [ref=e159]: Show completed
          - button "Check statuses" [ref=e160] [cursor=pointer]:
            - img [ref=e161]
            - text: Check statuses
          - button "New Idea" [ref=e164] [cursor=pointer]:
            - img [ref=e165]
            - text: New Idea
          - button "New Defect" [ref=e167] [cursor=pointer]:
            - img [ref=e168]
            - text: New Defect
          - button "New Docs" [ref=e177] [cursor=pointer]:
            - img [ref=e178]
            - text: New Docs
        - generic [ref=e180]:
          - generic [ref=e182]:
            - img [ref=e183]
            - textbox "Filter artifacts by text" [ref=e186]:
              - /placeholder: Filter by text…
          - combobox [ref=e187] [cursor=pointer]:
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
          - combobox [ref=e188] [cursor=pointer]:
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
          - combobox [ref=e189] [cursor=pointer]:
            - option "All types" [selected]
            - option "idea"
            - option "requirement"
            - option "plan-backend"
            - option "plan-frontend"
            - option "plan-test"
            - option "test"
            - option "prototype"
            - option "defect"
          - combobox [ref=e190] [cursor=pointer]:
            - option "All labels" [selected]
            - option "defect"
          - generic [ref=e191]: Release
          - combobox "Release" [ref=e192] [cursor=pointer]:
            - option "All releases" [selected]
            - option "Unassigned"
          - button "Reset" [ref=e193] [cursor=pointer]
        - table [ref=e195]:
          - rowgroup [ref=e196]:
            - row "Path Stage Status Priority Release Type Runs Created Modified" [ref=e197]:
              - columnheader "Path" [ref=e198] [cursor=pointer]:
                - generic [ref=e199]:
                  - text: Path
                  - img [ref=e200]
              - columnheader "Stage" [ref=e203] [cursor=pointer]:
                - generic [ref=e204]:
                  - text: Stage
                  - img [ref=e205]
              - columnheader "Status" [ref=e208] [cursor=pointer]:
                - generic [ref=e209]:
                  - text: Status
                  - img [ref=e210]
              - columnheader "Priority" [ref=e213] [cursor=pointer]:
                - generic [ref=e214]:
                  - text: Priority
                  - img [ref=e215]
              - columnheader "Release" [ref=e218] [cursor=pointer]:
                - generic [ref=e219]:
                  - text: Release
                  - img [ref=e220]
              - columnheader "Type" [ref=e223] [cursor=pointer]:
                - generic [ref=e224]:
                  - text: Type
                  - img [ref=e225]
              - columnheader "Runs" [ref=e228] [cursor=pointer]:
                - generic [ref=e229]:
                  - text: Runs
                  - img [ref=e230]
              - columnheader "Created" [ref=e233] [cursor=pointer]:
                - generic [ref=e234]:
                  - text: Created
                  - img [ref=e235]
              - columnheader "Modified" [ref=e238] [cursor=pointer]:
                - generic [ref=e239]:
                  - text: Modified
                  - img [ref=e240]
          - rowgroup [ref=e243]:
            - row "RC Idea A lifecycle/ideas/rc-idea-a.md ideas draft — — idea 0 May 16, 2026 May 16, 2026" [ref=e244] [cursor=pointer]:
              - cell "RC Idea A lifecycle/ideas/rc-idea-a.md" [ref=e245]:
                - generic [ref=e247]: RC Idea A
                - generic [ref=e248]: lifecycle/ideas/rc-idea-a.md
              - cell "ideas" [ref=e249]
              - cell "draft" [ref=e250]:
                - generic [ref=e251]: draft
              - cell "—" [ref=e252]
              - cell "—" [ref=e253]
              - cell "idea" [ref=e254]
              - cell "0" [ref=e255]
              - cell "May 16, 2026" [ref=e256]
              - cell "May 16, 2026" [ref=e257]
            - row "RC Idea B lifecycle/ideas/rc-idea-b.md ideas draft — — idea 0 May 16, 2026 May 16, 2026" [ref=e258] [cursor=pointer]:
              - cell "RC Idea B lifecycle/ideas/rc-idea-b.md" [ref=e259]:
                - generic [ref=e261]: RC Idea B
                - generic [ref=e262]: lifecycle/ideas/rc-idea-b.md
              - cell "ideas" [ref=e263]
              - cell "draft" [ref=e264]:
                - generic [ref=e265]: draft
              - cell "—" [ref=e266]
              - cell "—" [ref=e267]
              - cell "idea" [ref=e268]
              - cell "0" [ref=e269]
              - cell "May 16, 2026" [ref=e270]
              - cell "May 16, 2026" [ref=e271]
            - row "RC Idea C lifecycle/ideas/rc-idea-c.md ideas draft — — idea 0 May 16, 2026 May 16, 2026" [ref=e272] [cursor=pointer]:
              - cell "RC Idea C lifecycle/ideas/rc-idea-c.md" [ref=e273]:
                - generic [ref=e275]: RC Idea C
                - generic [ref=e276]: lifecycle/ideas/rc-idea-c.md
              - cell "ideas" [ref=e277]
              - cell "draft" [ref=e278]:
                - generic [ref=e279]: draft
              - cell "—" [ref=e280]
              - cell "—" [ref=e281]
              - cell "idea" [ref=e282]
              - cell "0" [ref=e283]
              - cell "May 16, 2026" [ref=e284]
              - cell "May 16, 2026" [ref=e285]
            - row "RC Pill Target lifecycle/ideas/rc-pill.md ideas draft — — idea 0 May 16, 2026 May 16, 2026" [ref=e286] [cursor=pointer]:
              - cell "RC Pill Target lifecycle/ideas/rc-pill.md" [ref=e287]:
                - generic [ref=e289]: RC Pill Target
                - generic [ref=e290]: lifecycle/ideas/rc-pill.md
              - cell "ideas" [ref=e291]
              - cell "draft" [ref=e292]:
                - generic [ref=e293]: draft
              - cell "—" [ref=e294]
              - cell "—" [ref=e295]
              - cell "idea" [ref=e296]
              - cell "0" [ref=e297]
              - cell "May 16, 2026" [ref=e298]
              - cell "May 16, 2026" [ref=e299]
            - row "RC WebSocket Target lifecycle/ideas/rc-ws.md ideas draft — — idea 0 May 16, 2026 May 16, 2026" [ref=e300] [cursor=pointer]:
              - cell "RC WebSocket Target lifecycle/ideas/rc-ws.md" [ref=e301]:
                - generic [ref=e303]: RC WebSocket Target
                - generic [ref=e304]: lifecycle/ideas/rc-ws.md
              - cell "ideas" [ref=e305]
              - cell "draft" [ref=e306]:
                - generic [ref=e307]: draft
              - cell "—" [ref=e308]
              - cell "—" [ref=e309]
              - cell "idea" [ref=e310]
              - cell "0" [ref=e311]
              - cell "May 16, 2026" [ref=e312]
              - cell "May 16, 2026" [ref=e313]
            - row "Smoke Defect Alpha lifecycle/defects/smoke-defect-01.md defects draft — — defect 0 May 16, 2026 May 16, 2026" [ref=e314] [cursor=pointer]:
              - cell "Smoke Defect Alpha lifecycle/defects/smoke-defect-01.md" [ref=e315]:
                - generic [ref=e317]: Smoke Defect Alpha
                - generic [ref=e318]: lifecycle/defects/smoke-defect-01.md
              - cell "defects" [ref=e319]
              - cell "draft" [ref=e320]:
                - generic [ref=e321]: draft
              - cell "—" [ref=e322]
              - cell "—" [ref=e323]
              - cell "defect" [ref=e324]
              - cell "0" [ref=e325]
              - cell "May 16, 2026" [ref=e326]
              - cell "May 16, 2026" [ref=e327]
            - row "Smoke Doc Approved lifecycle/docs/smoke-doc-approved.md docs approved — — doc 0 May 16, 2026 May 16, 2026" [ref=e328] [cursor=pointer]:
              - cell "Smoke Doc Approved lifecycle/docs/smoke-doc-approved.md" [ref=e329]:
                - generic [ref=e331]: Smoke Doc Approved
                - generic [ref=e332]: lifecycle/docs/smoke-doc-approved.md
              - cell "docs" [ref=e333]
              - cell "approved" [ref=e334]:
                - generic [ref=e335]: approved
              - cell "—" [ref=e336]
              - cell "—" [ref=e337]
              - cell "doc" [ref=e338]
              - cell "0" [ref=e339]
              - cell "May 16, 2026" [ref=e340]
              - cell "May 16, 2026" [ref=e341]
            - row "Smoke Idea 01 lifecycle/ideas/smoke-idea-01.md ideas draft — — idea 0 May 16, 2026 May 16, 2026" [ref=e342] [cursor=pointer]:
              - cell "Smoke Idea 01 lifecycle/ideas/smoke-idea-01.md" [ref=e343]:
                - generic [ref=e345]: Smoke Idea 01
                - generic [ref=e346]: lifecycle/ideas/smoke-idea-01.md
              - cell "ideas" [ref=e347]
              - cell "draft" [ref=e348]:
                - generic [ref=e349]: draft
              - cell "—" [ref=e350]
              - cell "—" [ref=e351]
              - cell "idea" [ref=e352]
              - cell "0" [ref=e353]
              - cell "May 16, 2026" [ref=e354]
              - cell "May 16, 2026" [ref=e355]
            - row "Smoke Idea 02 lifecycle/ideas/smoke-idea-02.md ideas draft — — idea 0 May 16, 2026 May 16, 2026" [ref=e356] [cursor=pointer]:
              - cell "Smoke Idea 02 lifecycle/ideas/smoke-idea-02.md" [ref=e357]:
                - generic [ref=e359]: Smoke Idea 02
                - generic [ref=e360]: lifecycle/ideas/smoke-idea-02.md
              - cell "ideas" [ref=e361]
              - cell "draft" [ref=e362]:
                - generic [ref=e363]: draft
              - cell "—" [ref=e364]
              - cell "—" [ref=e365]
              - cell "idea" [ref=e366]
              - cell "0" [ref=e367]
              - cell "May 16, 2026" [ref=e368]
              - cell "May 16, 2026" [ref=e369]
            - row "Smoke Idea 03 lifecycle/ideas/smoke-idea-03.md ideas draft — — idea 0 May 16, 2026 May 16, 2026" [ref=e370] [cursor=pointer]:
              - cell "Smoke Idea 03 lifecycle/ideas/smoke-idea-03.md" [ref=e371]:
                - generic [ref=e373]: Smoke Idea 03
                - generic [ref=e374]: lifecycle/ideas/smoke-idea-03.md
              - cell "ideas" [ref=e375]
              - cell "draft" [ref=e376]:
                - generic [ref=e377]: draft
              - cell "—" [ref=e378]
              - cell "—" [ref=e379]
              - cell "idea" [ref=e380]
              - cell "0" [ref=e381]
              - cell "May 16, 2026" [ref=e382]
              - cell "May 16, 2026" [ref=e383]
            - row "Smoke Idea 04 lifecycle/ideas/smoke-idea-04.md ideas draft — — idea 0 May 16, 2026 May 16, 2026" [ref=e384] [cursor=pointer]:
              - cell "Smoke Idea 04 lifecycle/ideas/smoke-idea-04.md" [ref=e385]:
                - generic [ref=e387]: Smoke Idea 04
                - generic [ref=e388]: lifecycle/ideas/smoke-idea-04.md
              - cell "ideas" [ref=e389]
              - cell "draft" [ref=e390]:
                - generic [ref=e391]: draft
              - cell "—" [ref=e392]
              - cell "—" [ref=e393]
              - cell "idea" [ref=e394]
              - cell "0" [ref=e395]
              - cell "May 16, 2026" [ref=e396]
              - cell "May 16, 2026" [ref=e397]
            - row "Smoke Idea 05 lifecycle/ideas/smoke-idea-05.md ideas approved — — idea 0 May 16, 2026 May 16, 2026" [ref=e398] [cursor=pointer]:
              - cell "Smoke Idea 05 lifecycle/ideas/smoke-idea-05.md" [ref=e399]:
                - generic [ref=e401]: Smoke Idea 05
                - generic [ref=e402]: lifecycle/ideas/smoke-idea-05.md
              - cell "ideas" [ref=e403]
              - cell "approved" [ref=e404]:
                - generic [ref=e405]: approved
              - cell "—" [ref=e406]
              - cell "—" [ref=e407]
              - cell "idea" [ref=e408]
              - cell "0" [ref=e409]
              - cell "May 16, 2026" [ref=e410]
              - cell "May 16, 2026" [ref=e411]
            - row "Smoke Idea 06 lifecycle/ideas/smoke-idea-06.md ideas approved — — idea 0 May 16, 2026 May 16, 2026" [ref=e412] [cursor=pointer]:
              - cell "Smoke Idea 06 lifecycle/ideas/smoke-idea-06.md" [ref=e413]:
                - generic [ref=e415]: Smoke Idea 06
                - generic [ref=e416]: lifecycle/ideas/smoke-idea-06.md
              - cell "ideas" [ref=e417]
              - cell "approved" [ref=e418]:
                - generic [ref=e419]: approved
              - cell "—" [ref=e420]
              - cell "—" [ref=e421]
              - cell "idea" [ref=e422]
              - cell "0" [ref=e423]
              - cell "May 16, 2026" [ref=e424]
              - cell "May 16, 2026" [ref=e425]
            - row "Smoke Idea 07 lifecycle/ideas/smoke-idea-07.md ideas approved — — idea 0 May 16, 2026 May 16, 2026" [ref=e426] [cursor=pointer]:
              - cell "Smoke Idea 07 lifecycle/ideas/smoke-idea-07.md" [ref=e427]:
                - generic [ref=e429]: Smoke Idea 07
                - generic [ref=e430]: lifecycle/ideas/smoke-idea-07.md
              - cell "ideas" [ref=e431]
              - cell "approved" [ref=e432]:
                - generic [ref=e433]: approved
              - cell "—" [ref=e434]
              - cell "—" [ref=e435]
              - cell "idea" [ref=e436]
              - cell "0" [ref=e437]
              - cell "May 16, 2026" [ref=e438]
              - cell "May 16, 2026" [ref=e439]
            - row "Smoke Requirement Alpha — Documentation lifecycle/docs/smoke-doc-linked.md docs draft — — doc 0 May 16, 2026 May 16, 2026" [ref=e440] [cursor=pointer]:
              - cell "Smoke Requirement Alpha — Documentation lifecycle/docs/smoke-doc-linked.md" [ref=e441]:
                - generic [ref=e443]: Smoke Requirement Alpha — Documentation
                - generic [ref=e444]: lifecycle/docs/smoke-doc-linked.md
              - cell "docs" [ref=e445]
              - cell "draft" [ref=e446]:
                - generic [ref=e447]: draft
              - cell "—" [ref=e448]
              - cell "—" [ref=e449]
              - cell "doc" [ref=e450]
              - cell "0" [ref=e451]
              - cell "May 16, 2026" [ref=e452]
              - cell "May 16, 2026" [ref=e453]
            - row "Smoke Requirement Alpha lifecycle/requirements/smoke-req-01.md requirements draft — — requirement 0 May 16, 2026 May 16, 2026" [ref=e454] [cursor=pointer]:
              - cell "Smoke Requirement Alpha lifecycle/requirements/smoke-req-01.md" [ref=e455]:
                - generic [ref=e457]: Smoke Requirement Alpha
                - generic [ref=e458]: lifecycle/requirements/smoke-req-01.md
              - cell "requirements" [ref=e459]
              - cell "draft" [ref=e460]:
                - generic [ref=e461]: draft
              - cell "—" [ref=e462]
              - cell "—" [ref=e463]
              - cell "requirement" [ref=e464]
              - cell "0" [ref=e465]
              - cell "May 16, 2026" [ref=e466]
              - cell "May 16, 2026" [ref=e467]
            - row "Smoke Requirement Beta lifecycle/requirements/smoke-req-02.md requirements planning — — requirement 0 May 16, 2026 May 16, 2026" [ref=e468] [cursor=pointer]:
              - cell "Smoke Requirement Beta lifecycle/requirements/smoke-req-02.md" [ref=e469]:
                - generic [ref=e471]: Smoke Requirement Beta
                - generic [ref=e472]: lifecycle/requirements/smoke-req-02.md
              - cell "requirements" [ref=e473]
              - cell "planning" [ref=e474]:
                - generic [ref=e475]: planning
              - cell "—" [ref=e476]
              - cell "—" [ref=e477]
              - cell "requirement" [ref=e478]
              - cell "0" [ref=e479]
              - cell "May 16, 2026" [ref=e480]
              - cell "May 16, 2026" [ref=e481]
            - row "Smoke Requirement Gamma lifecycle/requirements/smoke-req-03.md requirements in-development — — requirement 0 May 16, 2026 May 16, 2026" [ref=e482] [cursor=pointer]:
              - cell "Smoke Requirement Gamma lifecycle/requirements/smoke-req-03.md" [ref=e483]:
                - generic [ref=e485]: Smoke Requirement Gamma
                - generic [ref=e486]: lifecycle/requirements/smoke-req-03.md
              - cell "requirements" [ref=e487]
              - cell "in-development" [ref=e488]:
                - generic [ref=e489]: in-development
              - cell "—" [ref=e490]
              - cell "—" [ref=e491]
              - cell "requirement" [ref=e492]
              - cell "0" [ref=e493]
              - cell "May 16, 2026" [ref=e494]
              - cell "May 16, 2026" [ref=e495]
        - navigation "Table pagination" [ref=e496]:
          - generic [ref=e497]:
            - generic [ref=e498]: Rows per page
            - combobox "Rows per page" [ref=e499] [cursor=pointer]:
              - option "10"
              - option "25" [selected]
              - option "50"
              - option "100"
          - generic [ref=e500]: Showing 1–18 of 18
          - generic [ref=e501]:
            - button "Previous page" [disabled] [ref=e502]: ← Prev
            - generic [ref=e503]: Page
            - spinbutton "Jump to page" [disabled] [ref=e504]: "1"
            - generic [ref=e505]: of 1
            - button "Next page" [disabled] [ref=e506]: Next →
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