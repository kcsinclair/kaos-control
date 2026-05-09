---
title: Stages Distribution Pie Chart Widget
type: requirement
status: done
lineage: stages-distribution-pie-chart
created: "2026-05-09T00:00:00+10:00"
priority: high
parent: lifecycle/ideas/stages-distribution-pie-chart.md
labels:
    - feature
    - frontend
    - backend
    - vue
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

# Stages Distribution Pie Chart Widget

## Problem

The dashboard provides a Status Distribution donut chart showing how many artifacts sit in each workflow status (draft, clarifying, planning, etc.), but there is no equivalent visualisation for **lifecycle stages** — the directory-based categories (ideas, requirements, backend-plans, frontend-plans, test-plans, tests, releases, etc.). A user who wants to understand whether work is bunching up in early stages (ideas, requirements) versus progressing through plans and into tests must manually count artifacts across directories or mentally aggregate the artifact list. This friction reduces the dashboard's value as a project-health overview.

## Goals / Non-goals

### Goals

- Add a **Stages Distribution** pie chart widget to the dashboard that shows the count of artifacts in each lifecycle stage.
- Position the widget alongside the existing Status Distribution widget in the chart slot so both distributions are visible at a glance.
- Make pie slices clickable: clicking a slice navigates to the artifacts list view filtered to the selected stage, consistent with the click-through pattern being introduced in [[dashboard-clickable-filters]].
- Match the visual language of existing widgets (card styling, font sizes, colour palette, donut shape, responsive sizing).

### Non-goals

- Replacing or altering the existing Status Distribution widget.
- Adding a new "stage" filter dimension to the artifacts list view — the `stage` filter parameter already exists.
- Showing stage distribution over time (velocity-style); this is a point-in-time snapshot.
- Displaying stages that contain zero artifacts (empty slices add clutter; if needed later, this can be a follow-up).
- Custom colour configuration per stage in project config; colours are defined in the widget code.

## Detailed Requirements

### Functional

#### FR-1: Backend endpoint — stage distribution

Add a new API endpoint:

```
GET /api/p/:project/dashboard/stage-distribution
```

Response shape:

```json
{
  "distribution": [
    { "stage": "ideas", "count": 5 },
    { "stage": "requirements", "count": 12 },
    ...
  ]
}
```

Implementation notes:

- Query the existing `artifacts` table grouped by the `stage` column: `SELECT stage, COUNT(*) FROM artifacts WHERE type IN (<trackedTypes>) GROUP BY stage ORDER BY stage`.
- Respect `Dashboard.TrackedTypes` from project config, consistent with the other dashboard endpoints.
- Return an empty (non-nil) array when no artifacts exist.
- Do **not** exclude any status values (unlike `StatusDistribution` which excludes `done`/`abandoned`). All artifacts count toward their stage regardless of status.

#### FR-2: Frontend widget — StagesDistributionWidget

Create `web/src/components/dashboard/widgets/StagesDistributionWidget.vue`:

- Fetch data from the stage-distribution endpoint on mount and when the `project` prop changes.
- Render an ECharts donut chart (same library already used by `StatusDistributionWidget`).
- Chart configuration: donut shape with inner radius 40%, outer radius 70%, horizontal legend at bottom, tooltip showing `{stage}: {count} ({percent}%)`.
- Assign a distinct colour to each stage using a WCAG 2.1 AA compliant palette. Colours should be visually distinguishable from the status palette used in `StatusDistributionWidget`.
- Display "No artifacts yet" placeholder when the distribution is empty or all counts are zero.

#### FR-3: Click-through navigation

Each pie slice must be clickable. Clicking a slice navigates to:

```
/p/:project/artifacts?stage=<stage-value>
```

- Use `router.push` (not `window.location`).
- The ECharts `click` event on the series must resolve the clicked slice's `stage` value and navigate.
- The resulting URL must be bookmarkable and produce the correct filtered list when loaded directly.

#### FR-4: Widget registration and layout

- Register the widget in `registerWidgets.ts` with id `stages-distribution`, slot `chart`, order `0.5` (or another value that positions it between the existing Status Distribution and Velocity Chart widgets — i.e., immediately after Status Distribution).
- The widget must render correctly in both the single-column (< 1024 px) and two-column (>= 1024 px) dashboard layouts.

#### FR-5: Visual design

- Use the same card styling as other widgets: `var(--color-surface)` background, `var(--color-border)` border, `var(--radius-lg)` border radius, `var(--space-4)` padding.
- Widget title: "Stages Distribution".
- Chart height: 280 px (matching Status Distribution).
- `cursor: pointer` on chart segments to signal interactivity.

#### FR-6: Accessibility

- The chart container must have `role="img"` and a computed `aria-label` describing the distribution (e.g., "Stages distribution: ideas 5, requirements 12, ...").
- Pie chart segment keyboard navigation is exempt in this iteration (ECharts canvas limitation), but the container `aria-label` should mention that segments are clickable.

### Non-functional

#### NFR-1: Performance

- The backend query must execute in under 50 ms on a project with up to 1 000 artifacts (the `stage` column is already indexed).
- The widget must lazy-load via `defineAsyncComponent` to avoid increasing initial bundle size.

#### NFR-2: Responsiveness

- The chart must resize correctly when the viewport changes, using `ResizeObserver` (same pattern as `StatusDistributionWidget`).

#### NFR-3: Consistency

- The widget must follow the same architectural patterns as `StatusDistributionWidget`: ECharts tree-shaking, async component registration, `ResizeObserver` cleanup, `onUnmounted` chart disposal.

## Acceptance Criteria

- [ ] `GET /api/p/:project/dashboard/stage-distribution` returns a JSON object with a `distribution` array of `{ stage, count }` objects.
- [ ] The endpoint returns an empty array (not null) when no artifacts exist.
- [ ] The endpoint respects `Dashboard.TrackedTypes` from project config.
- [ ] A "Stages Distribution" donut chart widget appears on the dashboard between the Status Distribution and Velocity Chart widgets.
- [ ] Each lifecycle stage with at least one artifact is represented as a slice with the correct count.
- [ ] Stages with zero artifacts are omitted from the chart.
- [ ] Clicking a pie slice navigates to `/p/:project/artifacts?stage=<stage>` with the correct filter applied.
- [ ] The artifacts list view displays the correct filtered results after navigation.
- [ ] The filtered URL is bookmarkable: loading it directly in a new tab produces the same filtered list.
- [ ] Browser back-button returns the user to the dashboard after click-through navigation.
- [ ] `cursor: pointer` is shown on chart segments.
- [ ] The chart container has a descriptive `aria-label`.
- [ ] The widget displays "No artifacts yet" when there are no artifacts.
- [ ] The widget renders correctly at viewport widths below and above 1024 px.
- [ ] The chart resizes correctly on window resize.
- [ ] The widget is lazy-loaded (does not appear in the initial JS bundle).
- [ ] Existing dashboard widgets (Status Distribution, Velocity Chart, Summary Counts, Activity Feed) are unaffected.

## Resolved Questions

1. **Should done/abandoned artifacts be included in the stage distribution?** The Status Distribution widget excludes them. For stage distribution, including all artifacts gives a truer picture of where work has accumulated historically. This requirement specifies including all statuses, but the product owner may want to align with the Status Distribution's exclusion behaviour.

> Exclude done/abandoned artifacts

2. **Widget ordering mechanism** — The current `registerWidget` order field is an integer. Placing this widget between order 0 (Status Distribution) and order 1 (Velocity Chart) requires either fractional ordering or renumbering. The implementation should choose whichever approach the widget registry supports.

> Renumber
