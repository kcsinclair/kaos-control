---
title: 'performance.test.ts mocks stats API with field "total" but component expects "total_tickets"; renders only 2 of 4 stat cards'
type: defect
status: approved
lineage: dashboard-home-screen
parent: lifecycle/tests/dashboard-home-screen-6-test.md
labels:
  - defect
assignees:
  - role: test-developer
    who: agent
---

# performance.test.ts mocks stats API with wrong field name "total"; SummaryCountsWidget renders only 2 stat cards

## Reproduction Steps

1. Run the performance test:
   ```sh
   cd tests/web && npx vitest run performance.test.ts --config vitest.config.ts
   ```
2. Observe the test `mounts and renders summary counts within 500 ms` fail.

## Expected Behaviour

The performance test asserts that `SummaryCountsWidget` renders 4 `[role="figure"]` stat cards after the API resolves:

```ts
const cards = wrapper.findAll('[role="figure"]')
expect(cards).toHaveLength(4)      // fails — got 2
expect(cards[0].find('.summary-card-value').text()).toBe('5')
```

The first card ("Lifecycle Total") should show `5`.

## Actual Behaviour

The mock in `tests/web/performance.test.ts` (line 51–56) returns:

```ts
vi.mock('@/api/client', () => ({
  api: {
    get: vi.fn().mockResolvedValue({
      total: 5,          // ← wrong field name
      in_progress: 2,
      blocked: 1,
      completed_this_week: 1,
    }),
  },
}))
```

The `SummaryCountsWidget` component defines:

```ts
interface DashboardStats {
  total_tickets: number   // ← correct field name
  in_progress: number
  blocked: number
  completed_this_week: number
}
```

The backend (`internal/index/index.go`, `DashboardStatsRow`) and the integration tests both confirm the JSON field is `total_tickets`. Because the mock returns `total` instead of `total_tickets`, `stats.value.total_tickets` is `undefined` after the API call resolves.

Vue warns: `Invalid prop: type check failed for prop "value". Expected Number | String, got Undefined` and the "Lifecycle Total" `SummaryCountCard` receives `value=undefined`, preventing it (and at least one other card) from rendering with `role="figure"`. Only 2 of 4 stat-cards are found.

**Fix required** (in `tests/web/performance.test.ts`):

Rename the mock field from `total` to `total_tickets`:

```ts
get: vi.fn().mockResolvedValue({
  total_tickets: 5,
  in_progress: 2,
  blocked: 1,
  completed_this_week: 1,
}),
```

Apply the same correction at line 128 for the five-consecutive-mounts test:

```ts
vi.mocked(api.get).mockResolvedValueOnce({
  total_tickets: i,
  in_progress: 0,
  blocked: 0,
  completed_this_week: 0,
})
```

## Logs / Output

```
 FAIL  performance.test.ts > SummaryCountsWidget — mount and render performance > mounts and renders summary counts within 500 ms (API latency ≤ 50 ms)
AssertionError: expected [ DOMWrapper{ …(3) }, …(1) ] to have a length of 4 but got 2

- Expected
+ Received

- 4
+ 2

 ❯ performance.test.ts:95:19
     93|     // Verify the widget rendered four stat cards with the expected values.
     94|     const cards = wrapper.findAll('[role="figure"]')
     95|     expect(cards).toHaveLength(4)
       |                   ^

stderr | performance.test.ts
[Vue warn]: Invalid prop: type check failed for prop "value". Expected Number | String, got Undefined
  at <SummaryCountCard label="Lifecycle Total" value=undefined icon=fn ... >
  at <SummaryCountsWidget project="testproject" ref="VTU_COMPONENT" >
```
