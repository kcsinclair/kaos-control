---
title: "Tests: Recent Ideas and Defects Dashboard Widget"
type: test
status: approved
lineage: dashboard-recent-ideas-defects-widget
parent: lifecycle/test-plans/dashboard-recent-ideas-defects-widget-5-test.md
created: "2026-05-09T00:00:00+10:00"
---

# Tests: Recent Ideas and Defects Dashboard Widget

Companion artifact documenting the test suite built for the
[[dashboard-recent-ideas-defects-widget]] feature.

## Test Files

### Integration tests (Go)

`tests/integration/api_artifacts_filter_test.go`

Go integration tests covering multi-value `type` filter on
`GET /api/p/{project}/artifacts`. Starts a full HTTP server with a seeded
lifecycle project and drives the REST API directly.

`tests/integration/api_artifacts_sort_test.go`

Go integration tests covering the `sort` query parameter — valid sort
columns (`created:desc`, `created:asc`, `title:asc`), default order when
sort is omitted, and safety under invalid/malformed/injection values.

`tests/integration/api_artifacts_widget_query_test.go`

Go integration tests for the exact combined query the widget uses:
`?type=idea,defect&sort=created:desc&limit=7`. Covers limit application,
type exclusion, descending sort, accurate `total`, and edge cases
(fewer-than-limit and zero results).

### Web component tests (Vitest / Vue Test Utils)

`tests/web/RecentIdeasDefectsWidget.test.ts`

Component-level tests for `RecentIdeasDefectsWidget.vue` running under
happy-dom. Covers all six milestone-4 test cases.

`tests/web/DashboardView.test.ts` (extended)

Dashboard layout regression tests appended to the existing Vitest suite.
Covers TC1–TC4 of milestone 5: widget count, slot assignment,
`.dashboard-charts-top` and `.dashboard-charts-bottom` container placement.

## Scenarios Covered

### Milestone 1 — Backend API: multi-value `type` filter

| Test | Scenario |
|---|---|
| `TestArtifactTypeFilter_MultiValue`   | `?type=idea,defect` returns only ideas and defects (OR semantics) |
| `TestArtifactTypeFilter_SingleValue`  | `?type=idea` returns only ideas (backward compatibility) |
| `TestArtifactTypeFilter_ThreeTypes`   | `?type=idea,defect,ticket` returns all three types |
| `TestArtifactTypeFilter_Nonexistent`  | `?type=nonexistent` returns empty list, no error |
| `TestArtifactTypeFilter_Omitted`      | No `type` parameter returns all artifact types |

Seed: 3 ideas, 2 defects, 2 tickets per test (via `testFilterSeeds()`).

### Milestone 2 — Backend API: `sort` parameter

| Test | Scenario |
|---|---|
| `TestArtifactSort_CreatedDesc`    | `?sort=created:desc` — most recent created first |
| `TestArtifactSort_CreatedAsc`     | `?sort=created:asc` — oldest created first |
| `TestArtifactSort_TitleAsc`       | `?sort=title:asc` — alphabetical by title |
| `TestArtifactSort_Default`        | No `sort` — default order (lineage, idx, path) |
| `TestArtifactSort_InvalidColumn`  | `?sort=badcolumn:desc` — 200, falls back to default |
| `TestArtifactSort_MalformedValue` | `?sort=created` (no direction) — 200, falls back to default |
| `TestArtifactSort_SQLInjection`   | SQL injection in sort column — 200, table intact |

Sort tests use `makeArtifactDated()` and `seedAndIndexDatedArtifacts()` to
seed artifacts with explicit RFC3339 `created:` frontmatter timestamps spaced
1 hour apart, ensuring deterministic sort order.

### Milestone 3 — Combined widget query end-to-end

Seed: 10 ideas + 5 defects (with known created timestamps) + 3 requirements.
Query under test: `?type=idea,defect&sort=created:desc&limit=7`.

| Test | Scenario |
|---|---|
| `TestWidgetQuery_LimitApplied`         | Returns exactly 7 items |
| `TestWidgetQuery_OnlyIdeasAndDefects`  | No requirements in results |
| `TestWidgetQuery_SortedByCreatedDesc`  | Items ordered most-recent first |
| `TestWidgetQuery_TotalIsFullMatchCount`| `total` = 15 (not capped at 7) |
| `TestWidgetQuery_FewerThanLimit`       | 2 matching → returns 2, total = 2 |
| `TestWidgetQuery_ZeroResults`          | 0 matching → empty items, total = 0 |

### Milestone 4 — Frontend widget component

All tests in `tests/web/RecentIdeasDefectsWidget.test.ts`. The
`listArtifacts` function is mocked at module level; `useWebSocket` is mocked
to capture the registered callback.

| Test group | Scenarios |
|---|---|
| TC1 Renders items       | Correct item count, titles present, relative timestamps non-empty |
| TC2 Empty state         | "No recent ideas or defects" message; item list absent; error → empty |
| TC3 Navigation          | `<a>` href resolves to `/p/{project}/artifacts/{path}`; unique per item |
| TC4 Type badges         | `idea` badge → `.type-badge--idea`; `defect` → `.type-badge--defect`; mixed list |
| TC5 Live update         | `useWebSocket` called with `artifact.indexed`; callback triggers re-fetch with same params |
| TC6 Accessibility       | Item links are `<a>` (focusable); badges have `aria-label` containing type name |

### Milestone 5 — Dashboard layout regression

Tests appended to `tests/web/DashboardView.test.ts` in the
`"DashboardGrid — Milestone 5: layout with recent-ideas-defects widget"` suite.

| Test | Scenario |
|---|---|
| TC1 All 6 widgets       | `summary-counts`, `status-distribution`, `stages-distribution`, `recent-ideas-defects`, `velocity-chart`, `activity-feed` all render |
| TC2 Slot assignment     | `summary-counts` in `.dashboard-summary`; `activity-feed` in `.dashboard-panels` |
| TC3 Top row             | First 3 chart widgets (status, stages, recent-ideas) inside `.dashboard-charts-top` |
| TC4 Bottom row          | `velocity-chart` inside `.dashboard-charts-bottom`, not inside `.dashboard-charts-top` |
| TC5 Responsive collapse | Deferred to Playwright E2E (happy-dom does not evaluate CSS `@media` rules) |

## Deferred to Playwright E2E

- M5-TC5: single-column layout at viewport < 1024 px (CSS `@media` evaluation)
- Widget click-through navigation with real browser routing (back-button behaviour)
