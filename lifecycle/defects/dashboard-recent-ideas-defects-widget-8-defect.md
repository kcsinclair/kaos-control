---
title: "DashboardGrid missing .dashboard-charts-top / .dashboard-charts-bottom split"
type: defect
status: approved
lineage: dashboard-recent-ideas-defects-widget
parent: lifecycle/tests/dashboard-recent-ideas-defects-widget-6-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
---

# DashboardGrid missing .dashboard-charts-top / .dashboard-charts-bottom split

`DashboardGrid.vue` renders all chart-slot widgets inside a single
`.dashboard-charts` container. The spec requires the chart section to be split
into two sub-containers:

- `.dashboard-charts-top` — first three chart widgets (order 0, 1, 1.5):
  `status-distribution`, `stages-distribution`, `recent-ideas-defects`.
- `.dashboard-charts-bottom` — remaining chart widgets (order ≥ 2):
  `velocity-chart`.

Neither `.dashboard-charts-top` nor `.dashboard-charts-bottom` exist in the
current template.

## Reproduction Steps

1. Register four chart-slot widgets with orders 0, 1, 1.5, and 2:
   ```ts
   registerWidget('status-distribution',  StatusStub, { slot: 'chart', order: 0   })
   registerWidget('stages-distribution',  StagesStub, { slot: 'chart', order: 1   })
   registerWidget('recent-ideas-defects', RecentStub, { slot: 'chart', order: 1.5 })
   registerWidget('velocity-chart',       VeloStub,   { slot: 'chart', order: 2   })
   ```
2. Mount `DashboardGrid` with a valid `project` prop.
3. Assert `wrapper.find('.dashboard-charts-top').exists()` — fails.
4. Assert `wrapper.find('.dashboard-charts-bottom').exists()` — fails.

## Expected Behaviour

- `.dashboard-charts-top` wraps the first three chart widgets (orders 0, 1, 1.5)
  in a 3-column grid row.
- `.dashboard-charts-bottom` wraps remaining chart widgets (order ≥ 2) in a
  separate grid row, allowing `velocity-chart` to span 2 columns.
- `.dashboard-charts-top` must NOT contain `velocity-chart`.
- `.dashboard-charts-bottom` must NOT contain `status-distribution`,
  `stages-distribution`, or `recent-ideas-defects`.

## Actual Behaviour

All chart widgets are rendered flat inside a single `section.dashboard-charts`.
The DOM contains no `.dashboard-charts-top` or `.dashboard-charts-bottom`
elements. The top/bottom split required to correctly position the velocity chart
in its own row (spanning 2 of 3 columns) is absent.

## Logs / Output

```
FAIL  DashboardView.test.ts > DashboardGrid — Milestone 5: layout with recent-ideas-defects widget > TC3: first 3 chart-slot widgets render inside .dashboard-charts-top
AssertionError: expected false to be true // Object.is equality
 ❯ DashboardView.test.ts:526:26
    524|
    525|     const top = wrapper.find('.dashboard-charts-top')
    526|     expect(top.exists()).toBe(true)
       |                          ^

FAIL  DashboardView.test.ts > DashboardGrid — Milestone 5: layout with recent-ideas-defects widget > TC4: velocity-chart renders inside .dashboard-charts-bottom
AssertionError: expected false to be true // Object.is equality
 ❯ DashboardView.test.ts:553:29
    551|
    552|     const bottom = wrapper.find('.dashboard-charts-bottom')
    553|     expect(bottom.exists()).toBe(true)
       |                             ^
```

2 tests failing. The `.dashboard-charts` section referenced in the current
template does not satisfy the two-row layout contract.
