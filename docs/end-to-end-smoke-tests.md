# End-to-End Smoke Tests

Playwright-based smoke tests that drive the full Innovation Maker stack — Go binary, Vue SPA, WebSocket, and authentication — together in a real browser.

---

## Overview

Innovation Maker has two distinct test layers that serve different purposes and should not be conflated:

| Layer | Location | Runner | Purpose |
|-------|----------|--------|---------|
| Component tests | `tests/web/` | Vitest + happy-dom | Verify individual Vue components in isolation |
| E2E smoke tests | `tests/e2e/` | Playwright | Prove the full system is wired up end-to-end |

**Component tests** (≈337 tests) mount Vue components using `@vue/test-utils` against a synthetic happy-dom environment. They run fast, need no running server, and are the right home for UI logic, computed properties, emitted events, and prop contracts.

**E2E smoke tests** are different in kind. Each test spawns a real `kaos-control` binary, seeds a git-backed project with fixture data, opens a real browser (Chromium headless), and drives actual user flows. The goal is not regression coverage of every edge case — that is the component tests' job. Smoke tests prove that the entire system is wired up: HTTP auth, cookie sessions, the WebSocket hub, the file-watcher re-indexer, the artifact editor, the agent runner, and the graph view all work together with a real database and real disk I/O.

> **Historical note:** An earlier entry in `plans/PROJECT_PLAN.md` listed "Playwright or Vitest browser-mode smoke tests for core flows" as a single post-M6 item. In practice, the Vitest component suite was completed first. Smoke tests are tracked separately and are not yet wired into the default CI job — see [CI Integration](#ci-integration).

---

## Why Playwright

The core flows that matter for smoke testing span an out-of-process Go backend. Editing an artifact means a `PUT /api/p/:project/artifacts/*` HTTP round-trip that writes to disk; the file-watcher detects the change and fires a `file.changed` WebSocket event back to the browser. Transitioning an artifact status triggers `POST …/transition`, updates YAML frontmatter on disk, and creates a git commit. Starting an agent run involves HTTP + WebSocket streaming.

None of this is testable from within Vitest's happy-dom environment, which has no concept of a running Go process, no real network, and no real filesystem. A Vitest browser-mode run would still need the binary running externally and a test helper to manage its lifecycle — at that point the convenience gap over Playwright disappears.

Playwright provides:

- A real browser (Chromium) with real network and cookies.
- A fixture system (`test.extend`) for per-test server lifecycle management.
- `waitForURL`, `waitForResponse`, and `page.waitForFunction` for reliable async assertions without arbitrary sleeps.
- Built-in trace recording and an HTML reporter for post-mortem debugging.
- `PWDEBUG=1` step-through mode and `test:ui` interactive runner.

Playwright is isolated from the Vitest component test stack. `tests/e2e/` has its own `package.json` and `node_modules`; the two suites do not share configuration, dependencies, or globals.

---

## Test Environment Setup

### What runs

Each E2E test spawns an isolated instance of the compiled `./dist/kaos-control` binary. The binary is built by `make test-e2e` before Playwright starts. If you invoke Playwright directly (e.g., `pnpm --dir tests/e2e test:ui`), the harness checks for the binary at startup and calls `make build` automatically if it is absent.

No external services are required. No existing `~/.kaos-control` config is used. Each test gets its own temporary home directory and project root, so tests are completely isolated from the developer's local environment and from each other.

### How the harness bootstraps a server

The harness (`tests/e2e/harness/kaos-control.ts`) performs the following steps for every test worker:

1. **Locate the binary** — walks up from the harness file to find `go.mod`, then resolves `dist/kaos-control` relative to the repo root.

2. **Create temp directories** — two `os.tmpdir()`-prefixed directories:
   - `kc-home-XXXX` — stands in for `~/.kaos-control`
   - `kc-proj-XXXX` — the project root (the directory that contains `lifecycle/`)

3. **Seed fixture data** — copies `tests/e2e/fixtures/lifecycle/` into `kc-proj-XXXX/lifecycle/` recursively. The standard seed contains 14 artifacts: 10 ideas, 3 requirements, and 1 defect.

4. **Initialise git** — runs `git init -b main && git add -A && git commit -m "Initial fixture commit"` in the project root with a deterministic `Test Harness <test@kaos-e2e.local>` identity. The binary requires a git repo to create transition commits.

5. **Find a free port** — binds a TCP server on `127.0.0.1:0`, reads the OS-assigned port, then closes the server before handing the port to the binary.

6. **Write app config** — creates `kc-home-XXXX/config.yaml`:

   ```yaml
   server:
     listen: "127.0.0.1:<PORT>"
   auth:
     method: local
     session_ttl: 24h
   projects_dir: /tmp/kc-home-XXXX/projects
   data_dir: /tmp/kc-home-XXXX/data
   ```

7. **Register the test project** — writes `kc-home-XXXX/projects/testproject.yaml`:

   ```yaml
   name: testproject
   path: /tmp/kc-proj-XXXX
   owner: admin@kaos-e2e.local
   description: E2E smoke test project
   ```

8. **Spawn the binary** — launches `./dist/kaos-control -config <configPath>` with `LOG_LEVEL=warn` to suppress noise.

9. **Wait for health** — polls `GET /api/health` every 200 ms up to a 10-second timeout. A startup failure dumps the binary's captured stdout/stderr.

10. **Bootstrap the admin user** — calls the unauthenticated `POST /api/admin/users` endpoint to create `admin@kaos-e2e.local / TestPassword123!`.

### Fixture project config

The fixture `lifecycle/config.yaml` (at `tests/e2e/fixtures/lifecycle/config.yaml`) is a minimal project configuration that:

- Defines a single `stub-agent` using the `shell-stub` driver: `sleep 1 && printf '{"type":"result","subtype":"success","is_error":false}\n'`
- Tracks `idea`, `requirement`, and `defect` types on the dashboard.
- Defines roles: `product-owner`, `analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`.
- Sets `required_plans.requirement: [plan-backend, plan-frontend, plan-test]` to enable plan-gating.

The `stub-agent` is the key to the agent-run smoke test: it exercises the full run lifecycle (dialog → HTTP start → WebSocket events → terminal state) without requiring a Claude Code installation or API key.

### Environment variables

No environment variables are required beyond a working Node.js/pnpm install and a Go toolchain. The binary is controlled entirely through its `-config` flag. `LOG_LEVEL=warn` is set by the harness to keep test output readable.

### Cleanup

After each test, the harness sends `SIGTERM` to the binary and waits up to 5 seconds for a clean exit before escalating to `SIGKILL`. Both temp directories are then deleted. This runs in a Playwright worker-scoped fixture, so a single worker shares one server instance across all tests it runs.

---

## Covered Flows

The six flows live in `tests/e2e/flows/` and are numbered for stable ordering.

### Flow 00 — Harness smoke (`00-harness-smoke.spec.ts`)

**What it proves:** The harness itself can spawn and kill a server.

This minimal flow calls `spawnKaosControl()` directly (bypassing the fixture), hits `GET /api/health`, asserts HTTP 200, and calls `kill()`. It is a sanity check that the harness works before any browser is involved.

---

### Flow 01 — Login and project access (`01-login.spec.ts`)

**What it proves:** Auth redirect, session cookie, project data loading.

Two tests:

1. **Unauthenticated redirect** — navigates to `/p/testproject/dashboard` without a session and asserts the URL changes to `/login`.

2. **Authenticated dashboard** — uses the `loggedInPage` fixture (which drives the login form and acquires a session cookie) and navigates to the dashboard. Waits for the `SummaryCountsWidget` to render and asserts that the "Lifecycle Total" stat card shows a non-zero count, confirming that the 14 seed artifacts were indexed and served via the API.

**Auth flow detail:** `loginPage()` in `tests/e2e/harness/auth.ts` navigates to `/login`, fills the `#email` and `#password` inputs, submits the form, and waits for the URL to leave `/login`. The resulting session cookie is stored in Playwright's browser context and sent automatically on all subsequent requests.

---

### Flow 02 — Edit and save artifact (`02-edit-save.spec.ts`)

**What it proves:** Artifact editor writes to disk; file-watcher fires `file.changed` over WebSocket.

Steps:

1. Opens a WebSocket connection to `/api/p/testproject/ws` before navigating, so no events are missed.
2. Navigates to the artifact editor for `lifecycle/requirements/smoke-req-01.md`.
3. Waits for the CodeMirror editor (`.cm-content`) to be ready.
4. Appends a timestamped smoke marker (`smoke-test-marker-<timestamp>`) via keyboard (`Control+End` then `keyboard.type`).
5. Clicks the primary "Save" button.
6. Waits for a `.toast-message` containing "Saved".
7. Reads `smoke-req-01.md` directly from `kctest.projectRoot` on disk and asserts the marker is present — confirming the `PUT /api/p/testproject/artifacts/…` handler wrote the file.
8. Awaits the `file.changed` WebSocket event from the watcher.

---

### Flow 03 — Status transition (`03-transition.spec.ts`)

**What it proves:** Role-gated status transition updates frontmatter on disk and creates a git commit.

Steps:

1. Navigates to `lifecycle/requirements/smoke-req-01.md` in read mode.
2. Waits for the status badge (`.status-badge, [data-status]`) to become visible.
3. Intercepts the `POST …/transition` HTTP response via `page.waitForResponse`.
4. Clicks the interactive status badge to open the transition dropdown.
5. Selects the "clarifying" option from the `[role="listbox"]` or `.status-menu`.
6. Asserts the HTTP response status is 200.
7. Reads the file from disk and asserts `status: clarifying` in the frontmatter.
8. Runs `git log --oneline -1` in the project root and asserts the commit message matches `transition(smoke-req-01)` and includes `clarifying`.

The fixture seed sets `smoke-req-01.md` to `status: draft`. The transition to `clarifying` is a valid permitted transition from `draft`.

---

### Flow 04 — Agent run (`04-agent-run.spec.ts`)

**What it proves:** Agent run dialog → HTTP start → `agent.started` WebSocket event → run reaches `done` status.

Steps:

1. Opens a WebSocket connection to `/api/p/testproject/ws` and begins listening for `agent.started`.
2. Navigates to `/p/testproject/agents`.
3. Waits for `stub-agent` to appear in the agent list.
4. Clicks "Run Agent" to open the run dialog.
5. Selects the `stub-agent` chip from the agent list in the dialog.
6. Fills in `lifecycle/requirements/smoke-req-01.md` as the target path.
7. Clicks "Run".
8. Awaits the `agent.started` WebSocket event and captures the `run_id` from its payload.
9. Polls `GET /api/p/testproject/agents/runs/<run_id>` (with the browser session cookie) every 500 ms up to 10 seconds until `status` is `done` or `failed`.
10. Asserts `finalStatus === 'done'`.

The stub agent (`shell-stub` driver) sleeps 1 second then emits a success result — it is fast and deterministic. No Claude Code installation or Anthropic API key is needed.

---

### Flow 05 — Graph node click (`05-graph-click.spec.ts`)

**What it proves:** The map view renders the Cytoscape graph with real artifact data; clicking a node navigates to the artifact editor.

Steps:

1. Navigates to `/p/testproject/map`.
2. Waits for a `<canvas>` element to be visible (Cytoscape renders into canvas; timeout 15s).
3. Uses `page.waitForFunction` to verify `window.__cy` is exposed and that at least one node has a non-zero layout position (i.e., the fcose layout has finished).
4. Asserts `cy.nodes().length > 0`.
5. Uses `page.evaluate` to find the node whose `_raw.path` or `data('id')` matches `lifecycle/requirements/smoke-req-01.md` and calls `node.trigger('tap')`.
6. Waits for the URL to contain `smoke-req-01` (the node click triggers a Vue Router navigation to the artifact editor).
7. Asserts the final URL contains `smoke-req-01`.

The map view exposes `window.__cy` (the Cytoscape instance) for testability. The fixture seed provides 14 artifacts so the graph is non-trivial.

---

## Running the Tests

### Standard run

```sh
make test-e2e
```

This is the canonical way to run the full smoke suite. The `test-e2e` Make target:

1. Compiles the Go binary (`make build`) — embeds the Vue SPA from `web/dist/` into the binary.
2. Runs `cd tests/e2e && pnpm install && pnpm test`.

The binary must be built with a current `web/dist/`. If you have been running `make run` (which uses `go run` and serves `web/dist/` from disk), the binary in `./dist/` may be stale or absent — `make test-e2e` handles this.

### Interactive UI (Playwright Test Runner)

```sh
pnpm --dir tests/e2e test:ui
```

Opens the Playwright interactive UI. Lets you select individual flows, step through actions, and inspect snapshots. The harness detects that the binary is absent and builds it automatically.

### Step-through debugger

```sh
pnpm --dir tests/e2e test:debug
```

Sets `PWDEBUG=1`, which pauses the browser before each action and opens Playwright Inspector. Useful for tracing assertions on selectors that are not matching.

### Running a single flow

```sh
pnpm --dir tests/e2e test flows/02-edit-save.spec.ts
```

Or by test title grep:

```sh
pnpm --dir tests/e2e test --grep "saves content to disk"
```

### Reading results

The HTML report is written to `tests/e2e/playwright-report/index.html`. Open it in a browser for a per-test timeline with trace zip links. On a local run the list reporter outputs pass/fail to the terminal.

On failure, the harness captures the binary's stdout and stderr and includes them in the test error. Look for lines like:

```
Server startup failed:
stderr: 2026/05/16 10:00:00 ERROR config: …
```

To inspect a saved trace:

```sh
pnpm --dir tests/e2e exec playwright show-trace tests/e2e/playwright-report/data/<trace>.zip
```

### Parallelism

`playwright.config.ts` sets `workers: 4`. Each worker owns one `kctest` server instance (worker-scoped fixture). With 6 flows, the suite typically finishes in under 30 seconds on a developer machine.

---

## CI Integration

The smoke tests are **not** wired into the default CI job that runs on every pull request. They are available via `make test-e2e` and are the intended gate for releases and manual pre-merge checks.

### Why they are gated

- They require a compiled Go binary and a Playwright browser download.
- Each run takes 15–30 seconds wall-clock time (binary startup, real browser, filesystem I/O).
- The default CI job is optimised for fast feedback: unit tests, integration tests, and frontend Vitest run in parallel in under two minutes.

### Running them in CI

Add a separate CI job (e.g., a GitHub Actions `workflow_dispatch` or a scheduled nightly job):

```yaml
jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - name: Install pnpm
        run: npm install -g pnpm
      - name: Install Playwright browsers
        run: pnpm --dir tests/e2e install && pnpm --dir tests/e2e exec playwright install chromium
      - name: Build web assets
        run: make build-web
      - name: Run smoke tests
        run: make test-e2e
      - uses: actions/upload-artifact@v4
        if: failure()
        with:
          name: playwright-report
          path: tests/e2e/playwright-report/
```

### Playwright browser caching

Playwright downloads browser binaries to a platform-specific cache directory (e.g., `~/.cache/ms-playwright` on Linux). In CI, cache this directory keyed on the Playwright version from `tests/e2e/package.json`:

```yaml
- uses: actions/cache@v4
  with:
    path: ~/.cache/ms-playwright
    key: playwright-${{ hashFiles('tests/e2e/package.json') }}
```

Only Chromium is needed — Playwright defaults to testing against Chromium unless additional browsers are configured in `playwright.config.ts`. The `firefox` and `webkit` entries are absent from the config by design; add them only if cross-browser coverage is required.

---

## Extending the Suite

### Adding a new flow

1. Choose the next sequence number. If the last flow is `05-graph-click.spec.ts`, name yours `06-my-flow.spec.ts`.
2. Import from `../fixtures.js`:

   ```typescript
   import { test, expect } from '../fixtures.js'
   ```

3. Use the `loggedInPage` fixture for any flow that requires an authenticated session. Use `kctest` directly for flows that test auth itself (e.g., verifying a 401 redirect).

   ```typescript
   test.describe('Flow 06 — My new flow', () => {
     test('does something meaningful end-to-end', async ({ kctest, loggedInPage: page }) => {
       await page.goto(`${kctest.baseURL}/p/testproject/dashboard`)
       // assertions...
     })
   })
   ```

4. Keep flows independent. Each test spawns its own server and gets a fresh copy of the fixture data. Do not write a flow that depends on state left by a previous flow.

### Naming conventions

| Convention | Example |
|------------|---------|
| File name | `NN-short-description.spec.ts` (NN = zero-padded integer) |
| `test.describe` | `'Flow NN — Human-readable description'` |
| Test title | Active verb phrase: `'saves content to disk and fires file.changed WS event'` |

### Adding fixture data

Fixture files live in `tests/e2e/fixtures/lifecycle/`. They are plain markdown files with the standard kaos-control frontmatter. The harness copies this entire directory into a fresh temp project for every test worker.

If your new flow needs a specific artifact type or status:

1. Add the artifact file to the appropriate subdirectory (e.g., `fixtures/lifecycle/backend-plans/smoke-plan-01.md`).
2. Ensure `fixtures/lifecycle/config.yaml` includes the artifact's `stage` dir in the `stages` list if it is a new stage.
3. Update the node-count assertion in `05-graph-click.spec.ts` if it becomes sensitive to the total fixture count.

Keep fixtures minimal and deterministic. Avoid using `Date.now()` or any runtime-generated values in fixture frontmatter — content must be stable across runs.

### Asserting on WebSocket events

Use the `connectProjectWs` helper from `tests/e2e/harness/ws.ts`:

```typescript
import { connectProjectWs } from '../harness/ws.js'

test('receives artifact.indexed after save', async ({ kctest, loggedInPage: page }) => {
  const ws = connectProjectWs(kctest.baseURL, 'testproject')

  // ... perform the action that triggers the event ...

  const event = await ws.waitFor('artifact.indexed', 5_000)
  expect(event.payload).toMatchObject({ path: 'lifecycle/requirements/smoke-req-01.md' })
  ws.close()
})
```

Key points:

- Connect **before** performing the action, not after. The WebSocket connection takes a moment to establish, and events are emitted as soon as the backend processes the request.
- `waitFor` checks already-received events first, so it is safe to call even if the event arrived while you were setting up.
- Always call `ws.close()` at the end of the test to prevent open handle warnings.
- The default timeout is 5 seconds. For actions that involve disk I/O or agent runs, pass a longer timeout (8 000–10 000 ms).

### When to write a smoke test vs a component test

Write a **smoke test** when:

- The flow spans the HTTP boundary (browser → Go backend).
- Correctness depends on real disk writes, git commits, or WebSocket events.
- You are verifying a wiring concern: "does this button actually call the right API endpoint and reflect the result?"

Write a **component test** when:

- The behaviour is entirely within a Vue component (computed values, emitted events, slot rendering, prop validation).
- The flow can be fully mocked at the API boundary without losing confidence.
- You need to exercise many variants quickly (e.g., testing all status badge colours).

A heuristic: if the test needs `kctest.projectRoot` or a real `WebSocket`, it belongs in the E2E suite. If it only needs `mount()` from `@vue/test-utils`, it belongs in `tests/web/`.
