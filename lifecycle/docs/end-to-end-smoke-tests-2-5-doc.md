---
title: End-to-End Smoke Tests
type: doc
status: done
lineage: end-to-end-smoke-tests
created: "2026-05-16T08:00:44+10:00"
completed: "2026-05-16"
priority: normal
parent: lifecycle/ideas/end-to-end-smoke-tests.md
output: docs/end-to-end-smoke-tests.md
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

## Produced

Full documentation written to `docs/end-to-end-smoke-tests.md`.

### Sections covered

- **Overview** — distinction between the Vitest + happy-dom component test suite (`tests/web/`, ≈337 tests) and the Playwright E2E smoke tests; clarification that smoke tests prove end-to-end wiring, not regression coverage.
- **Why Playwright** — rationale for choosing Playwright over Vitest browser mode; flows must span an out-of-process Go backend (auth, WebSocket, disk I/O), which requires a real browser and a real binary.
- **Test Environment Setup** — step-by-step bootstrap: binary compilation, temp home/project directories, fixture copy (`tests/e2e/fixtures/lifecycle/`), git init, free-port binding, app config and project registration, `/api/health` poll, admin user bootstrap. No environment variables required.
- **Covered Flows** — detailed writeups for all six flows (00–05): harness smoke, login/project access, edit-and-save with `file.changed` WS assertion, status transition with frontmatter + git verification, stub agent run with `agent.started` WS event and `done` status poll, and graph node click navigation.
- **Running the Tests** — `make test-e2e`, interactive UI (`pnpm --dir tests/e2e test:ui`), debugger mode (`test:debug`), single-flow targeting by file or grep, reading the HTML report, trace inspection.
- **CI Integration** — rationale for keeping smoke tests out of the default PR CI job; example GitHub Actions job YAML; Playwright browser caching strategy.
- **Extending the Suite** — naming conventions, skeleton template, fixture data guidelines, `connectProjectWs` helper usage with annotated example, smoke test vs component test decision heuristic.
