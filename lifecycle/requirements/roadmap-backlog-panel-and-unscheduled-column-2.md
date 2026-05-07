---
title: Roadmap Backlog Panel and Unscheduled Column
type: requirement
status: approved
lineage: roadmap-backlog-panel-and-unscheduled-column
parent: ideas/roadmap-backlog-panel-and-unscheduled-column.md
labels:
    - feature
    - frontend
    - roadmaps
    - usability
    - vue
release: May2026
assignees:
    - role: product-owner
      who: agent
---

# Roadmap Backlog Panel and Unscheduled Column

## Problem

The Roadmap Gantt view has two usability gaps:

1. **Unscheduled releases are invisible on the timeline.** Releases without dates are listed as flat cards beneath the Gantt chart, disconnected from the time axis. Users cannot visually compare unscheduled work against time-boxed releases or drag/plan it into the timeline context.

2. **Artifacts not assigned to any release are invisible on the Roadmap.** There is no surface in the Roadmap view that shows artifacts (ideas, tickets, etc.) that have no `release` field. Users must leave the Roadmap and navigate to the Artifact List to discover unassigned work, breaking the planning flow.

The current "Unscheduled" card list conflates two distinct concepts — releases without dates and artifacts without a release — under one label, making neither easy to act on.

## Goals / Non-goals

### Goals

- G1: Display the "Unscheduled" release as a distinct, dateless column at the right edge of the Gantt chart so it is visually co-located with time-boxed releases.
- G2: Render an Unscheduled Gantt bar anchored to the bottom-right of the chart area, styled to be visually distinct from date-bound bars (no time span implied).
- G3: Replace the existing beneath-the-chart "Unscheduled" card section with a "Backlog" panel that lists artifacts not assigned to any release.
- G4: Allow users to click any Backlog card to open the artifact edit modal, enabling in-place release assignment without leaving the Roadmap.

### Non-goals

- N1: Drag-and-drop from the Backlog panel onto a Gantt bar or column (future enhancement).
- N2: Inline editing of artifact fields directly in the Backlog card (clicking opens the existing modal/editor).
- N3: Changes to the RoadmapGraphView (3D/2D force-graph); this requirement targets the Gantt view only.
- N4: Backend API changes for artifact querying — the frontend already has access to artifact data via existing endpoints.

## Detailed Requirements

### Functional

#### FR1 — Unscheduled Column on the Gantt Timeline

- FR1.1: When one or more releases lack both `start_date` and `end_date`, the Gantt chart MUST render an additional column at the far right of the time axis, labelled "Unscheduled".
- FR1.2: The Unscheduled column MUST have no date label in the header; it uses the text "Unscheduled" instead.
- FR1.3: The Unscheduled column width MUST match the width of a single time-unit column at the current granularity.
- FR1.4: The Unscheduled column MUST be separated from the last dated column by a visual divider (e.g., a heavier or dashed vertical border) to indicate the break from the time axis.
- FR1.5: When no unscheduled releases exist, the Unscheduled column MUST NOT be rendered.

#### FR2 — Unscheduled Release Bars

- FR2.1: Each unscheduled release MUST be rendered as a Gantt row with its bar positioned within the Unscheduled column.
- FR2.2: Unscheduled bars MUST be visually distinct from scheduled bars — use a muted or hatched fill pattern, or a distinct colour from the `statusColor` palette, to communicate the absence of a time commitment.
- FR2.3: Unscheduled bars MUST stack vertically in the Unscheduled column (one row per release), ordered alphabetically by release name.
- FR2.4: Clicking an unscheduled release bar MUST emit `clickRelease` and open the `ReleaseDetailModal`, identical to clicking a scheduled bar.
- FR2.5: The release detail badge (idea count, defect count) MUST be shown on unscheduled bars when the data is available, consistent with scheduled bars.

#### FR3 — Backlog Panel

- FR3.1: Below the Gantt chart (in the position currently occupied by the "Unscheduled" cards), the view MUST render a collapsible panel labelled "Backlog".
- FR3.2: The Backlog panel MUST list all artifacts in the project that have no `release` frontmatter field (or an empty/null value), excluding artifact types `release` and `sprint` (which are organisational, not work items).
- FR3.3: Each Backlog card MUST display: artifact title, type badge, status badge, and lineage slug.
- FR3.4: Backlog cards MUST be visually consistent with artifact cards used elsewhere in the application (left-border colour by status, consistent typography).
- FR3.5: The panel MUST show a count of backlog items in its header (e.g., "Backlog (12)").
- FR3.6: When the Backlog is empty, the panel MUST display an empty-state message (e.g., "All artifacts are assigned to a release.").
- FR3.7: The Backlog panel MUST default to collapsed state to keep the Gantt chart prominent. The collapsed/expanded state SHOULD persist across page navigations within the session.

#### FR4 — Backlog Card Interaction

- FR4.1: Clicking a Backlog card MUST open the artifact in the artifact editor (via `router.push` to `/p/:project/artifacts/:path`), consistent with how `ReleaseDetailModal` opens artifacts.
- FR4.2: When the user assigns a `release` to a Backlog artifact and saves, the artifact MUST disappear from the Backlog panel on the next WebSocket `artifact.indexed` event without requiring a page reload.
- FR4.3: Conversely, when an artifact's `release` field is cleared, it MUST appear in the Backlog panel upon the next index update.

#### FR5 — Data Sourcing

- FR5.1: The Backlog panel MUST source its data from the existing artifact list API (`GET /p/:project/artifacts`), filtering client-side for artifacts where `release` is absent/null and `type` is not `release` or `sprint`.
- FR5.2: The panel MUST refresh reactively via the existing WebSocket `artifact.indexed` events — no polling.

### Non-functional

- NFR1: The Unscheduled column and Backlog panel MUST render without layout shifts or reflows when the release list or artifact list updates via WebSocket.
- NFR2: The Backlog panel MUST handle up to 500 artifacts without perceptible scroll jank (virtualised list if needed).
- NFR3: All new UI elements MUST be keyboard-navigable and use semantic HTML (buttons, not divs with click handlers, for interactive elements).
- NFR4: The Backlog panel collapsed/expanded state MUST NOT be persisted to the server; session-level state (e.g., `sessionStorage` or Pinia store) is sufficient.

## Acceptance Criteria

- [ ] Unscheduled releases appear as rows in a rightmost "Unscheduled" column on the Gantt chart, not as cards below it
- [ ] The Unscheduled column is visually separated from dated columns and has no date header
- [ ] Unscheduled release bars are visually distinct (colour/pattern) from scheduled bars
- [ ] Clicking an unscheduled bar opens `ReleaseDetailModal` with full release details
- [ ] The old "Unscheduled" cards section below the Gantt chart is removed
- [ ] A collapsible "Backlog" panel appears below the Gantt chart showing artifacts with no release assignment
- [ ] Backlog panel header shows item count (e.g., "Backlog (N)")
- [ ] Each Backlog card shows title, type, status, and lineage
- [ ] Clicking a Backlog card navigates to the artifact editor for that artifact
- [ ] Assigning a release to a Backlog artifact causes it to disappear from the panel reactively (via WebSocket)
- [ ] Removing a release assignment from an artifact causes it to appear in the Backlog panel reactively
- [ ] Backlog panel defaults to collapsed; state persists within the session
- [ ] Empty Backlog shows a meaningful empty-state message
- [ ] All interactive elements are keyboard-accessible
- [ ] Related: [[roadmap-backlog-panel-and-unscheduled-column]]

## Resolved Questions

- OQ1: Should the Backlog panel support filtering or sorting (e.g., by type, status, priority), or is a flat list sufficient for v1?

> A filter with type, status and priority would be great.

- OQ2: Should the Unscheduled column width scale if there are many unscheduled releases, or remain fixed at one column-width with vertical stacking only?

> Fixed with with vertical stacking.

- OQ3: When the Gantt view mode is set to a narrow granularity (e.g., week) with many columns, should the Unscheduled column be pinned/sticky on the right edge during horizontal scroll?

> Yes.
