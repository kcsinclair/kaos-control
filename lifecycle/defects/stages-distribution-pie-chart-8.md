---
title: "DashboardView SummaryCountsWidget tests use wrong role selector and wrong mock field name"
type: defect
status: approved
lineage: stages-distribution-pie-chart
parent: lifecycle/tests/stages-distribution-pie-chart-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
created: "2026-05-09T00:00:00+10:00"
---

# DashboardView SummaryCountsWidget tests use wrong role selector and wrong mock field name

## Reproduction Steps

1. Run the DashboardView component tests:
   ```
   cd tests/web && pnpm vitest run DashboardView.test.ts
   ```
2. Observe the four failing tests in the `SummaryCountsWidget — summary counts after API response` describe block:
   - `renders four stat cards on mount`
   - `shows zero counts while waiting for the API (initial state)`
   - `displays counts returned by the API after the response resolves`
   - `keeps zero counts when the API call fails (graceful degradation)`

## Expected Behaviour

All four tests should pass. Specifically:

- `renders four stat cards on mount`: finds all 4 summary cards.
- `shows zero counts while waiting for the API`: reads zero values from all 4 cards.
- `displays counts returned by the API after the response resolves`: first card shows the `total_tickets` value (12) returned by the mock.
- `keeps zero counts when the API call fails`: reads zero values from all 4 cards.

## Actual Behaviour

**Issue 1 — Wrong role selector.**
`SummaryCountCard.vue` assigns `role="link"` to interactive cards (those with a non-null `:to` prop) and `role="figure"` to non-interactive ones. Two of the four cards ("Lifecycle Total" and "Blocked") have `:to` set to a route object, so they receive `role="link"`. The tests query `wrapper.findAll('[role="figure"]')` and find only 2 elements instead of 4.

**Issue 2 — Wrong API mock field name.**
The module-level `vi.mock('@/api/client')` (line 41 of `DashboardView.test.ts`) and the per-test mock in `displays counts returned by the API after the response resolves` (line 254) both return a `total` field:
```js
{ total: 12, in_progress: 3, blocked: 1, completed_this_week: 5 }
```
The backend returns `total_tickets` (per `DashboardStatsRow` in `internal/index/index.go:1630`), and `SummaryCountsWidget` reads `stats.total_tickets`. Because the mock field is named `total`, the component receives `undefined` for the first card value rather than `12`.

## Logs / Output

```
 FAIL  DashboardView.test.ts > SummaryCountsWidget … > renders four stat cards on mount
AssertionError: expected [ DOMWrapper{ …(3) }, …(1) ] to have a length of 4 but got 2
 ❯ DashboardView.test.ts:238:19

 FAIL  DashboardView.test.ts > SummaryCountsWidget … > shows zero counts while waiting for the API (initial state)
AssertionError: expected [ '0', '0' ] to deeply equal [ '0', '0', '0', '0' ]
 ❯ DashboardView.test.ts:249:20

 FAIL  DashboardView.test.ts > SummaryCountsWidget … > displays counts returned by the API after the response resolves
AssertionError: expected '3' to be '12' // Object.is equality
 ❯ DashboardView.test.ts:268:57

 FAIL  DashboardView.test.ts > SummaryCountsWidget … > keeps zero counts when the API call fails (graceful degradation)
AssertionError: expected [ '0', '0' ] to deeply equal [ '0', '0', '0', '0' ]
 ❯ DashboardView.test.ts:286:20

Tests  5 failed | 60 passed (65)
```

**Fixes required in `tests/web/DashboardView.test.ts`:**

1. Replace all `wrapper.findAll('[role="figure"]')` with `wrapper.findAll('.summary-card')` in the `SummaryCountsWidget` describe block (lines 237, 247, 266, 283).
2. Update the default module mock (line 41) and the per-test mock (line 254) to use `total_tickets` instead of `total`:
   ```js
   // module-level mock
   api.get.mockResolvedValue({ total_tickets: 0, in_progress: 0, blocked: 0, completed_this_week: 0 })
   // per-test mock
   api.get.mockResolvedValueOnce({ total_tickets: 12, in_progress: 3, blocked: 1, completed_this_week: 5 })
   ```
