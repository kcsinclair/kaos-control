---
title: Roadmap Gantt Period Display Options
type: requirement
status: approved
lineage: roadmap-gantt-period-options
created: "2026-05-09T00:00:00+10:00"
priority: high
parent: lifecycle/ideas/roadmap-gantt-period-options.md
labels:
    - roadmaps
    - frontend
    - enhancement
    - usability
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

# Roadmap Gantt Period Display Options

## Problem

The Roadmap Gantt view (`GanttChart.vue`) computes its time range by scanning all release dates and padding outward by one column unit on each side. When releases are spread across a wide date range or when many columns have no release bars, the chart wastes horizontal space on empty columns, making it harder to read. Conversely, there is no way for a user to lock the time axis to a predictable, fixed window — the visible range shifts whenever releases are added or rescheduled.

Users need two complementary controls: (1) an autoscale mode that trims the time axis to only the columns actually occupied by release bars, and (2) a fixed-period mode that pins the axis to a user-selected calendar window with horizontal scrolling when content overflows.

## Goals / Non-goals

### Goals

- Give users explicit control over the Gantt time axis via a period-mode selector on the existing Roadmap toolbar.
- Provide an **Autoscale** mode that eliminates empty leading and trailing columns, showing only the range spanned by scheduled releases.
- Provide a **Fixed-period** mode with selectable windows: Month, Quarter, Half-Year, Year — anchored to the current date — with horizontal scrolling when content extends beyond the visible area.
- Default to Autoscale for a clean out-of-the-box experience.
- Persist the user's period-mode selection per session so it survives tab switches within the same session.

### Non-goals

- Arbitrary user-defined date ranges (date-picker). Only the four predefined fixed periods are in scope.
- Persisting the period-mode selection across browser sessions or to the server. Session-level (in-memory / Pinia state) is sufficient.
- Changes to the granularity control — it remains independent of period mode.
- Changes to the Graph view — period options apply only to the Gantt view.
- Modifying how "Unscheduled" releases are displayed — the sticky unscheduled column is unchanged.

## Detailed Requirements

### Functional

**FR-1 Period-mode selector**
Add a segmented control or dropdown to the Roadmap toolbar (alongside the existing granularity control) with two options: **Autoscale** and **Fixed Period**. When Fixed Period is selected, a secondary control appears allowing the user to choose one of: Month, Quarter, Half-Year, Year.

**FR-2 Autoscale mode**
When Autoscale is active, the `timeRange` computation must:
- Set `start` to the earliest `start_date` among all scheduled releases (snapped to the current granularity boundary).
- Set `end` to the latest `end_date` among all scheduled releases (snapped to the current granularity boundary).
- Not add padding columns beyond the snapped boundaries.
- If no releases are scheduled, fall back to showing a single column containing today's date.

**FR-3 Fixed-period mode**
When a fixed period is selected, the `timeRange` computation must:
- Anchor the window start to the beginning of the current calendar period containing today (e.g., start of current month for Month, start of current quarter for Quarter).
- Set the window end to the end of that same period.
- Clamp column generation to this window regardless of release dates.
- Releases with bars partially outside the window must render only the visible portion (clipped at window boundaries), not be hidden entirely.

**FR-4 Horizontal scrolling**
When content overflows the fixed-period window (i.e., more columns would be needed at the current granularity than fit the window, or release bars extend beyond the visible area), the Gantt container must become horizontally scrollable. The release label column on the left should remain fixed (sticky) during horizontal scroll so users always see which release each bar belongs to.

**FR-5 Interaction with granularity**
The period mode and granularity controls are independent. Selecting a fine granularity (e.g., Week) with a large fixed period (e.g., Year) must produce a scrollable chart with many columns — the 200-column safety cap in `GanttChart.vue` must be respected, and if exceeded the granularity should be coarsened automatically with a visual indicator.

**FR-6 Default behaviour**
The default period mode on first load must be **Autoscale**. If the user switches modes, the selection is held in Pinia state for the duration of the session.

**FR-7 URL / route state (optional)**
Period mode and fixed-period value may optionally be reflected in the route query string so that links can be shared with a specific view. This is desirable but not required for the initial implementation.

### Non-functional

**NFR-1 Performance**
Switching period modes must re-render the Gantt within 100 ms for up to 50 releases. No additional API calls are required — all data is already available in the releases store.

**NFR-2 Responsiveness**
The period-mode selector must not cause the toolbar to wrap or overflow on viewports ≥ 1024 px wide. On narrower viewports, controls may stack vertically.

**NFR-3 Accessibility**
All new controls must be keyboard-navigable and include appropriate ARIA labels.

## Acceptance Criteria

- [ ] A period-mode selector (Autoscale / Fixed Period) is visible on the Roadmap toolbar when Gantt view is active.
- [ ] Selecting Autoscale trims the time axis to exactly the range spanned by scheduled releases — no empty leading/trailing columns.
- [ ] Autoscale with no scheduled releases displays a single column containing today.
- [ ] Selecting Fixed Period → Month shows only the current calendar month.
- [ ] Selecting Fixed Period → Quarter shows only the current calendar quarter.
- [ ] Selecting Fixed Period → Half-Year shows only the current half-year.
- [ ] Selecting Fixed Period → Year shows only the current calendar year.
- [ ] Release bars that extend outside a fixed-period window are clipped at the window boundary, not hidden.
- [ ] Horizontal scrolling activates when the Gantt content overflows the fixed-period window.
- [ ] The release label column remains sticky (fixed left) during horizontal scroll.
- [ ] The default mode on first load is Autoscale.
- [ ] Switching between modes does not trigger additional API requests.
- [ ] The period mode selection persists across tab switches within the same session.
- [ ] Granularity and period mode operate independently — all combinations render correctly.
- [ ] The 200-column safety cap is respected; exceeding it coarsens granularity automatically.
- [ ] All new controls are keyboard-navigable with ARIA labels.
- [ ] Toolbar does not overflow on viewports ≥ 1024 px.

Related: [[roadmap-gantt-period-options]]

## Resolved Questions

1. Should the fixed-period window anchor be configurable (e.g., "this quarter" vs. "next quarter"), or is the current period always sufficient for the initial implementation?

> Current period for initial implementation.

2. When a release bar is clipped at the window boundary, should there be a visual indicator (e.g., arrow or fade) to signal that the bar extends beyond the visible area?

> Yes a visual indicator when clipped is good.

3. Should the period-mode default be configurable in `lifecycle/config.yaml`, or is hardcoding Autoscale acceptable?

> Yes, make it configurable.
