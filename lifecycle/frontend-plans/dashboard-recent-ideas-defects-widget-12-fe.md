---
title: "Frontend Plan: Update Recent Ideas and Defects Widget Limit to 7"
type: plan-frontend
status: done
lineage: dashboard-recent-ideas-defects-widget
parent: lifecycle/requirements/dashboard-recent-ideas-defects-widget-10.md
---

# Frontend Plan: Update Recent Ideas and Defects Widget Limit to 7

This plan covers frontend-side work required by [[dashboard-recent-ideas-defects-widget-10]]. The `RecentIdeasDefectsWidget.vue` component already uses `limit: 7` — no source code change is needed. The frontend plan focuses on visual verification that the widget renders 7 items without layout issues.

---

## Milestone 1: Visual verification — 7 items at standard viewport

### Description

Verify that the widget renders 7 items without overflow, truncation, or scrollbar on viewports >= 1024 px. This is a manual verification step against a running dev server with a project that has at least 7 ideas/defects.

### Files to change

- None (verification only).

### Acceptance criteria

- The widget displays all 7 items in its list without introducing a vertical scrollbar within the widget container.
- No item title is truncated or clipped.
- The widget card does not overflow its parent grid cell in the `.dashboard-charts-top` row.
- Type badges and timestamps remain fully visible for all 7 items.

---

## Milestone 2: Visual verification — 7 items at narrow viewport

### Description

Verify that on viewports narrower than 1024 px (where the dashboard stacks to a single column), the widget with 7 items renders correctly in the stacked layout.

### Files to change

- None (verification only).

### Acceptance criteria

- At viewport width < 1024 px, the widget stacks vertically with the other chart widgets.
- All 7 items are visible without horizontal overflow.
- The widget does not push other stacked widgets off-screen or create unexpected whitespace.

---

## Milestone 3: Verify empty state and fewer-than-7 rendering

### Description

Confirm that existing edge-case rendering is unaffected: the empty-state message still appears when there are zero ideas/defects, and a project with fewer than 7 matching artifacts renders only the available items.

### Files to change

- None (verification only).

### Acceptance criteria

- With 0 ideas/defects, the widget displays "No recent ideas or defects".
- With 3 ideas/defects, the widget displays exactly 3 items (no blank rows or padding artefacts).

---

## Cross-references

- [[dashboard-recent-ideas-defects-widget-11-be]] — Backend plan (Go integration test updates + requirement artifact update).
- [[dashboard-recent-ideas-defects-widget-13-test]] — Test plan (Vitest assertion update + test-plan artifact update).
