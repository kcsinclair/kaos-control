---
title: Test Plan — Roadmap Backlog Panel and Unscheduled Column
type: plan-test
status: done
lineage: roadmap-backlog-panel-and-unscheduled-column
priority: high
parent: lifecycle/requirements/roadmap-backlog-panel-and-unscheduled-column-2.md
release: May2026
---

# Test Plan — Roadmap Backlog Panel and Unscheduled Column

This plan covers integration tests for the Unscheduled column on the Gantt chart and the Backlog panel. Tests run against a live kaos-control instance with a test project containing fixtures for scheduled releases, unscheduled releases, and artifacts with/without release assignments.

Cross-references: [[roadmap-backlog-panel-and-unscheduled-column]] (frontend plan for implementation details), [[roadmap-backlog-panel-and-unscheduled-column]] (backend plan — no API changes, but verification gates inform test assumptions).

---

## Milestone 1 — Test Fixtures and Helpers

### Description

Create test fixtures: a project directory with lifecycle artifacts covering the required scenarios — scheduled releases, unscheduled releases, artifacts assigned to releases, and artifacts with no release. Add helper functions for navigating to the Roadmap Gantt view, querying DOM elements, and waiting for WebSocket-driven updates.

### Files to change

- `tests/fixtures/roadmap-backlog/` (new directory)
  - `lifecycle/config.yaml` — minimal project config.
  - `lifecycle/releases/` — at least 2 scheduled releases (with dates) and 2 unscheduled releases (no dates).
  - `lifecycle/ideas/` — at least 3 idea artifacts: 1 assigned to a scheduled release, 1 assigned to an unscheduled release, 1 with no release field.
  - `lifecycle/requirements/` — at least 1 requirement with no release field (to verify multiple types appear in Backlog).
- `tests/helpers/roadmap.ts` or `tests/helpers/roadmap.go` (new)
  - Helper to navigate to `/p/<project>/roadmap` and wait for Gantt chart render.
  - Helper to wait for WebSocket-driven DOM updates (poll for element presence/absence with timeout).

### Acceptance criteria

- [ ] Fixture project loads successfully in kaos-control and appears on the Roadmap view.
- [ ] Fixtures include both scheduled and unscheduled releases.
- [ ] Fixtures include artifacts with release assignments, without release assignments, and of type `release`/`sprint` (which should be excluded from Backlog).

---

## Milestone 2 — Unscheduled Column Tests

### Description

Verify the Unscheduled column renders correctly on the Gantt timeline, including conditional rendering, visual separation, and sticky behaviour.

### Files to change

- `tests/roadmap-backlog-gantt-column_test.go` or `tests/roadmap-backlog-gantt-column.test.ts` (new)
  - **Test: Unscheduled column renders when unscheduled releases exist (FR1.1)**
    - Load fixture project with unscheduled releases.
    - Navigate to Roadmap Gantt view.
    - Assert an element with text "Unscheduled" exists in the Gantt header row.
    - Assert the column has no date label.
  - **Test: Unscheduled column is absent when no unscheduled releases exist (FR1.5)**
    - Load fixture project where all releases have dates.
    - Navigate to Roadmap Gantt view.
    - Assert no "Unscheduled" column header exists.
  - **Test: Unscheduled column has visual divider (FR1.4)**
    - Assert the Unscheduled column header/cell has a distinct left border style (dashed or heavier than regular column borders).
  - **Test: Unscheduled column is sticky during horizontal scroll (OQ3)**
    - Scroll the Gantt chart horizontally.
    - Assert the Unscheduled column remains visible at the right edge.

- `lifecycle/tests/roadmap-backlog-panel-and-unscheduled-column-gantt-test.md` (new artifact)
  - Test artifact documenting what this test file covers.

### Acceptance criteria

- [ ] Test confirms Unscheduled column appears when unscheduled releases exist.
- [ ] Test confirms Unscheduled column is hidden when no unscheduled releases exist.
- [ ] Test confirms visual divider is present.
- [ ] Test confirms sticky positioning during scroll.

---

## Milestone 3 — Unscheduled Release Bar Tests

### Description

Verify unscheduled releases render as bars within the Unscheduled column with correct styling, ordering, interaction, and badge display.

### Files to change

- `tests/roadmap-backlog-release-bars_test.go` or `tests/roadmap-backlog-release-bars.test.ts` (new)
  - **Test: Each unscheduled release renders as a row with a bar (FR2.1)**
    - Assert the number of unscheduled bar elements matches the number of unscheduled releases in fixtures.
  - **Test: Unscheduled bars are visually distinct (FR2.2)**
    - Assert unscheduled bars have a different CSS class or computed style (e.g., hatched pattern, muted opacity) compared to scheduled bars.
  - **Test: Unscheduled bars are ordered alphabetically (FR2.3)**
    - Assert the bar labels appear in alphabetical order top-to-bottom.
  - **Test: Clicking an unscheduled bar opens ReleaseDetailModal (FR2.4)**
    - Click an unscheduled bar.
    - Assert the `ReleaseDetailModal` appears with the correct release name.
  - **Test: Summary badge appears on unscheduled bars (FR2.5)**
    - Assert that an unscheduled release with assigned artifacts shows the idea/defect count badge.

### Acceptance criteria

- [ ] All unscheduled releases render as individual rows.
- [ ] Visual distinction from scheduled bars is verified.
- [ ] Alphabetical ordering is confirmed.
- [ ] Click interaction opens the detail modal.
- [ ] Summary badges display correctly.

---

## Milestone 4 — Backlog Panel Tests

### Description

Verify the Backlog panel renders, filters, and interacts correctly.

### Files to change

- `tests/roadmap-backlog-panel_test.go` or `tests/roadmap-backlog-panel.test.ts` (new)
  - **Test: Backlog panel renders below the Gantt chart (FR3.1)**
    - Assert a panel with header text matching "Backlog" exists below the Gantt chart.
  - **Test: Backlog lists artifacts without release, excluding release/sprint types (FR3.2)**
    - Assert backlog cards match the expected count from fixtures (artifacts with no release, minus any release/sprint type artifacts).
    - Assert no `release` or `sprint` type artifacts appear in the Backlog.
  - **Test: Backlog cards show title, type, status, lineage (FR3.3)**
    - For each card, assert the presence of title text, type badge, status badge, and lineage slug.
  - **Test: Backlog header shows count (FR3.5)**
    - Assert header text matches pattern `Backlog (N)` where N is the expected count.
  - **Test: Backlog defaults to collapsed (FR3.7)**
    - On initial page load, assert the Backlog card list is not visible (collapsed state).
    - Click the header to expand; assert cards become visible.
  - **Test: Empty backlog shows empty-state message (FR3.6)**
    - Load a fixture project where all artifacts have release assignments.
    - Assert the empty-state message is displayed.
  - **Test: Backlog filters work (OQ1)**
    - Expand the Backlog panel.
    - Select a type filter; assert only matching artifacts are shown.
    - Select a status filter; assert further filtering applies.
    - Clear filters; assert all backlog items reappear.

### Acceptance criteria

- [ ] Backlog panel renders in the correct position with correct content.
- [ ] Type exclusion (`release`, `sprint`) is verified.
- [ ] Card content (title, type, status, lineage) is verified.
- [ ] Count header is accurate.
- [ ] Collapse/expand behaviour works correctly.
- [ ] Empty state is tested.
- [ ] Filters narrow the displayed list correctly.

---

## Milestone 5 — Backlog Interaction and Reactive Update Tests

### Description

Verify clicking a Backlog card navigates to the artifact editor, and that assigning/clearing a release field causes reactive updates via WebSocket.

### Files to change

- `tests/roadmap-backlog-interaction_test.go` or `tests/roadmap-backlog-interaction.test.ts` (new)
  - **Test: Clicking a Backlog card navigates to the artifact editor (FR4.1)**
    - Click a backlog card.
    - Assert the browser navigates to `/p/:project/artifacts/:path`.
  - **Test: Assigning a release removes artifact from Backlog reactively (FR4.2)**
    - Open an artifact from the Backlog.
    - Set the `release` field to an existing release name.
    - Save the artifact.
    - Navigate back to the Roadmap.
    - Assert the artifact no longer appears in the Backlog panel (without page reload, via WebSocket event).
  - **Test: Clearing a release adds artifact to Backlog reactively (FR4.3)**
    - Open an artifact that has a release assignment.
    - Clear the `release` field.
    - Save the artifact.
    - Navigate back to the Roadmap.
    - Assert the artifact now appears in the Backlog panel.

### Acceptance criteria

- [ ] Card click navigates to the correct artifact editor route.
- [ ] Assigning a release causes the artifact to disappear from Backlog reactively.
- [ ] Clearing a release causes the artifact to appear in Backlog reactively.
- [ ] Updates occur without page reload, driven by WebSocket `artifact.indexed` events.

---

## Milestone 6 — Accessibility and Performance Tests

### Description

Verify keyboard navigation and performance under load.

### Files to change

- `tests/roadmap-backlog-a11y_test.go` or `tests/roadmap-backlog-a11y.test.ts` (new)
  - **Test: All interactive elements are keyboard-accessible (NFR3)**
    - Tab through the Gantt chart and Backlog panel.
    - Assert focus moves to unscheduled bars, Backlog collapse toggle, filter dropdowns, and each card in sequence.
    - Press Enter on a focused card; assert navigation occurs.
  - **Test: Collapse toggle has correct ARIA attributes**
    - Assert `aria-expanded` is `false` when collapsed and `true` when expanded.
    - Assert `aria-controls` references the Backlog card list element ID.
  - **Test: Backlog handles 500 artifacts without scroll jank (NFR2)**
    - Load a fixture project with 500 artifacts without release assignments.
    - Expand the Backlog panel.
    - Scroll through the list.
    - Assert render and scroll complete within acceptable thresholds (no frame drops > 100ms).

### Acceptance criteria

- [ ] Keyboard navigation reaches all interactive elements in logical tab order.
- [ ] ARIA attributes are correct on the collapse toggle.
- [ ] 500-item list renders and scrolls without perceptible jank.
