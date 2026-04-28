---
title: "Hide Done Items by Default — Test Plan"
type: plan-test
status: done
lineage: hide-done-items-by-default
parent: lifecycle/requirements/hide-done-items-by-default-2.md
---

# Hide Done Items by Default — Test Plan

Integration tests verifying that the "Show completed" toggle works correctly across all three artifact views. Tests interact with a running kaos-control instance via its API and a headless browser (or DOM-level assertions against the Vue components). See [[hide-done-items-by-default]] frontend plan for implementation details.

## Milestone 1: Test Data Setup Helpers

### Description

Create test fixture helpers that seed a project with artifacts spanning all statuses, including at least one each of `done`, `rejected`, and `abandoned`. This shared setup will be used by all subsequent test suites.

### Files to Change

- `tests/helpers/seed_artifacts.ts` (or equivalent test helper) — add a function that creates artifacts via the API with statuses: `draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `done`, `rejected`, `abandoned`.

### Acceptance Criteria

- [ ] A helper function exists that seeds at least 8 artifacts, one per status.
- [ ] The helper is importable by all test files in subsequent milestones.
- [ ] Seeded artifacts are retrievable via `GET /p/:project/artifacts` and include all statuses.

## Milestone 2: ArtifactListView Toggle Tests

### Description

Test that the artifact list view hides terminal-status artifacts by default and reveals them when the toggle is checked.

### Files to Change

- `tests/hide-done-items/artifact-list-toggle.test.ts` — integration tests for ArtifactListView.

### Acceptance Criteria

- [ ] **Test: default state hides terminal items** — on initial load, the artifact table does not contain rows with status `done`, `rejected`, or `abandoned`.
- [ ] **Test: toggle reveals terminal items** — after checking "Show completed", all artifacts (including terminal) appear in the table.
- [ ] **Test: toggle hides terminal items again** — unchecking the toggle re-hides terminal-status rows.
- [ ] **Test: count reflects visible set** — the displayed count in the header matches the number of visible rows, not the total from the API.
- [ ] **Test: toggle is keyboard accessible** — the toggle can be focused and activated via keyboard (Tab + Space).

## Milestone 3: KanbanBoardView Toggle Tests

### Description

Test that the kanban board hides the Done column by default and shows it when the toggle is checked.

### Files to Change

- `tests/hide-done-items/kanban-toggle.test.ts` — integration tests for KanbanBoardView.

### Acceptance Criteria

- [ ] **Test: default state hides Done column** — on initial load, no column named "Done" is rendered on the board.
- [ ] **Test: toggle reveals Done column** — after checking "Show completed", the "Done" column appears with its cards.
- [ ] **Test: column counts are accurate** — each visible column's count badge matches the number of cards displayed in it.
- [ ] **Test: other columns unaffected** — the Backlog, In-Progress, and Blocked columns show the same cards regardless of the toggle state.
- [ ] **Test: toggle resets on navigation** — navigate away and return; the Done column is hidden again.

## Milestone 4: GraphView Toggle Tests

### Description

Test that the graph view hides terminal-status nodes and their edges by default, and reveals them when the toggle is checked.

### Files to Change

- `tests/hide-done-items/graph-toggle.test.ts` — integration tests for GraphView.

### Acceptance Criteria

- [ ] **Test: default state hides terminal nodes** — on initial load, graph nodes with status `done`, `rejected`, or `abandoned` are not present.
- [ ] **Test: edges to hidden nodes are removed** — edges that would connect to a hidden terminal node are absent from the rendered graph.
- [ ] **Test: toggle reveals terminal nodes** — after checking "Show completed", terminal nodes and their edges reappear.
- [ ] **Test: node count reflects visible set** — the "N / M nodes" display in GraphFilters shows the filtered count excluding terminal nodes when hidden.
- [ ] **Test: toggle resets on navigation** — navigate away and return; terminal nodes are hidden again.

## Milestone 5: Cross-View Consistency Tests

### Description

Test that the toggle behaviour is consistent and independent across views — toggling in one view does not affect another, and each view resets on mount.

### Files to Change

- `tests/hide-done-items/cross-view-consistency.test.ts` — integration tests spanning multiple views.

### Acceptance Criteria

- [ ] **Test: toggling in list view does not affect kanban** — check "Show completed" on ArtifactListView, navigate to KanbanBoardView; the Done column remains hidden.
- [ ] **Test: toggling in kanban does not affect graph** — check "Show completed" on KanbanBoardView, navigate to GraphView; terminal nodes remain hidden.
- [ ] **Test: each view resets independently** — check the toggle on all three views, navigate away and back to each; all three are reset to unchecked.
- [ ] **Test: no extra API calls on toggle** — toggling "Show completed" does not trigger additional network requests to the backend; filtering is purely client-side on already-fetched data.
