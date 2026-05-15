---
title: End-to-End Smoke Tests
type: doc
status: in-development
lineage: end-to-end-smoke-tests
created: "2026-05-16T08:00:44+10:00"
priority: normal
parent: lifecycle/ideas/end-to-end-smoke-tests.md
labels:
    - playwright
    - testing
    - qa
    - integration
    - frontend
    - backend
---

# End-to-End Smoke Tests

Documentation covering the design, setup, and execution of Playwright-based end-to-end smoke tests that drive the full system: Go binary, Vue SPA, WebSocket, and auth — together in a real browser.

## Overview

Explain the distinction between the existing Vitest + happy-dom component tests (~337 tests in `tests/web/`) and these E2E smoke tests. Clarify that smoke tests prove the system is wired up end-to-end; they are not regression coverage. Reference the `plans/PROJECT_PLAN.md` line that conflated the two and note that component tests are done while smoke tests are not yet started.

## Why Playwright

Document the rationale for choosing Playwright over Vitest browser mode: flows must span an out-of-process Go backend (auth, WebSocket, file I/O), which is an E2E job. Note that Playwright adds a dependency but stays cleanly isolated from the component test stack.

## Test Environment Setup

Describe how the test environment is bootstrapped:
- Whether tests run against a freshly-built `./dist/kaos-control` binary with a temp `~/.kaos-control` config, or against `make run`.
- How the `lifecycle/` fixture directory is seeded (own fixture or reuse `tests/fixtures/`).
- Any required environment variables or config files.

## Covered Flows

Document each of the five initial smoke test flows:
1. **Login → project picker → open project** — auth round-trip, session cookie, project listing.
2. **Open an artifact, edit, save** — file written to disk, re-indexed, WebSocket `artifact.indexed` event received and reflected in the UI.
3. **Transition an artifact** — role-gated transition succeeds, status persists, commit created.
4. **Start an agent run** — run dialog → run starts → progress events stream over WS → run appears in run history.
5. **Open the 3D graph and click a node** — graph loads with real data, node click navigates to the editor.

## Running the Tests

Document the `make test-e2e` target: how to invoke it, what it builds/starts, how to read results, and how to run a single flow in isolation. Note that these are gated behind `make test-e2e` rather than wired into CI as an automatic job.

## CI Integration

Describe the CI strategy: whether these run as a separate slower job with a real browser, what triggers them, and any caching considerations for the Playwright browser binaries.

## Extending the Suite

Guidance for adding new smoke test flows: naming conventions, fixture patterns, how to assert on WebSocket events, and when a new flow warrants a smoke test versus a component test.
