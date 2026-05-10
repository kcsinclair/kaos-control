---
title: Update Recent Ideas and Defects Widget Limit from 6 to 7
type: requirement
status: blocked
lineage: dashboard-recent-ideas-defects-widget
parent: lifecycle/defects/dashboard-recent-ideas-defects-widget-9-defect.md
labels:
    - frontend
    - feature
    - vue
    - defect-fix
assignees:
    - role: product-owner
      who: agent
---

# Update Recent Ideas and Defects Widget Limit from 6 to 7

## Problem

The Recent Ideas and Defects dashboard widget was originally specified to display 6 items (see [[dashboard-recent-ideas-defects-widget-2]]). The implementation was subsequently updated to fetch 7 items (`limit: 7`), but the test suite and feature specification were not updated to match. This causes a test failure: the test asserts `limit: 6` while the code sends `limit: 7`.

The defect at [[dashboard-recent-ideas-defects-widget-9-defect]] confirms the intended behaviour is `limit: 7`. The code is correct; the test and specification are stale.

## Goals / Non-goals

### Goals

- Align the test suite with the current intended limit of 7 items.
- Update the original requirement specification ([[dashboard-recent-ideas-defects-widget-2]]) to reflect the new limit so that all documentation, code, and tests are consistent.
- Ensure the widget renders correctly with 7 items (no layout overflow or truncation).

### Non-goals

- No changes to the widget's visual design, layout position, or styling beyond accommodating the additional item.
- No changes to the backend API; the `limit` query parameter already supports arbitrary integer values.
- No changes to WebSocket live-update behaviour or other widget functionality.

## Detailed Requirements

### Functional

1. **Test update** -- The integration test at `tests/web/RecentIdeasDefectsWidget.test.ts` must assert `limit: 7` in the `listArtifacts` call expectation, replacing the current `limit: 6` assertion.
2. **Test-plan artifact update** -- The test-plan artifact at [[dashboard-recent-ideas-defects-widget-5-test]] must be updated to state that the widget fetches 7 items, not 6.
3. **Requirement artifact update** -- The requirement at [[dashboard-recent-ideas-defects-widget-2]] must be updated: all references to "6 most recent" items should read "7 most recent" in both the body text and acceptance criteria.
4. **Widget display count** -- The widget must display up to 7 items. If fewer than 7 ideas/defects exist, it displays all available items. The empty-state message ("No recent ideas or defects") remains unchanged when zero items exist.
5. **No code change to the widget** -- `RecentIdeasDefectsWidget.vue` already uses `limit: 7`; no source code change is required in the component itself.

### Non-functional

1. **Layout integrity** -- The widget must not overflow its container or introduce a scrollbar when displaying 7 items. Verify visually on viewports >= 1024 px and on narrow (< 1024 px stacked) layouts.
2. **Test reliability** -- The corrected test must pass deterministically. Run the test suite at least twice to confirm no flakiness.

## Acceptance Criteria

- [ ] `npx --prefix tests/web vitest run --root tests/web RecentIdeasDefectsWidget` passes with all assertions green.
- [ ] The test file asserts `limit: 7` in the `listArtifacts` spy expectation.
- [ ] The test-plan artifact ([[dashboard-recent-ideas-defects-widget-5-test]]) states the expected limit is 7.
- [ ] The requirement artifact ([[dashboard-recent-ideas-defects-widget-2]]) references 7 items, not 6, in all relevant sections.
- [ ] The widget renders 7 items without layout overflow on viewports >= 1024 px.
- [ ] The widget stacks correctly on narrow viewports (< 1024 px) with 7 items.
- [ ] No other existing tests regress as a result of this change.
- [ ] Related lineage: [[dashboard-recent-ideas-defects-widget]]

## Open Questions

None -- the defect clearly states the intended limit is 7 and identifies exactly which artifacts need updating.
