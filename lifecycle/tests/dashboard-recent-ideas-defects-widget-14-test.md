---
title: "Tests: Update Recent Ideas and Defects Widget Limit to 7"
type: test
status: draft
lineage: dashboard-recent-ideas-defects-widget
parent: lifecycle/test-plans/dashboard-recent-ideas-defects-widget-13-test.md
---

# Tests: Update Recent Ideas and Defects Widget Limit to 7

Companion artifact for [[dashboard-recent-ideas-defects-widget-13-test]]. Documents
the test-side changes made to align all test code and artifacts with the `limit: 7`
widget behaviour introduced in [[dashboard-recent-ideas-defects-widget-10]].

## Test files changed

### `tests/web/RecentIdeasDefectsWidget.test.ts`

Updated the existing Vitest component test that asserts the widget calls
`listArtifacts` with the correct parameters:

- **Test name** changed from `'calls listArtifacts with type=idea,defect, sort=created:desc, limit=6'`
  to `'calls listArtifacts with type=idea,defect, sort=created:desc, limit=7'`.
- **Assertion** `expect.objectContaining({ limit: 6 })` updated to `limit: 7`.

All 20 tests in the suite pass (verified twice for flakiness).

### `tests/integration/api_artifacts_widget_query_test.go`

Updated the Go integration tests that exercise the combined widget query
`?type=idea,defect&sort=created:desc&limit=7`:

- URL strings changed from `limit=6` to `limit=7` across all six test functions.
- Bounds checks updated from `> 6` / `!= 6` to `> 7` / `!= 7` in
  `TestWidgetQuery_LimitApplied`.
- Comments updated to reference the correct limit of 7.

## Artifact updates

### `lifecycle/test-plans/dashboard-recent-ideas-defects-widget-5-test.md`

Milestone 3 updated: query string, test case descriptions, and acceptance
criteria now read `limit=7` instead of `limit=6` (scenarios 1, 4, 5).

### `lifecycle/tests/dashboard-recent-ideas-defects-widget-6-test.md`

Milestone 3 section updated: query under test and the scenario table descriptions
(`TestWidgetQuery_LimitApplied`, `TestWidgetQuery_TotalIsFullMatchCount`) now
reference `limit=7`.

## Scenarios covered

| Milestone | Scenario | Outcome |
|---|---|---|
| 1 — Vitest assertion | `listArtifacts` called with `limit: 7` | passes |
| 2 — test-plan artifact | All `limit=6` refs → `limit=7` | artifact consistent |
| 3 — lifecycle test artifact | All `limit=6` refs → `limit=7` | artifact consistent |
| 4 — Go integration: `TestWidgetQuery_LimitApplied` | Returns exactly 7 items | passes |
| 4 — Go integration: `TestWidgetQuery_OnlyIdeasAndDefects` | No requirements in results | passes |
| 4 — Go integration: `TestWidgetQuery_SortedByCreatedDesc` | Items ordered most-recent first | passes |
| 4 — Go integration: `TestWidgetQuery_TotalIsFullMatchCount` | `total` = 15 (not capped) | passes |
| 4 — Go integration: `TestWidgetQuery_FewerThanLimit` | 2 matching → 2 items, total = 2 | passes |
| 4 — Go integration: `TestWidgetQuery_ZeroResults` | 0 matching → empty, total = 0 | passes |
