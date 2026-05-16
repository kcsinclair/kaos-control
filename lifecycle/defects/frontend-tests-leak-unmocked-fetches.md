---
title: Frontend Vitest Suite Leaks Un-Mocked Fetches, Logged as ECONNREFUSED Errors
type: defect
status: planning
lineage: frontend-tests-leak-unmocked-fetches
created: "2026-05-13T11:55:00+10:00"
priority: low
labels:
    - defect
    - frontend
    - test
    - hygiene
release: KC-Release3
---

# Frontend Vitest Suite Leaks Un-Mocked Fetches, Logged as ECONNREFUSED Errors

## Reproduction Steps

1. From the repo root, run `cd tests/web && pnpm test`.
2. Wait for the full suite to finish.
3. Observe the trailing summary block, which reads (paraphrased):

   ```
   Test Files  76 passed (76)
        Tests  1248 passed (1248)
       Errors  4 errors
   ELIFECYCLE  Test failed.
   ```

4. Scroll back through the output and locate the four `Serialized Error`
   blocks. Each one contains:

   ```
   Error: connect ECONNREFUSED ::1:8080
       at createConnectionError (node:net:1686:14)
       at afterConnectMultiple (node:net:1716:16)
   ```

   plus the IPv4 sibling on `127.0.0.1:8080`.

## Expected Behaviour

The vitest suite should exit with status 0 and zero unhandled errors when
all assertions pass. No network connections should be attempted during
the test run — all `fetch`/WS interactions must be mocked.

## Actual Behaviour

Four tests reach a code path that calls `fetch()` (or an API helper that
calls `fetch()` internally) without a mock in place. The fetch resolves
its URL against the synthetic test origin
(`tests/web/vitest.config.ts:16` sets `happyDOM.url = http://localhost:8080`),
attempts a real TCP connection to `localhost:8080`, and fails with
`ECONNREFUSED`. The errors are unhandled rejections inside the test
worker, so all assertions still pass but vitest's process-level
`ELIFECYCLE` post-step flags the run as failed.

The four ECONNREFUSED errors are consistent run-to-run, indicating a
small fixed set of leaking call sites rather than flaky network
behaviour. The port number `8080` is not significant — it's the happy-dom
origin string used to make relative URLs resolvable inside the fake DOM;
no real server is expected to listen there.

## Logs / Output

Representative `Serialized Error` excerpt from a recent run:

```
Serialized Error: {
  errors: [
    { stack: 'Error: connect ECONNREFUSED ::1:8080\n    at createConnectionError (node:net:1686:14)\n    at afterConnectMultiple (node:net:1716:16)',
      message: 'connect ECONNREFUSED ::1:8080',
      errno: -61, code: 'ECONNREFUSED', syscall: 'connect', address: '::1', port: 8080,
      constructor: 'Function<Error>', name: 'Error', toString: 'Function<toString>' },
    { stack: 'Error: connect ECONNREFUSED 127.0.0.1:8080\n    at createConnectionError (node:net:1686:14)\n    at afterConnectMultiple (node:net:1716:16)',
      message: 'connect ECONNREFUSED 127.0.0.1:8080',
      errno: -61, code: 'ECONNREFUSED', syscall: 'connect', address: '127.0.0.1', port: 8080,
      constructor: 'Function<Error>', name: 'Error', toString: 'Function<toString>' }
  ],
  code: 'ECONNREFUSED'
}
```

## Suggested Investigation

1. **Surface which tests leak.** Add a temporary `unhandledRejection`
   listener in `tests/web/test-setup.ts` (if it exists, or set up via
   `vitest.config.ts` setupFiles) that records the test name from
   `expect.getState().currentTestName` and prints it on each
   unhandled `ECONNREFUSED`. Re-run; the four offenders will identify
   themselves.

2. **Common pattern to look for.** Components that subscribe in
   `onMounted()` to stores whose actions call `fetch()` without the
   test mocking that store's `api.*` helpers. The leak typically
   happens because the test mocks the *function under test* but not
   an *unrelated side-effect* triggered by `onMounted`.

3. **Fix shape.** Either mock the offending API helper in the test, or
   stop the unrelated `onMounted` side-effect from firing in unit
   tests (e.g. via `shallowMount` for the parent, or by guarding the
   side-effect behind a prop the test omits).

## Out of Scope

- The synthetic test origin in `vitest.config.ts` is changing
  separately (`http://localhost:8080` → `http://test.local`) so the
  port number `8080` stops appearing in unrelated grep results. That
  change does NOT fix this defect — it just removes a red herring.
- The Vite dev-server proxy in `web/vite.config.ts` was also stale
  on `:8080` and is being changed to `:8042` (the kaos-control default
  port). Unrelated to this leak.

## Notes

This defect is hygiene-only. No assertions fail, no user-visible
feature is affected. It does, however, mean the frontend test step
exits non-zero in CI, which can mask a real regression that the same
exit code would otherwise reveal. Worth fixing before the next
release branch settles.
