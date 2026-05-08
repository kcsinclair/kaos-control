---
title: AgentsRunsView agent-name sort tests have incorrect expected order
type: defect
status: in-development
lineage: sortable-table-columns
parent: lifecycle/tests/sortable-table-columns-11-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# AgentsRunsView agent-name sort tests have incorrect expected order

## Reproduction Steps

1. Run `pnpm vitest run AgentsRunsView.sort.test.ts` from `tests/web/`.
2. Observe 2 failures in the **"AgentsRunsView — Agent column sort"** describe block.

## Expected Behaviour

The tests should assert the correct alphabetical sort order for the three
fixture agent names:

- Ascending: `backend-developer` → `qa` → `requirements-analyst`
  (`b < q < r`)
- Descending: `requirements-analyst` → `qa` → `backend-developer`

## Actual Behaviour

The test expectations are transposed. They assert:

- Ascending test: `names[0]` = `requirements-analyst`,
  `names[1]` = `backend-developer`, `names[2]` = `qa`
  — this is neither ascending nor descending alphabetical order.
- Descending test: `names[0]` = `qa`, `names[2]` = `requirements-analyst`
  — this would be the wrong end of the descending sequence
  (`requirements-analyst` sorts last descending, not first).

The implementation (`useSortableTable`) produces the correct ascending order
(`backend-developer`, `qa`, `requirements-analyst`), but the hardcoded
expectations fail against it.

## Logs / Output

```
FAIL  AgentsRunsView.sort.test.ts > AgentsRunsView — Agent column sort > clicking Agent header sorts runs alphabetically by agent name (ascending)
AssertionError: expected 'backend-developer' to be 'requirements-analyst' // Object.is equality

- Expected
+ Received

- requirements-analyst
+ backend-developer

 ❯ AgentsRunsView.sort.test.ts:145:22
    143|     const names = getAgentNames(wrapper)
    144|     expect(names[0]).toBe('requirements-analyst')
       |                      ^
    145|     expect(names[1]).toBe('backend-developer')
    146|     expect(names[2]).toBe('qa')

FAIL  AgentsRunsView.sort.test.ts > AgentsRunsView — Agent column sort > clicking Agent header again sorts descending
AssertionError: expected 'requirements-analyst' to be 'qa' // Object.is equality

- Expected
+ Received

- qa
+ requirements-analyst

 ❯ AgentsRunsView.sort.test.ts:162:22
    160|     const names = getAgentNames(wrapper)
    161|     expect(names[0]).toBe('qa')
       |                      ^
    162|     expect(names[2]).toBe('requirements-analyst')
```

## Fix

In `tests/web/AgentsRunsView.sort.test.ts`:

**Ascending test (line ~144–147):** change to:
```ts
expect(names[0]).toBe('backend-developer')
expect(names[1]).toBe('qa')
expect(names[2]).toBe('requirements-analyst')
```

**Descending test (line ~161–163):** change to:
```ts
expect(names[0]).toBe('requirements-analyst')
expect(names[1]).toBe('qa')
expect(names[2]).toBe('backend-developer')
```
