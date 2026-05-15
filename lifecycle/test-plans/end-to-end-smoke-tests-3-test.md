---
title: End-to-end smoke tests for core flows — Test Plan
type: plan-test
status: done
lineage: end-to-end-smoke-tests
priority: normal
parent: lifecycle/requirements/end-to-end-smoke-tests-2.md
labels:
    - testing
    - qa
    - feature
release: KC-Release2
---

# End-to-end smoke tests for core flows — Test Plan

This is a single test plan rather than the usual backend / frontend / test
trio because the work is almost entirely test-infrastructure. No production
backend or frontend changes are required to deliver F1–F4 of the
requirement; the only production touch is wiring `make test-e2e` and adding
the step to `lifecycle/devops/test.yaml`. Both belong with the test work.

## Architecture

```
tests/e2e/
├── package.json              # @playwright/test + small deps
├── playwright.config.ts      # workers, reporter, base setup
├── tsconfig.json             # node 20+ target
├── README.md                 # how to run / add flows
├── fixtures/
│   ├── lifecycle/            # seeded markdown artifacts
│   │   ├── config.yaml       # roles, agents (with shell-stub driver), gates
│   │   ├── ideas/*.md
│   │   ├── requirements/*.md
│   │   ├── defects/*.md
│   │   └── releases/         # release records seeded via API in setup
│   └── seed-helpers.ts       # programmatic seeding utilities
├── harness/
│   ├── kaos-control.ts       # spawn + ready + teardown
│   ├── auth.ts               # bootstrap first user, login helper
│   └── ws.ts                 # subscribe to project WS, collect events
└── flows/
    ├── 01-login.spec.ts
    ├── 02-edit-save.spec.ts
    ├── 03-transition.spec.ts
    ├── 04-agent-run.spec.ts
    └── 05-graph-click.spec.ts
```

## Milestone 1 — Test framework + harness

**Description.** Stand up Playwright in `tests/e2e/`, write the binary
spawn / teardown harness, prove that a test can boot a fresh kaos-control
instance, hit `/api/health`, and shut it down cleanly.

**Files to create.**

- `tests/e2e/package.json`:
  ```json
  {
    "name": "kaos-control-e2e",
    "private": true,
    "type": "module",
    "scripts": {
      "test": "playwright test",
      "test:ui": "playwright test --ui",
      "test:debug": "PWDEBUG=1 playwright test"
    },
    "devDependencies": {
      "@playwright/test": "^1.50.0",
      "typescript": "^5.4.0"
    }
  }
  ```
- `tests/e2e/playwright.config.ts`: workers=4, reporter=list+html,
  headless by default, retries=0 (deterministic per NF2).
- `tests/e2e/tsconfig.json`: target ES2022, module ESNext, strict.
- `tests/e2e/harness/kaos-control.ts`: exports `spawnKaosControl()`
  returning `{ baseURL, kctestDir, projectRoot, kill }`. Steps:
  1. Resolve `repoRoot/dist/kaos-control`. If missing or older than
     `repoRoot/internal` mtime, run `make build` synchronously.
  2. `mkdtemp` two temp dirs: one for `KAOS_CONTROL_HOME`, one for the
     project root.
  3. Copy `tests/e2e/fixtures/lifecycle/` into the project root as a
     bare git repo (init + initial commit).
  4. Find a free port via `net.createServer().listen(0)`.
  5. Write `<KAOS_CONTROL_HOME>/config.yaml` with `server.listen: :PORT`,
     `data_dir`, `projects_dir`.
  6. Write `<KAOS_CONTROL_HOME>/projects/testproject.yaml` pointing
     at the temp project root.
  7. Spawn `dist/kaos-control` with `KAOS_CONTROL_HOME` env. Pipe
     stdout / stderr to per-test buffers (printed only on failure).
  8. Poll `/api/health` until 200 or 10 s timeout.
  9. Return the descriptor; `kill()` sends SIGTERM, awaits with a
     5 s deadline before SIGKILL, then `rm -rf` the temp dirs.

**Acceptance.**

- A trivial `flows/00-harness-smoke.spec.ts` (kept only for M1):
  spawn, GET `/api/health`, expect 200, kill. Passes in < 5 s on a
  laptop.
- Running it twice in parallel (`workers: 2`) shows two distinct
  random ports in the logs and both succeed.
- Killing the test with Ctrl-C leaves no orphaned `kaos-control`
  processes in `ps` and no temp dirs in `/tmp`.

## Milestone 2 — Auth + project access fixtures

**Description.** Wire the auth-bootstrap step and a Playwright
`test.beforeAll` that logs the test browser session in. Without this
every flow would have to repeat the login dance.

**Files.**

- `tests/e2e/harness/auth.ts`: `bootstrapUser({ email, password,
  baseURL })` → POSTs to `/api/admin/users` and returns the auth-store
  user id; `loginPage(page, { email, password })` drives the SPA login
  form to acquire session cookies in the browser context.
- A Playwright fixture at `tests/e2e/fixtures.ts` that exposes
  `kctest`, `loggedInPage` (browser context with valid cookies for the
  seeded admin user). Tests opt in via `test.extend(...)`.

**Acceptance.**

- A new `flows/01-login.spec.ts` (the F3 flow #1) uses `loggedInPage`
  to land on `/p/testproject/dashboard`; the seeded "Lifecycle Total"
  card reads a non-zero value.
- Without `loggedInPage`, navigating to `/p/testproject/dashboard`
  redirects to `/login`. Test asserts both behaviours.

## Milestone 3 — Edit, save, and transition flows

**Description.** Implement F3 flows #2 (edit & save) and #3 (transition).
These are the highest-signal smoke flows because they exercise file
write, re-index, WebSocket fan-out, and git commit in one path.

**Files.**

- `tests/e2e/flows/02-edit-save.spec.ts`:
  1. Navigate to a seeded artifact.
  2. Acquire its current body via `expect(page.locator('.cm-content'))`
     (CodeMirror 6).
  3. Type a deterministic change (e.g. append "smoke-test-marker").
  4. Click Save. Wait for the save-success toast.
  5. Read the file from disk via `fs.promises.readFile` against
     `kctest.projectRoot`. Assert it contains the marker.
  6. Subscribe to the project WS before the save; assert a
     `file.changed` event for the artifact path arrived within 2 s of
     the save response.
- `tests/e2e/flows/03-transition.spec.ts`:
  1. Navigate to a seeded `requirement` artifact whose status is
     `draft`. Click Change Status, pick `clarifying`.
  2. Wait for the response, assert HTTP 200 in the network log.
  3. Read the file: frontmatter `status: clarifying`.
  4. Run `git log --oneline -1` on the project repo (via Node `child_process`).
     Assert the most recent commit subject matches
     `transition(<lineage>): draft → clarifying`.

**Acceptance.**

- Both flows pass on a clean run. Running with `--repeat-each=10`
  passes 10/10 — flake budget is zero.
- Reverting [14f4c89](https://github.com/) (the `total_tickets` rename)
  on a branch makes flow #2's dashboard-stats subassertion fail
  with a clear message — proves the suite catches contract drift.
  (No need to actually do this; it's a verification step the
  implementer performs once before merge.)

## Milestone 4 — Agent run + graph click

**Description.** Implement F3 flows #4 and #5. Flow #4 needs a stubbed
agent driver to avoid requiring Claude Code on PATH.

**Files.**

- Update `tests/e2e/fixtures/lifecycle/config.yaml`: add an agent
  whose `driver: shell-stub` runs an `echo` script that prints stream-
  json events to stdout and exits 0 after a configurable delay. The
  driver type either reuses an existing one (search the agent package)
  or, if not present, gets a small addition under `internal/agent/` —
  in which case this milestone gains a `Files to change` row for the
  Go side.
- `tests/e2e/flows/04-agent-run.spec.ts`:
  1. Navigate to `/p/testproject/agents`. Click Run for the
     stub agent. Confirm in the run dialog.
  2. Subscribe to WS; assert `agent.started` arrives within 2 s.
  3. Assert the run row appears in the runs list with status
     `running`. Wait for the stub to finish (5 s timeout). Assert
     final status is `done`.
- `tests/e2e/flows/05-graph-click.spec.ts`:
  1. Navigate to `/p/testproject/graph?layout=fcose`. Wait for the
     Cytoscape root to render. Assert `node.length` matches the seed
     count plus the synthetic Backlog/Unscheduled nodes (per
     [internal/http/releases.go](internal/http/releases.go#L361)).
  2. Click on a known seeded artifact's node (Cytoscape exposes
     positions; use `page.locator('canvas').click({ position: ... })`
     after computing the node's pixel coordinates from the layout
     event).
  3. Assert URL becomes `/p/testproject/artifacts/<expected-path>`.

**Acceptance.**

- Flow #4 passes without Claude Code installed.
- Flow #5 passes against the seeded fixture's actual node count.
- All five flows together run in under 30 s on a laptop (NF1).

## Milestone 5 — Make targets, DevOps pipeline, README

**Description.** Wire the suite into the project's normal build / test
infrastructure.

**Files to change.**

- `Makefile`: new `test-e2e` target that does:
  ```make
  test-e2e: build
  	cd tests/e2e && pnpm install && pnpm test
  ```
  And new `test-all`:
  ```make
  test-all: test-unit test-integration test-e2e
  	cd tests/web && pnpm test
  ```
- `lifecycle/devops/test.yaml`: append a step:
  ```yaml
  - name: E2E smoke tests
    description: Playwright flows against a fresh ./dist/kaos-control binary
    command: make test-e2e
    timeout: 5m
  ```
- `tests/e2e/README.md`:
  - One-line summary.
  - How to run: `make test-e2e`, `pnpm --dir tests/e2e test:ui`,
    `pnpm --dir tests/e2e test:debug`.
  - How to add a flow: create `flows/NN-name.spec.ts`, use the
    `loggedInPage` fixture, follow the existing pattern.
  - How to debug a failing flow: read the trace at
    `playwright-report/index.html`, open the trace viewer with
    `pnpm exec playwright show-trace <zip>`.
  - Where the harness logs go on failure (per NF3).

**Acceptance.**

- `make test-e2e` passes on a clean checkout in under 60 s
  (binary already built; otherwise add ~5 s for the build).
- `make test-all` passes end-to-end.
- The DevOps pipeline run in the UI shows the E2E step succeed.
- Following the README from cold, a new contributor can add a sixth
  flow in under 30 minutes (validate by walking through the steps with
  one).

## Milestone 6 — Verification

**Description.** Final round to prove the suite catches the class of
bug the requirement was written for.

**Steps.**

1. Branch off `main`. On the branch, revert one of the recent contract
   fixes — e.g. set `web/src/components/dashboard/widgets/SummaryCountsWidget.vue`
   back to reading `stats.total`.
2. Run `make test-e2e`. Expect flow #1 (login + project picker, which
   asserts the dashboard renders a non-zero Lifecycle Total) to fail.
3. Run again with `--reporter=list --workers=1 -- --grep '01-'` to
   confirm the failure is in the right flow with a clear assertion
   message.
4. Discard the branch. Document the verification in the merge PR.

**Acceptance.**

- The E2E suite caught at least one historical contract-drift bug
  during this milestone's verification (proves the smoke flows are
  load-bearing, not just busy-work).
- Merge PR's description includes the exact reproduction the verifier
  used.

## Risks and mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Cytoscape node-click coordinates are flaky | Medium | Use Cytoscape's exposed JS API via `page.evaluate(() => cy.elements()...)` to programmatically click rather than mouse coordinates |
| Spawning the binary on macOS triggers a Gatekeeper prompt on first run | Low | Document in README that `dist/kaos-control` should be `chmod +x` and ideally run once manually first; not a blocker |
| `make build` race when several test files all try to build simultaneously | Medium | Make the binary build idempotent (check mtime against `internal/`) and serialise via a file lock in the harness |
| Playwright install adds ~250 MB to dev environments | Low | This is the cost of E2E testing; document, accept |
| Future Claude Code CLI availability changes break flow #4 | Low–Medium | The `shell-stub` driver has no external dep; flow #4 stays green if Claude Code disappears from the test fixture |

## Test data

A small, deterministic seed is committed under `tests/e2e/fixtures/lifecycle/`:

- 10 ideas (mixed statuses: 4 draft, 3 approved, 2 done, 1 abandoned)
- 3 requirements (draft, planning, in-development)
- 1 defect
- 1 release record (KC-E2E-Test, dated 2026-01-01 → 2026-01-31)

Pinning specific values keeps assertions in the flows readable
(`expect(card).toHaveText('14')` instead of `toBeGreaterThan(0)`).

## How this fits

After this lineage lands, the project's test pyramid is:

| Layer | Tests | Where | What it catches |
|---|---|---|---|
| Component | ~1000 | `tests/web/` (Vitest + happy-dom) | Component logic, mock-shape contracts |
| Integration | ~500 | `tests/integration/` (Go + `integration` build tag) | Backend handler shape, persistence, indexing, transitions |
| **Smoke (this work)** | **5** | **`tests/e2e/` (Playwright + real binary)** | **Frontend ↔ backend contract drift, SPA wiring, WS, auth, git side-effects** |

5 flows is the target ceiling. If a sixth flow seems essential, it's
probably regression coverage that belongs in the component or
integration layer.
