---
title: Tests — Roadmap Backlog Panel and Unscheduled Column
type: test
status: in-qa
lineage: roadmap-backlog-panel-and-unscheduled-column
parent: lifecycle/test-plans/roadmap-backlog-panel-and-unscheduled-column-5-test.md
---

# Tests — Roadmap Backlog Panel and Unscheduled Column

Integration and component tests covering the Unscheduled column on the Gantt chart and the Backlog panel, as specified in the test plan.

---

## Test Files

### `tests/integration/roadmap_backlog_panel_test.go`

Go integration tests (build tag `integration`) exercising the API layer that backs the Backlog panel.

Scenarios covered:

- **FR3.2 — Type field present for client filtering**: `GET /artifacts?release=__unassigned__` returns items with a populated `type` field, allowing the client to exclude `release` and `sprint` typed artifacts.
- **FR3.3 — Card fields present**: Each artifact in the unassigned list carries `title`, `type`, `status`, and `lineage` fields required to render a Backlog card.
- **FR3.5 — Count accuracy**: The `total` field in the API response matches the expected number of unassigned artifacts; the `items` array length is consistent.
- **FR3.6 — Empty state**: When all artifacts have release assignments, `GET /artifacts?release=__unassigned__` returns an empty `items` array.
- **OQ1 — Type filter**: `GET /artifacts?release=__unassigned__&type=<t>` returns only artifacts of the requested type.
- **OQ1 — Status filter**: `GET /artifacts?release=__unassigned__&status=<s>` returns only artifacts with the requested status.
- **FR3.1 — Roadmap graph Backlog node**: `GET /releases/graph` always includes a synthetic `release:backlog` node and attaches unassigned artifacts to it via `assigned` edges.

### `tests/integration/roadmap_backlog_interaction_test.go`

Go integration tests exercising reactive update behaviour.

Scenarios covered:

- **FR4.2 — Assign release removes from backlog**: Updating an artifact via `PUT /artifacts/*` to add a `release` field causes it to disappear from `GET /artifacts?release=__unassigned__` and appear under the named release filter.
- **FR4.3 — Clear release adds to backlog**: Updating an artifact to omit the `release` field causes it to appear in `GET /artifacts?release=__unassigned__` and disappear from the old release filter.
- **FR4.2/FR4.3 — WebSocket `artifact.indexed` event**: A `PUT /artifacts/*` update broadcasts an `artifact.indexed` WebSocket event, enabling the client to refresh the Backlog panel reactively without a page reload.
- **FR4.1 — Card path navigable**: Each artifact in the unassigned list carries a non-empty `path` field that resolves successfully via `GET /artifacts/:path`.

### `tests/web/GanttChart.unscheduled.test.ts`

Vitest + `@vue/test-utils` component tests for `GanttChart.vue`.

Scenarios covered:

- **FR1.1 — Unscheduled column renders**: `col-header--unscheduled` element exists with text "Unscheduled" when releases with no dates are present.
- **FR1.5 — Unscheduled column absent**: `col-header--unscheduled` does not exist when all releases have dates.
- **FR1.4 — Visual divider class**: `col-header--unscheduled` class is present (carries `border-left: 2px dashed` in scoped CSS).
- **OQ3 — Sticky class**: `col-header--unscheduled` carries `position: sticky` in component CSS.
- **FR1.1 — No date label**: Unscheduled column header text is exactly "Unscheduled" with no date digits.
- **FR2.1 — Each unscheduled release renders a bar**: `release-bar--unscheduled` element count matches unscheduled release count.
- **FR2.2 — Visual distinction**: Unscheduled bars have `release-bar--unscheduled` class; scheduled bars do not.
- **FR2.3 — Alphabetical ordering**: Unscheduled bar names appear top-to-bottom in alphabetical order.
- **FR2.4 — Click emits clickRelease**: Clicking an unscheduled bar emits `clickRelease` with the correct release id.
- **FR2.5 — Summary badge**: Badge appears on bars when `releaseDetails` contains non-zero `idea_count` or `defect_count`; absent when counts are zero.
- **Grid placeholder**: Scheduled rows include an `.unscheduled-cell` placeholder when the unscheduled column is visible.

### `tests/web/BacklogPanel.test.ts`

Vitest + `@vue/test-utils` component tests for `BacklogPanel.vue`.

Scenarios covered:

- **FR3.1 — Panel renders**: `section.backlog-panel[aria-label="Backlog"]` is present.
- **FR3.5 — Count header**: Header text matches `Backlog (N)` with the correct artifact count.
- **FR3.7 — Default collapsed**: Card list absent on initial mount; appears after toggle click.
- **FR3.7 — Toggle collapse/expand**: Two clicks collapse then expand the panel.
- **NFR3 — ARIA `aria-expanded`**: False when collapsed, true when expanded.
- **NFR3 — ARIA `aria-controls`**: Toggle's `aria-controls` matches the list element's `id`.
- **FR3.3 — Card content**: Each card shows title, type badge, status badge, and lineage.
- **FR4.1 — Card click emits openArtifact**: Click emits the artifact `path`.
- **FR3.6 — Empty state**: `backlog-empty` element appears when artifact list is empty or all filtered out.
- **OQ1 — Type filter**: Selecting a type narrows displayed cards to matching type.
- **OQ1 — Status filter**: Selecting a status narrows displayed cards to matching status.
- **OQ1 — Clear filter**: Clearing type selection restores all cards.
- **OQ1 — Combined filters**: Type and status filters apply with AND logic.
- **OQ1 — Filters hidden when collapsed**: Filter dropdowns absent in collapsed state.
- **NFR3 — Toggle keyboard-activatable**: Toggle button is a `<button>` element.
- **NFR3 — Card aria-label**: Each card has an `aria-label` containing the artifact title.
- **NFR2 — 500-item mount**: Panel mounts with 500 artifacts without throwing.
- **NFR2 — 500-item expand**: Expanding renders all 500 cards.

---

## Coverage Notes

- Visual layout (sticky scroll, pixel-level borders, frame-rate jank) cannot be verified without a real browser. The component tests assert the CSS classes that carry these rules; end-to-end browser tests are out of scope for this suite.
- The `artifact.indexed` WebSocket test uses the project hub directly (`env.proj.Hub`) to avoid network-level WebSocket connections in CI.
- Performance threshold (NFR2: no frame drop > 100 ms) is approximated by asserting that mounting and expanding a 500-item panel does not throw; real frame-rate measurement requires Playwright or similar.
