---
title: End-to-End Smoke Tests for Server Startup
type: test
status: draft
lineage: end-to-end-smoke-tests
parent: lifecycle/defects/end-to-end-smoke-tests-9-defect.md
---

# End-to-End Smoke Tests for Server Startup

This test suite covers the core functionality of the kaos-control server startup and basic API endpoints.

## Test Coverage

The tests verify that:
1. The server can be successfully started with the serve command and config flag (fixes defect where `-d` or `serve` was missing)
2. The health endpoint is accessible after startup
3. Basic HTTP responses are returned correctly
4. Server properly shuts down when requested

## Test Files

- `tests/e2e/flows/00-harness-smoke.spec.ts` - Basic harness smoke test
- `tests/e2e/flows/01-login.spec.ts` - Login functionality
- `tests/e2e/flows/02-edit-save.spec.ts` - Artifact editing and saving
- `tests/e2e/flows/03-transition.spec.ts` - Artifact state transitions

The test suite ensures the core server functionality works as expected before more complex functionality is tested.

## Notes

This test suite specifically addresses defect [end-to-end-smoke-tests-9-defect.md](lifecycle/defects/end-to-end-smoke-tests-9-defect.md) which reported that all E2E tests were failing due to the server startup command missing the `serve` argument. The fix ensures that the harness correctly starts the server with `['serve', '-config', configPath]` instead of just `['-config', configPath]`.