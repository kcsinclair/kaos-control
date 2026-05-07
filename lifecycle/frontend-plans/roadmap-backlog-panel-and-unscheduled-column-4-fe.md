---
title: Frontend Plan — Roadmap Backlog Panel and Unscheduled Column
type: plan-frontend
status: in-development
lineage: roadmap-backlog-panel-and-unscheduled-column
priority: high
parent: lifecycle/requirements/roadmap-backlog-panel-and-unscheduled-column-2.md
release: May2026
---

# Frontend Plan — Roadmap Backlog Panel and Unscheduled Column

This plan covers all Vue 3 / TypeScript frontend changes: refactoring the Gantt chart to render unscheduled releases as a sticky column instead of flat cards, and adding a collapsible Backlog panel below the chart for artifacts without a release assignment.

Cross-references: [[roadmap-backlog-panel-and-unscheduled-column]] (backend plan — no API changes expected), [[roadmap-backlog-panel-and-unscheduled-column]] (test plan for integration and visual tests).

---

## Milestone 1 — Unscheduled Column on the Gantt Timeline

### Description

Replace the current "Unscheduled" card list below the Gantt rows with a dedicated column at the right edge of the time axis. This column is only rendered when unscheduled releases exist (FR1.5), has no date label (FR1.2), matches single-column width (FR1.3), and is visually separated by a heavier/dashed border (FR1.4). The column must be sticky on the right edge during horizontal scroll (OQ3 resolution).

### Files to change

- `web/src/components/releases/GanttChart.vue`
  - **Template**: Remove the existing `<template v-if="unscheduled.length > 0">` block (lines 296–310) that renders `.unscheduled-heading` and `.unscheduled-cards`.
  - **Template**: Add an "Unscheduled" column header cell to `.gantt-header`, conditionally rendered when `unscheduled.length > 0`. The cell should use text "Unscheduled" instead of a date label, and be styled as `position: sticky; right: 0` for horizontal scroll pinning.
  - **Template**: In each `.gantt-row` for scheduled bars, add a blank sticky cell on the right to maintain the grid when the Unscheduled column is visible.
  - **Script**: Add a computed `hasUnscheduled` boolean for template conditionals.
  - **Style**: Add `.col-header--unscheduled` with sticky positioning, `border-left: 2px dashed var(--color-border)` as the visual divider, and matching single-column width.

### Acceptance criteria

- [ ] When unscheduled releases exist, an "Unscheduled" column appears at the far right of the time axis header.
- [ ] The column has no date label — only the text "Unscheduled".
- [ ] The column width matches a single time-unit column at the current granularity.
- [ ] A dashed vertical border separates the Unscheduled column from the last dated column (FR1.4).
- [ ] The column is `position: sticky; right: 0` so it remains visible during horizontal scroll (OQ3).
- [ ] When no unscheduled releases exist, the column is not rendered (FR1.5).
- [ ] The old `.unscheduled-heading` and `.unscheduled-cards` sections are removed.

---

## Milestone 2 — Unscheduled Release Bars

### Description

Render each unscheduled release as a Gantt row with its bar positioned within the Unscheduled column. Bars must be visually distinct (muted/hatched fill — FR2.2), stack vertically (FR2.3), support click-to-detail (FR2.4), and show summary badges (FR2.5).

### Files to change

- `web/src/components/releases/GanttChart.vue`
  - **Template**: Below the scheduled rows, add a `<div v-for="r in unscheduledSorted" ...>` block that renders `.gantt-row` elements. Each row has an empty track area (spanning the dated columns) and a sticky Unscheduled cell containing the release bar `<button>`.
  - **Script**: Add a computed `unscheduledSorted` that sorts `unscheduled` alphabetically by `r.name` (FR2.3).
  - **Script**: Reuse `summaryBadge()` and `statusColor()` for badge display — ensure unscheduled bars call `summaryBadge(r)` (FR2.5).
  - **Template**: The bar button emits `clickRelease` with the release ID (FR2.4), same as scheduled bars.
  - **Style**: Add `.release-bar--unscheduled` with a CSS hatched pattern (`repeating-linear-gradient`) or muted opacity to visually distinguish from scheduled bars (FR2.2). Use the same `statusColor` for the base but overlay the pattern.

### Acceptance criteria

- [ ] Each unscheduled release appears as a row with its bar inside the Unscheduled column.
- [ ] Unscheduled bars use a hatched/muted visual style distinct from scheduled bars (FR2.2).
- [ ] Rows are ordered alphabetically by release name (FR2.3).
- [ ] Clicking an unscheduled bar emits `clickRelease` and opens `ReleaseDetailModal` (FR2.4).
- [ ] Summary badge (idea count, defect count) is shown on unscheduled bars when data is available (FR2.5).
- [ ] No layout shifts when releases are added/removed via WebSocket updates (NFR1).

---

## Milestone 3 — Backlog Panel Component

### Description

Create a new `BacklogPanel.vue` component that lists all artifacts with no `release` field, excluding `release` and `sprint` types (FR3.2). The panel is collapsible (FR3.7), shows a count header (FR3.5), and supports filtering by type, status, and priority (OQ1 resolution).

### Files to change

- `web/src/components/releases/BacklogPanel.vue` (new)
  - **Props**: `project: string`, `artifacts: ArtifactRow[]` (pre-filtered backlog items passed from parent).
  - **State**: `collapsed: boolean` (default `true`, FR3.7), persisted to `sessionStorage` under key `backlog-panel-collapsed` (NFR4). Filter state: `filterType`, `filterStatus`, `filterPriority` (dropdowns, OQ1).
  - **Template**:
    - Header bar: `<button>` toggling collapse, showing "Backlog (N)" count (FR3.5), plus filter dropdowns (type, status, priority) visible only when expanded.
    - Scrollable list of `<button class="backlog-card">` items (FR3.3): title, type badge (coloured), status badge, lineage slug. Left-border colour by status, consistent with artifact cards elsewhere (FR3.4).
    - Empty state message when no items match filters (FR3.6).
  - **Emits**: `openArtifact(path: string)` — parent handles navigation.
  - **Style**: Consistent with existing card styles. Use semantic `<button>` elements for keyboard accessibility (NFR3). Keyboard navigation: cards are focusable, Enter/Space activates.

- `web/src/views/project/RoadmapView.vue`
  - **Script**: Import `BacklogPanel` and `useArtifactsStore`. On mount (alongside release fetch), call `artifactsStore.fetchList(project, { limit: 500 })` to load artifacts. Compute `backlogArtifacts` as items where `fm.release` is empty/null and `type` is not `release` or `sprint` (FR3.2, FR5.1).
  - **Script**: Subscribe to WebSocket `artifact.indexed` events. On receiving one, re-fetch the artifact list to update the Backlog reactively (FR5.2, FR4.2, FR4.3).
  - **Script**: Add `onOpenBacklogArtifact(path)` handler that calls `router.push({ name: 'artifact', params: { project, path } })` (FR4.1).
  - **Template**: Place `<BacklogPanel>` below `<GanttChart>`, passing `backlogArtifacts` and handling `@openArtifact`.

### Acceptance criteria

- [ ] A collapsible "Backlog" panel appears below the Gantt chart (FR3.1).
- [ ] The panel defaults to collapsed; collapsed/expanded state persists in `sessionStorage` within the session (FR3.7, NFR4).
- [ ] The header shows "Backlog (N)" with the count of matching items (FR3.5).
- [ ] Each card displays title, type badge, status badge, and lineage slug (FR3.3).
- [ ] Cards use left-border colour by status, matching existing artifact card styling (FR3.4).
- [ ] Filter dropdowns for type, status, and priority are present and functional (OQ1).
- [ ] Empty state message is shown when no artifacts match (FR3.6).
- [ ] Clicking a card navigates to the artifact editor via `router.push` (FR4.1).
- [ ] All interactive elements are `<button>` elements and keyboard-navigable (NFR3).

---

## Milestone 4 — Reactive Backlog Updates via WebSocket

### Description

Ensure the Backlog panel updates reactively when artifacts gain or lose a `release` assignment, without requiring a page reload. This milestone wires up WebSocket event handling and verifies the end-to-end reactive flow.

### Files to change

- `web/src/views/project/RoadmapView.vue`
  - **Script**: In the existing WebSocket connection (managed by `releasesStore.connectWs`), listen for `artifact.indexed` events. The releases store already handles `release.*` events; extend the WS message handler (or add a parallel listener) to trigger `artifactsStore.fetchList(project, { limit: 500 })` on `artifact.indexed` events. This re-fetches the full list, and the computed `backlogArtifacts` filter will automatically add/remove items.
  - Alternatively, if the releases store's WS connection is not easily extended, open a second WS connection from `RoadmapView` specifically for `artifact.indexed` events, or refactor the store to expose an `onMessage` callback.

- `web/src/stores/releases.ts` (minor)
  - If needed, add an `onWsMessage(callback)` registration method so `RoadmapView` can subscribe to raw WS messages without opening a duplicate connection.

### Acceptance criteria

- [ ] Assigning a `release` to a backlog artifact and saving causes the artifact to disappear from the Backlog panel without page reload (FR4.2).
- [ ] Clearing an artifact's `release` field causes it to appear in the Backlog panel without page reload (FR4.3).
- [ ] No duplicate WebSocket connections are opened.
- [ ] Updates do not cause layout shifts or reflows in the Gantt chart or Backlog panel (NFR1).

---

## Milestone 5 — Performance and Accessibility Polish

### Description

Ensure the Backlog panel handles large artifact counts (up to 500, NFR2) without scroll jank, and verify all keyboard accessibility requirements (NFR3).

### Files to change

- `web/src/components/releases/BacklogPanel.vue`
  - If performance testing reveals scroll jank with 500 cards, implement a virtualised list (e.g., a simple CSS `contain: content` approach or a lightweight virtual scroll utility). Only add virtualisation if needed — test first.
  - Verify `tabindex`, `role`, and `aria-label` attributes on all interactive elements.
  - Ensure the collapse/expand toggle has `aria-expanded` and `aria-controls` attributes.

### Acceptance criteria

- [ ] The Backlog panel scrolls smoothly with 500 items (NFR2).
- [ ] All interactive elements (cards, filter dropdowns, collapse toggle) are reachable via Tab and activatable via Enter/Space (NFR3).
- [ ] Collapse toggle has `aria-expanded` and `aria-controls` attributes.
- [ ] Semantic HTML is used throughout — no `<div>` click handlers for interactive elements (NFR3).
