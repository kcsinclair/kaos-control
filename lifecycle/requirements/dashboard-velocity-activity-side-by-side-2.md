---
title: 'Dashboard: Velocity and Activity Side-by-Side Layout'
type: requirement
status: blocked
lineage: dashboard-velocity-activity-side-by-side
created: "2026-05-10T00:00:00+10:00"
priority: normal
parent: lifecycle/ideas/dashboard-velocity-activity-side-by-side.md
labels:
    - frontend
    - enhancement
    - usability
    - vue
    - testing
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

# Dashboard: Velocity and Activity Side-by-Side Layout

## Problem

The Completion Velocity chart and the Recent Activity panel on the dashboard are stacked vertically. On typical desktop viewports this forces the user to scroll to see both widgets simultaneously, reducing the dashboard's value as an at-a-glance overview. Key information that should be visible together -- recent throughput trends alongside the latest activity -- is split across the fold.

## Goals / Non-goals

### Goals

- Place the Completion Velocity chart and the Recent Activity panel side by side in a responsive two-column layout so both are visible without scrolling on standard desktop resolutions (>= 1280 px wide).
- Degrade gracefully on narrower viewports: columns should stack vertically below a defined breakpoint so content remains usable on tablets and smaller screens.
- Maintain existing widget functionality (real-time WebSocket updates, chart interactivity, activity feed links) without regression.
- Update all affected component tests and any Playwright / integration tests that assert on dashboard element positions or visibility.

### Non-goals

- Redesigning the content or data sources of either widget.
- Adding new widgets or changing the ordering of other dashboard sections.
- Making the two-column ratio user-configurable or drag-and-drop rearrangeable.
- Backend or API changes -- this is a purely frontend layout change.

## Detailed Requirements

### Functional

#### FR-1: Two-column container

Introduce a two-column layout container that wraps the Completion Velocity widget and the Recent Activity widget. The container must use CSS Grid or Flexbox (no JavaScript-driven layout).

| Column | Content |
|---|---|
| Left | Completion Velocity chart |
| Right | Recent Activity panel |

Both columns should have equal width by default (`1fr 1fr` or `flex: 1`). The container must sit within the existing dashboard layout flow at the same position the two widgets currently occupy.

#### FR-2: Responsive breakpoint

Below a viewport width of **768 px**, the layout must collapse to a single column with the Completion Velocity chart above the Recent Activity panel (preserving the current visual order). The breakpoint value should be defined as a CSS custom property or named constant for maintainability.

#### FR-3: Minimum widget dimensions

Each widget must remain usable at its narrowest rendered width. Specifically:

- The Completion Velocity chart must not render narrower than **360 px** so that axis labels and bars remain legible. If the available column width falls below this minimum, the layout should switch to single-column mode.
- The Recent Activity panel must retain its current minimum content width and not truncate activity entries.

#### FR-4: Chart resize handling

The Completion Velocity chart (ECharts-based) must respond to column width changes. An ECharts `resize()` call must be triggered when:

- The component mounts in the new layout.
- The viewport crosses the responsive breakpoint (layout switch).
- The browser window is resized within two-column mode.

A `ResizeObserver` on the chart container is the preferred mechanism.

#### FR-5: Visual consistency

- Spacing between the two columns must match the existing vertical gap between dashboard widget rows.
- Widget card styling (background, border-radius, padding, shadow) must remain unchanged.
- The two widgets must align at the top of the row (no vertical centring).

### Non-functional

#### NFR-1: No layout shift on load

The two-column layout must not cause visible content layout shift (CLS) during page load. Widget containers should reserve their height via CSS min-height or aspect-ratio hints.

#### NFR-2: Performance

The layout change must not introduce additional JavaScript bundle size beyond what is required for the responsive logic. No new dependencies.

#### NFR-3: Accessibility

- The reading order in the DOM must be logical (velocity then activity) regardless of visual layout mode.
- Widgets must remain navigable via keyboard in the same order as before.

## Acceptance Criteria

- [ ] On viewports >= 1280 px wide, the Completion Velocity chart and Recent Activity panel render side by side in a single row, both fully visible without vertical scrolling (assuming standard dashboard content above).
- [ ] On viewports < 768 px, the two widgets stack vertically with the velocity chart on top.
- [ ] Between 768 px and 1279 px, the side-by-side layout is maintained but columns flex to fill available width; neither widget is narrower than its minimum usable width.
- [ ] Resizing the browser window across the breakpoint transitions the layout smoothly without page reload or visible glitch.
- [ ] The ECharts velocity chart resizes correctly when column width changes (bars and labels remain proportional and legible).
- [ ] No regressions in widget functionality: WebSocket-driven updates, chart tooltips, activity feed links, and [[dashboard-clickable-filters]] click-throughs all work as before.
- [ ] Existing component unit tests are updated to assert the new layout structure.
- [ ] Any Playwright or integration tests that reference dashboard widget positioning or visibility are updated and pass.
- [ ] The layout does not introduce measurable Cumulative Layout Shift (CLS > 0) on initial dashboard load.
- [ ] The DOM reading order remains velocity-then-activity in both layout modes.

## Open Questions

1. **Column width ratio** -- Should both columns be equal width (`1fr 1fr`), or should the Recent Activity panel be narrower (e.g., `3fr 2fr`) since it is a text list while the chart benefits from more horizontal space? Default assumption is equal width pending product-owner input.
2. **Max-height / scroll on activity panel** -- If the activity feed is long, should the Recent Activity panel have a fixed max-height with internal scrolling to match the chart height, or should it grow naturally and potentially extend below the chart? Current behaviour (natural height) is assumed.
