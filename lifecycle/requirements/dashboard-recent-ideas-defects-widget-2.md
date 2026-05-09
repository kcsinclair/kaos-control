---
title: Recent Ideas and Defects Dashboard Widget
type: requirement
status: planning
lineage: dashboard-recent-ideas-defects-widget
created: "2026-05-09"
priority: high
parent: lifecycle/ideas/dashboard-recent-ideas-defects-widget.md
labels:
    - frontend
    - feature
    - vue
    - enhancement
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

# Recent Ideas and Defects Dashboard Widget

## Problem

The dashboard currently provides no at-a-glance visibility into the most recent ideas and defects. Users must navigate to individual artifact lists to discover what has been filed recently. This adds unnecessary clicks and delays triage, especially when new defects are time-sensitive.

## Goals / Non-goals

### Goals

- Surface the latest ideas and defects directly on the dashboard so users can spot new items without navigating away.
- Make each item clickable, opening the artifact in the artifact view.
- Restructure the dashboard column layout so the chart/widget column is wider, with a top row divided into thirds and the Completion Velocity widget spanning two-thirds width beneath the two pie charts.

### Non-goals

- This widget does not replace the existing Recent Activity feed; it complements it with a type-filtered view.
- No filtering, searching, or pagination within the widget — it shows only the most recent 6 items.
- No backend API changes beyond what is needed to fetch the 6 most recent ideas and defects (no new aggregation or analytics endpoints).

## Detailed Requirements

### Functional

1. **New widget: "Recent Ideas and Defects"**
   - Registered in the widget registry with slot and order values that place it in the top row of the charts column, alongside the two pie charts.
   - Fetches the 6 most recent artifacts where `type` is `idea` or `defect`, sorted by `created` descending.
   - Each item displays: artifact title, type badge (`idea` or `defect`), and relative timestamp (e.g. "2 hours ago").
   - Each item is a clickable link that navigates to the artifact detail view (`/p/{project}/artifacts/{path}`).
   - The widget subscribes to the `artifact.indexed` WebSocket event and refreshes its data when an idea or defect is created, updated, or deleted.

2. **Data source**
   - The widget fetches data from an API endpoint that returns recent artifacts filtered by type.
   - If no existing endpoint supports filtering by type with a limit, a new endpoint or query-parameter extension (e.g. `GET /p/{project}/artifacts?type=idea,defect&sort=created:desc&limit=6`) must be added.

3. **Dashboard layout restructure**
   - The charts column (left in the two-column layout) must become wider relative to the panels column (right).
   - The top row of the charts column is divided into three equal-width cells:
     - Cell 1: Status Distribution (pie chart).
     - Cell 2: Stages Pie Chart (if it exists; otherwise this cell is available for future use — see Open Questions).
     - Cell 3: Recent Ideas and Defects widget.
   - Below the top row, the Completion Velocity widget spans two-thirds of the column width, aligned beneath the two pie charts.

4. **Widget positioning relative to Recent Activity**
   - The new widget is in the charts column (left), not the panels column where Recent Activity lives. The idea's instruction to position it "above the existing Recent Activity widget in its column" is interpreted as placing it in the same visual area but within the restructured layout described above.

### Non-functional

1. **Performance** — The widget must render within 200 ms of data arrival. The API query must complete within 300 ms for projects with up to 1 000 artifacts.
2. **Responsiveness** — On viewports narrower than 1024 px, the three-column top row collapses to a single column (widgets stack vertically).
3. **Accessibility** — Clickable items must be keyboard-navigable and have appropriate ARIA labels. Type badges must meet WCAG 2.1 AA contrast requirements.
4. **Consistency** — The widget's card style, spacing, font sizes, and colours must match the existing dashboard widgets (e.g. ActivityFeedWidget's entry styling).

## Acceptance Criteria

- [ ] A "Recent Ideas and Defects" widget appears on the dashboard in the top row of the charts column, occupying one-third width alongside Status Distribution and a third widget.
- [ ] The widget displays up to 6 items, each showing title, type badge, and relative timestamp.
- [ ] Items are sorted by creation date, most recent first.
- [ ] Clicking an item navigates to the artifact detail view for that artifact.
- [ ] The widget live-updates when an `artifact.indexed` WebSocket event fires for an idea or defect.
- [ ] The Completion Velocity widget spans two-thirds of the charts column width, beneath the two pie charts.
- [ ] The charts column is wider than the panels column (previously equal `1fr` vs fixed `340px`; new ratio gives more space to charts).
- [ ] On narrow viewports (< 1024 px), the top-row widgets stack vertically and the layout remains usable.
- [ ] All clickable items are keyboard-accessible with visible focus indicators.
- [ ] Type badges use distinct colours for `idea` and `defect` and meet WCAG 2.1 AA contrast.
- [ ] Widget data loads within 300 ms and renders within 200 ms on a dataset of 500 artifacts.
- [ ] Related lineage: [[dashboard-recent-ideas-defects-widget]]

## Resolved Questions

1. **Stages Pie Chart**: The idea references a "Stages Pie Chart" as one of the three widgets in the top row. No such widget currently exists in the widget registry — only Status Distribution and Completion Velocity are registered chart widgets. Should a new Stages Pie Chart widget be created as part of this work, or should the top row contain only two pie charts (Status Distribution + the new widget) with Completion Velocity below?

> The Stages Pie Chart will be added before this work.

2. **API endpoint**: Should the backend expose a dedicated endpoint for recent ideas/defects, or should the existing artifacts list endpoint be extended with `type` and `sort` query parameters? The latter is more general-purpose.

> Extend the endpoint with query parameters

3. **Empty state**: What should the widget display when there are no ideas or defects in the project? A simple "No recent ideas or defects" message, or should the widget be hidden entirely?

> Yes, display "No recent ideas or defects"

4. **Column width ratio**: The idea says the charts column should be "wider" but does not specify an exact ratio. Should the two-column split change from `1fr 340px` to something like `2fr 340px`, or should the panels column also become flexible (e.g. `2fr 1fr`)?

> each column should be one third wide, flexible panels is a good option.
