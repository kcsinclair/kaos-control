---
created: "2026-07-14T19:34:44+10:00"
title: AgentsRunsView projectQueue tests crash because agents store mock lacks runs array
type: defect
status: done
lineage: project-queue-view
parent: lifecycle/tests/project-queue-view-5-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Reproduction Steps

1. Run the Vitest suite for `AgentsRunsView.projectQueue.test.ts`:
   ```sh
   cd tests/web && pnpm test AgentsRunsView.projectQueue.test.ts
   ```
2. Observe 7 test case failures with `TypeError: Cannot read properties of undefined (reading 'length')`.

## Expected Behaviour

The store mock for `@/stores/agents` in `AgentsRunsView.projectQueue.test.ts` should define all reactive state properties used by the template of `AgentsRunsView.vue`, including `runs` and `loading`.

## Actual Behaviour

The mock for `useAgentsStore` only defines `activeRuns` and `agents`. Because `runs` is missing, `store.runs` evaluates to `undefined`, causing a rendering crash when the component tries to access `store.runs.length`.

## Logs / Output

```
TypeError: Cannot read properties of undefined (reading 'length')
 ❯ Proxy._sfc_render ../../web/src/views/project/AgentsRunsView.vue:249:37
    247|
    248|         <div v-if="store.loading" class="state-msg">Loading…</div>
    249|         <div v-else-if="!store.runs.length" class="state-msg">No runs …
```
