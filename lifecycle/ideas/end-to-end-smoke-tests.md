---
title: End-to-end smoke tests for core flows
type: idea
status: blocked
lineage: end-to-end-smoke-tests
created: "2026-04-28T10:00:00+10:00"
priority: normal
labels:
    - testing
    - qa
    - feature
release: KC-Release1
assignees:
    - role: product-owner
      who: agent
---

# End-to-end smoke tests for core flows

The existing test suite at `tests/web/` runs ~337 component tests under Vitest + happy-dom. That covers component behaviour but not whole-system flows: nothing currently boots the Go binary, drives a real browser, and asserts that the SPA + server + WebSocket + auth all work together.

The line in `plans/PROJECT_PLAN.md` ("Playwright or Vitest browser-mode smoke tests for core flows") conflates component tests with E2E smoke tests. The component tests are done; the smoke tests are not started.

## Scope

A small number of E2E flows that prove the system is wired up end-to-end. Not regression coverage — that's what the component tests are for. Suggested initial flows:

1. **Login → project picker → open project** — auth round-trip, session cookie, project listing.
2. **Open an artifact, edit, save** — file written to disk, re-indexed, WebSocket `artifact.indexed` event received and reflected in the UI.
3. **Transition an artifact** — role-gated transition succeeds, status persists, commit created.
4. **Start an agent run** — run dialog → run starts → progress events stream over WS → run appears in run history.
5. **Open the 3D graph and click a node** — graph loads with real data, node click navigates to the editor.

## Approach

Two viable choices:

- **Playwright** — separate test runner, real Chromium, well-suited to driving the production binary. Adds a dependency but stays cleanly isolated from the component test stack.
- **Vitest browser mode** — keeps continuity with the existing Vitest setup but is less mature for E2E flows that span an out-of-process backend.

Recommendation: Playwright, because the flows need to drive both the SPA and the Go server (including auth and WebSocket), and that is an out-of-process E2E job rather than a component test.

## Open Questions

- Should the smoke tests run against a freshly-built `./dist/kaos-control` binary with a temp `~/.kaos-control` config, or against `make run`?
- Do the smoke tests need their own seeded `lifecycle/` fixture, or can they reuse `tests/fixtures/`?
- Should they be wired into CI as a separate job (slower, real browser) or gated behind a `make test-e2e` target only?
