---
title: End-to-end smoke tests for core flows — Requirements
type: requirement
status: approved
lineage: end-to-end-smoke-tests
priority: normal
parent: lifecycle/ideas/end-to-end-smoke-tests.md
labels:
    - testing
    - qa
    - feature
release: KC-Release1
---

# End-to-end smoke tests for core flows — Requirements

## Problem

The current frontend test suite at [tests/web/](tests/web/) runs over a thousand
component tests under Vitest + happy-dom. That suite is comprehensive at the
unit level but cannot catch a class of bug that has bitten this project
repeatedly: drift between the frontend's expectations and the running
backend's contract.

Concrete examples from the recent commit history:

- Dashboard widgets read field names that no longer matched the backend
  response shape (`stats.total` vs `total_tickets`, `data.items` vs
  `data.distribution`, `data.items` vs `data.buckets`). All three widgets'
  component tests passed because each test mocked the wrong shape that
  matched its widget. The bugs only surfaced when the user opened the live
  dashboard.
- The releases store envelope-unwrapping bug — the spread error that caused
  the 409 conflict — was invisible to component tests because the test
  mocked `listReleases` to return an unwrapped array, while the real backend
  returned a `{releases: [...]}` envelope.
- The `ERR_INVALID_URL` issue caused 39 unhandled rejections in the test
  suite without failing any tests.

Each of these required either the user to find them visually in the running
app, or for a backend change to expose the gap during routine testing. There
is no automated check that boots the actual binary, drives a real browser,
and confirms the headline flows still work.

## Goals

- Detect frontend/backend contract drift before a developer or user does.
- Detect SPA route + auth wiring breakage as soon as it lands.
- Detect WebSocket channel breakage for the live-progress paths
  (artifact-indexed, agent.progress, pipeline.step.output, feed.new).
- Be cheap enough to run on every commit during local development —
  ideally under 30 seconds for the smoke suite.
- Compose with existing test infrastructure (`make test-unit`,
  `make test-integration`, frontend Vitest); do not replace any of it.

## Non-goals

- Replacing component or integration tests. Smoke tests are a sanity layer,
  not regression coverage.
- Visual regression / screenshot diffing. Out of scope; can be a follow-up.
- Cross-browser matrix. Chromium only for the first cut.
- Performance benchmarking. Out of scope.
- Testing every artifact-list filter, every kanban column, every devops
  pipeline shape. Smoke tests prove the spine is wired up; component tests
  prove the joints work.

## Detailed requirements

### Functional

#### F1 — Test framework

- Use **Playwright** (`@playwright/test`) — separate test runner, real
  Chromium, well-suited to driving the production binary. The Vitest
  browser mode alternative was considered and rejected: it shares a
  process with the SPA, which is the opposite of what an E2E test needs.
- Tests live in a new sibling directory: `tests/e2e/`. Sibling-of, not
  child-of, `tests/web/` so they do not share node_modules and Vitest
  config.
- Tests are TypeScript, target Node 20+.

#### F2 — Test harness

The harness is responsible for spinning up an isolated kaos-control
instance for each test run.

- Each `npx playwright test` invocation:
  1. Builds (or reuses) `./dist/kaos-control` via `make build` if the
     binary is older than the latest source change in `cmd/` /
     `internal/` / `web/dist/`.
  2. Creates a temp directory `KCTEST=$(mktemp -d)` for `~/.kaos-control`
     and a separate temp directory for the project root.
  3. Seeds a minimal `lifecycle/` fixture under the project root: ten
     ideas, three requirements, one defect, one release. Seed content
     is fixture data committed under `tests/e2e/fixtures/`.
  4. Writes app config to `$KCTEST/config.yaml` pointing at a random
     free port and the project registry directory.
  5. Writes a project entry YAML to `$KCTEST/projects/testproject.yaml`.
  6. Spawns the binary as a child process with
     `KAOS_CONTROL_HOME=$KCTEST` and waits for `/api/health` to return
     200.
  7. Bootstraps a single test user via the auth-less first-user
     endpoint: `POST /api/admin/users`.
- After the run, the child process is signalled `SIGTERM`, awaited, and
  the temp directories removed.
- The harness must run multiple test files in parallel against
  separate binary instances on separate ports.

#### F3 — Initial smoke flows

Five flows must be covered for the first release. Each flow must use a
real browser session (cookie-based auth, real WebSocket connection) and
assert both the visible UI state and any expected on-disk side effect.

1. **Login & project picker.** `POST /api/auth/login` with seeded
   credentials, navigate to `/projects`, assert the `testproject` card
   is visible, click it, assert the URL becomes `/p/testproject/dashboard`.
2. **Artifact edit and save.** Navigate to an artifact in the editor.
   Modify the body. Save. Assert: response is 200, the file on disk
   contains the new body, the artifact list shows the updated mtime,
   a `file.changed` WebSocket event was received in the test browser.
3. **Workflow transition.** From the artifact editor, click Change
   Status, choose a target. Assert: response is 200, the file's
   frontmatter `status:` field is updated, the page header reflects
   the new status, a `transition(<lineage>): <from> → <to>` git commit
   was created in the project repo.
4. **Agent run.** Open the agent dialog for a configured agent, click
   Run. Assert: a `pipeline.run.started`-style WebSocket event arrives,
   the run appears in the runs list with status `running`. Kill the run.
   Assert: status flips to `killed`. (No real Claude Code subprocess
   is invoked — the test uses a stub `echo`-based agent driver
   configured in the test fixture's `lifecycle/config.yaml`.)
5. **3D graph render and click.** Navigate to `/p/testproject/graph`.
   Assert the canvas renders (Cytoscape root element present, node
   count matches the seeded artifact count). Click a node. Assert the
   URL becomes `/p/testproject/artifacts/<that-node-path>`.

#### F4 — Run targets

- A new `make test-e2e` target: builds the binary if needed, then runs
  Playwright. Exit status is the runner's exit status.
- A new `make test-all` target: `test-unit && test-integration && test-e2e
  && (cd tests/web && pnpm test)`.
- The DevOps pipeline at `lifecycle/devops/test.yaml` adds a new step
  for the E2E suite, gated to run last.

### Non-functional

- **NF1 — Runtime.** Full smoke suite under 30 s on a developer laptop;
  under 60 s on CI. Achieved by running flows in parallel where
  practical and using a single binary instance per test file.
- **NF2 — Determinism.** No flake budget. A test that passes intermittently
  is treated as broken until stabilised. The harness uses a fresh temp
  directory per test file and a fixed seed corpus, so there is no shared
  mutable state between test files.
- **NF3 — Failure clarity.** On failure, Playwright's HTML report is
  produced at `tests/e2e/playwright-report/`. A failed run also captures
  a video of the failing trace, the binary's stderr, and the contents
  of `$KCTEST/data/<project>/index.db` (sqlite dump for post-mortem).
- **NF4 — Isolation.** Tests must not require any preinstalled state.
  Specifically: no requirement that Claude Code CLI be installed
  (agents run via stubbed driver), no requirement that Ollama be
  reachable, no requirement that any port other than the dynamically
  allocated one is free.
- **NF5 — Documentation.** A new `tests/e2e/README.md` covers: how to
  run the suite, how to debug a failing test (`PWDEBUG=1`,
  `--ui` mode, the trace viewer), how to add a new flow.

## Acceptance criteria

- [ ] `make test-e2e` exits 0 on a clean checkout against `main` (with
      the binary built).
- [ ] All five flows in F3 pass in under 30 s on a developer laptop.
- [ ] Deliberately introducing one of the historical regressions
      (the dashboard `total` → `total_tickets` rename, reverted on a
      branch) causes the relevant smoke flow to fail with a clear
      message — verified by the implementer before merge.
- [ ] Killing the test run mid-flow with Ctrl-C cleans up the
      spawned binary and the temp directory.
- [ ] No flaky tests in 10 consecutive runs; if one is flaky, it
      blocks merge.
- [ ] `tests/e2e/README.md` documents add-a-flow workflow.
- [ ] `lifecycle/devops/test.yaml` runs the E2E suite as its last
      step.
- [ ] DevOps pipeline `make test-e2e` step succeeds in under 60 s.

## Dependencies

- The CLI scaffolder ([cli-init-scaffold idea](../ideas/cli-init-scaffold.md))
  would simplify the harness's project bootstrap. The smoke tests should
  use the harness directly today; if `kaos-control init` lands first,
  the harness can switch to using it.
- The configurable agent driver: F3 flow #4 needs a stubbed agent that
  doesn't require Claude Code to be installed. Either reuse the existing
  driver interface with a `shell-stub` driver type, or accept that flow
  #4 is skipped when Claude Code is not on PATH.

## Notes

- A previous version of this lineage's idea ([end-to-end-smoke-tests.md](../ideas/end-to-end-smoke-tests.md))
  was auto-blocked by the open-questions handler. The relevant questions
  are now answered in this requirement (F2 covers test fixtures and
  binary handling; F4 covers CI integration).

  Open answers:
  - **Build vs `make run`?** Use `./dist/kaos-control` built from current
    source. `make run` is for the developer's interactive session; tests
    must not collide with it.
  - **Fresh fixtures vs reuse `tests/fixtures/`?** Fresh fixtures
    committed under `tests/e2e/fixtures/`. Reusing `tests/fixtures/` would
    couple two test layers and cause edits in one to break the other.
  - **CI integration?** `make test-e2e` on its own and as the last step
    of the existing `lifecycle/devops/test.yaml` pipeline. No separate CI
    job until there is CI.
