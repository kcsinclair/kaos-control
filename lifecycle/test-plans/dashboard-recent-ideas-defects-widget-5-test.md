---
title: "Test Plan: Recent Ideas and Defects Dashboard Widget"
type: plan-test
status: approved
lineage: dashboard-recent-ideas-defects-widget
parent: lifecycle/requirements/dashboard-recent-ideas-defects-widget-2.md
created: "2026-05-09"
---

# Test Plan: Recent Ideas and Defects Dashboard Widget

This plan covers integration tests for both the backend API extensions and the frontend widget. Tests target the changes described in [[dashboard-recent-ideas-defects-widget]] backend plan (-3-be) and frontend plan (-4-fe).

Test code lives in `tests/` (integration test directory). Test lifecycle artifacts live in `lifecycle/tests/`.

---

## Milestone 1: Backend API — multi-value `type` filter tests

### Description

Test that the `type` query parameter on `GET /api/p/{project}/artifacts` correctly handles comma-separated values.

### Files to change

- `tests/api_artifacts_filter_test.go` (new or extend existing)
  - **Setup**: Seed the test project's `lifecycle/` directory with at least 3 ideas, 2 defects, and 2 requirements artifacts (varying types).
  - **Test cases**:
    1. `?type=idea,defect` — returns only artifacts of type `idea` or `defect`, excludes other types.
    2. `?type=idea` — returns only ideas (single-value backward compatibility).
    3. `?type=idea,defect,requirement` — returns all three types.
    4. `?type=nonexistent` — returns empty list, no error.
    5. Empty/omitted `type` — returns all artifact types (existing behaviour preserved).
  - **Assertions**: Check response `items` array contains only artifacts with matching types; check `total` reflects full matching count.

### Acceptance criteria

- All five test cases pass.
- Multi-value filter returns the union of all specified types (OR semantics).
- No regression in single-value or omitted `type` behaviour.

---

## Milestone 2: Backend API — `sort` parameter tests

### Description

Test the new `sort` query parameter for correctness and safety.

### Files to change

- `tests/api_artifacts_sort_test.go` (new or extend existing)
  - **Setup**: Seed artifacts with distinct, known `created` timestamps (e.g. stagger by 1 hour).
  - **Test cases**:
    1. `?sort=created:desc` — first item has the most recent `created` date.
    2. `?sort=created:asc` — first item has the oldest `created` date.
    3. `?sort=title:asc` — items are sorted alphabetically by title.
    4. No `sort` parameter — items are ordered by `lineage, idx, path` (default, existing behaviour).
    5. Invalid sort value (e.g. `?sort=badcolumn:desc`) — falls back to default order, no error.
    6. Malformed sort value (e.g. `?sort=created`) — falls back to default order, no error.
    7. SQL injection attempt (e.g. `?sort=created;DROP TABLE artifacts--:desc`) — falls back to default order, no error, table unaffected.

### Acceptance criteria

- Valid sort columns produce correctly ordered results.
- Invalid/malformed/malicious sort values silently fall back to default order without errors.
- Default order is unchanged when `sort` is omitted.

---

## Milestone 3: Combined widget query end-to-end test

### Description

Test the exact query the frontend widget will use: `?type=idea,defect&sort=created:desc&limit=6`.

### Files to change

- `tests/api_artifacts_widget_query_test.go` (new or extend existing)
  - **Setup**: Seed 10 ideas and 5 defects with known creation dates, plus 3 requirements.
  - **Test cases**:
    1. Combined query returns at most 6 items.
    2. All returned items are type `idea` or `defect` (no requirements).
    3. Items are sorted by `created` descending (most recent first).
    4. `total` in response equals the full count of matching ideas + defects (15), not capped at 6.
    5. When fewer than 6 ideas+defects exist (e.g. 2 total), returns only 2 items with correct total.
    6. When zero ideas+defects exist, returns empty `items` array and `total: 0`.

### Acceptance criteria

- The combined query correctly applies type filter, sort, and limit together.
- Pagination metadata (`total`) is accurate regardless of `limit`.
- Edge cases (fewer than limit, zero results) handled correctly.

---

## Milestone 4: Frontend widget integration tests

### Description

Test the `RecentIdeasDefectsWidget` component's rendering, navigation, and live-update behaviour.

### Files to change

- `tests/dashboard_widget_test.go` (new or extend existing) — if testing via the HTTP-served SPA with a headless browser or similar E2E tool.
- Alternatively, `web/src/components/dashboard/widgets/__tests__/RecentIdeasDefectsWidget.spec.ts` — if using Vitest + Vue Test Utils for component-level tests.
  - **Test cases**:
    1. **Renders items**: Mount with mocked API returning 4 items → widget displays 4 entries with correct titles, type badges, and timestamps.
    2. **Empty state**: Mount with mocked API returning 0 items → widget displays "No recent ideas or defects" message.
    3. **Navigation**: Click an item → router navigates to `/p/{project}/artifacts/{path}`.
    4. **Type badges**: Each item renders the correct badge text (`idea` or `defect`) with the correct CSS class.
    5. **Live update**: Simulate `artifact.indexed` WebSocket event → widget refetches data (verify fetch called twice: once on mount, once after event).
    6. **Accessibility**: All items are focusable via keyboard; type badges have `aria-label` attributes.

### Acceptance criteria

- Widget correctly renders items from the API response.
- Empty state displays appropriate message.
- Click navigation works.
- WebSocket event triggers data refresh.
- Keyboard focus and ARIA labels are present.

---

## Milestone 5: Dashboard layout regression tests

### Description

Verify the layout restructure does not break existing widgets and the new layout renders correctly.

### Files to change

- `tests/dashboard_layout_test.go` or `web/src/components/dashboard/__tests__/DashboardGrid.spec.ts`
  - **Test cases**:
    1. **Widget count**: Dashboard renders all 6 registered widgets (summary-counts, status-distribution, stages-distribution, recent-ideas-defects, velocity-chart, activity-feed).
    2. **Slot assignment**: summary-counts is in `summary` slot; status-distribution, stages-distribution, recent-ideas-defects, velocity-chart are in `chart` slot; activity-feed is in `panel` slot.
    3. **Top row structure**: The first 3 chart-slot widgets are rendered inside the `.dashboard-charts-top` container.
    4. **Velocity placement**: velocity-chart renders inside `.dashboard-charts-bottom`.
    5. **Responsive collapse**: At viewport width < 1024px, `.dashboard-charts-top` has a single-column layout (test via computed styles or class assertions).

### Acceptance criteria

- All registered widgets render on the dashboard.
- Layout containers exist and contain the expected widgets.
- No widget is missing or duplicated after the restructure.

---

## Milestone 6: Test lifecycle artifact

### Description

Create the corresponding test artifact in `lifecycle/tests/` documenting what the test code covers.

### Files to change

- `lifecycle/tests/dashboard-recent-ideas-defects-widget-6-test.md` (new file)
  - Frontmatter: `type: test`, `status: draft`, `lineage: dashboard-recent-ideas-defects-widget`, `parent: lifecycle/test-plans/dashboard-recent-ideas-defects-widget-5-test.md`.
  - Body: summary of test coverage from milestones 1–5, mapping test files to the requirements they verify.

### Acceptance criteria

- Test artifact exists and has correct frontmatter.
- It accurately describes the test coverage.
