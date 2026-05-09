---
title: "SummaryCountsWidget first card label is 'Lifecycle Total' instead of 'Total Tickets'"
type: defect
status: approved
lineage: stages-distribution-pie-chart
parent: lifecycle/tests/stages-distribution-pie-chart-6-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
created: "2026-05-09"
---

# SummaryCountsWidget first card label is "Lifecycle Total" instead of "Total Tickets"

## Reproduction Steps

1. Run the DashboardView component tests:
   ```
   cd tests/web && pnpm vitest run DashboardView.test.ts
   ```
2. Observe the `SummaryCountsWidget — summary counts after API response > displays correct card labels` test.

## Expected Behaviour

The first summary card should render the label **"Total Tickets"**, matching the backend JSON field name (`total_tickets`) and the test specification.

## Actual Behaviour

The component renders **"Lifecycle Total"** as the first card's label (see `web/src/components/dashboard/widgets/SummaryCountsWidget.vue:40`):

```vue
<SummaryCountCard
  label="Lifecycle Total"
  :value="stats.total_tickets"
  ...
/>
```

The test expects `labels` to contain `'Total Tickets'` but the actual labels array is:
`['Lifecycle Total', 'In Progress', 'Blocked', 'Completed This Week']`.

## Logs / Output

```
FAIL  DashboardView.test.ts > SummaryCountsWidget — summary counts after API response > displays correct card labels
AssertionError: expected [ 'Lifecycle Total', …(3) ] to include 'Total Tickets'
 ❯ DashboardView.test.ts:308:20
    306|
    307|     const labels = wrapper.findAll('.summary-card-label').map(el => el.text())
    308|     expect(labels).toContain('Total Tickets')
       |                    ^
    309|     expect(labels).toContain('In Progress')
    310|     expect(labels).toContain('Blocked')
```

**File:** `web/src/components/dashboard/widgets/SummaryCountsWidget.vue`, line 40.

**Fix:** Change `label="Lifecycle Total"` to `label="Total Tickets"` on the first `SummaryCountCard`.
