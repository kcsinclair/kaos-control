---
title: Dashboard Home Screen
type: requirement
status: done
lineage: dashboard-home-screen
created: "2026-05-06T00:00:00+10:00"
priority: normal
parent: lifecycle/ideas/dashboard-home-screen.md
labels:
    - feature
    - frontend
    - vue
assignees:
    - role: product-owner
      who: agent
---

# Dashboard Home Screen

## Problem

When users open a project they land on an artifact list or graph view with no summarised health indicators. Understanding project status requires manually navigating multiple views and mentally aggregating information. There is no single pane of glass showing completion velocity, status distribution, or recent activity.

## Goals / Non-goals

### Goals

- Provide an at-a-glance project health summary as the default landing page per project.
- Show artifact completion velocity over time (time-series chart of artifacts transitioning to `done`).
- Show current status distribution of non-done tickets (pie/donut chart).
- Surface the project activity feed prominently alongside the visual summaries.
- Make the dashboard extensible so new widgets can be added without restructuring the page.
- Ensure the layout is responsive across desktop and tablet viewports.

### Non-goals

- Real-time collaboration cursors or multi-user presence indicators.
- User-customisable widget arrangement or drag-and-drop layout (future enhancement).
- Filtering or drill-down within dashboard charts (charts link to existing views instead).
- Backend API changes beyond what is needed to serve the dashboard data.

## Detailed Requirements

### Functional

1. **Navigation placement** — A "Dashboard" item must appear as the first entry in the left navigation panel for every project. Clicking it renders the dashboard view.
2. **Default landing** — When a project is opened (via project list or direct URL), the router must resolve to the dashboard route by default.
3. **Completion velocity chart** — A time-series line or bar chart displaying the count of artifacts that transitioned to status `done` per day/week (toggle). Data source: activity feed events with `status → done`.
4. **Status distribution chart** — A pie or donut chart showing the breakdown of current statuses for all artifacts of type `ticket` that are not in status `done` or `abandoned`.
5. **Activity feed panel** — The right-hand column displays the existing project activity feed (most recent first), reusing or composing the existing activity feed component.
6. **Summary counts** — Display numeric counts for: total tickets, in-progress tickets, blocked tickets, and artifacts completed this week.
7. **Responsive layout** — On viewports ≥ 1024 px, render a two-column layout (charts left, feed right). Below 1024 px, stack vertically (charts on top, feed below).
8. **Extensibility** — The dashboard must use a widget-slot architecture so that new chart or summary components can be registered without modifying the dashboard page component itself.

### Non-functional

1. **Performance** — Dashboard must render meaningful content (at least summary counts) within 500 ms of route entry on a project with ≤ 500 artifacts.
2. **Accessibility** — Charts must include accessible descriptions (aria-label or sr-only text summarising the data). Colour choices must meet WCAG 2.1 AA contrast.
3. **Bundle size** — Any new charting library added must not increase the gzipped JS bundle by more than 80 KB.

## Acceptance Criteria

- [ ] "Dashboard" appears as the first item in the left nav for a project.
- [ ] Opening a project routes to `/projects/:id/dashboard` by default.
- [ ] Completion velocity chart renders with accurate data sourced from activity feed transitions.
- [ ] Status distribution chart renders showing non-done ticket statuses.
- [ ] Activity feed panel displays recent activity entries and updates via WebSocket push.
- [ ] Summary count widgets show total, in-progress, blocked, and completed-this-week counts.
- [ ] Layout is two-column on desktop (≥ 1024 px) and single-column on narrow viewports.
- [ ] New widgets can be added by registering a component without editing the dashboard page template.
- [ ] Page renders summary counts within 500 ms on a 500-artifact project.
- [ ] Charts have accessible labels; colours pass WCAG AA contrast check.
- [ ] No new charting dependency exceeds 80 KB gzipped.

## Resolved Questions

- Should the completion velocity chart default to daily or weekly granularity, and should the user be able to toggle between them?

> Daily, Weekly and Monthly summaries would be great.

- Should charts be interactive (tooltips on hover, click-to-filter) or purely informational in v1?

> Some interactive features would be great.

- Is there a preferred charting library (e.g., Chart.js, Apache ECharts, lightweight D3 wrapper) given the existing bundle and three.js dependency?

> The Apache ECharts look fantastic.

- Should the activity feed on the dashboard be limited to N most recent entries with a "View all" link, or infinite-scroll?

> Limit to N rows and a view all link.
