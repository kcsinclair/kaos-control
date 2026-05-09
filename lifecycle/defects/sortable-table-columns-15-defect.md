---
title: AgentsRunsView sort tests — missing @/api/config mock causes 9 unhandled rejections
type: defect
status: done
lineage: sortable-table-columns
parent: lifecycle/tests/sortable-table-columns-13-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
release: KC-Feature-Sprint
---

# AgentsRunsView sort tests — missing @/api/config mock causes 9 unhandled rejections

All 9 tests in `tests/web/AgentsRunsView.sort.test.ts` report as passing, but
the suite exits with code **1** because every test produces an unhandled promise
rejection. Vitest treats unhandled rejections as suite-level errors, so CI will
record this run as a failure even though no individual assertion fails.

## Reproduction Steps

1. `cd tests/web`
2. `pnpm exec vitest run AgentsRunsView.sort.test.ts`
3. Observe all 9 tests pass but 9 unhandled-rejection errors are reported and
   exit code is **1**.

## Expected Behaviour

The suite exits with code 0. No unhandled rejections are emitted. All 9 tests
pass cleanly.

## Actual Behaviour

Every test mounts `AgentsRunsView`, which in `onMounted` calls
`configStore.fetchRoles(project)`. That calls `getRoles` from `@/api/config`,
which issues `fetch('/api/p/testproject/roles')`. In the happy-dom test
environment, relative URLs have no base URL, so `new URL('/api/…')` throws
`ERR_INVALID_URL`. The rejection is never caught, producing one unhandled
rejection per test (9 total). Vitest captures these as suite errors and exits
with code 1.

The test already mocks `@/api/agents` but omits a mock for `@/api/config`.

## Logs / Output

```
 ✓ AgentsRunsView.sort.test.ts  (9 tests) 95ms

⎯⎯⎯⎯⎯⎯ Unhandled Errors ⎯⎯⎯⎯⎯⎯

Vitest caught 9 unhandled errors during the test run.

⎯⎯⎯⎯ Unhandled Rejection ⎯⎯⎯⎯⎯
TypeError: Failed to parse URL from /api/p/testproject/roles
Caused by: TypeError: Invalid URL
 ❯ new URL node:internal/url:819:25
 ❯ new URL node_modules/.pnpm/happy-dom@14.12.3/.../happy-dom/src/url/URL.ts:9:15
 ❯ new Request node:internal/deps/undici/undici:11063:25
 ❯ fetch ...
 ❯ request ../../web/src/api/client.ts:30:21
 ❯ Object.get ../../web/src/api/client.ts:65:29
 ❯ Module.getRoles ../../web/src/api/config.ts:13:14
 ❯ Proxy.fetchRoles ../../web/src/stores/projectConfig.ts:12:24

 Test Files  1 passed (1)
      Tests  9 passed (9)
     Errors  9 errors
   Start at  15:27:12
   Duration  889ms
```

## Fix

Add a `vi.mock('@/api/config', ...)` block alongside the existing
`vi.mock('@/api/agents', ...)` in `tests/web/AgentsRunsView.sort.test.ts`:

```ts
vi.mock('@/api/config', () => ({
  getRoles: vi.fn().mockResolvedValue({ roles: [] }),
}))
```

This prevents `configStore.fetchRoles` from making a real HTTP call and
eliminates all 9 unhandled rejections.
