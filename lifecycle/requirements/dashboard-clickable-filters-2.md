---
title: Dashboard Items as Clickable Filters
type: requirement
status: blocked
lineage: dashboard-clickable-filters
created: "2026-05-09"
priority: normal
parent: lifecycle/ideas/dashboard-clickable-filters.md
labels:
    - frontend
    - enhancement
    - usability
    - vue
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

# Dashboard Items as Clickable Filters

## Problem

The dashboard currently displays summary counts, status distribution, velocity data, and an activity feed as read-only widgets. When a user sees "3 Blocked" or a pie-chart wedge for "in-development", they must manually navigate to the artifacts list, then select the correct filter values to see the relevant artifacts. This context-switch adds friction during triage and means the dashboard is a passive summary rather than an actionable launchpad.

## Goals / Non-goals

### Goals

- Every numeric count and chart segment on the dashboard that represents a queryable subset of artifacts becomes a clickable link that navigates to the artifacts list view (`/p/:project/artifacts`) with the matching filter(s) pre-applied.
- The navigation must use the existing `ArtifactFilter` query-parameter interface so that the resulting list view is a normal filtered view (bookmarkable, shareable, back-button friendly).
- The user should perceive the click targets as interactive (cursor, hover state, accessible focus ring).

### Non-goals

- Inline expansion of artifact lists within dashboard widgets (out of scope; the target is always the list view).
- Adding new filter dimensions to the artifacts list view. Only existing filter parameters (`status`, `stage`, `type`, `label`, `priority`, `release`, `q`, `lineage`) are used.
- Changes to the Velocity Chart widget. Its bars represent time-bucketed completion counts, not a filterable artifact subset, so they are excluded.
- Changes to the Activity Feed widget. Individual feed entries already link to their artifact; no additional filter navigation is required.
- Backend API changes. All required data is already available on the frontend.

## Detailed Requirements

### Functional

#### FR-1: SummaryCountsWidget click-through

Each `SummaryCountCard` in the Summary Counts widget must be clickable and navigate to the artifacts list with the filter that produces the same set of artifacts the count represents.

| Card | Filter applied |
|---|---|
| Lifecycle Total | _(no filter — show all artifacts)_ |
| In Progress | `status=in-development` |
| Blocked | `status=blocked` |
| Completed This Week | `status=done` (note: the backend stat already scopes to "this week"; the list view will show all `done` artifacts — see Open Questions) |

Navigation must use `router.push` with query parameters, not `window.location`.

#### FR-2: StatusDistributionWidget click-through

Each segment (wedge/slice) of the status-distribution pie chart must be clickable. Clicking a segment navigates to the artifacts list filtered by `status=<segment status value>` (e.g., `status=clarifying`).

The ECharts click event (`bindEvents` / `bindEvents` / `bindEvents` / chart `click` handler) must be used; the handler must resolve the clicked segment's status key and call `router.push`.

#### FR-3: Filter parameters in URL

Navigations must produce URLs of the form:

```
/p/{project}/artifacts?status=blocked
```

These URLs must be valid when loaded directly (deep-link / bookmark). The artifacts list view already reads filters from query parameters, so no new parsing logic should be needed.

#### FR-4: Visual affordance

All clickable elements must indicate interactivity:

- `SummaryCountCard`: `cursor: pointer` on the card; subtle hover state (e.g., slight elevation or background shift). The entire card is the click target.
- Status Distribution pie chart: `cursor: pointer` on segments (ECharts supports per-item cursor config). Segment highlight on hover (ECharts default emphasis is acceptable).

#### FR-5: Accessibility

- `SummaryCountCard` click targets must be keyboard-focusable and activatable with Enter/Space.
- Appropriate `role` and `aria-label` attributes must be present (e.g., `role="link"`, `aria-label="View 3 blocked artifacts"`).
- Pie chart segments are exempt from keyboard navigation requirements in this iteration (ECharts canvas limitation), but the chart container should include an `aria-label` describing that segments are clickable.

### Non-functional

#### NFR-1: No new API calls

Click-through navigation must not introduce additional backend requests beyond what the artifacts list view already issues when filters are applied.

#### NFR-2: Performance

Router navigation from dashboard to filtered list must feel instantaneous (< 100 ms to route change). No loading spinner should appear on the dashboard side.

#### NFR-3: Maintainability

Filter mappings (card → query params, chart segment → query params) should be co-located with the widget that uses them, not buried in a shared utility. If a new widget is added later, its click-through behaviour should follow the same pattern without modifying shared code.

## Acceptance Criteria

- [ ] Clicking each Summary Counts card navigates to `/p/:project/artifacts` with the correct `status` filter pre-applied in the URL.
- [ ] Clicking a pie-chart segment in the Status Distribution widget navigates to the artifacts list filtered to that status.
- [ ] The artifacts list view displays the correct filtered results after navigation (no extra manual filtering needed).
- [ ] The filtered URL is bookmarkable: loading it directly in a new tab produces the same filtered list.
- [ ] Browser back-button returns the user to the dashboard after a click-through navigation.
- [ ] All Summary Count cards show `cursor: pointer` and a visible hover/focus state.
- [ ] Summary Count cards are keyboard-accessible (Tab to focus, Enter/Space to activate).
- [ ] No new backend API endpoints or modifications are required.
- [ ] Existing dashboard widget functionality (real-time WebSocket updates, responsive layout) is unaffected.

## Open Questions

1. **"Completed This Week" scope mismatch** — The Summary Counts widget's "Completed This Week" card counts artifacts completed in the current week (server-side), but the artifacts list `status=done` filter shows _all_ done artifacts regardless of completion date. Should we (a) add a date-range filter parameter to the list view to scope to the current week, (b) accept the mismatch and link to all `done` artifacts, or (c) omit the click-through on this card until date filtering exists?

2. **Multiple-status filters** — The "In Progress" card currently maps to `status=in-development`. Should it also include `status=in-qa` and `status=in-progress`, or is `in-development` the sole intended status? If multiple statuses are desired, the list view's filter interface would need to support multi-value status parameters (e.g., `status=in-development&status=in-qa`).
