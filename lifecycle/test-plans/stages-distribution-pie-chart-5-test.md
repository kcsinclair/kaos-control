---
title: "Test Plan: Stages Distribution Pie Chart"
type: plan-test
status: approved
lineage: stages-distribution-pie-chart
parent: lifecycle/requirements/stages-distribution-pie-chart-2.md
created: "2026-05-09"
---

# Test Plan: Stages Distribution Pie Chart

## Overview

Integration and end-to-end tests for the [[stages-distribution-pie-chart]] feature, covering the backend API endpoint and the frontend widget behaviour. Tests follow the existing patterns in `tests/` and use the same test infrastructure as the dashboard test suite.

## Milestone 1: Backend API tests — stage-distribution endpoint

**Description:** Test the `GET /api/p/:project/dashboard/stage-distribution` endpoint for correct response shape, filtering, and edge cases.

**Files to change:**

- `tests/api/dashboard_test.go` (or equivalent existing dashboard test file) — add test cases for the stage-distribution endpoint.

**Test cases:**

1. **Happy path** — Create artifacts across multiple stages (ideas, requirements, backend-plans). Call the endpoint and verify the response contains the correct stage names and counts.
2. **Empty project** — Call the endpoint on a project with no artifacts. Verify the response is `{"distribution": []}` (empty array, not null).
3. **TrackedTypes filtering** — Configure `Dashboard.TrackedTypes` to a subset of types. Create artifacts of tracked and non-tracked types. Verify only tracked-type artifacts are counted.
4. **Done/abandoned exclusion** — Create artifacts with `done` and `abandoned` statuses. Verify they are excluded from the stage counts.
5. **Mixed statuses** — Create artifacts with various statuses (draft, in-development, done). Verify only non-done/non-abandoned artifacts are counted per stage.
6. **Single stage** — All artifacts in one stage. Verify response has exactly one entry.
7. **Alphabetical ordering** — Verify the distribution array is sorted by stage name.

**Acceptance criteria:**

- [ ] All 7 test cases pass.
- [ ] The response shape matches `{"distribution": [{"stage": string, "count": number}, ...]}`.
- [ ] The endpoint returns HTTP 200 for valid requests.
- [ ] The endpoint returns an empty array (not null/undefined) for empty projects.

## Milestone 2: Backend unit tests — StageDistribution index method

**Description:** Unit test the `StageDistribution` method on the index directly, separate from HTTP handling.

**Files to change:**

- `internal/index/index_test.go` (or existing index test file) — add test cases.

**Test cases:**

1. **Correct grouping** — Insert artifacts into multiple stages, call `StageDistribution`, verify counts match.
2. **Empty database** — Call on empty index, verify non-nil empty slice returned.
3. **TrackedTypes default** — Call with nil/empty tracked types, verify fallback to `["ticket"]`.
4. **Status exclusion** — Insert done/abandoned artifacts, verify they are excluded.

**Acceptance criteria:**

- [ ] All 4 test cases pass.
- [ ] Method returns `[]StageCount{}` (not nil) for empty results.
- [ ] Method correctly reuses `trackedTypesClause`.

## Milestone 3: Frontend widget tests — StagesDistributionWidget

**Description:** Test the widget component's rendering, data fetching, click-through navigation, and accessibility attributes.

**Files to change:**

- `tests/web/StagesDistributionWidget.test.ts` — new test file (following the pattern of existing widget tests in `tests/web/`).

**Test cases:**

1. **Renders chart with data** — Mock the API to return a multi-stage distribution. Verify the chart container is rendered (not the empty state).
2. **Empty state** — Mock the API to return `{"distribution": []}`. Verify "No artifacts yet" placeholder is displayed.
3. **All-zero counts** — Mock the API to return stages with all counts zero. Verify "No artifacts yet" is shown.
4. **Click-through navigation** — Simulate a click event on the chart. Verify `router.push` is called with the correct route (name: `artifacts`, query: `{ stage: <clicked-stage> }`).
5. **Accessibility** — Verify the chart container has `role="img"` and a descriptive `aria-label` that includes stage names and counts.
6. **Project prop change** — Change the `project` prop. Verify data is re-fetched.
7. **Error handling** — Mock the API to return an error. Verify the empty state is shown (graceful degradation).

**Acceptance criteria:**

- [ ] All 7 test cases pass.
- [ ] Tests use the same mocking/test utilities as existing widget tests.
- [ ] No test depends on real API calls or network.

## Milestone 4: Widget registration tests

**Description:** Verify the widget is correctly registered in the widget registry with the expected slot and order.

**Files to change:**

- `tests/web/widgetRegistry.test.ts` — add test case for `stages-distribution` widget.

**Test cases:**

1. **Registration** — Verify that `stages-distribution` is registered with `slot: 'chart'` and `order: 1`.
2. **Ordering** — Verify the chart-slot widgets are ordered: `status-distribution` (0), `stages-distribution` (1), `velocity-chart` (2).
3. **No duplicates** — Verify calling registration twice does not create duplicate entries (existing test pattern).

**Acceptance criteria:**

- [ ] All 3 test cases pass.
- [ ] Existing widget registration tests continue to pass (status-distribution order may need updating from 0 to 0 — unchanged — and velocity-chart from 1 to 2).

## Milestone 5: End-to-end dashboard integration

**Description:** Verify the full flow from dashboard load through click-through navigation to the filtered artifacts list.

**Files to change:**

- `tests/web/DashboardView.test.ts` — add integration test cases.

**Test cases:**

1. **Widget visible on dashboard** — Load the dashboard view. Verify the "Stages Distribution" widget title is present.
2. **Click-through produces correct URL** — Click a stage slice. Verify the URL changes to `/p/:project/artifacts?stage=<stage>`.
3. **Filtered list matches** — After click-through, verify the artifacts list shows only artifacts from the selected stage.
4. **Back navigation** — After click-through, press back. Verify the dashboard is restored.
5. **Bookmarkable URL** — Navigate directly to `/p/:project/artifacts?stage=requirements`. Verify the correct filter is applied.
6. **Existing widgets unaffected** — Verify Status Distribution, Velocity Chart, Summary Counts, and Activity Feed still render correctly alongside the new widget.

**Acceptance criteria:**

- [ ] All 6 test cases pass.
- [ ] The dashboard renders without console errors.
- [ ] No regressions in existing dashboard widget tests.
