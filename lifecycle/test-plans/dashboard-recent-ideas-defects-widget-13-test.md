---
title: "Test Plan: Update Recent Ideas and Defects Widget Limit to 7"
type: plan-test
status: in-development
lineage: dashboard-recent-ideas-defects-widget
parent: lifecycle/requirements/dashboard-recent-ideas-defects-widget-10.md
---

# Test Plan: Update Recent Ideas and Defects Widget Limit to 7

This plan covers test-side changes required by [[dashboard-recent-ideas-defects-widget-10]]. The widget code already sends `limit: 7`, but the Vitest component test asserts `limit: 6`. The test-plan artifact [[dashboard-recent-ideas-defects-widget-5-test]] also references `limit=6` and must be updated for consistency.

---

## Milestone 1: Update Vitest assertion — listArtifacts limit parameter

### Description

The test at `tests/web/RecentIdeasDefectsWidget.test.ts` contains an assertion that the widget calls `listArtifacts` with `limit: 6`. This must be changed to `limit: 7` to match the actual widget behaviour.

### Files to change

- `tests/web/RecentIdeasDefectsWidget.test.ts`
  - **Test case** `'calls listArtifacts with type=idea,defect, sort=created:desc, limit=6'` (line ~511):
    - Update the test name from `limit=6` to `limit=7`.
    - Change the `expect.objectContaining({ ... limit: 6 })` assertion to `limit: 7`.

### Acceptance criteria

- The test name reads `limit=7`.
- The assertion checks `limit: 7`.
- `npx --prefix tests/web vitest run --root tests/web RecentIdeasDefectsWidget` passes with all assertions green.

---

## Milestone 2: Update test-plan artifact to reflect limit of 7

### Description

The test-plan artifact [[dashboard-recent-ideas-defects-widget-5-test]] documents the test coverage for this feature. Milestone 3 of that artifact references the combined widget query as `?type=idea,defect&sort=created:desc&limit=6`. This must be updated to `limit=7`.

### Files to change

- `lifecycle/test-plans/dashboard-recent-ideas-defects-widget-5-test.md`
  - **Milestone 3 heading/description**: Change `limit=6` to `limit=7` in the query string `?type=idea,defect&sort=created:desc&limit=6`.
  - **Milestone 3 test case 1**: "Combined query returns at most 6 items" → Change `6` to `7`.
  - **Milestone 3 test case 4**: "`total` in response equals the full count of matching ideas + defects (15), not capped at 6" → Change `6` to `7`.
  - **Milestone 3 test case 5**: "When fewer than 6 ideas+defects exist" → Change `6` to `7`.
  - **Milestone 3 acceptance criteria**: "type filter, sort, and limit together" — no number change needed, but verify the bullet about `limit` is consistent.

### Acceptance criteria

- All references to the item limit in [[dashboard-recent-ideas-defects-widget-5-test]] read `7` instead of `6`.
- No unrelated content in the artifact is modified.
- The artifact remains valid markdown with correct frontmatter.

---

## Milestone 3: Update test lifecycle artifact to reflect limit of 7

### Description

The test artifact [[dashboard-recent-ideas-defects-widget-6-test]] documents what the test code covers. Its Milestone 3 section references `limit=6` in the query under test. This must be updated for consistency.

### Files to change

- `lifecycle/tests/dashboard-recent-ideas-defects-widget-6-test.md`
  - **Milestone 3 section**: Change the query string from `limit=6` to `limit=7` in the description.
  - **Test scenario table**: Update any references to `limit=6` (e.g. `TestWidgetQuery_LimitApplied` description, `TestWidgetQuery_TotalIsFullMatchCount` description, `TestWidgetQuery_FewerThanLimit` description).

### Acceptance criteria

- All references to the item limit in [[dashboard-recent-ideas-defects-widget-6-test]] read `7` instead of `6`.
- No unrelated content in the artifact is modified.

---

## Milestone 4: Run full test suite and confirm no regressions

### Description

Run both the Vitest web tests and Go integration tests to confirm no regressions.

### Files to change

- None (verification only).

### Acceptance criteria

- `npx --prefix tests/web vitest run --root tests/web RecentIdeasDefectsWidget` passes with all assertions green.
- Run the test suite at least twice to confirm no flakiness (per requirement non-functional §2).
- `go test ./tests/integration/ -run TestWidgetQuery` passes with all assertions green.
- No other existing tests regress.

---

## Cross-references

- [[dashboard-recent-ideas-defects-widget-11-be]] — Backend plan (Go integration test updates + requirement artifact update).
- [[dashboard-recent-ideas-defects-widget-12-fe]] — Frontend plan (visual verification of 7-item rendering).
