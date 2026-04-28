---
title: Hide Done Items by Default
type: requirement
status: draft
lineage: hide-done-items-by-default
created: "2026-04-28"
parent: lifecycle/ideas/hide-done-items-by-default.md
---

# Hide Done Items by Default

## Problem

All artifact views currently display every artifact regardless of status, including those that are `done`, `rejected`, or `abandoned`. As the number of completed artifacts grows, active work gets buried in noise. Users spend time scanning past items that are no longer actionable, reducing the effectiveness of every screen.

## Goals / Non-goals

### Goals

1. Reduce visual noise by hiding terminal-status artifacts (`done`, `rejected`, `abandoned`) by default on every screen that lists or visualises artifacts.
2. Provide a simple, discoverable toggle so users can reveal hidden items when they need to review completed work.
3. Apply the behaviour consistently across all artifact-rendering surfaces.

### Non-goals

- Persisting the toggle state across sessions or page navigations — resetting to "hidden" on each page load is intentional.
- Providing per-status granularity (e.g. hide `done` but show `rejected`) — a single toggle covers all terminal statuses.
- Changing the backend API response — filtering is purely a frontend concern.

## Detailed Requirements

### Functional

1. **Terminal statuses** — the statuses treated as terminal (hidden by default) are: `done`, `rejected`, `abandoned`.
2. **Toggle control** — each view that renders artifacts must include a checkbox (or equivalent toggle) in its page header area, labelled "Show completed" (or similar concise label). The toggle must be unchecked by default on every page load.
3. **Affected views** — the filter and toggle must appear on:
   - `ArtifactListView` (table/list)
   - `KanbanBoardView` (kanban board)
   - `GraphView` (2D and 3D graph)
4. **Filter behaviour** — when the toggle is unchecked, artifacts whose `status` matches any terminal status must be excluded from rendering. When checked, all artifacts are shown.
5. **Kanban specifics** — on the kanban board, when the toggle is unchecked the "Done" column (which maps to `done`, `abandoned`, `rejected` per `lifecycle/config.yaml`) should either be hidden entirely or rendered empty. When the toggle is checked, the column and its cards must appear normally.
6. **Graph specifics** — on the graph view, hidden artifacts must also be removed as nodes (and their edges pruned). Revealing them via the toggle re-adds nodes and edges.
7. **Counts / badges** — if any view displays artifact counts (column counts on kanban, total counts in list headers), the counts must reflect the currently visible set, not the full unfiltered set.
8. **No backend changes** — the API continues to return all artifacts. Filtering is performed client-side after data is fetched.

### Non-functional

1. **Performance** — filtering must not introduce perceptible latency; it should operate on the already-fetched dataset in memory.
2. **Accessibility** — the toggle must be keyboard-focusable and have an accessible label (e.g. `aria-label` or associated `<label>` element).
3. **Responsiveness** — the toggle must remain usable at all supported viewport widths.

## Acceptance Criteria

- [ ] On `ArtifactListView`, artifacts with status `done`, `rejected`, or `abandoned` are not shown on initial page load.
- [ ] On `KanbanBoardView`, the Done column (and its cards) is hidden or empty on initial page load.
- [ ] On `GraphView`, nodes with terminal status are absent from the graph on initial page load.
- [ ] Each of the three views displays a "Show completed" toggle in its header area.
- [ ] Checking the toggle immediately reveals all previously hidden artifacts without a page reload or additional API call.
- [ ] Unchecking the toggle re-hides terminal-status artifacts.
- [ ] Navigating away from a view and returning resets the toggle to unchecked (hidden).
- [ ] Artifact counts displayed in any view header or column header reflect only the visible (filtered) set.
- [ ] The toggle is keyboard-accessible and has a visible focus indicator.
- [ ] No backend API changes are required; filtering is client-side only.

## Open Questions

1. **Toggle label** — should the label read "Show completed", "Show done", or something else? The idea suggests "Show done" but this requirement uses "Show completed" to encompass `rejected` and `abandoned` as well.

> Show completed is great.

2. **Graph edge handling** — when a hidden node is an intermediate ancestor in a lineage chain, should edges be re-routed to connect visible ancestors/descendants, or should disconnected subgraphs simply appear? (Recommend: remove node and its edges; accept potential disconnection.)

> remove node and its edges.
