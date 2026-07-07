---
title: ProjectQueuePanel realtime tests fail to inspect mock WebSocket calls due to new instances
type: defect
status: in-development
lineage: project-queue-view
parent: lifecycle/tests/project-queue-view-5-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Reproduction Steps

1. Run the Vitest suite for `ProjectQueuePanel.realtime.test.ts`:
   ```sh
   cd tests/web && pnpm test ProjectQueuePanel.realtime.test.ts
   ```
2. Observe 8 test case failures with `TypeError: Cannot read properties of undefined (reading '0')`.

## Expected Behaviour

The test mock for `@/api/ws` should expose a persistent mock object or function so that the test code and the component under test share the same reference to `getAppWs().on`, enabling inspection of `on.mock.calls`.

## Actual Behaviour

`getAppWs: vi.fn(() => ({ on: vi.fn(() => () => {}) }))` returns a new object on every call. Therefore, `vi.mocked(await import('@/api/ws')).getAppWs()` returns a different mock object from the one that registered the handler inside the component or store, leading to empty/undefined mock calls.

## Logs / Output

```
 FAIL  ProjectQueuePanel.realtime.test.ts > ProjectQueuePanel real-time updates > adds job to pending list when queue.added event for current project
TypeError: Cannot read properties of undefined (reading '0')
 ❯ ProjectQueuePanel.realtime.test.ts:214:42
    212|     // Simulate a queue.added event for the current project
    213|     const wsMock = vi.mocked(await import('@/api/ws')).getAppWs()
    214|     const onCall = wsMock.on.mock.calls[0][0]
```
