---
title: "Tests: End-to-end Smoke Tests"
type: test
status: done
lineage: end-to-end-smoke-tests
parent: lifecycle/test-plans/end-to-end-smoke-tests-3-test.md
---

# Tests: End-to-end Smoke Tests

This artifact documents the E2E Playwright smoke suite written for the
`end-to-end-smoke-tests` lineage.

## Scenarios covered

### Milestone 1 — Harness smoke (`flows/00-harness-smoke.spec.ts`)

- Spawns a fresh `dist/kaos-control` binary in a temp directory.
- Copies fixture artifacts into a temporary git repository.
- Polls `GET /api/health` until 200 or 10 s timeout.
- Asserts HTTP 200 response.
- Kills the process cleanly and removes temp dirs.

### Milestone 2 — Auth + project access (`flows/01-login.spec.ts`)

- Verifies unauthenticated navigation to `/p/testproject/dashboard` redirects
  to `/login`.
- Bootstraps an admin user via `POST /api/admin/users` (unauthenticated
  bootstrap endpoint).
- Drives the SPA login form (`#email`, `#password`, submit button).
- Asserts the dashboard renders with a non-zero "Lifecycle Total" stat card.

### Milestone 3 — Edit & save (`flows/02-edit-save.spec.ts`)

- Navigates to `lifecycle/requirements/smoke-req-01.md` in the artifact editor.
- Appends a unique `smoke-test-marker-<timestamp>` string via CodeMirror (`.cm-content`).
- Clicks the Save button and waits for the "Saved" toast.
- Reads the file from disk and asserts it contains the marker.
- Subscribes to the project WebSocket and asserts a `file.changed` event arrives
  within 8 s of the save.

### Milestone 3 — Transition (`flows/03-transition.spec.ts`)

- Navigates to `lifecycle/requirements/smoke-req-01.md` (draft status).
- Clicks "Change Status" button to open `TransitionDialog`.
- Selects "clarifying" and confirms.
- Asserts the API response is HTTP 200.
- Reads the file from disk: frontmatter must contain `status: clarifying`.
- Runs `git log --oneline -1` on the project repo: commit subject must match
  `transition(smoke-req-01): ... clarifying`.

### Milestone 4 — Agent run (`flows/04-agent-run.spec.ts`)

- Navigates to `/p/testproject/agents`.
- Clicks "Run Agent", selects the `stub-agent` chip, provides a target path.
- Subscribes to WS and asserts `agent.started` event arrives within 8 s.
- Polls `GET /api/p/testproject/agents/runs/:run_id` until status is `done`
  (up to 10 s). Asserts `done` — no Claude Code required.

### Milestone 4 — Graph/Map (`flows/05-graph-click.spec.ts`)

- Navigates to `/p/testproject/map`.
- Waits for the Cytoscape canvas to render and `window.__cy` to be exposed.
- Waits for layout to stabilise (nodes have non-zero positions).
- Asserts `cy.nodes().length` is greater than zero.
- Programmatically fires a `tap` event on the `smoke-req-01` node via
  `window.__cy`.
- Asserts the URL changes to include `smoke-req-01`.

### Milestone 5 — Makefile + pipeline integration

- `make test-e2e` target wires `cd tests/e2e && pnpm install && pnpm test`.
- `make test-all` runs unit + integration + e2e + web tests.
- `lifecycle/devops/test.yaml` includes an "E2E smoke tests" step.

## Shell-stub driver (`internal/agent/shell_stub.go`)

Added a `ShellStubDriver` to `internal/agent/` and registered it under the
`"shell-stub"` key in the agent `Manager`. The driver runs a configurable
shell command (`shell_command` in `lifecycle/config.yaml`) and exits 0. This
allows flow 04 to test the full agent run lifecycle without Claude Code.

## Test files

- `tests/e2e/package.json` — `@playwright/test` + TypeScript
- `tests/e2e/playwright.config.ts` — workers=4, list+html reporters, retries=0
- `tests/e2e/tsconfig.json` — ES2022/ESNext strict config
- `tests/e2e/README.md` — run/debug/contribute guide
- `tests/e2e/harness/kaos-control.ts` — binary spawn + health-poll + teardown
- `tests/e2e/harness/auth.ts` — bootstrap user + SPA login helper
- `tests/e2e/harness/ws.ts` — WebSocket subscription + event collector
- `tests/e2e/fixtures.ts` — Playwright `test.extend` with `kctest` and `loggedInPage`
- `tests/e2e/fixtures/lifecycle/config.yaml` — per-project config with stub-agent
- `tests/e2e/fixtures/lifecycle/ideas/smoke-idea-01..10.md` — 10 seeded ideas
- `tests/e2e/fixtures/lifecycle/requirements/smoke-req-01..03.md` — 3 requirements
- `tests/e2e/fixtures/lifecycle/defects/smoke-defect-01.md` — 1 defect
- `tests/e2e/fixtures/seed-helpers.ts` — seed metadata constants
- `tests/e2e/flows/00-harness-smoke.spec.ts` — M1 harness smoke
- `tests/e2e/flows/01-login.spec.ts` — M2 auth + dashboard
- `tests/e2e/flows/02-edit-save.spec.ts` — M3 edit + WS event
- `tests/e2e/flows/03-transition.spec.ts` — M3 status transition
- `tests/e2e/flows/04-agent-run.spec.ts` — M4 agent run via stub driver
- `tests/e2e/flows/05-graph-click.spec.ts` — M4 map node click
- `internal/agent/shell_stub.go` — shell-stub driver implementation
